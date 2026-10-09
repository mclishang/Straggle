// Package kill accepts end-process requests and drives the graceful → force
// escalation.
package kill

import (
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/windows"

	"Straggle/internal/i18n"
	"Straggle/internal/model"
)

// createNoWindow keeps taskkill from flashing a console window.
const createNoWindow = 0x08000000

// Killer is the platform capability the state machine needs. Taskkill is the
// closest Windows equivalent of a signal: it posts WM_CLOSE to window processes
// and CTRL_CLOSE_EVENT to console processes. Force uses a verified process handle.
type Killer interface {
	// Graceful asks the process to close.
	Graceful(pid int32, startedAt int64) error
	// Force terminates the process.
	Force(pid int32, startedAt int64) error
	// Alive verifies the creation time and reports whether the process still exists.
	Alive(pid int32, startedAt int64) (bool, error)
}

type taskkillKiller struct{}

// NewPlatformKiller returns the Windows implementation.
func NewPlatformKiller() Killer { return &taskkillKiller{} }

func (k *taskkillKiller) Graceful(pid int32, startedAt int64) error {
	h, err := openVerified(pid, startedAt, 0)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(h)
	// Keeping the process object open prevents PID reuse while taskkill resolves it.
	return runTaskkill("/PID", strconv.Itoa(int(pid)))
}

func (k *taskkillKiller) Force(pid int32, startedAt int64) error {
	h, err := openVerified(pid, startedAt, windows.PROCESS_TERMINATE)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(h)
	if err := windows.TerminateProcess(h, 1); err != nil {
		return fmt.Errorf("%s", i18n.T("kill.terminateFailed", err))
	}
	return nil
}

var errGone = errors.New("process identity no longer exists")

func openVerified(pid int32, startedAt int64, access uint32) (windows.Handle, error) {
	if startedAt <= 0 {
		return 0, fmt.Errorf("%s", i18n.T("kill.identityUnknown"))
	}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.SYNCHRONIZE|access, false, uint32(pid))
	if err != nil {
		if errors.Is(err, windows.ERROR_INVALID_PARAMETER) {
			return 0, errGone
		}
		return 0, fmt.Errorf("%s", i18n.T("kill.identityFailed", err))
	}
	var creation, exit, kernel, user windows.Filetime
	if err := windows.GetProcessTimes(h, &creation, &exit, &kernel, &user); err != nil {
		windows.CloseHandle(h)
		return 0, fmt.Errorf("%s", i18n.T("kill.identityFailed", err))
	}
	if creation.Nanoseconds()/int64(time.Millisecond) != startedAt {
		windows.CloseHandle(h)
		return 0, errGone
	}
	return h, nil
}

func (k *taskkillKiller) Alive(pid int32, startedAt int64) (bool, error) {
	h, err := openVerified(pid, startedAt, 0)
	if errors.Is(err, errGone) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer windows.CloseHandle(h)
	status, err := windows.WaitForSingleObject(h, 0)
	if err != nil {
		return false, fmt.Errorf("%s", i18n.T("kill.identityFailed", err))
	}
	return status == uint32(windows.WAIT_TIMEOUT), nil
}

func runTaskkill(args ...string) error {
	cmd := exec.Command("taskkill", args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
	_, err := cmd.CombinedOutput()
	if err != nil {
		code := -1
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		}
		// 128 means the process is gone; 1 means access denied or the process
		// cannot be ended, which is common for an orphaned console process.
		switch code {
		case 128:
			return fmt.Errorf("%s", i18n.T("kill.noSuchProcess"))
		case 1:
			return fmt.Errorf("%s", i18n.T("kill.denied"))
		default:
			return fmt.Errorf("%s", i18n.T("kill.taskkill", code))
		}
	}
	return nil
}

// Manager serializes end requests per PID and runs the escalation.
type Manager struct {
	killer Killer
	onDone func(model.KillReport)

	// Grace is the window the process gets to exit on its own, Poll the interval
	// at which it is checked.
	Grace time.Duration
	Poll  time.Duration

	mu       sync.Mutex
	inflight map[int32]bool
}

// NewManager builds a manager that waits 10 seconds before escalating.
func NewManager(k Killer, onDone func(model.KillReport)) *Manager {
	return &Manager{
		killer:   k,
		onDone:   onDone,
		Grace:    10 * time.Second,
		Poll:     250 * time.Millisecond,
		inflight: map[int32]bool{},
	}
}

// Start accepts an end request and returns immediately; the outcome is reported
// through onDone. The second return value is the reason for a refusal.
func (m *Manager) Start(pid int32, startedAt int64) (bool, string) {
	if pid <= 4 {
		return false, i18n.T("kill.systemPid")
	}
	if startedAt <= 0 {
		return false, i18n.T("kill.identityUnknown")
	}
	m.mu.Lock()
	if m.inflight[pid] {
		m.mu.Unlock()
		return false, i18n.T("kill.inflight")
	}
	m.inflight[pid] = true
	m.mu.Unlock()

	go m.run(pid, startedAt)
	return true, ""
}

func (m *Manager) run(pid int32, startedAt int64) {
	defer func() {
		m.mu.Lock()
		delete(m.inflight, pid)
		m.mu.Unlock()
	}()

	if m.finished(pid, startedAt, model.StageGone, "kill.gone") {
		return
	}

	gracefulErr := m.killer.Graceful(pid, startedAt)
	if errors.Is(gracefulErr, errGone) {
		m.report(model.KillReport{PID: pid, OK: true, Stage: model.StageGone, Message: i18n.T("kill.gone")})
		return
	}
	deadline := time.Now().Add(m.Grace)
	for time.Now().Before(deadline) {
		if m.finished(pid, startedAt, model.StageTerm, "kill.term") {
			return
		}
		time.Sleep(m.Poll)
	}
	if m.finished(pid, startedAt, model.StageTerm, "kill.term") {
		return
	}

	if err := m.killer.Force(pid, startedAt); err != nil {
		if errors.Is(err, errGone) {
			m.report(model.KillReport{PID: pid, OK: true, Stage: model.StageGone, Message: i18n.T("kill.gone")})
			return
		}
		msg := i18n.T("kill.forceFailed", err.Error())
		if gracefulErr != nil {
			msg += i18n.T("kill.graceFailed", gracefulErr.Error())
		}
		m.report(model.KillReport{PID: pid, OK: false, Stage: model.StageDenied, Message: msg})
		return
	}
	for i := 0; i < 40; i++ {
		if m.finished(pid, startedAt, model.StageKill, "kill.kill") {
			return
		}
		time.Sleep(m.Poll)
	}
	m.report(model.KillReport{PID: pid, OK: false, Stage: model.StageDenied, Message: i18n.T("kill.stillRunning")})
}

func (m *Manager) finished(pid int32, startedAt int64, stage, message string) bool {
	alive, err := m.killer.Alive(pid, startedAt)
	if err != nil {
		m.report(model.KillReport{PID: pid, Stage: model.StageDenied, Message: err.Error()})
		return true
	}
	if !alive {
		m.report(model.KillReport{PID: pid, OK: true, Stage: stage, Message: i18n.T(message)})
		return true
	}
	return false
}

func (m *Manager) report(r model.KillReport) {
	if m.onDone != nil {
		m.onDone(r)
	}
}
