// Fake backend for the browser preview, loaded on demand when the URL carries
// ?mock=1. It renders the same data shape as the real one (model.Entry) so the
// interface can be developed and screenshotted without WebView2 or the Go
// backend. The application never loads this file.

import { setLanguage } from '../src/strings.js';

const MIN = 60_000;
const HOUR = 60 * MIN;

// The preview stands in for the backend, so it also stands in for the backend's
// own text catalogue (internal/i18n).
const TEXT = {
  en: {
    underMinute: 'under a minute',
    minutes: (n) => `${n} minutes`,
    hours: (n) => `${n} hours`,
    hoursMinutes: (h, m) => `${h} hours ${m} minutes`,
    days: (n) => `${n} days`,
    clock: (d) =>
      `${new Intl.DateTimeFormat('en', { month: 'short', day: '2-digit' }).format(d)} at ${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`,
    parentGone: (pid) => `Parent process ${pid} has exited`,
    parentAlive: (pid, name) => `Parent process ${pid} (${name})`,
    terminalGone: 'The terminal that started it has closed',
    term: 'Ended gracefully',
    kill: 'Force-killed',
    debugName: 'node (debug)',
    database: 'database',
  },
  'zh-CN': {
    underMinute: '不到 1 分钟',
    minutes: (n) => `${n} 分钟`,
    hours: (n) => `${n} 小时`,
    hoursMinutes: (h, m) => `${h} 小时 ${m} 分钟`,
    days: (n) => `${n} 天`,
    clock: (d) =>
      `${d.getMonth() + 1}月${String(d.getDate()).padStart(2, '0')}日 ${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`,
    parentGone: (pid) => `父进程 ${pid} 已退出`,
    parentAlive: (pid, name) => `父进程 ${pid}（${name}）`,
    terminalGone: '启动它的终端已经关闭',
    term: '已优雅结束',
    kill: '已强制结束',
    debugName: 'node 调试',
    database: '数据库',
  },
};

const query = () => new URLSearchParams(location.search);

function previewTheme() {
  const q = query().get('theme');
  if (q === 'dark' || q === 'light') return q;
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

export function createMockBridge() {
  let language = setLanguage(query().get('lang'));
  let T = TEXT[language];

  const ago = (minutes) => Date.now() - minutes * MIN;

  const duration = (minutes) => {
    if (minutes < 1) return T.underMinute;
    if (minutes < 60) return T.minutes(Math.round(minutes));
    if (minutes < 24 * 60) {
      const h = Math.floor(minutes / 60);
      const m = Math.round(minutes % 60);
      return m === 0 ? T.hours(h) : T.hoursMinutes(h, m);
    }
    return T.days(Math.floor(minutes / (24 * 60)));
  };

  const startTime = (minutes) => T.clock(new Date(Date.now() - minutes * MIN));

  const entry = (spec) => {
    const startedAt = ago(spec.minutes);
    const addrs = (spec.ips ?? ['127.0.0.1']).map((ip) => ({
      addr: ip.includes(':') ? `[${ip}]:${spec.port}` : `${ip}:${spec.port}`,
      ip,
      port: spec.port,
      scope: ip === '0.0.0.0' || ip === '::' ? 'all' : ip.startsWith('127.') ? 'local' : 'lan',
    }));
    const scope = addrs.some((a) => a.scope === 'all')
      ? 'all'
      : addrs.some((a) => a.scope === 'lan')
        ? 'lan'
        : 'local';
    return {
      key: `${spec.proto ?? 'tcp'}:${spec.port}:${spec.pid}`,
      port: spec.port,
      proto: spec.proto ?? 'tcp',
      pid: spec.pid,
      processName: spec.prog ?? `${spec.name}.exe`,
      displayName: spec.name,
      headline: `${spec.port} · ${spec.name}`,
      supporting: [`PID ${spec.pid}`, spec.project, duration(spec.minutes)].join(' · '),
      cmdline: spec.cmdline,
      exe: spec.exe ?? '',
      cwd: spec.cwd ?? '',
      cwdKnown: !!spec.cwd,
      project: spec.project,
      addrs,
      scope,
      startedAt,
      uptimeHuman: duration(spec.minutes),
      startedHuman: startTime(spec.minutes),
      orphan: !!spec.orphan,
      orphanReason: spec.orphanReason ?? '',
      parentPid: spec.parentPid ?? 0,
      parentName: spec.parentName ?? '',
      parentAlive: !spec.orphan,
      parentNote: spec.orphan
        ? T.parentGone(spec.parentPid)
        : T.parentAlive(spec.parentPid, spec.parentName ?? 'explorer'),
      readable: true,
      denied: false,
      deniedReason: '',
    };
  };

  const buildEntries = () => [
    entry({
      port: 3000,
      pid: 48213,
      name: 'node',
      prog: 'node.exe',
      exe: 'C:\\Program Files\\nodejs\\node.exe',
      project: 'agent-web',
      cwd: 'C:\\work\\agent-web',
      cmdline: 'node server.js --port 3000',
      minutes: 12,
      orphan: true,
      orphanReason: T.parentGone(48190),
      parentPid: 48190,
    }),
    entry({
      port: 5173,
      pid: 48501,
      name: 'vite',
      prog: 'node.exe',
      exe: 'C:\\Program Files\\nodejs\\node.exe',
      project: 'agent-web',
      cwd: 'C:\\work\\agent-web',
      cmdline: 'node C:\\work\\agent-web\\node_modules\\vite\\bin\\vite.js',
      minutes: 9,
      orphan: true,
      orphanReason: T.terminalGone,
      parentPid: 48490,
    }),
    entry({
      port: 8080,
      pid: 47777,
      name: 'python',
      prog: 'python.exe',
      exe: 'C:\\Python312\\python.exe',
      project: 'api-server',
      cwd: 'C:\\work\\api-server',
      cmdline: 'python -m uvicorn app.main:app --port 8080',
      minutes: 31,
      orphan: true,
      orphanReason: T.parentGone(47600),
      parentPid: 47600,
    }),
    entry({
      port: 5432,
      pid: 31200,
      name: 'postgres',
      prog: 'postgres.exe',
      exe: 'C:\\Program Files\\PostgreSQL\\16\\bin\\postgres.exe',
      project: T.database,
      cwd: 'C:\\Program Files\\PostgreSQL\\16\\data',
      cmdline: 'postgres -D "C:\\Program Files\\PostgreSQL\\16\\data"',
      minutes: 2 * 24 * 60,
      parentPid: 31000,
      parentName: 'services.exe',
    }),
    entry({
      port: 9229,
      pid: 48501,
      name: T.debugName,
      prog: 'node.exe',
      exe: 'C:\\Program Files\\nodejs\\node.exe',
      project: 'agent-web',
      cwd: 'C:\\work\\agent-web',
      cmdline: 'node --inspect=9229 C:\\work\\agent-web\\node_modules\\vite\\bin\\vite.js',
      minutes: 9,
      orphan: true,
      orphanReason: T.terminalGone,
      parentPid: 48490,
    }),
  ];

  let entries = buildEntries();
  let settings = {
    background: true,
    onlyLocal: false,
    notifyOrphans: true,
    confirmKill: true,
    intervalSec: 5,
    theme: '',
    language: query().get('lang') ?? '',
  };
  const handlers = new Map();

  const emit = (event, payload) => {
    for (const fn of handlers.get(event) ?? []) fn(payload);
  };

  // The theme the preview shows: an explicit choice first, then ?theme=, then
  // the browser preference. On a real machine resolveDark does this in Go.
  const build = () => ({
    generatedAt: Date.now(),
    theme: settings.theme || previewTheme(),
    language,
    settings,
    entries,
    warning: '',
    listeningCount: entries.length,
    orphanCount: entries.filter((e) => e.orphan).length,
    truncated: false,
  });

  return {
    GetSnapshot: async () => build(),
    GetSettings: async () => ({ ...settings }),
    ScanNow: async () => build(),
    GetAppInfo: async () => ({
      name: 'Straggle',
      version: '0.1.1',
      os: 'windows/amd64',
      dataDir: 'C:\\Users\\demo\\AppData\\Roaming\\Straggle',
      repoUrl: 'https://github.com/mclishang/Straggle',
    }),
    SaveSettings: async (next) => {
      const languageChanged = next.language !== settings.language;
      settings = { ...next };
      if (languageChanged) {
        // The real backend rescans and reports the new language in the snapshot;
        // the fake one rebuilds the text it fabricates and does the same.
        language = setLanguage(settings.language);
        T = TEXT[language];
        entries = buildEntries();
        emit('snapshot:update', build());
      }
      return settings;
    },
    SetTheme: async (theme) => {
      settings = { ...settings, theme };
      emit('theme:changed', { dark: settings.theme === 'dark' });
      emit('snapshot:update', build());
      return settings;
    },
    SetIncludeUDP: async () => build(),
    KillProcess: async (pid) => {
      window.setTimeout(() => {
        entries = entries.filter((e) => e.pid !== pid);
        emit('kill:result', { pid, ok: true, stage: 'term', message: T.term });
        emit('snapshot:update', build());
      }, 900);
      return { pids: [pid], skipped: {} };
    },
    KillProcesses: async (pids) => {
      const list = [...pids];
      window.setTimeout(() => {
        entries = entries.filter((e) => !list.includes(e.pid));
        for (const pid of list) {
          emit('kill:result', { pid, ok: true, stage: 'kill', message: T.kill });
        }
        emit('snapshot:update', build());
      }, 1200);
      return { pids: list, skipped: {} };
    },
    CopyText: async () => {},
    OpenPath: async () => {},
    OpenRepo: async () => {},
    on: (event, handler) => {
      if (!handlers.has(event)) handlers.set(event, []);
      handlers.get(event).push(handler);
    },
    _hour: HOUR,
  };
}
