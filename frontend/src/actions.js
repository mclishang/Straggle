// Application-level actions: navigation, refresh, ending processes, copying and
// settings. Views call these functions instead of touching the bridge.

import { bridge } from './bridge.js';
import {
  askConfirm,
  closeDialog,
  entries,
  notify,
  patch,
  showToast,
  state,
} from './store.js';
import { S, setLanguage } from './strings.js';

// The three destinations share one content area: switching only changes state,
// the subscribers repaint.
export function switchTab(screen) {
  if (state.screen !== screen) patch({ screen, menu: null });
}

// Window buttons (the traffic lights drawn by the frontend); the real action
// happens on the Go side.
export function windowControl(action) {
  bridge.windowControl(action);
}

// The port list and the details pane are two columns of one screen: selecting a
// row expands it, there is no routing involved.
export function selectPort(key) {
  patch({ detailKey: key });
}

export function closeDetail() {
  patch({ detailKey: null });
}

// ---------- Theme ----------

const prefersDark = () => window.matchMedia?.('(prefers-color-scheme: dark)').matches ?? false;

// The theme in effect: state.theme comes from the backend, and until it arrives
// the system preference decides (which is also what color-scheme does while
// <html> has no data-theme).
export const currentTheme = () => state.theme ?? (prefersDark() ? 'dark' : 'light');

// The only place that writes <html data-theme>. Colours come from the
// light-dark() tokens, so switching color-scheme is enough.
export function applyTheme(theme) {
  state.theme = theme === 'dark' ? 'dark' : 'light';
  document.documentElement.dataset.theme = state.theme;
  notify();
}

// The theme button: apply first, persist second, roll back if that fails. The
// settings returned by the backend are written back into the snapshot, otherwise
// the next "save settings" would send a stale theme.
export function toggleTheme() {
  return updateSetting('theme', currentTheme() === 'dark' ? 'light' : 'dark');
}

let settingsQueue = Promise.resolve();
const pendingSettings = new Map();
let savedSettings = null;
let savedTheme = currentTheme();

export function applySnapshot(snap) {
  if (!snap) return;
  // '' means follow the system language, which setLanguage resolves through the locale.
  setLanguage(snap.language ?? '');
  state.snapshot = snap;
  if (pendingSettings.size === 0) savedSettings = { ...snap.settings };
  if (!pendingSettings.has('theme') && snap.theme) savedTheme = snap.theme;
  state.snapshot.settings = { ...snap.settings };
  for (const [key, edit] of pendingSettings) state.snapshot.settings[key] = edit.value;
  if (snap.theme) applyTheme(pendingSettings.get('theme')?.value ?? snap.theme);
  // Drop the checked and in-flight marks of entries that are gone.
  const pids = new Set((snap.entries ?? []).map((e) => e.pid));
  for (const pid of [...state.selection.keys()]) {
    if (!pids.has(pid)) state.selection.delete(pid);
  }
  for (const pid of [...state.killing]) {
    if (!pids.has(pid)) state.killing.delete(pid);
  }
  notify();
}

export async function refresh() {
  if (state.refreshing) return;
  patch({ refreshing: true });
  try {
    const snap = await bridge.scan();
    applySnapshot(snap);
  } catch (err) {
    showToast(S.errors.refresh(err?.message ?? err));
  } finally {
    patch({ refreshing: false });
  }
}

// "Include UDP bindings" needs a fresh scan on the backend; its snapshot
// replaces the current one.
export async function setIncludeUDP(value) {
  try {
    const snap = await bridge.setIncludeUDP(value);
    if (snap) applySnapshot(snap);
  } catch (err) {
    showToast(S.errors.refresh(err?.message ?? err));
  }
}

export async function updateSetting(key, value) {
  if (!state.snapshot?.settings) return;
  savedSettings ??= { ...state.snapshot.settings };
  const edit = { value };
  pendingSettings.set(key, edit);
  state.snapshot.settings = { ...state.snapshot.settings, [key]: value };
  if (key === 'theme') applyTheme(value);
  notify();
  const save = async () => {
    try {
      const saved = await bridge.updateSetting(key, value);
      if (!saved) throw new Error(S.errors.noBackend);
      savedSettings = saved;
      if (key === 'theme') savedTheme = saved.theme;
      if (pendingSettings.get(key) === edit) pendingSettings.delete(key);
      if (saved && state.snapshot) {
        state.snapshot.settings = { ...saved };
        for (const [pendingKey, pending] of pendingSettings) {
          state.snapshot.settings[pendingKey] = pending.value;
        }
        setLanguage(saved.language);
      }
    } catch (err) {
      if (pendingSettings.get(key) === edit) pendingSettings.delete(key);
      state.snapshot.settings = { ...savedSettings };
      for (const [pendingKey, pending] of pendingSettings) {
        state.snapshot.settings[pendingKey] = pending.value;
      }
      if (key === 'theme' && !pendingSettings.has(key)) applyTheme(savedTheme);
      showToast(S.settings.savedFailed);
    }
    if (key === 'theme' && state.snapshot?.settings) {
      applyTheme((pendingSettings.get('theme')?.value ?? state.snapshot.settings.theme) || currentTheme());
    }
    notify();
  };
  const result = settingsQueue.then(save);
  settingsQueue = result;
  return result;
}

export async function killPids(pids, { confirmed = false } = {}) {
  const list = [...new Set(pids)].filter((pid) => !state.killing.has(pid));
  if (list.length === 0) {
    showToast(S.cleanup.nothingSelected);
    return;
  }
  if (!confirmed && state.snapshot?.settings?.confirmKill) {
    askConfirm({
      title: S.dialog.killTitle(list.length),
      body: S.dialog.killBody,
      confirmText: S.dialog.confirm,
      onConfirm: () => {
        closeDialog();
        killPids(list, { confirmed: true });
      },
    });
    return;
  }

  for (const pid of list) state.killing.add(pid);
  notify();
  try {
    const res = await bridge.kill(list);
    const started = new Set(res?.pids ?? []);
    const skipped = res?.skipped ?? {};
    for (const pid of list) {
      if (!started.has(pid)) state.killing.delete(pid);
    }
    const reasons = Object.values(skipped);
    if (reasons.length > 0) showToast(reasons.join(S.common.listJoin));
  } catch (err) {
    for (const pid of list) state.killing.delete(pid);
    showToast(String(err?.message ?? err));
  }
  notify();
}

export function onKillResult(report) {
  state.killing.delete(report.pid);
  notify();
  const name = entries().find((e) => e.pid === report.pid)?.headline ?? `PID ${report.pid}`;
  showToast(
    report.ok
      ? S.errors.killDone(name, report.message)
      : S.errors.killFailed(name, report.message),
  );
}

export async function copyText(text, okMessage = S.common.copyOk) {
  if (!text) {
    showToast(S.common.nothingToCopy);
    return;
  }
  try {
    if (navigator.clipboard?.writeText) {
      try {
        await navigator.clipboard.writeText(text);
        showToast(okMessage);
        return;
      } catch {
        // fall through to the Go-side clipboard
      }
    }
    await bridge.copy(text);
    showToast(okMessage);
  } catch (err) {
    showToast(S.common.copyFail(err?.message ?? err));
  }
}

export async function openPath(path) {
  if (!path) return;
  try {
    await bridge.openPath(path);
  } catch (err) {
    showToast(String(err?.message ?? err));
  }
}

export async function openRepo() {
  try {
    await bridge.openRepo();
  } catch (err) {
    showToast(String(err?.message ?? err));
  }
}
