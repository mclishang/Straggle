// Text catalogue. Every user-visible string lives here, in both languages, and
// S always points at the catalogue of the active one.
//
// The Go side keeps its own catalogue (internal/i18n) for text it renders
// itself: errors, tray menu, notifications and humanized time.

export const LANGS = ['en', 'zh-CN'];

const en = {
  appName: 'Straggle',

  // Window buttons drawn by the frontend (the window is frameless).
  window: {
    group: 'Window controls',
    close: 'Close',
    minimise: 'Minimise',
    zoom: 'Maximise / restore',
  },

  // Theme button in the top bar (light ↔ dark).
  theme: {
    toDark: 'Switch to dark',
    toLight: 'Switch to light',
  },

  nav: {
    ports: 'Ports',
    cleanup: 'Cleanup',
    settings: 'Settings',
  },

  ports: {
    title: 'Local ports',
    searchPlaceholder: 'Search port, process or command',
    updated: (ago, count) => `Updated ${ago} · ${count} listening ports`,
    emptyTitle: 'No listening ports found',
    emptyBody: 'Nothing is listening on this machine right now. Start a service and refresh.',
    searchEmptyTitle: 'No matching results',
    searchEmptyBody: 'Try another port number, process name or command fragment.',
    truncated: (n) => `Many ports found, showing the first ${n}.`,
  },

  detail: {
    cmd: 'Command',
    pid: 'Process ID',
    addr: 'Listening on',
    started: 'Started',
    cwd: 'Working directory',
    orphanNote: 'The terminal of this process has closed, but the port is still held.',
    kill: 'End process',
    killing: 'Ending…',
    copy: 'Copy command',
    open: 'Open folder',
    close: 'Close details',
    unknown: 'Unknown',
    denied: 'Cannot read this process (administrator rights may be required)',
    startedUptime: (started, uptime) => `${started} · running for ${uptime}`,
    addrItem: (addr, scope) => `${addr} (${scope})`,
    addrJoin: ', ',
  },

  scope: {
    local: 'this machine only',
    lan: 'local network',
    all: 'all interfaces',
  },

  cleanup: {
    title: 'Orphan processes',
    caption: 'These processes lost their terminal but still hold a port.',
    killSelected: (n) => `End selected processes (${n})`,
    emptyTitle: 'No orphan processes',
    emptyBody: 'Every listening port still has a live terminal session behind it.',
    portsHeld: (n) => `${n} ports`,
    nothingSelected: 'No process to end',
  },

  settings: {
    title: 'Settings',
    background: { title: 'Keep running', sub: 'Stay in the tray after the window closes' },
    onlyLocal: { title: 'Loopback listeners only', sub: 'Hide 0.0.0.0 and LAN ports' },
    notifyOrphans: { title: 'Orphan notification', sub: 'Notify me when a dead terminal still holds a port' },
    confirmKill: { title: 'Confirm before ending', sub: 'Ask once before every cleanup' },
    interval: 'Refresh interval',
    intervalNote: 'Shorter is more timely and uses more power',
    seconds: (n) => `${n} seconds`,
    language: 'Language',
    languageNote: 'Applies to the window and the tray menu',
    languageSystem: 'System default',
    languageEnglish: 'English',
    languageChinese: '简体中文',
    version: 'Version',
    platform: 'Platform',
    dataDir: 'Config directory',
    repo: 'Open source',
    savedFailed: 'Could not save the settings, rolled back',
  },

  filters: {
    byPort: 'By port',
    byUptime: 'By uptime',
    byProject: 'By project',
    includeUdp: 'Include UDP bindings',
  },

  dialog: {
    cancel: 'Cancel',
    confirm: 'End process',
    killTitle: (n) => (n > 1 ? `End the ${n} selected processes?` : 'End this process?'),
    killBody: 'Straggle asks the process to close first and force-kills it after 10 seconds.',
  },

  updated: {
    justNow: 'just now',
    seconds: (n) => `${n}s ago`,
    minutes: (n) => `${n}m ago`,
    hours: (n) => `${n}h ago`,
  },

  errors: {
    init: (message) => `Initialization failed: ${message}`,
    refresh: (message) => `Refresh failed: ${message}`,
    themeSave: (message) => `Could not save the theme: ${message}`,
    killDone: (name, message) => `${name}: ${message}`,
    killFailed: (name, message) => `${name} could not be ended: ${message}`,
    noBackend: 'Not connected to the backend',
    noOrphans: 'No new orphan processes',
  },

  common: {
    nav: 'Main navigation',
    refresh: 'Refresh',
    filter: 'Filter and sort',
    orphan: 'Orphan',
    copyOk: 'Command copied to the clipboard',
    copyFail: (message) => `Copy failed: ${message}`,
    nothingToCopy: 'Nothing to copy',
    listJoin: '; ',
    offline: 'Not connected to the backend (the interface is in preview mode)',
  },
};

const zh = {
  appName: 'Straggle',

  window: {
    group: '窗口控制',
    close: '关闭',
    minimise: '最小化',
    zoom: '最大化 / 还原',
  },

  theme: {
    toDark: '切换到深色',
    toLight: '切换到浅色',
  },

  nav: {
    ports: '端口',
    cleanup: '清理',
    settings: '设置',
  },

  ports: {
    title: '本地端口',
    searchPlaceholder: '搜索端口、进程或命令',
    updated: (ago, count) => `已更新 ${ago} · 共 ${count} 个监听端口`,
    emptyTitle: '没有发现监听端口',
    emptyBody: '本机当前没有进程在监听端口。启动一个服务后再刷新试试。',
    searchEmptyTitle: '没有匹配的结果',
    searchEmptyBody: '换个端口号、进程名或命令片段再试。',
    truncated: (n) => `端口较多，仅显示前 ${n} 项。`,
  },

  detail: {
    cmd: '运行命令',
    pid: '进程 ID',
    addr: '监听地址',
    started: '启动时间',
    cwd: '工作目录',
    orphanNote: '这个进程的终端已经关闭，端口却还被占着。',
    kill: '结束进程',
    killing: '正在结束…',
    copy: '复制命令',
    open: '打开目录',
    close: '关闭详情',
    unknown: '未知',
    denied: '无法读取该进程的信息（可能需要管理员权限）',
    startedUptime: (started, uptime) => `${started} · 已运行 ${uptime}`,
    addrItem: (addr, scope) => `${addr}（${scope}）`,
    addrJoin: '，',
  },

  scope: {
    local: '仅本机',
    lan: '局域网',
    all: '所有网卡',
  },

  cleanup: {
    title: '清理孤儿进程',
    caption: '这些进程的终端已经关闭，但端口仍被占用。',
    killSelected: (n) => `结束所选进程（${n}）`,
    emptyTitle: '没有孤儿进程',
    emptyBody: '所有监听端口背后都还有活着的终端会话，无需清理。',
    portsHeld: (n) => `占用 ${n} 个端口`,
    nothingSelected: '没有可结束的进程',
  },

  settings: {
    title: '设置',
    background: { title: '后台常驻', sub: '关闭窗口后继续在托盘运行' },
    onlyLocal: { title: '只显示本机监听', sub: '隐藏 0.0.0.0 与局域网端口' },
    notifyOrphans: { title: '孤儿进程提醒', sub: '终端关闭后端口仍被占用时通知我' },
    confirmKill: { title: '结束进程前确认', sub: '每次清理都先弹一次确认' },
    interval: '刷新间隔',
    intervalNote: '间隔越短越及时，也越耗电',
    seconds: (n) => `${n} 秒`,
    language: '语言',
    languageNote: '窗口与托盘菜单同时生效',
    languageSystem: '跟随系统',
    languageEnglish: 'English',
    languageChinese: '简体中文',
    version: '版本',
    platform: '平台',
    dataDir: '配置目录',
    repo: '开源仓库',
    savedFailed: '设置保存失败，已回滚',
  },

  filters: {
    byPort: '按端口',
    byUptime: '按运行时长',
    byProject: '按项目',
    includeUdp: '包含 UDP 绑定',
  },

  dialog: {
    cancel: '取消',
    confirm: '结束进程',
    killTitle: (n) => (n > 1 ? `结束所选的 ${n} 个进程？` : '结束这个进程？'),
    killBody: '将先发送关闭请求，10 秒后仍未退出则强制结束。',
  },

  updated: {
    justNow: '刚刚',
    seconds: (n) => `${n} 秒前`,
    minutes: (n) => `${n} 分钟前`,
    hours: (n) => `${n} 小时前`,
  },

  errors: {
    init: (message) => `初始化失败：${message}`,
    refresh: (message) => `刷新失败：${message}`,
    themeSave: (message) => `主题保存失败：${message}`,
    killDone: (name, message) => `${name}：${message}`,
    killFailed: (name, message) => `${name} 未能结束：${message}`,
    noBackend: '未连接到后端',
    noOrphans: '没有新的孤儿进程',
  },

  common: {
    nav: '主导航',
    refresh: '刷新',
    filter: '筛选与排序',
    orphan: '孤儿进程',
    copyOk: '命令已复制到剪贴板',
    copyFail: (message) => `复制失败：${message}`,
    nothingToCopy: '没有可复制的内容',
    listJoin: '；',
    offline: '未连接到后端（界面处于预览模式）',
  },
};

const catalogues = { en, 'zh-CN': zh };

const systemLanguage = () =>
  String(navigator.language ?? '').toLowerCase().startsWith('zh') ? 'zh-CN' : 'en';

export const normalizeLanguage = (lang) =>
  LANGS.includes(lang) ? lang : systemLanguage();

export let S = en;

let active = 'en';

export const currentLanguage = () => active;

export function setLanguage(lang) {
  const next = normalizeLanguage(lang);
  active = next;
  S = catalogues[next];
  document.documentElement.lang = next;
  return next;
}

setLanguage(systemLanguage());
