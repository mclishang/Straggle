package scan

import (
	"os"
	"strings"
	"sync"
	"time"

	gnet "github.com/shirou/gopsutil/v4/net"
	gproc "github.com/shirou/gopsutil/v4/process"

	"Straggle/internal/i18n"
	"Straggle/internal/model"
)

// Options controls what one scan covers.
type Options struct {
	IncludeUDP bool // include UDP bindings, not just TCP listeners
	OnlyLocal  bool // keep loopback listeners only
}

// Scanner is the scanning layer.
type Scanner interface {
	Scan(opts Options) ([]model.Entry, error)
}

// GopsutilScanner reads listening sockets and process metadata through
// gopsutil, which uses the Windows IP helper API underneath.
type GopsutilScanner struct {
	mu      sync.Mutex
	cache   map[int32]cachedProc
	ttl     time.Duration
	timeout time.Duration
	sysRoot string
}

type cachedProc struct {
	info ProcInfo
	at   time.Time
}

// NewScanner builds the real scanner.
func NewScanner() *GopsutilScanner {
	return &GopsutilScanner{
		cache:   map[int32]cachedProc{},
		ttl:     15 * time.Second,
		timeout: 400 * time.Millisecond,
		sysRoot: os.Getenv("SystemRoot"),
	}
}

// Scan enumerates listening ports, enriches them with process metadata and
// returns the entries sorted by port.
func (s *GopsutilScanner) Scan(opts Options) ([]model.Entry, error) {
	socks, err := collectSockets(opts.IncludeUDP)
	if err != nil && len(socks) == 0 {
		return nil, err
	}
	groups := MergeSockets(socks)
	infos := s.enrichAll(groups)
	now := time.Now()
	out := make([]model.Entry, 0, len(groups))
	for _, g := range groups {
		if opts.OnlyLocal && GroupScope(g) != model.ScopeLocal {
			continue
		}
		p, ok := infos[g.PID]
		if !ok {
			p = ProcInfo{PID: g.PID, Denied: true, DeniedReason: i18n.T("scan.denied")}
		}
		out = append(out, BuildEntry(g, p, now, s.sysRoot))
	}
	return out, nil
}

// Invalidate drops a cached process so the next scan re-reads it.
func (s *GopsutilScanner) Invalidate(pid int32) {
	s.mu.Lock()
	delete(s.cache, pid)
	s.mu.Unlock()
}

func collectSockets(includeUDP bool) ([]RawSocket, error) {
	kinds := []string{"tcp"}
	if includeUDP {
		kinds = append(kinds, "udp")
	}
	var out []RawSocket
	var firstErr error
	for _, kind := range kinds {
		conns, err := gnet.Connections(kind)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		for _, c := range conns {
			if c.Pid <= 0 || c.Laddr.Port == 0 {
				continue
			}
			switch kind {
			case "tcp":
				if !strings.EqualFold(strings.TrimSpace(c.Status), "LISTEN") {
					continue
				}
			case "udp":
				// UDP has no LISTEN state, so only low ports count as services;
				// the ephemeral ports an application opens are skipped.
				if c.Laddr.Port >= 32768 {
					continue
				}
			}
			out = append(out, RawSocket{Proto: kind, IP: c.Laddr.IP, Port: c.Laddr.Port, PID: c.Pid})
		}
	}
	if len(out) == 0 && firstErr != nil {
		return nil, firstErr
	}
	return out, nil
}

func (s *GopsutilScanner) enrichAll(groups []Group) map[int32]ProcInfo {
	res := map[int32]ProcInfo{}
	seen := map[int32]bool{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for _, g := range groups {
		if seen[g.PID] {
			continue
		}
		seen[g.PID] = true
		wg.Add(1)
		sem <- struct{}{}
		go func(pid int32) {
			defer wg.Done()
			defer func() { <-sem }()
			info := s.info(pid)
			mu.Lock()
			res[pid] = info
			mu.Unlock()
		}(g.PID)
	}
	wg.Wait()
	return res
}

func (s *GopsutilScanner) info(pid int32) ProcInfo {
	s.mu.Lock()
	if c, ok := s.cache[pid]; ok && time.Since(c.at) < s.ttl {
		s.mu.Unlock()
		return c.info
	}
	if len(s.cache) > 512 {
		s.cache = map[int32]cachedProc{}
	}
	s.mu.Unlock()

	info := s.fetch(pid)

	s.mu.Lock()
	s.cache[pid] = cachedProc{info: info, at: time.Now()}
	s.mu.Unlock()
	return info
}

// fetch guards a single process read with a timeout: a process owned by another
// user or an elevated one can hang.
func (s *GopsutilScanner) fetch(pid int32) ProcInfo {
	ch := make(chan ProcInfo, 1)
	go func() { ch <- readProc(pid) }()
	select {
	case info := <-ch:
		return info
	case <-time.After(s.timeout):
		return ProcInfo{PID: pid, Denied: true, DeniedReason: i18n.T("scan.timeout")}
	}
}

func readProc(pid int32) ProcInfo {
	info := ProcInfo{PID: pid}
	p, err := gproc.NewProcess(pid)
	if err != nil {
		info.Denied = true
		info.DeniedReason = i18n.T("scan.inaccessible")
		return info
	}
	if name, err := p.Name(); err == nil {
		info.Name = strings.TrimSpace(name)
	}
	if cmd, err := p.Cmdline(); err == nil {
		info.Cmdline = strings.TrimSpace(cmd)
	}
	if exe, err := p.Exe(); err == nil {
		info.Exe = strings.TrimSpace(exe)
	}
	if ct, err := p.CreateTime(); err == nil {
		info.StartedAt = ct
	}
	if ppid, err := p.Ppid(); err == nil {
		info.Ppid = ppid
	}
	if cwd, err := p.Cwd(); err == nil {
		info.Cwd = strings.TrimSpace(cwd)
		info.CwdKnown = info.Cwd != ""
	}
	info.Readable = info.Name != "" || info.Cmdline != ""
	if !info.Readable {
		info.Denied = true
		info.DeniedReason = i18n.T("scan.denied")
		return info
	}

	cur := info.Ppid
	childStarted := info.StartedAt
	seen := map[int32]bool{pid: true}
	for cur > 4 {
		if seen[cur] || childStarted <= 0 {
			return info
		}
		seen[cur] = true
		ap, err := gproc.NewProcess(cur)
		if err != nil {
			if exists, checkErr := gproc.PidExists(cur); checkErr == nil && !exists {
				info.AncestorsComplete = true
			} else if cur == info.Ppid {
				// An inaccessible parent is not evidence that it exited.
				info.ParentAlive = true
			}
			return info
		}
		if cur == info.Ppid {
			info.ParentAlive = true
		}
		created, err := ap.CreateTime()
		if err != nil {
			return info
		}
		if created > childStarted {
			if cur == info.Ppid {
				info.ParentAlive = false
				info.ParentReused = true
			}
			info.AncestorsComplete = true
			return info
		}
		name, err := ap.Name()
		if err != nil {
			return info
		}
		a := Ancestor{PID: cur, Name: strings.TrimSpace(name)}
		if cur == info.Ppid {
			info.ParentName = a.Name
		}
		info.Ancestors = append(info.Ancestors, a)
		next, err := ap.Ppid()
		if err != nil {
			return info
		}
		childStarted = created
		cur = next
	}
	info.ParentAlive = info.ParentAlive || info.Ppid > 0
	info.AncestorsComplete = true
	return info
}
