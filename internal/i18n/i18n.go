// Package i18n holds the text the Go side renders and selects the language it is
// rendered in. The catalogue pairs both languages per key, so a key can never
// exist in only one of them.
package i18n

import (
	"fmt"
	"strings"
	"sync"

	"golang.org/x/sys/windows"
)

// Lang is a supported interface language.
type Lang string

// Supported languages. An empty stored value means "follow the system".
const (
	EN Lang = "en"
	ZH Lang = "zh-CN"
)

var (
	mu  sync.RWMutex
	cur = EN
)

// Parse maps a stored setting to a language, falling back to the system language.
func Parse(v string) Lang {
	switch Lang(strings.TrimSpace(v)) {
	case EN:
		return EN
	case ZH:
		return ZH
	default:
		return Detect()
	}
}

// Valid reports whether v is a language the settings screen may store, including
// the empty value that means "follow the system".
func Valid(v string) bool {
	switch Lang(v) {
	case "", EN, ZH:
		return true
	default:
		return false
	}
}

// Detect returns the language Windows is configured for, defaulting to English.
func Detect() Lang {
	langs, err := windows.GetUserPreferredUILanguages(windows.MUI_LANGUAGE_NAME)
	if err != nil || len(langs) == 0 {
		return EN
	}
	if strings.HasPrefix(strings.ToLower(langs[0]), "zh") {
		return ZH
	}
	return EN
}

// Set selects the language used by T.
func Set(l Lang) {
	mu.Lock()
	cur = l
	mu.Unlock()
}

// Current returns the selected language.
func Current() Lang {
	mu.RLock()
	defer mu.RUnlock()
	return cur
}

// messages maps a key to its English and Simplified Chinese text. Templates use
// fmt verbs; both texts of a key take the same arguments.
var messages = map[string][2]string{
	"app.startFailed": {"Straggle failed to start: %v", "Straggle 启动失败：%v"},

	"common.unknown": {"Unknown", "未知"},
	"common.joiner":  {"; ", "；"},

	"error.saveSettings":  {"Cannot save settings: %v", "保存设置失败：%v"},
	"error.notReady":      {"The application is not ready yet", "应用尚未就绪"},
	"error.nothingToCopy": {"Nothing to copy", "没有可复制的内容"},
	"error.emptyPath":     {"The path is empty", "路径为空"},
	"error.pathMissing":   {"The path does not exist: %s", "路径不存在：%s"},
	"error.openPath":      {"Cannot open the folder: %v", "打开失败：%v"},
	"error.scanFailed":    {"Port scan failed: %v", "端口扫描失败：%v"},
	"error.staleEntry":    {"The process information is out of date, refresh and try again", "进程信息已过期，请刷新后重试"},
	"error.skippedEntry":  {"PID %d: %s", "PID %d：%s"},

	"guard.systemPid": {"A system process cannot be ended", "系统进程不能结束"},
	"guard.self":      {"Straggle cannot end itself", "不能结束 Straggle 自身"},
	"guard.critical":  {"Refused to end the critical system process %s", "已阻止结束关键系统进程 %s"},
	"guard.systemDir": {"Refused to end a process inside the Windows system directory", "已阻止结束 Windows 系统目录下的进程"},

	"kill.systemPid":       {"System processes (PID ≤ 4) cannot be ended", "系统进程（PID ≤ 4）不能结束"},
	"kill.inflight":        {"This process is already being ended", "该进程正在结束中"},
	"kill.identityUnknown": {"Cannot end a process without a verified creation time; refresh and try again", "无法结束创建时间未确认的进程，请刷新后重试"},
	"kill.identityFailed":  {"Cannot verify the process identity: %v", "无法确认进程身份：%v"},
	"kill.terminateFailed": {"Cannot terminate the process: %v", "无法终止进程：%v"},
	"kill.gone":            {"The process had already exited", "进程已经退出"},
	"kill.term":            {"Ended gracefully", "已优雅结束"},
	"kill.kill":            {"Force-killed", "已强制结束"},
	"kill.forceFailed":     {"Force-kill failed: %v", "强制结束失败：%v"},
	"kill.graceFailed":     {" (the graceful request failed too: %v)", "（优雅结束同样失败：%v）"},
	"kill.stillRunning":    {"The process is still running, administrator rights may be required", "进程仍在运行，可能需要管理员权限"},
	"kill.noSuchProcess":   {"The process does not exist", "进程不存在"},
	"kill.denied":          {"The system refused to end this process (exit code 1)", "系统拒绝结束该进程（退出码 1）"},
	"kill.taskkill":        {"taskkill failed (exit code %d)", "taskkill 失败（退出码 %d）"},

	"scan.unknownProcess": {"Unknown process", "未知进程"},
	"scan.debugSuffix":    {" (debug)", " 调试"},
	"scan.unknownDir":     {"Unknown folder", "未知目录"},
	"scan.denied":         {"Cannot read this process (administrator rights may be required)", "无法读取进程信息（可能需要管理员权限）"},
	"scan.timeout":        {"Timed out reading this process (administrator rights may be required)", "读取进程信息超时（可能需要管理员权限）"},
	"scan.inaccessible":   {"Cannot access this process (administrator rights may be required)", "无法访问该进程（可能需要管理员权限）"},
	"scan.noParent":       {"No parent process recorded", "无父进程记录"},
	"scan.parent":         {"Parent process %d", "父进程 %d"},
	"scan.parentNamed":    {"Parent process %d (%s)", "父进程 %d（%s）"},
	"scan.parentGone":     {"Parent process %d has exited", "父进程 %d 已退出"},
	"scan.terminalGone":   {"The terminal that started it has closed", "启动它的终端已经关闭"},

	"time.underMinute":  {"under a minute", "不到 1 分钟"},
	"time.minuteOne":    {"%d minute", "%d 分钟"},
	"time.minutes":      {"%d minutes", "%d 分钟"},
	"time.hourOne":      {"%d hour", "%d 小时"},
	"time.hours":        {"%d hours", "%d 小时"},
	"time.hoursMinutes": {"%s %s", "%s %s"},
	"time.dayOne":       {"%d day", "%d 天"},
	"time.days":         {"%d days", "%d 天"},
	"time.today":        {"Today %s", "今天 %s"},
	"time.yesterday":    {"Yesterday %s", "昨天 %s"},
	"time.stamped":      {"%s at %s", "%s %s"},
	"time.layoutDay":    {"Jan 02", "01月02日"},
	"time.layoutDate":   {"Jan 02, 2006", "2006年01月02日"},

	"tray.show":        {"Show window", "显示窗口"},
	"tray.showHint":    {"Restore the main window", "恢复主界面"},
	"tray.hide":        {"Hide window", "隐藏窗口"},
	"tray.hideHint":    {"Hide to the tray", "隐藏到托盘"},
	"tray.cleanup":     {"Clean orphan processes (%d)", "清理孤儿进程（%d）"},
	"tray.cleanupHint": {"Open the cleanup screen", "打开清理页"},
	"tray.quit":        {"Quit", "退出"},
	"tray.quitHint":    {"Quit Straggle", "退出 Straggle"},
	"tray.tooltip":     {"Straggle · %d listening · %d orphan", "Straggle · 监听 %d · 孤儿 %d"},

	"notify.orphansTitle":   {"Orphan processes found", "发现孤儿进程"},
	"notify.orphansBodyOne": {"%d port is held by a process whose terminal has closed", "有 %d 个端口被已关闭终端的进程占用"},
	"notify.orphansBody":    {"%d ports are held by processes whose terminal has closed", "有 %d 个端口被已关闭终端的进程占用"},
}

// T renders the message for key in the selected language. An unknown key returns
// the key itself, which makes the mistake visible instead of hiding it.
func T(key string, args ...any) string {
	text, ok := messages[key]
	if !ok {
		return key
	}
	tmpl := text[0]
	if Current() == ZH {
		tmpl = text[1]
	}
	if len(args) == 0 {
		return tmpl
	}
	return fmt.Sprintf(tmpl, args...)
}

// Tn renders one and many depending on n: English inflects nouns, Simplified
// Chinese does not and stores the same text under both keys.
func Tn(one, many string, n int, args ...any) string {
	if Current() == EN && n == 1 {
		return T(one, args...)
	}
	return T(many, args...)
}
