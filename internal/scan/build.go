package scan

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/windows"

	"Straggle/internal/humanize"
	"Straggle/internal/i18n"
	"Straggle/internal/model"
)

// genericRuntimes are process names that say nothing on their own (node running
// vite, for instance), so the command line decides what the entry is called.
var genericRuntimes = map[string]bool{
	"node": true, "nodejs": true, "python": true, "python3": true, "pythonw": true,
	"java": true, "javaw": true, "deno": true, "bun": true, "ruby": true,
	"php": true, "dotnet": true, "py": true,
}

// toolHints are common development tools in priority order.
var toolHints = []string{
	"vite", "next", "nuxt", "webpack", "rollup", "esbuild",
	"nodemon", "tsx", "ts-node", "vitest", "jest", "storybook",
	"playwright", "tsc", "eslint", "prettier", "npm", "pnpm", "yarn", "uvicorn", "gunicorn",
}

// DisplayName builds the process name shown as a list title, replacing a generic
// runtime with the tool its command line names.
func DisplayName(processName, cmdline string, port uint32) string {
	base := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(filepath.Base(strings.TrimSpace(processName))), ".exe"))
	if base == "" || base == "." || base == "unknown" {
		base = i18n.T("scan.unknownProcess")
	}
	low := strings.ToLower(cmdline)
	if genericRuntimes[base] {
		if hint := firstHint(low); hint != "" {
			base = hint
		}
	}
	if strings.Contains(low, "--inspect") || (base == "node" && port == 9229) {
		base += i18n.T("scan.debugSuffix")
	}
	return base
}

func firstHint(cmdline string) string {
	for _, h := range toolHints {
		if containsMarker(cmdline, h) {
			return h
		}
	}
	return ""
}

// containsMarker looks for needle in hay and requires a non-word character on
// both sides, so a tool name buried inside a longer word never matches:
// node_modules\vite\bin\vite.js hits vite, a1b2vite9c does not.
func containsMarker(hay, needle string) bool {
	if needle == "" || hay == "" {
		return false
	}
	from := 0
	for {
		i := strings.Index(hay[from:], needle)
		if i < 0 {
			return false
		}
		i += from
		var before byte = ' '
		if i > 0 {
			before = hay[i-1]
		}
		var after byte = ' '
		if i+len(needle) < len(hay) {
			after = hay[i+len(needle)]
		}
		if !isWordByte(before) && !isWordByte(after) {
			return true
		}
		from = i + 1
		if from >= len(hay) {
			return false
		}
	}
}

// isWordByte counts only ASCII letters and digits as word characters, so
// vite, vite-cli and /vite/ all match.
func isWordByte(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9')
}

// ProjectOf infers the project a process belongs to: working directory first,
// then an absolute path in the command line, then the executable's directory.
// The bool reports whether the answer came from the working directory.
func ProjectOf(p ProcInfo) (string, bool) {
	if p.CwdKnown {
		if name := cleanBase(p.Cwd); name != "" {
			return name, true
		}
	}
	if dir := dirFromCmdline(p.Cmdline); dir != "" {
		if name := cleanBase(dir); name != "" {
			return name, false
		}
	}
	if p.Exe != "" {
		if name := cleanBase(filepath.Dir(p.Exe)); name != "" {
			return name, false
		}
	}
	return i18n.T("scan.unknownDir"), false
}

func cleanBase(dir string) string {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return ""
	}
	base := filepath.Base(filepath.Clean(dir))
	switch base {
	case "", ".", string(filepath.Separator), "/":
		return ""
	}
	return base
}

// dirFromCmdline returns the directory of the first absolute path in a command
// line.
func dirFromCmdline(cmdline string) string {
	args, err := windows.DecomposeCommandLine(cmdline)
	if err != nil {
		return ""
	}
	for i, arg := range args {
		if arg == "--cwd" && i+1 < len(args) && filepath.IsAbs(args[i+1]) {
			return args[i+1]
		}
		if dir, ok := strings.CutPrefix(arg, "--cwd="); ok && filepath.IsAbs(dir) {
			return dir
		}
	}
	for _, tok := range args {
		if !filepath.IsAbs(tok) {
			continue
		}
		dir := filepath.Dir(tok)
		if cleanBase(dir) != "" {
			return dir
		}
	}
	return ""
}

// BuildEntry assembles one entry the frontend can render directly from a port
// group and its process metadata.
func BuildEntry(g Group, p ProcInfo, now time.Time, sysRoot string) model.Entry {
	display := DisplayName(p.Name, p.Cmdline, g.Port)
	addrs := make([]model.Addr, 0, len(g.IPs))
	for _, ip := range g.IPs {
		addrs = append(addrs, model.Addr{
			Addr:  AddrLabel(ip, g.Port),
			IP:    ip,
			Port:  g.Port,
			Scope: ScopeOf(ip),
		})
	}
	project, cwdKnown := ProjectOf(p)
	e := model.Entry{
		Key:          fmt.Sprintf("%s:%d:%d", g.Proto, g.Port, g.PID),
		Port:         g.Port,
		Proto:        g.Proto,
		PID:          g.PID,
		ProcessName:  strings.TrimSpace(p.Name),
		DisplayName:  display,
		Headline:     fmt.Sprintf("%d · %s", g.Port, display),
		Cmdline:      p.Cmdline,
		Exe:          p.Exe,
		Cwd:          p.Cwd,
		CwdKnown:     cwdKnown,
		Project:      project,
		Addrs:        addrs,
		Scope:        GroupScope(g),
		StartedAt:    p.StartedAt,
		ParentPID:    p.Ppid,
		ParentName:   p.ParentName,
		ParentAlive:  p.ParentAlive,
		Readable:     p.Readable,
		Denied:       p.Denied,
		DeniedReason: p.DeniedReason,
	}
	if p.StartedAt > 0 {
		started := time.UnixMilli(p.StartedAt)
		e.UptimeHuman = humanize.Duration(now.Sub(started))
		e.StartedHuman = humanize.StartTime(started, now)
	} else {
		e.UptimeHuman = i18n.T("common.unknown")
		e.StartedHuman = i18n.T("common.unknown")
	}

	parts := []string{fmt.Sprintf("PID %d", g.PID)}
	if project != "" {
		parts = append(parts, project)
	}
	if e.UptimeHuman != "" {
		parts = append(parts, e.UptimeHuman)
	}
	e.Supporting = strings.Join(parts, " · ")

	switch {
	case p.Ppid <= 0:
		e.ParentNote = i18n.T("scan.noParent")
	case p.ParentAlive:
		name := strings.TrimSuffix(p.ParentName, ".exe")
		if name != "" {
			e.ParentNote = i18n.T("scan.parentNamed", p.Ppid, name)
		} else {
			e.ParentNote = i18n.T("scan.parent", p.Ppid)
		}
	default:
		e.ParentNote = i18n.T("scan.parentGone", p.Ppid)
	}

	e.Orphan, e.OrphanReason = IsOrphan(p, now, sysRoot)
	return e
}
