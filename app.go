package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"Straggle/internal/i18n"
	"Straggle/internal/kill"
	"Straggle/internal/model"
	"Straggle/internal/scan"
	"Straggle/internal/settings"
	"Straggle/internal/theme"
	"Straggle/internal/tray"
)

const (
	appName = "Straggle"
	version = "0.1.1" // shown in Settings; wails.json stamps the executable with the same number

	repoURL    = "https://github.com/mclishang/Straggle"
	maxEntries = 300 // render cap; above it the UI says the list is truncated
)

// App is the object bound to the frontend: the UI calls window.go.main.App.*,
// and scan results and kill outcomes arrive as events (snapshot:update,
// kill:result, theme:changed, navigate, orphans:new).
type App struct {
	ctx     context.Context
	store   *settings.Store
	scanner scan.Scanner
	manager *kill.Manager
	tray    *tray.Controller

	scanMu sync.Mutex
	snapMu sync.RWMutex
	snap   model.Snapshot

	orphanMu    sync.Mutex
	lastOrphans map[string]bool

	themeMu sync.RWMutex
	dark    bool

	stop       chan struct{}
	stopOnce   sync.Once
	wake       chan struct{}
	quitting   atomic.Bool
	includeUDP atomic.Bool
}

// NewApp wires the backend: settings store, scanner, kill manager and tray.
func NewApp(icon []byte) *App {
	a := &App{
		store:   settings.New(),
		scanner: scan.NewScanner(),
		stop:    make(chan struct{}),
		wake:    make(chan struct{}, 1),
	}
	a.manager = kill.NewManager(kill.NewPlatformKiller(), a.onKillDone)
	a.tray = tray.New(tray.Actions{
		Show:    a.showWindow,
		Hide:    a.hideWindow,
		Cleanup: a.showCleanup,
		Quit:    a.quit,
	}, icon)
	return a
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	cfg := a.store.Load()
	i18n.Set(i18n.Parse(cfg.Language))
	a.tray.SetLabels(a.trayLabels())
	a.setDark(a.resolveDark(), false)
	go a.tray.Start()
	go a.ticker()
	go theme.Watch(ctx, 2*time.Second, a.onSystemTheme)
	go a.scanAndBroadcast()
}

func (a *App) shutdown(ctx context.Context) {
	a.stopOnce.Do(func() { close(a.stop) })
	a.tray.Stop()
}

// beforeClose runs before the window closes; a true return intercepts the close
// and hides the window to the tray instead.
func (a *App) beforeClose(ctx context.Context) bool {
	if a.quitting.Load() {
		return false
	}
	if a.store.Current().Background && a.tray.Available() {
		wruntime.WindowHide(ctx)
		return true
	}
	return false
}

// GetSnapshot returns the current snapshot, scanning synchronously on the first
// call after startup.
func (a *App) GetSnapshot() model.Snapshot {
	if s := a.Snapshot(); s.GeneratedAt != 0 {
		return s
	}
	return a.ScanNow()
}

// ScanNow rescans immediately and returns the fresh snapshot.
func (a *App) ScanNow() model.Snapshot {
	a.scanAndBroadcast()
	return a.Snapshot()
}

// SetIncludeUDP switches "include UDP bindings" and rescans immediately.
func (a *App) SetIncludeUDP(v bool) model.Snapshot {
	a.includeUDP.Store(v)
	a.scanAndBroadcast()
	return a.Snapshot()
}

// Snapshot returns the latest snapshot held in memory.
func (a *App) Snapshot() model.Snapshot {
	a.snapMu.RLock()
	defer a.snapMu.RUnlock()
	return a.snap
}

// GetAppInfo returns metadata about the running application.
func (a *App) GetAppInfo() model.AppInfo {
	return model.AppInfo{
		Name:    appName,
		Version: version,
		OS:      runtime.GOOS + "/" + runtime.GOARCH,
		DataDir: a.store.Dir(),
		RepoURL: repoURL,
	}
}

// SaveSettings persists settings and rescans, so a change such as "loopback
// listeners only" or the language takes effect at once.
func (a *App) SaveSettings(v model.Settings) (model.Settings, error) {
	saved, err := a.store.Save(v)
	if err != nil {
		return saved, fmt.Errorf("%s", i18n.T("error.saveSettings", err.Error()))
	}
	i18n.Set(i18n.Parse(saved.Language))
	a.tray.SetLabels(a.trayLabels())
	a.refreshTrayText()
	a.wakeScan()
	return saved, nil
}

// GetSettings returns the current persisted preferences.
func (a *App) GetSettings() model.Settings {
	return a.store.Current()
}

// SetTheme switches light/dark, persists it and rescans so the frontend gets the
// new theme in the next snapshot.
func (a *App) SetTheme(name string) (model.Settings, error) {
	cfg := a.store.Current()
	cfg.Theme = name
	saved, err := a.store.Save(cfg)
	if err != nil {
		return saved, fmt.Errorf("%s", i18n.T("error.saveSettings", err.Error()))
	}
	a.setDark(a.resolveDark(), true)
	a.wakeScan()
	return saved, nil
}

// KillProcess ends one process. It returns as soon as the request is accepted;
// the outcome arrives on the kill:result event.
func (a *App) KillProcess(pid int32) (model.KillStarted, error) {
	return a.KillProcesses([]int32{pid})
}

// KillProcesses ends several processes at once (used by the cleanup screen).
func (a *App) KillProcesses(pids []int32) (model.KillStarted, error) {
	started := model.KillStarted{PIDs: []int32{}, Skipped: map[int32]string{}}
	var reasons []string

	index := map[int32]model.Entry{}
	for _, e := range a.Snapshot().Entries {
		if _, ok := index[e.PID]; !ok {
			index[e.PID] = e
		}
	}
	for _, pid := range pids {
		e, ok := index[pid]
		if !ok {
			reason := i18n.T("error.staleEntry")
			started.Skipped[pid] = reason
			reasons = append(reasons, i18n.T("error.skippedEntry", pid, reason))
			continue
		}
		if reason := guardProcess(e); reason != "" {
			started.Skipped[pid] = reason
			reasons = append(reasons, i18n.T("error.skippedEntry", pid, reason))
			continue
		}
		accepted, reason := a.manager.Start(pid, e.StartedAt)
		if !accepted {
			started.Skipped[pid] = reason
			reasons = append(reasons, i18n.T("error.skippedEntry", pid, reason))
			continue
		}
		started.PIDs = append(started.PIDs, pid)
	}
	if len(started.PIDs) == 0 && len(reasons) > 0 {
		return started, fmt.Errorf("%s", strings.Join(reasons, i18n.T("common.joiner")))
	}
	return started, nil
}

// CopyText writes text to the system clipboard.
func (a *App) CopyText(text string) error {
	if a.ctx == nil {
		return fmt.Errorf("%s", i18n.T("error.notReady"))
	}
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("%s", i18n.T("error.nothingToCopy"))
	}
	wruntime.ClipboardSetText(a.ctx, text)
	return nil
}

// OpenPath reveals a directory in File Explorer.
func (a *App) OpenPath(path string) error {
	p := strings.TrimSpace(path)
	if p == "" {
		return fmt.Errorf("%s", i18n.T("error.emptyPath"))
	}
	if _, err := os.Stat(p); err != nil {
		return fmt.Errorf("%s", i18n.T("error.pathMissing", p))
	}
	if err := exec.Command("explorer", p).Start(); err != nil {
		return fmt.Errorf("%s", i18n.T("error.openPath", err.Error()))
	}
	return nil
}

// OpenRepo opens the public repository in the default browser. The address is
// fixed in Go, so the frontend cannot use this binding to open anything else.
func (a *App) OpenRepo() error {
	if a.ctx == nil {
		return fmt.Errorf("%s", i18n.T("error.notReady"))
	}
	wruntime.BrowserOpenURL(a.ctx, repoURL)
	return nil
}

// CloseWindow behaves like the system close button: it asks beforeClose first,
// hiding to the tray when "keep running" is on and quitting otherwise. It goes
// through the policy directly instead of posting WM_CLOSE, because finding the
// window by title also matches unrelated windows that happen to carry it.
func (a *App) CloseWindow() {
	if a.ctx == nil {
		return
	}
	if a.beforeClose(a.ctx) {
		return
	}
	a.quit()
}

// MinimiseWindow minimises the window.
func (a *App) MinimiseWindow() {
	if a.ctx != nil {
		wruntime.WindowMinimise(a.ctx)
	}
}

// ToggleMaximiseWindow maximises or restores the window.
func (a *App) ToggleMaximiseWindow() {
	if a.ctx != nil {
		wruntime.WindowToggleMaximise(a.ctx)
	}
}

func (a *App) scanAndBroadcast() {
	if !a.scanMu.TryLock() {
		return // a scan is still running, skip this tick
	}
	defer a.scanMu.Unlock()

	cfg := a.store.Current()
	entries, err := a.scanner.Scan(scan.Options{OnlyLocal: cfg.OnlyLocal, IncludeUDP: a.includeUDP.Load()})
	warning := ""
	if err != nil {
		warning = i18n.T("error.scanFailed", err.Error())
		entries = nil
	}
	truncated := false
	if len(entries) > maxEntries {
		entries = entries[:maxEntries]
		truncated = true
	}

	orphans := 0
	for _, e := range entries {
		if e.Orphan {
			orphans++
		}
	}
	themeName := "light"
	if a.isDark() {
		themeName = "dark"
	}
	snap := model.Snapshot{
		GeneratedAt:    time.Now().UnixMilli(),
		Theme:          themeName,
		Language:       string(i18n.Current()),
		Settings:       cfg,
		Entries:        entries,
		Warning:        warning,
		ListeningCount: len(entries),
		OrphanCount:    orphans,
		Truncated:      truncated,
	}

	a.snapMu.Lock()
	a.snap = snap
	a.snapMu.Unlock()

	if a.ctx != nil {
		wruntime.EventsEmit(a.ctx, "snapshot:update", snap)
	}
	a.afterScan(snap)
}

func (a *App) afterScan(snap model.Snapshot) {
	a.refreshTrayText()

	current := map[string]bool{}
	for _, e := range snap.Entries {
		if e.Orphan {
			current[e.Key] = true
		}
	}
	a.orphanMu.Lock()
	first := a.lastOrphans == nil
	added := 0
	for k := range current {
		if !a.lastOrphans[k] {
			added++
		}
	}
	a.lastOrphans = current
	a.orphanMu.Unlock()

	if first || added == 0 || !snap.Settings.NotifyOrphans || a.ctx == nil {
		return
	}
	msg := i18n.Tn("notify.orphansBodyOne", "notify.orphansBody", snap.OrphanCount, snap.OrphanCount)
	notified := a.tray.Notify(i18n.T("notify.orphansTitle"), msg)
	wruntime.EventsEmit(a.ctx, "orphans:new", map[string]any{
		"added":    added,
		"total":    snap.OrphanCount,
		"message":  msg,
		"notified": notified,
	})
}

func (a *App) trayLabels() tray.Labels {
	return tray.Labels{
		Show:        i18n.T("tray.show"),
		ShowHint:    i18n.T("tray.showHint"),
		Hide:        i18n.T("tray.hide"),
		HideHint:    i18n.T("tray.hideHint"),
		CleanupHint: i18n.T("tray.cleanupHint"),
		Quit:        i18n.T("tray.quit"),
		QuitHint:    i18n.T("tray.quitHint"),
	}
}

// refreshTrayText re-renders the counts shown by the tray from the latest
// snapshot.
func (a *App) refreshTrayText() {
	if !a.tray.Available() {
		return
	}
	snap := a.Snapshot()
	a.tray.SetTooltip(i18n.T("tray.tooltip", snap.ListeningCount, snap.OrphanCount))
	a.tray.SetCleanupLabel(i18n.T("tray.cleanup", snap.OrphanCount))
}

func (a *App) onKillDone(r model.KillReport) {
	if s, ok := a.scanner.(*scan.GopsutilScanner); ok {
		s.Invalidate(r.PID)
	}
	if a.ctx != nil {
		wruntime.EventsEmit(a.ctx, "kill:result", r)
	}
	a.wakeScan()
}

func (a *App) wakeScan() {
	select {
	case a.wake <- struct{}{}:
	default:
	}
}

func (a *App) ticker() {
	for {
		interval := time.Duration(a.store.Current().IntervalSec) * time.Second
		if interval <= 0 {
			interval = 5 * time.Second
		}
		timer := time.NewTimer(interval)
		select {
		case <-a.stop:
			timer.Stop()
			return
		case <-a.wake:
			timer.Stop()
		case <-timer.C:
		}
		a.scanAndBroadcast()
	}
}

// resolveDark decides between dark and light: an explicit user choice wins, and
// only the untouched "follow the system" state tracks Windows.
func (a *App) resolveDark() bool {
	switch a.store.Current().Theme {
	case model.ThemeLight:
		return false
	case model.ThemeDark:
		return true
	default:
		return theme.IsDark()
	}
}

// onSystemTheme is the theme.Watch callback: the system preference only applies
// while the user has not picked a theme themselves.
func (a *App) onSystemTheme(dark bool) {
	if a.store.Current().Theme == model.ThemeSystem {
		a.setDark(dark, true)
	}
}

func (a *App) setDark(dark bool, emit bool) {
	a.themeMu.Lock()
	changed := a.dark != dark
	a.dark = dark
	a.themeMu.Unlock()
	if a.ctx == nil || !changed {
		return
	}
	if dark {
		wruntime.WindowSetBackgroundColour(a.ctx, 0x0D, 0x15, 0x15, 255)
	} else {
		wruntime.WindowSetBackgroundColour(a.ctx, 0xF4, 0xFB, 0xFB, 255)
	}
	if emit {
		wruntime.EventsEmit(a.ctx, "theme:changed", map[string]any{"dark": dark})
	}
}

func (a *App) isDark() bool {
	a.themeMu.RLock()
	defer a.themeMu.RUnlock()
	return a.dark
}

func (a *App) showWindow() {
	if a.ctx == nil {
		return
	}
	wruntime.WindowShow(a.ctx)
	wruntime.WindowUnminimise(a.ctx)
}

func (a *App) hideWindow() {
	if a.ctx != nil {
		wruntime.WindowHide(a.ctx)
	}
}

func (a *App) showCleanup() {
	a.showWindow()
	if a.ctx != nil {
		wruntime.EventsEmit(a.ctx, "navigate", map[string]string{"screen": "cleanup"})
	}
}

func (a *App) quit() {
	a.quitting.Store(true)
	if a.ctx != nil {
		wruntime.Quit(a.ctx)
	}
}

// blockedNames are critical system processes that may not be ended even when
// they are the ones holding a port.
var blockedNames = map[string]bool{
	"system": true, "system idle process": true, "registry": true, "memcompression": true,
	"csrss.exe": true, "wininit.exe": true, "services.exe": true, "lsass.exe": true,
	"winlogon.exe": true, "smss.exe": true, "svchost.exe": true, "dwm.exe": true,
	"explorer.exe": true, "audiodg.exe": true,
}

// guardProcess is the safety net in front of every kill; a non-empty return
// value is the reason for the refusal.
func guardProcess(e model.Entry) string {
	if e.PID <= 4 {
		return i18n.T("guard.systemPid")
	}
	if e.PID == int32(os.Getpid()) {
		return i18n.T("guard.self")
	}
	name := strings.ToLower(filepath.Base(strings.TrimSpace(e.Exe)))
	if name == "" || name == "." {
		// The image path is unreadable, so fall back to the real process name and
		// only then to the display name.
		name = strings.ToLower(strings.TrimSpace(e.ProcessName))
	}
	if name == "" || name == "." {
		name = strings.ToLower(strings.TrimSpace(e.DisplayName))
	}
	if blockedNames[name] || blockedNames[name+".exe"] {
		return i18n.T("guard.critical", name)
	}
	sysRoot := os.Getenv("SystemRoot")
	if e.Exe != "" && sysRoot != "" &&
		strings.HasPrefix(strings.ToLower(filepath.Clean(e.Exe)), strings.ToLower(filepath.Clean(sysRoot))) {
		return i18n.T("guard.systemDir")
	}
	return ""
}
