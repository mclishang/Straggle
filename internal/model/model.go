// Package model defines Straggle's domain model: the JSON contract shared by the
// scanner, the kill layer and the frontend.
package model

// Scope is how far a listening address can be reached.
type Scope string

const (
	// ScopeLocal is reachable from this machine only (127.0.0.1 / ::1).
	ScopeLocal Scope = "local"
	// ScopeLAN is reachable from the local network (bound to a NIC address).
	ScopeLAN Scope = "lan"
	// ScopeAll is reachable on every interface (0.0.0.0 / ::).
	ScopeAll Scope = "all"
)

// Addr is one listening address, already formatted for display.
type Addr struct {
	Addr  string `json:"addr"`  // 127.0.0.1:3000 or [::1]:3000
	IP    string `json:"ip"`    // raw IP
	Port  uint32 `json:"port"`  // port
	Scope Scope  `json:"scope"` // reach
}

// Entry is one process listening on one port.
type Entry struct {
	Key          string `json:"key"`
	Port         uint32 `json:"port"`
	Proto        string `json:"proto"`
	PID          int32  `json:"pid"`
	ProcessName  string `json:"processName"` // image name (node.exe); empty when unreadable
	DisplayName  string `json:"displayName"` // list title, may be an inferred tool name (vite)
	Headline     string `json:"headline"`    // "3000 · node"
	Supporting   string `json:"supporting"`  // "PID 48213 · agent-web · 12 minutes"
	Cmdline      string `json:"cmdline"`
	Exe          string `json:"exe"`
	Cwd          string `json:"cwd"`
	CwdKnown     bool   `json:"cwdKnown"`
	Project      string `json:"project"`
	Addrs        []Addr `json:"addrs"`
	Scope        Scope  `json:"scope"`
	StartedAt    int64  `json:"startedAt"`    // Unix milliseconds
	UptimeHuman  string `json:"uptimeHuman"`  // "12 minutes"
	StartedHuman string `json:"startedHuman"` // "Today 14:02"

	Orphan       bool   `json:"orphan"`
	OrphanReason string `json:"orphanReason"`

	ParentPID   int32  `json:"parentPid"`
	ParentName  string `json:"parentName"`
	ParentAlive bool   `json:"parentAlive"`
	ParentNote  string `json:"parentNote"` // "Parent process 48190 has exited"

	Readable     bool   `json:"readable"`
	Denied       bool   `json:"denied"`
	DeniedReason string `json:"deniedReason"`
}

// Theme values. The empty string follows the system, which is also the state of a
// user who has never pressed the theme button.
const (
	ThemeSystem = ""
	ThemeLight  = "light"
	ThemeDark   = "dark"
)

// Settings is the user's preferences, persisted to %APPDATA%\Straggle\settings.json.
type Settings struct {
	Background    bool   `json:"background"`    // keep running in the tray after the window closes
	OnlyLocal     bool   `json:"onlyLocal"`     // hide 0.0.0.0 and LAN listeners
	NotifyOrphans bool   `json:"notifyOrphans"` // notify when a port is left held by a dead terminal
	ConfirmKill   bool   `json:"confirmKill"`   // ask before ending processes
	IntervalSec   int    `json:"intervalSec"`   // refresh interval: 1 / 5 / 10
	Theme         string `json:"theme"`         // "" follows the system / light / dark
	Language      string `json:"language"`      // "" follows the system / en / zh-CN
}

// DefaultSettings returns the factory defaults.
func DefaultSettings() Settings {
	return Settings{
		Background:    true,
		OnlyLocal:     false,
		NotifyOrphans: true,
		ConfirmKill:   true,
		IntervalSec:   5,
	}
}

// Snapshot is the result of one scan and the single source of truth for the UI.
type Snapshot struct {
	GeneratedAt    int64    `json:"generatedAt"`
	Theme          string   `json:"theme"`    // light / dark
	Language       string   `json:"language"` // en / zh-CN, with "" resolved to the system
	Settings       Settings `json:"settings"`
	Entries        []Entry  `json:"entries"`
	Warning        string   `json:"warning"`
	ListeningCount int      `json:"listeningCount"`
	OrphanCount    int      `json:"orphanCount"`
	Truncated      bool     `json:"truncated"`
}

// KillStarted is the receipt for an accepted kill request; the outcome arrives
// later on the kill:result event.
type KillStarted struct {
	PIDs    []int32          `json:"pids"`
	Skipped map[int32]string `json:"skipped"` // PID → why it was not accepted
}

// KillReport is the final outcome for one process.
type KillReport struct {
	PID     int32  `json:"pid"`
	OK      bool   `json:"ok"`
	Stage   string `json:"stage"` // term | kill | gone | denied
	Message string `json:"message"`
}

// Kill stages.
const (
	StageTerm   = "term"   // ended gracefully
	StageKill   = "kill"   // force-killed
	StageGone   = "gone"   // already exited before the request
	StageDenied = "denied" // not permitted, or the force-kill failed
)

// AppInfo describes the running application.
type AppInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	OS      string `json:"os"`
	DataDir string `json:"dataDir"`
	RepoURL string `json:"repoUrl"`
}
