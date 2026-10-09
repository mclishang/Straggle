// Minimal subscription store: views read state and call notify() after changing
// it. Facts such as ports, processes and settings belong to the backend
// snapshot; this holds interface state only.

import { S } from './strings.js';

const listeners = new Set();

export const state = {
  screen: 'ports', // ports | cleanup | settings
  detailKey: null, // key of the entry shown in the details pane; null = none
  snapshot: null, // backend snapshot, the single source of truth
  appInfo: null,
  offline: false, // no backend (browser preview or a backend failure)
  refreshing: false,
  search: '',
  sort: 'port', // port | uptime | project
  includeUdp: false,
  theme: null, // 'light' | 'dark'; null = not known yet, so <html> has no data-theme
  menu: null, // 'tune' | null
  selection: new Map(), // pid -> checked (an unrecorded orphan counts as checked)
  killing: new Set(), // pids currently being ended
  toast: null, // { text }
  banner: null, // { text }
  dialog: null, // { title, body, confirmText, onConfirm }
};

let toastTimer = null;

export function subscribe(fn) {
  listeners.add(fn);
}

export function notify() {
  for (const fn of listeners) {
    try {
      fn();
    } catch (err) {
      console.error('[straggle] render failed', err);
    }
  }
}

export function patch(partial) {
  Object.assign(state, partial);
  notify();
}

export function showToast(text, ms = 3200) {
  state.toast = { text };
  notify();
  if (toastTimer) clearTimeout(toastTimer);
  toastTimer = setTimeout(() => {
    state.toast = null;
    notify();
  }, ms);
}

export function showBanner(text) {
  state.banner = { text };
  notify();
}

export function askConfirm({ title, body, confirmText, onConfirm }) {
  state.dialog = { title, body, confirmText, onConfirm };
  notify();
}

export function closeDialog() {
  state.dialog = null;
  notify();
}

// ---- Derived queries ----

export function entries() {
  return state.snapshot?.entries ?? [];
}

export function entryByKey(key) {
  return entries().find((e) => e.key === key) ?? null;
}

export function orphans() {
  return entries().filter((e) => e.orphan);
}

// Orphans are grouped per process: one process can hold several ports (several
// entries with the same PID), and the cleanup screen lists processes.
export function orphanProcesses() {
  const byPid = new Map();
  for (const entry of orphans()) {
    const found = byPid.get(entry.pid);
    if (found) {
      found.ports.push(entry.port);
    } else {
      byPid.set(entry.pid, { entry, ports: [entry.port] });
    }
  }
  return [...byPid.values()];
}

export function visibleEntries() {
  const q = state.search.trim().toLowerCase();
  const list = entries().filter((e) => {
    if (!q) return true;
    const hay = [
      String(e.port),
      e.headline,
      e.displayName,
      e.processName,
      e.cmdline,
      e.project,
      e.cwd,
      String(e.pid),
      ...(e.addrs ?? []).map((a) => a.addr),
    ];
    return hay.some((v) => v && String(v).toLowerCase().includes(q));
  });

  switch (state.sort) {
    case 'uptime':
      // Longest running first, which is the earliest start time.
      list.sort(
        (a, b) =>
          (a.startedAt || Number.MAX_SAFE_INTEGER) - (b.startedAt || Number.MAX_SAFE_INTEGER) ||
          a.port - b.port,
      );
      break;
    case 'project':
      list.sort(
        (a, b) => String(a.project).localeCompare(String(b.project), 'zh') || a.port - b.port,
      );
      break;
    default:
      list.sort((a, b) => a.port - b.port || a.pid - b.pid);
  }
  return list;
}

export function isSelected(pid) {
  return state.selection.has(pid) ? state.selection.get(pid) : true;
}

export function setSelected(pid, value) {
  state.selection.set(pid, value);
  notify();
}

// "Updated x ago": the backend sends a timestamp, the wording is derived here.
export function updatedAgo() {
  const ts = state.snapshot?.generatedAt;
  if (!ts) return '—';
  const sec = Math.max(0, Math.round((Date.now() - ts) / 1000));
  if (sec < 5) return S.updated.justNow;
  if (sec < 60) return S.updated.seconds(sec);
  const min = Math.floor(sec / 60);
  if (min < 60) return S.updated.minutes(min);
  return S.updated.hours(Math.floor(min / 60));
}
