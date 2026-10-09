package tests

import (
	"testing"
	"time"

	"Straggle/internal/i18n"
	"Straggle/internal/model"
	"Straggle/internal/scan"
)

func TestScopeOf(t *testing.T) {
	cases := []struct {
		ip   string
		want model.Scope
	}{
		{"127.0.0.1", model.ScopeLocal},
		{"::1", model.ScopeLocal},
		{"0.0.0.0", model.ScopeAll},
		{"::", model.ScopeAll},
		{"192.168.1.7", model.ScopeLAN},
		{"fe80::1%eth0", model.ScopeLAN},
		{"", model.ScopeAll},
	}
	for _, c := range cases {
		if got := scan.ScopeOf(c.ip); got != c.want {
			t.Errorf("ScopeOf(%q) = %q, want %q", c.ip, got, c.want)
		}
	}
}

func TestMergeSocketsGroupsByPidPortProto(t *testing.T) {
	groups := scan.MergeSockets([]scan.RawSocket{
		{Proto: "tcp", IP: "127.0.0.1", Port: 3000, PID: 11},
		{Proto: "tcp", IP: "::1", Port: 3000, PID: 11},
		{Proto: "tcp", IP: "0.0.0.0", Port: 5173, PID: 22},
		{Proto: "tcp", IP: "127.0.0.1", Port: 3000, PID: 0}, // system PID: dropped
	})
	if len(groups) != 2 {
		t.Fatalf("want 2 groups, got %d: %+v", len(groups), groups)
	}
	first := groups[0]
	if first.Port != 3000 || first.PID != 11 || len(first.IPs) != 2 {
		t.Fatalf("IPv4 and IPv6 must merge into one group: %+v", first)
	}
	if first.IPs[0] != "127.0.0.1" {
		t.Fatalf("IPv4 must come first: %v", first.IPs)
	}
	if got := scan.GroupScope(first); got != model.ScopeLocal {
		t.Fatalf("group scope = %q, want local", got)
	}
	if got := scan.GroupScope(groups[1]); got != model.ScopeAll {
		t.Fatalf("0.0.0.0 group scope = %q, want all", got)
	}
}

func TestDisplayName(t *testing.T) {
	cases := []struct {
		lang    i18n.Lang
		name    string
		cmdline string
		port    uint32
		want    string
	}{
		{i18n.EN, "node.exe", `node C:\dev\app\node_modules\vite\bin\vite.js`, 5173, "vite"},
		{i18n.EN, "node.exe", "node server.js --port 3000", 3000, "node"},
		{i18n.EN, "node.exe", "node --inspect=9229 app.js", 9229, "node (debug)"},
		{i18n.EN, "python.exe", "", 8080, "python"},
		{i18n.EN, "postgres.exe", `postgres -D "C:\Program Files\PostgreSQL\16\data"`, 5432, "postgres"},
		{i18n.ZH, "node.exe", "node --inspect=9229 app.js", 9229, "node 调试"},
		{i18n.ZH, "", "", 8080, "未知进程"},
	}
	for _, c := range cases {
		i18n.Set(c.lang)
		if got := scan.DisplayName(c.name, c.cmdline, c.port); got != c.want {
			t.Errorf("[%s] DisplayName(%q, %q, %d) = %q, want %q", c.lang, c.name, c.cmdline, c.port, got, c.want)
		}
	}
}

func TestIsOrphan(t *testing.T) {
	now := time.Now()
	sysRoot := `C:\Windows`
	base := scan.ProcInfo{
		PID:               4200,
		Name:              "node.exe",
		Exe:               `C:\Program Files\nodejs\node.exe`,
		StartedAt:         now.Add(-10 * time.Minute).UnixMilli(),
		Ppid:              900,
		ParentName:        "cmd.exe",
		ParentAlive:       true,
		Readable:          true,
		Ancestors:         []scan.Ancestor{{PID: 900, Name: "cmd.exe"}},
		AncestorsComplete: true,
	}
	cases := []struct {
		name       string
		mutate     func(*scan.ProcInfo)
		wantOrphan bool
	}{
		{"terminal still alive", func(p *scan.ProcInfo) {}, false},
		{"terminal gone", func(p *scan.ProcInfo) {
			p.ParentName = "npm.cmd"
			p.Ancestors = []scan.Ancestor{{PID: 900, Name: "node.exe"}}
		}, true},
		{"parent exited", func(p *scan.ProcInfo) {
			p.ParentAlive = false
		}, true},
		{"parent is explorer", func(p *scan.ProcInfo) {
			p.ParentName = "explorer.exe"
			p.Ancestors = []scan.Ancestor{{PID: 900, Name: "explorer.exe"}}
		}, false},
		{"parent is a service", func(p *scan.ProcInfo) {
			p.ParentName = "services.exe"
			p.Ancestors = []scan.Ancestor{{PID: 900, Name: "services.exe"}}
		}, false},
		{"process in the system directory", func(p *scan.ProcInfo) {
			p.Exe = `C:\Windows\System32\svchost.exe`
			p.ParentName = "services.exe"
		}, false},
		{"metadata unreadable", func(p *scan.ProcInfo) {
			p.Readable = false
			p.Denied = true
		}, false},
		{"just started", func(p *scan.ProcInfo) {
			p.StartedAt = now.Add(-1 * time.Second).UnixMilli()
			p.ParentAlive = false
		}, false},
		{"parent chain unreadable", func(p *scan.ProcInfo) {
			p.ParentName = ""
			p.Ancestors = nil
			p.AncestorsComplete = false
		}, false},
		{"reused parent is explorer", func(p *scan.ProcInfo) {
			p.ParentReused = true
			p.ParentAlive = false
			p.ParentName = "explorer.exe"
			p.Ancestors = []scan.Ancestor{{PID: 900, Name: "explorer.exe"}}
		}, true},
		{"incomplete non-terminal chain", func(p *scan.ProcInfo) {
			p.ParentName = "node.exe"
			p.Ancestors = []scan.Ancestor{{PID: 900, Name: "node.exe"}}
			p.AncestorsComplete = false
		}, false},
		{"terminal beyond three ancestors", func(p *scan.ProcInfo) {
			p.ParentName = "node.exe"
			p.Ancestors = []scan.Ancestor{{Name: "node.exe"}, {Name: "node.exe"}, {Name: "node.exe"}, {Name: "pwsh.exe"}}
		}, false},
		{"desktop application launcher exited", func(p *scan.ProcInfo) {
			p.Name = "goland64.exe"
			p.Exe = `G:\GoLand 2026.2.3\bin\goland64.exe`
			p.ParentAlive = false
			p.Ancestors = nil
		}, false},
	}
	for _, c := range cases {
		i18n.Set(i18n.EN)
		info := base
		c.mutate(&info)
		got, reason := scan.IsOrphan(info, now, sysRoot)
		if got != c.wantOrphan {
			t.Errorf("%s: IsOrphan = %v (%s), want %v", c.name, got, reason, c.wantOrphan)
		}
		if got && reason == "" {
			t.Errorf("%s: an orphan must come with a reason", c.name)
		}
	}
}

func TestProjectOfWindowsArguments(t *testing.T) {
	cases := []struct{ command, want string }{
		{`node "C:\Program Files\demo\server.js"`, "demo"},
		{`node server.js --cwd="C:\work\my project"`, "my project"},
		{`node server.js --cwd "C:\work\my project"`, "my project"},
		{`node \\server\share\demo\server.js`, "demo"},
	}
	for _, c := range cases {
		got, known := scan.ProjectOf(scan.ProcInfo{Cmdline: c.command})
		if got != c.want || known {
			t.Errorf("ProjectOf(%q) = %q, %v; want %q, false", c.command, got, known, c.want)
		}
	}
}

func TestIsOrphanReasonIsLocalized(t *testing.T) {
	now := time.Now()
	info := scan.ProcInfo{
		PID:         4200,
		Name:        "node.exe",
		Exe:         `C:\Program Files\nodejs\node.exe`,
		StartedAt:   now.Add(-10 * time.Minute).UnixMilli(),
		Ppid:        900,
		ParentAlive: false,
		Readable:    true,
	}
	i18n.Set(i18n.EN)
	orphan, reason := scan.IsOrphan(info, now, `C:\Windows`)
	if !orphan || reason != "Parent process 900 has exited" {
		t.Fatalf("English reason = %q (orphan=%v)", reason, orphan)
	}
	i18n.Set(i18n.ZH)
	orphan, reason = scan.IsOrphan(info, now, `C:\Windows`)
	if !orphan || reason != "父进程 900 已退出" {
		t.Fatalf("Chinese reason = %q (orphan=%v)", reason, orphan)
	}
}
