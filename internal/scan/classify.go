// Package scan enumerates local listening ports, enriches them with process
// metadata, and decides which of them are orphans.
package scan

import (
	"net"
	"sort"
	"strconv"
	"strings"

	"Straggle/internal/model"
)

// RawSocket is one listening entry straight from the system, neither merged nor
// enriched.
type RawSocket struct {
	Proto string // tcp / udp
	IP    string
	Port  uint32
	PID   int32
}

// Group is every listening address of one process on one port, with IPv4 and
// IPv6 merged.
type Group struct {
	Proto string
	Port  uint32
	PID   int32
	IPs   []string
}

// MergeSockets groups raw sockets by (protocol, port, PID), so a service that
// listens on IPv4 and IPv6 shows up as a single entry.
func MergeSockets(socks []RawSocket) []Group {
	type key struct {
		proto string
		port  uint32
		pid   int32
	}
	var order []key
	seen := map[key]map[string]bool{}
	for _, s := range socks {
		if s.Port == 0 || s.PID <= 0 {
			continue
		}
		k := key{proto: strings.ToLower(s.Proto), port: s.Port, pid: s.PID}
		ips, ok := seen[k]
		if !ok {
			ips = map[string]bool{}
			seen[k] = ips
			order = append(order, k)
		}
		ip := strings.TrimSpace(s.IP)
		if ip == "" {
			ip = "0.0.0.0"
		}
		ips[ip] = true
	}
	out := make([]Group, 0, len(order))
	for _, k := range order {
		ips := make([]string, 0, len(seen[k]))
		for ip := range seen[k] {
			ips = append(ips, ip)
		}
		sort.Slice(ips, func(i, j int) bool { return ipLess(ips[i], ips[j]) })
		out = append(out, Group{Proto: k.proto, Port: k.port, PID: k.pid, IPs: ips})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Port != out[j].Port {
			return out[i].Port < out[j].Port
		}
		if out[i].PID != out[j].PID {
			return out[i].PID < out[j].PID
		}
		return out[i].Proto < out[j].Proto
	})
	return out
}

// ipLess keeps IPv4 ahead of IPv6 so the rendered order is stable.
func ipLess(a, b string) bool {
	av4 := strings.Contains(a, ".")
	bv4 := strings.Contains(b, ".")
	if av4 != bv4 {
		return av4
	}
	return a < b
}

// ScopeOf reports how far a listening address can be reached. IPv6 link-local
// addresses may carry a %zone suffix.
func ScopeOf(ip string) model.Scope {
	raw := strings.TrimSpace(ip)
	if i := strings.Index(raw, "%"); i >= 0 {
		raw = raw[:i]
	}
	parsed := net.ParseIP(raw)
	if parsed == nil {
		if raw == "" || raw == "0.0.0.0" || raw == "::" {
			return model.ScopeAll
		}
		return model.ScopeLAN
	}
	switch {
	case parsed.IsUnspecified():
		return model.ScopeAll
	case parsed.IsLoopback():
		return model.ScopeLocal
	default:
		return model.ScopeLAN
	}
}

// GroupScope returns the widest scope among the group's addresses.
func GroupScope(g Group) model.Scope {
	scope := model.ScopeLocal
	for _, ip := range g.IPs {
		switch ScopeOf(ip) {
		case model.ScopeAll:
			return model.ScopeAll
		case model.ScopeLAN:
			scope = model.ScopeLAN
		}
	}
	return scope
}

// AddrLabel formats a single listening address.
func AddrLabel(ip string, port uint32) string {
	host := strings.TrimSpace(ip)
	if host == "" {
		host = "0.0.0.0"
	}
	if strings.Contains(host, ":") {
		return "[" + host + "]:" + strconv.FormatUint(uint64(port), 10)
	}
	return host + ":" + strconv.FormatUint(uint64(port), 10)
}
