import { html, nothing } from 'lit';
import { S } from '../strings.js';
import { state } from '../store.js';
import { chip, icon } from '../components/ui.js';
import { closeDetail, copyText, killPids, openPath } from '../actions.js';

// Details pane on the right: expands when a row is selected, 320dp wide. It
// shares the screen with the list instead of being a route of its own, because
// two columns beat hopping between screens on a desktop.

const scopeLabel = (scope) => S.scope[scope] ?? S.scope.all;

const addrText = (entry) => {
  const items = entry.addrs ?? [];
  if (items.length === 0) return S.detail.unknown;
  return items.map((a) => S.detail.addrItem(a.addr, scopeLabel(a.scope))).join(S.detail.addrJoin);
};

const def = ({ label, value, mono = false, action }) => html`
  <div class="def">
    <dt>${label}</dt>
    <dd>
      <span class=${mono ? 'mono' : ''}>${value}</span>
      ${action ?? nothing}
    </dd>
  </div>
`;

export function inspector(entry) {
  const killing = state.killing.has(entry.pid);
  const started = entry.startedAt
    ? S.detail.startedUptime(entry.startedHuman, entry.uptimeHuman)
    : S.detail.unknown;

  return html`
    <aside class="pane">
      <header class="pane__head">
        <span class="pane__title">${entry.port} · ${entry.displayName || entry.processName}</span>
        ${entry.orphan ? chip(S.common.orphan, 'chip--error') : nothing}
        <md-icon-button aria-label=${S.detail.close} @click=${closeDetail}>
          ${icon('close')}
        </md-icon-button>
      </header>
      <div class="pane__body scroll">
        ${entry.denied
          ? html`<div class="strip">
              ${icon('error')}<span>${entry.deniedReason || S.detail.denied}</span>
            </div>`
          : nothing}
        <dl class="defs">
          ${def({ label: S.detail.cmd, value: entry.cmdline || S.detail.unknown, mono: true })}
          ${def({ label: S.detail.pid, value: `${entry.pid} · ${entry.parentNote}` })}
          ${def({ label: S.detail.addr, value: addrText(entry) })}
          ${def({ label: S.detail.started, value: started })}
          ${def({
            label: S.detail.cwd,
            value: entry.cwd || S.detail.unknown,
            mono: true,
            action: html`
              <md-icon-button
                aria-label=${S.detail.open}
                ?disabled=${!entry.cwdKnown}
                @click=${() => openPath(entry.cwd)}
              >
                ${icon('folder_open')}
              </md-icon-button>
            `,
          })}
        </dl>
        ${entry.orphan ? html`<p class="page-sub">${S.detail.orphanNote}</p>` : nothing}
      </div>
      <footer class="pane__foot">
        <md-filled-button
          ?disabled=${killing || entry.denied}
          @click=${() => killPids([entry.pid])}
        >
          <md-icon slot="icon">stop_circle</md-icon>
          ${killing ? S.detail.killing : S.detail.kill}
        </md-filled-button>
        <md-text-button ?disabled=${!entry.cmdline} @click=${() => copyText(entry.cmdline)}>
          <md-icon slot="icon">content_copy</md-icon>
          ${S.detail.copy}
        </md-text-button>
      </footer>
    </aside>
  `;
}
