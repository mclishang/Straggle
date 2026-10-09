import assert from 'node:assert/strict';
import test from 'node:test';

globalThis.document = { documentElement: { lang: '', dataset: {} } };
globalThis.window = {
  go: { main: { App: {} } },
  matchMedia: () => ({ matches: false }),
};

const { bridge } = await import('../src/bridge.js');
const { state } = await import('../src/store.js');
const { applySnapshot, toggleTheme, updateSetting } = await import('../src/actions.js');
const { currentLanguage, S } = await import('../src/strings.js');
await bridge.init();

test('settings writes preserve queued edits, recover from failure, and serialize themes', async () => {
  let settings = { onlyLocal: false, background: true, intervalSec: 5, theme: '', language: 'en' };
  const backend = window.go.main.App;
  const calls = [];
  let release;
  backend.GetSettings = async () => ({ ...settings });
  backend.SaveSettings = async (next) => {
    calls.push({ ...next });
    await new Promise((resolve) => { release = resolve; });
    settings = { ...next };
    return { ...settings };
  };
  backend.SetTheme = async (theme) => {
    settings = { ...settings, theme };
    return { ...settings };
  };
  const snapshot = () => ({ settings: { ...settings }, language: settings.language, theme: settings.theme || 'light', entries: [] });
  applySnapshot(snapshot());

  const first = updateSetting('onlyLocal', true);
  const second = updateSetting('intervalSec', 10);
  await new Promise(setImmediate);
  assert.equal(calls.length, 1);
  applySnapshot(snapshot());
  assert.equal(state.snapshot.settings.intervalSec, 10);
  release();
  await first;
  await new Promise(setImmediate);
  assert.equal(calls.length, 2);
  assert.equal(calls[1].onlyLocal, true);
  assert.equal(state.snapshot.settings.intervalSec, 10);
  release();
  await second;
  assert.equal(settings.intervalSec, 10);

  let fail = true;
  backend.SaveSettings = async (next) => {
    if (fail) { fail = false; throw new Error('disk unavailable'); }
    settings = { ...next };
    return { ...settings };
  };
  const rejected = updateSetting('onlyLocal', false);
  const later = updateSetting('background', false);
  await rejected;
  await later;
  assert.equal(settings.onlyLocal, true);
  assert.equal(settings.background, false);
  assert.equal(state.snapshot.settings.onlyLocal, true);

  await Promise.all([toggleTheme(), updateSetting('intervalSec', 1), updateSetting('language', 'zh-CN')]);
  assert.equal(settings.theme, 'dark');
  assert.equal(settings.intervalSec, 1);
  assert.equal(currentLanguage(), 'zh-CN');
  assert.equal(S.settings.languageEnglish, 'English');
  assert.equal(S.settings.languageChinese, '\u7b80\u4f53\u4e2d\u6587');

  backend.SetTheme = async () => { throw new Error('theme save failed'); };
  await toggleTheme();
  assert.equal(state.theme, 'dark');
  assert.equal(state.snapshot.settings.theme, 'dark');
});
