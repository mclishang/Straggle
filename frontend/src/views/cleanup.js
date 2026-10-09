import { html } from 'lit';
import { S } from '../strings.js';
import { isSelected, orphanProcesses, setSelected, state } from '../store.js';
import { emptyState, lead, pageHead, refreshButton, stackRow } from '../components/ui.js';
import { killPids, refresh } from '../actions.js';

// The cleanup screen lists orphans per process: one process can hold several
// ports at once (several entries with the same PID).
function orphanItem({ entry, ports }) {
  const selected = isSelected(entry.pid);
  const killing = state.killing.has(entry.pid);
  const sub =
    ports.length > 1 ? `${entry.supporting} · ${S.cleanup.portsHeld(ports.length)}` : entry.supporting;

  return stackRow({
    lead: lead('terminal', entry.denied),
    title: entry.headline,
    sub,
    end: html`
      <md-switch
        ?selected=${selected}
        ?disabled=${killing}
        @click=${(ev) => ev.stopPropagation()}
        @change=${(ev) => setSelected(entry.pid, ev.target.selected)}
      ></md-switch>
    `,
    onClick: () => setSelected(entry.pid, !selected),
  });
}

export function cleanupView() {
  const list = orphanProcesses();
  const selected = list.filter((p) => isSelected(p.entry.pid));
  const pending = selected.filter((p) => !state.killing.has(p.entry.pid));

  return html`
    ${pageHead({
      title: S.cleanup.title,
      sub: S.cleanup.caption,
      actions: refreshButton(refresh, state.refreshing),
    })}
    <div class="content__body scroll">
      ${list.length === 0
        ? emptyState({
            iconName: 'check_circle',
            title: S.cleanup.emptyTitle,
            body: S.cleanup.emptyBody,
          })
        : html`
            <div class="list">${list.map(orphanItem)}</div>
            <div class="actions-bar">
              <md-filled-button
                ?disabled=${pending.length === 0}
                @click=${() => killPids(selected.map((p) => p.entry.pid))}
              >
                <md-icon slot="icon">stop_circle</md-icon>
                ${S.cleanup.killSelected(selected.length)}
              </md-filled-button>
            </div>
          `}
    </div>
  `;
}
