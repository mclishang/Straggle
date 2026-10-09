// Entry point: assembles the shell (top bar, side navigation, content area),
// subscribes to backend events and wires the keyboard.

import './theme/tokens.css';
import './theme/app.css';
import '@fontsource/roboto/400.css';
import '@fontsource/roboto/500.css';
import '@fontsource/roboto/700.css';
import 'material-symbols/rounded.css';
import '@material/web/all.js';

import { html, nothing, render } from 'lit';
import { bridge } from './bridge.js';
import { S, setLanguage } from './strings.js';
import {
  closeDialog,
  entryByKey,
  notify,
  orphans,
  showBanner,
  showToast,
  state,
  subscribe,
} from './store.js';
import {
  applySnapshot,
  applyTheme,
  closeDetail,
  currentTheme,
  killPids,
  onKillResult,
  refresh,
  switchTab,
  toggleTheme,
} from './actions.js';
import { appBar, appBarActions, dialogView, navItems, toastView } from './components/ui.js';
import { windowControls } from './components/window-controls.js';
import { cleanupView } from './views/cleanup.js';
import { portsView } from './views/ports.js';
import { settingsView } from './views/settings.js';

const app = document.getElementById('app');

render(
  html`
    <div class="app">
      ${appBar()}
      <div class="app__body">
        <nav class="sidebar" aria-label=${S.common.nav}></nav>
        <main class="content"></main>
      </div>
      <div class="overlay-host"></div>
    </div>
    <div class="win-host"></div>
  `,
  app,
);

const navEl = app.querySelector('.sidebar');
const contentEl = app.querySelector('.content');
const overlayEl = app.querySelector('.overlay-host');
// The actions area on the right of the top bar (the theme button) follows the
// state, so it is filled on every repaint.
const barToolsEl = app.querySelector('.appbar__tools');
// The window buttons belong to the window, not to a screen: pinned top right.
const winEl = app.querySelector('.win-host');

function viewFor(screen) {
  switch (screen) {
    case 'cleanup':
      return cleanupView();
    case 'settings':
      return settingsView();
    default:
      return portsView();
  }
}

function paint() {
  render(windowControls(), winEl);
  navEl.setAttribute('aria-label', S.common.nav);
  render(viewFor(state.screen), contentEl);
  render(
    appBarActions({ dark: currentTheme() === 'dark', onToggleTheme: toggleTheme }),
    barToolsEl,
  );
  render(
    navItems({
      active: state.screen,
      onSelect: switchTab,
      orphanCount: state.snapshot?.orphanCount ?? 0,
    }),
    navEl,
  );
  render(
    html`${dialogView(state.dialog)}${state.toast ? toastView(state.toast) : nothing}`,
    overlayEl,
  );
}

function onKeydown(ev) {
  if (ev.key === 'Escape') {
    if (state.dialog) {
      closeDialog();
      return;
    }
    if (state.detailKey) closeDetail();
    return;
  }
  if ((ev.ctrlKey && ev.key.toLowerCase() === 'r') || ev.key === 'F5') {
    ev.preventDefault();
    refresh();
  }
}

async function boot() {
  // ?lang= has to be applied before the mock backend is loaded, because the fake
  // data carries text in the language it was built with.
  const params = new URLSearchParams(location.search);
  if (params.get('lang')) setLanguage(params.get('lang'));

  const mode = await bridge.init();
  state.offline = mode === 'offline';

  // Deep links for the preview mode (?mock=1), used for screenshots and
  // debugging; the Wails runtime ignores them because mode !== 'mock'.
  //   ?mock=1&screen=cleanup&key=tcp:3000:48213&theme=dark&lang=en&dialog=kill
  if (mode === 'mock') {
    const theme = params.get('theme');
    if (theme === 'dark' || theme === 'light') {
      applyTheme(theme);
    }
    const screen = params.get('screen');
    if (screen && ['ports', 'cleanup', 'settings'].includes(screen)) {
      state.screen = screen;
    }
    if (params.get('key')) state.detailKey = params.get('key');
  }

  subscribe(paint);
  paint();

  bridge.on('snapshot:update', (snap) => applySnapshot(snap));
  bridge.on('kill:result', (report) => onKillResult(report));
  bridge.on('theme:changed', (payload) => {
    applyTheme(payload?.dark ? 'dark' : 'light');
  });
  bridge.on('navigate', (payload) => {
    if (payload?.screen) switchTab(payload.screen);
  });
  bridge.on('orphans:new', (payload) => showBanner(payload?.message ?? S.errors.noOrphans));

  if (state.offline) showToast(S.common.offline, 6000);

  try {
    const snap = await bridge.snapshot();
    if (snap) applySnapshot(snap);
    state.appInfo = await bridge.appInfo();
  } catch (err) {
    showToast(S.errors.init(err?.message ?? err));
  }

  // Preview mode can open the kill confirmation directly, which makes the flow
  // easy to capture.
  if (mode === 'mock' && params.get('dialog') === 'kill') {
    const target = state.detailKey ? entryByKey(state.detailKey) : orphans()[0];
    if (target) killPids([target.pid]);
  }
  notify();

  // "Updated x seconds ago" needs a repaint every second.
  window.setInterval(() => {
    if (state.screen === 'ports') notify();
  }, 1000);

  window.addEventListener('keydown', onKeydown);
  document.addEventListener('visibilitychange', () => {
    if (!document.hidden) refresh();
  });

  // The traffic light group greys out while the window has no focus.
  const syncWindowFocus = () =>
    document.documentElement.toggleAttribute('data-window-inactive', !document.hasFocus());
  window.addEventListener('focus', syncWindowFocus);
  window.addEventListener('blur', syncWindowFocus);
  syncWindowFocus();
}

boot();
