// The bridge to the Go backend.
//   Wails runtime: window.go.main.App.* and window.runtime.EventsOn
//   browser preview (?mock=1): preview/mock-bridge.js, loaded on demand
//   neither: the offline state, so the interface says so instead of going blank

import { S } from './strings.js';

let backend = null;
let runtime = null;

export const bridge = {
  mode: 'offline',

  async init() {
    if (typeof window !== 'undefined' && window.go?.main?.App) {
      backend = window.go.main.App;
      runtime = window.runtime ?? null;
      this.mode = 'wails';
      return this.mode;
    }
    const wantsMock = new URLSearchParams(location.search).get('mock') === '1';
    if (wantsMock) {
      const mod = await import('../preview/mock-bridge.js');
      backend = mod.createMockBridge();
      runtime = null;
      this.mode = 'mock';
      return this.mode;
    }
    this.mode = 'offline';
    return this.mode;
  },

  on(event, handler) {
    if (this.mode === 'wails' && runtime?.EventsOn) {
      runtime.EventsOn(event, (payload) => handler(payload));
      return;
    }
    if (this.mode === 'mock' && backend?.on) {
      backend.on(event, handler);
    }
  },

  async snapshot() {
    if (!backend) return null;
    return backend.GetSnapshot();
  },

  async scan() {
    if (!backend) return null;
    return backend.ScanNow();
  },

  async setIncludeUDP(value) {
    if (!backend) return null;
    return backend.SetIncludeUDP(value);
  },

  async appInfo() {
    if (!backend) return null;
    return backend.GetAppInfo();
  },

  async updateSetting(key, value) {
    if (!backend) return null;
    if (key === 'theme') return backend.SetTheme(value);
    const settings = await backend.GetSettings();
    return backend.SaveSettings({ ...settings, [key]: value });
  },

  async kill(pids) {
    if (!backend) return { pids: [], skipped: {} };
    if (pids.length === 1) return backend.KillProcess(pids[0]);
    return backend.KillProcesses(pids);
  },

  async copy(text) {
    if (!backend) throw new Error(S.errors.noBackend);
    return backend.CopyText(text);
  },

  async openPath(path) {
    if (!backend) throw new Error(S.errors.noBackend);
    return backend.OpenPath(path);
  },

  // The repository link from Settings: the Go side opens it in the default
  // browser, so the address never comes from the frontend.
  async openRepo() {
    if (!backend) throw new Error(S.errors.noBackend);
    return backend.OpenRepo();
  },

  // Window buttons (close / minimise / maximise / restore). The preview and the
  // offline mode have no real window, so only the looks are kept.
  async windowControl(action) {
    const method = {
      close: 'CloseWindow',
      minimise: 'MinimiseWindow',
      zoom: 'ToggleMaximiseWindow',
    }[action];
    if (!method || typeof backend?.[method] !== 'function') return null;
    return backend[method]();
  },
};
