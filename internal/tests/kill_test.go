package tests

import (
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	gproc "github.com/shirou/gopsutil/v4/process"

	"Straggle/internal/i18n"
	"Straggle/internal/kill"
	"Straggle/internal/model"
)

// fakeKiller simulates processes in memory, so the whole 10-second watchdog
// state machine runs in milliseconds.
type fakeKiller struct {
	mu         sync.Mutex
	alive      map[int32]bool
	createdAt  map[int32]int64
	graceful   []int32
	forced     []int32
	graceErr   error
	forceErr   error
	aliveErr   error
	dieOnForce bool
}

func newFakeKiller() *fakeKiller {
	return &fakeKiller{alive: map[int32]bool{}, createdAt: map[int32]int64{}}
}

func (f *fakeKiller) Alive(pid int32, startedAt int64) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.aliveErr != nil {
		return false, f.aliveErr
	}
	if !f.alive[pid] {
		return false, nil
	}
	if startedAt > 0 && f.createdAt[pid] != 0 && f.createdAt[pid] != startedAt {
		return false, nil
	}
	return true, nil
}

func TestUnknownIdentityIsRejected(t *testing.T) {
	f := newFakeKiller()
	m, _ := newTestManager(f)
	if accepted, reason := m.Start(100, 0); accepted || reason == "" {
		t.Fatal("unknown creation time must be refused")
	}
}

func TestIdentityReadFailureCannotKill(t *testing.T) {
	f := newFakeKiller()
	f.aliveErr = errors.New("identity unavailable")
	m, ch := newTestManager(f)
	m.Start(100, 1)
	report := waitReport(t, ch)
	if report.OK || report.Stage != model.StageDenied || len(f.graceful) != 0 || len(f.forcedPids()) != 0 {
		t.Fatalf("identity failure must stop before ending a process: %+v", report)
	}
}

func TestPlatformKillerVerifiesIdentity(t *testing.T) {
	pid := int32(os.Getpid())
	p, err := gproc.NewProcess(pid)
	if err != nil {
		t.Fatal(err)
	}
	started, err := p.CreateTime()
	if err != nil {
		t.Fatal(err)
	}
	k := kill.NewPlatformKiller()
	if alive, err := k.Alive(pid, started); !alive || err != nil {
		t.Fatalf("current process must match: alive=%v, error=%v", alive, err)
	}
	if alive, err := k.Alive(pid, started+1); alive || err != nil {
		t.Fatalf("wrong creation time must not match: alive=%v, error=%v", alive, err)
	}
	if err := k.Force(pid, started+1); err == nil {
		t.Fatal("force must refuse a mismatched identity")
	}
	if err := k.Graceful(pid, started+1); err == nil {
		t.Fatal("graceful must refuse a mismatched identity")
	}
	if _, err := k.Alive(pid, 0); err == nil {
		t.Fatal("missing creation time must return an error")
	}
}

func (f *fakeKiller) Graceful(pid int32, _ int64) error {
	f.mu.Lock()
	f.graceful = append(f.graceful, pid)
	f.mu.Unlock()
	return f.graceErr
}

func (f *fakeKiller) Force(pid int32, _ int64) error {
	f.mu.Lock()
	f.forced = append(f.forced, pid)
	die := f.dieOnForce
	err := f.forceErr
	if die {
		f.alive[pid] = false
	}
	f.mu.Unlock()
	return err
}

func (f *fakeKiller) kill(pid int32) {
	f.mu.Lock()
	f.alive[pid] = false
	f.mu.Unlock()
}

func (f *fakeKiller) forcedPids() []int32 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]int32{}, f.forced...)
}

func newTestManager(f *fakeKiller) (*kill.Manager, chan model.KillReport) {
	ch := make(chan model.KillReport, 4)
	m := kill.NewManager(f, func(r model.KillReport) { ch <- r })
	m.Grace = 150 * time.Millisecond
	m.Poll = 5 * time.Millisecond
	return m, ch
}

func waitReport(t *testing.T, ch chan model.KillReport) model.KillReport {
	t.Helper()
	select {
	case r := <-ch:
		return r
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for the kill report")
		return model.KillReport{}
	}
}

func TestGracefulPath(t *testing.T) {
	f := newFakeKiller()
	f.alive[100] = true
	m, ch := newTestManager(f)

	if ok, reason := m.Start(100, 1); !ok {
		t.Fatalf("request should be accepted, refused with: %s", reason)
	}
	go func() {
		time.Sleep(20 * time.Millisecond)
		f.kill(100)
	}()

	report := waitReport(t, ch)
	if !report.OK || report.Stage != model.StageTerm {
		t.Fatalf("want a graceful end, got %+v", report)
	}
	if len(f.forcedPids()) != 0 {
		t.Fatalf("a graceful end must not force-kill: %v", f.forcedPids())
	}
}

func TestEscalatesToForceAfterGrace(t *testing.T) {
	f := newFakeKiller()
	f.alive[200] = true
	f.dieOnForce = true
	m, ch := newTestManager(f)

	start := time.Now()
	if ok, _ := m.Start(200, 1); !ok {
		t.Fatal("request should be accepted")
	}
	report := waitReport(t, ch)
	if !report.OK || report.Stage != model.StageKill {
		t.Fatalf("want a force-kill, got %+v", report)
	}
	if elapsed := time.Since(start); elapsed < m.Grace {
		t.Fatalf("the force-kill must not happen before the grace window: %v", elapsed)
	}
	if len(f.forcedPids()) != 1 {
		t.Fatalf("want exactly one force-kill: %v", f.forcedPids())
	}
}

func TestAlreadyGone(t *testing.T) {
	f := newFakeKiller()
	m, ch := newTestManager(f)
	if ok, _ := m.Start(300, 1); !ok {
		t.Fatal("request should be accepted")
	}
	report := waitReport(t, ch)
	if !report.OK || report.Stage != model.StageGone {
		t.Fatalf("want \"already gone\", got %+v", report)
	}
}

func TestPidReuseIsTreatedAsGone(t *testing.T) {
	f := newFakeKiller()
	f.alive[400] = true
	f.createdAt[400] = 999 // differs from the requested creation time: the PID was reused
	m, ch := newTestManager(f)
	if ok, _ := m.Start(400, 111); !ok {
		t.Fatal("request should be accepted")
	}
	report := waitReport(t, ch)
	if report.Stage != model.StageGone {
		t.Fatalf("a reused PID must abort the request, got %+v", report)
	}
	if len(f.forcedPids()) != 0 {
		t.Fatalf("a reused PID must never be touched: %v", f.forcedPids())
	}
}

func TestForceDenied(t *testing.T) {
	f := newFakeKiller()
	f.alive[500] = true
	f.forceErr = errors.New("access denied (exit code 5)")
	m, ch := newTestManager(f)
	ok, _ := m.Start(500, 1)
	if !ok {
		t.Fatal("request should be accepted")
	}
	report := waitReport(t, ch)
	if report.OK || report.Stage != model.StageDenied {
		t.Fatalf("want a failure report, got %+v", report)
	}
	if report.Message == "" {
		t.Fatal("a failure must carry a readable reason")
	}
}

func TestRejectionReasonsAreLocalized(t *testing.T) {
	f := newFakeKiller()
	m, _ := newTestManager(f)

	i18n.Set(i18n.EN)
	ok, reason := m.Start(4, 0)
	if ok || reason == "" {
		t.Fatal("PID ≤ 4 must be refused with a reason")
	}
	if want := i18n.T("kill.systemPid"); reason != want {
		t.Fatalf("English refusal = %q, want %q", reason, want)
	}

	i18n.Set(i18n.ZH)
	_, reason = m.Start(4, 0)
	if want := i18n.T("kill.systemPid"); reason != want {
		t.Fatalf("Chinese refusal = %q, want %q", reason, want)
	}
}

func TestDuplicateStartRejected(t *testing.T) {
	f := newFakeKiller()
	f.alive[600] = true
	m, ch := newTestManager(f)
	if ok, _ := m.Start(600, 1); !ok {
		t.Fatal("the first request should be accepted")
	}
	if ok, reason := m.Start(600, 1); ok || reason == "" {
		t.Fatal("a second request for the same PID must be refused")
	}
	f.kill(600)
	waitReport(t, ch)
	select {
	case <-ch:
		t.Fatal("no second report should be produced")
	default:
	}
}
