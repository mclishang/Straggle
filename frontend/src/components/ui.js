// Reusable interface fragments: anything a Material Web component can do is left
// to it, this file only composes them and picks colours from tokens.css.
import { html, nothing } from 'lit';
import { S } from '../strings.js';
import { closeDialog, state } from '../store.js';
import { windowControl } from '../actions.js';

export const icon = (name, filled = false) =>
  html`<md-icon class=${filled ? 'is-filled' : ''}>${name}</md-icon>`;

// Top bar: brand plus the mount point of the actions area (appBarActions fills
// it on every repaint). Double-clicking empty space maximises, as a macOS title
// bar does.
function onBarDoubleClick(ev) {
  if (ev.target.closest('button, input, a')) return;
  windowControl('zoom');
}

export const appBar = () => html`
  <header class="appbar" @dblclick=${onBarDoubleClick}>
    <span class="appbar__brand">${S.appName}</span>
    <div class="appbar__tools"></div>
  </header>
`;

// Actions area on the right of the top bar: the theme button. Its icon shows
// what the switch would turn the interface into.
export const appBarActions = ({ dark, onToggleTheme }) => {
  const label = dark ? S.theme.toLight : S.theme.toDark;
  return html`
    <md-icon-button title=${label} aria-label=${label} @click=${onToggleTheme}>
      ${icon(dark ? 'light_mode' : 'dark_mode')}
    </md-icon-button>
  `;
};

function navItem(key, iconName, label, active, onSelect, badge = 0) {
  const isActive = active === key;
  return html`
    <button
      type="button"
      class="nav ${isActive ? 'is-active' : ''}"
      @click=${() => onSelect(key)}
    >
      ${icon(iconName, isActive)}<span>${label}</span>
      ${badge > 0 ? html`<span class="nav__badge">${badge}</span>` : nothing}
    </button>
  `;
}

// The three destinations of the side navigation (its container, <nav
// class="sidebar">, lives in the shell).
export const navItems = ({ active, onSelect, orphanCount = 0 }) => html`
  ${navItem('ports', 'lan', S.nav.ports, active, onSelect)}
  ${navItem('cleanup', 'delete_sweep', S.nav.cleanup, active, onSelect, orphanCount)}
  ${navItem('settings', 'settings', S.nav.settings, active, onSelect)}
`;

export const pageHead = ({ title, sub, actions }) => html`
  <div class="page-head">
    <div class="page-head__text">
      <h1 class="page-title">${title}</h1>
      ${sub ? html`<p class="page-sub">${sub}</p>` : nothing}
    </div>
    ${actions ? html`<div class="page-head__actions">${actions}</div>` : nothing}
  </div>
`;

export const refreshButton = (onClick, busy) => html`
  <md-icon-button aria-label=${S.common.refresh} ?disabled=${busy} @click=${onClick}>
    <md-icon class=${busy ? 'spin' : ''}>refresh</md-icon>
  </md-icon-button>
`;

export function searchBar({ value, onInput, onFilter, filterActive }) {
  return html`
    <div class="toolbar">
      <div class="search">
        <md-icon>search</md-icon>
        <input
          type="search"
          placeholder=${S.ports.searchPlaceholder}
          .value=${value}
          @input=${onInput}
        />
        <md-icon-button
          aria-label=${S.common.filter}
          @click=${onFilter}
        >
          ${icon('tune', filterActive)}
        </md-icon-button>
      </div>
    </div>
  `;
}

export const lead = (iconName, muted = false) => html`
  <span class="row__lead ${muted ? 'row__lead--muted' : ''}">${icon(iconName)}</span>
`;

export const chip = (text, tone = '') => html`<span class="chip ${tone}">${text}</span>`;

// The real program shown at the end of a row: the process image name first, then
// the executable file name, and an honest "unknown" when neither can be read
// (usually a permissions problem) instead of a guess.
const programOf = (entry) => {
  const name = (entry.processName ?? '').trim();
  if (name) return name;
  const exe = (entry.exe ?? '').trim();
  return exe ? exe.split(/[\\/]/).pop() : S.detail.unknown;
};

// A port row: port | display name | badges | description | real program. The
// description and the program columns are right-aligned into straight lines, so
// the badges sit next to the name rather than between them.
export const portRow = ({ entry, active, onClick }) => html`
  <button type="button" class="row ${active ? 'is-active' : ''}" @click=${onClick}>
    ${lead('lan', entry.denied)}
    <span class="row__port">${entry.port}</span>
    <span class="row__name">${entry.displayName || entry.processName}</span>
    ${entry.orphan
      ? html`<span class="row__chips">
          ${chip(S.common.orphan, 'chip--error')}
        </span>`
      : nothing}
    <span class="row__meta">${entry.supporting}</span>
    <span class="row__prog" title=${entry.exe || ''}>${programOf(entry)}</span>
  </button>
`;

// Two-line row (settings and cleanup items): title on top, description below,
// the control on the right. The row is a div because it contains interactive
// components such as md-switch, which must not be nested inside a button.
export const stackRow = ({ lead: leadNode, title, sub, end, onClick }) => html`
  <div class="row row--stack" @click=${onClick}>
    ${leadNode ?? nothing}
    <span class="row__stack">
      <span class="row__title">${title}</span>
      ${sub ? html`<span class="row__sub">${sub}</span>` : nothing}
    </span>
    ${end ? html`<span class="row__end">${end}</span>` : nothing}
  </div>
`;

export const emptyState = ({ iconName = 'info', title, body }) => html`
  <div class="empty">
    ${icon(iconName)}
    <div class="empty__title">${title}</div>
    ${body ? html`<div class="empty__body">${body}</div>` : nothing}
  </div>
`;

export const warningStrip = (text) => html`
  <div class="strip">${icon('error')}<span>${text}</span></div>
`;

export const bannerView = (banner) => html`
  <div class="banner">
    ${icon('info')}
    <span class="banner__text">${banner.text}</span>
  </div>
`;

export const toastView = (toast) => html`<div class="toast">${toast.text}</div>`;

export function dialogView(dialog) {
  if (!dialog) return nothing;
  const confirm = () => {
    const action = dialog.onConfirm;
    closeDialog();
    action?.();
  };
  return html`
    <md-dialog .open=${true} @closed=${closeDialog}>
      <div slot="headline">${dialog.title}</div>
      <div slot="content">${dialog.body}</div>
      <div slot="actions">
        <md-text-button @click=${closeDialog}>${S.dialog.cancel}</md-text-button>
        <md-filled-button @click=${confirm}>${dialog.confirmText ?? S.dialog.confirm}</md-filled-button>
      </div>
    </md-dialog>
  `;
}

// The filter menu, built from Material Web's md-menu anchored to the tune button.
export function filterMenu({ open, anchor, onClose }) {
  if (!open) return nothing;
  const item = (label, checked, onClick) => html`
    <md-menu-item @click=${onClick}>
      <div slot="headline" class="menu-headline">
        <span>${label}</span>
        ${checked ? icon('check') : nothing}
      </div>
    </md-menu-item>
  `;
  return html`
    <md-menu .open=${true} .anchorElement=${anchor} @closed=${onClose}>
      ${item(S.filters.byPort, state.sort === 'port', () => onPick(onClose, 'sort', 'port'))}
      ${item(S.filters.byUptime, state.sort === 'uptime', () => onPick(onClose, 'sort', 'uptime'))}
      ${item(S.filters.byProject, state.sort === 'project', () => onPick(onClose, 'sort', 'project'))}
      <md-divider></md-divider>
      ${item(S.filters.includeUdp, state.includeUdp, () =>
        onPick(onClose, 'includeUdp', !state.includeUdp),
      )}
    </md-menu>
  `;
}

let onPick = () => {};
export function setFilterPicker(fn) {
  onPick = fn;
}
