package scan

import (
	"path/filepath"
	"strings"
	"time"

	"Straggle/internal/i18n"
)

// Ancestor is one link of the ancestor chain, used to tell whether a terminal
// host is still alive.
type Ancestor struct {
	PID  int32
	Name string
}

// ProcInfo is the process metadata read from the system.
type ProcInfo struct {
	PID               int32
	Name              string
	Cmdline           string
	Exe               string
	Cwd               string
	CwdKnown          bool
	StartedAt         int64 // Unix milliseconds
	Ppid              int32
	ParentName        string
	ParentAlive       bool
	ParentReused      bool
	Ancestors         []Ancestor
	AncestorsComplete bool
	Readable          bool
	Denied            bool
	DeniedReason      string
}

// terminalHosts are terminal/shell images: a process under one of them is still
// part of a live terminal session.
var terminalHosts = map[string]bool{
	"cmd.exe":             true,
	"powershell.exe":      true,
	"pwsh.exe":            true,
	"wt.exe":              true,
	"windowsterminal.exe": true,
	"conhost.exe":         true,
	"openconsole.exe":     true,
	"bash.exe":            true,
	"sh.exe":              true,
	"zsh.exe":             true,
	"fish.exe":            true,
	"wsl.exe":             true,
	"git-bash.exe":        true,
	"mintty.exe":          true,
	"alacritty.exe":       true,
	"wezterm-gui.exe":     true,
	"wezterm.exe":         true,
	"kitty.exe":           true,
	"tmux.exe":            true,
	"screen.exe":          true,
	"nu.exe":              true,
	"xonsh.exe":           true,
	"code.exe":            true, // VS Code's integrated terminal counts as a live host
}

// guiHosts are hosts a user or the system started on purpose; a process under
// one of them is never an orphan.
var guiHosts = map[string]bool{
	"explorer.exe":  true,
	"services.exe":  true,
	"svchost.exe":   true,
	"wininit.exe":   true,
	"taskeng.exe":   true,
	"runtimebroker": true,
	"code.exe":      true,
	"cursor.exe":    true,
	"windsurf.exe":  true,
	"trae.exe":      true,
	"idea64.exe":    true,
	"goland64.exe":  true,
	"pycharm64.exe": true,
	"devenv.exe":    true,
}

// HasTerminalAncestor reports whether a terminal, shell or editor host is still
// present in the ancestor chain.
func HasTerminalAncestor(p ProcInfo) bool {
	for _, a := range p.Ancestors {
		name := baseName(a.Name)
		if v, ok := terminalHosts[name]; ok && v {
			return true
		}
	}
	if v, ok := terminalHosts[baseName(p.ParentName)]; ok && v {
		return true
	}
	return false
}

func hasGUIHostAncestor(p ProcInfo) bool {
	if guiHosts[baseName(p.ParentName)] {
		return true
	}
	for _, a := range p.Ancestors {
		if guiHosts[baseName(a.Name)] {
			return true
		}
	}
	return false
}

func baseName(s string) string {
	return strings.ToLower(filepath.Base(strings.TrimSpace(s)))
}

// desktopApps are desktop applications (IDE, editor, browser, messenger) whose
// launcher normally exits right after starting them — GoLand handing over to
// itself, for instance. Their parent exiting says nothing about the port they
// hold, so they are never orphans.
var desktopApps = map[string]bool{
	"goland64.exe": true, "idea64.exe": true, "pycharm64.exe": true, "webstorm64.exe": true,
	"clion64.exe": true, "rider64.exe": true, "datagrip64.exe": true, "studio64.exe": true,
	"code.exe": true, "code - insiders.exe": true, "cursor.exe": true, "windsurf.exe": true,
	"trae.exe": true, "zed.exe": true, "devenv.exe": true, "sublime_text.exe": true,
	"notepad++.exe": true, "explorer.exe": true, "chrome.exe": true, "msedge.exe": true,
	"firefox.exe": true, "wechat.exe": true, "weixin.exe": true, "qq.exe": true,
	"dingtalk.exe": true, "feishu.exe": true, "lark.exe": true, "telegram.exe": true,
	"slack.exe": true, "discord.exe": true, "spotify.exe": true,
}

func isDesktopApp(name, exe string) bool {
	if desktopApps[baseName(name)] {
		return true
	}
	if strings.TrimSpace(exe) != "" && desktopApps[strings.ToLower(filepath.Base(exe))] {
		return true
	}
	return false
}

func isSystemExe(exe, sysRoot string) bool {
	if strings.TrimSpace(exe) == "" || strings.TrimSpace(sysRoot) == "" {
		return false
	}
	return strings.HasPrefix(strings.ToLower(filepath.Clean(exe)), strings.ToLower(filepath.Clean(sysRoot)))
}

// IsOrphan decides whether a port is still held by a process whose terminal is
// gone, and returns the human-readable reason. It prefers a missed orphan over a
// wrong kill:
//
//  1. unreadable, a system process (PID ≤ 4 or inside the Windows directory), a
//     common desktop application, or younger than 3 seconds → not an orphan;
//  2. the parent exited, or its creation time proves the parent PID was reused → orphan;
//  3. a complete verified ancestor chain has no terminal host and is not a
//     deliberate launcher such as explorer.exe or services.exe → orphan;
//  4. anything else is not an orphan.
func IsOrphan(p ProcInfo, now time.Time, sysRoot string) (bool, string) {
	if !p.Readable || p.Denied {
		return false, ""
	}
	if p.PID <= 4 {
		return false, ""
	}
	if p.StartedAt > 0 {
		if now.Sub(time.UnixMilli(p.StartedAt)) < 3*time.Second {
			return false, ""
		}
	}
	if sysRoot != "" && isSystemExe(p.Exe, sysRoot) {
		return false, ""
	}
	if isDesktopApp(p.Name, p.Exe) {
		return false, ""
	}
	if p.ParentReused {
		return true, i18n.T("scan.parentGone", p.Ppid)
	}
	if hasGUIHostAncestor(p) {
		return false, ""
	}
	if !p.ParentAlive {
		if p.Ppid > 0 {
			return true, i18n.T("scan.parentGone", p.Ppid)
		}
		return true, i18n.T("scan.terminalGone")
	}
	if !p.AncestorsComplete {
		// The parent chain is unreadable, so nothing proves the terminal is gone.
		return false, ""
	}
	if !HasTerminalAncestor(p) {
		return true, i18n.T("scan.terminalGone")
	}
	return false, ""
}
