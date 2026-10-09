import { html, nothing } from 'lit';
import { S } from '../strings.js';
import { entryByKey, patch, state, updatedAgo, visibleEntries } from '../store.js';
import {
  bannerView,
  emptyState,
  filterMenu,
  pageHead,
  portRow,
  refreshButton,
  searchBar,
  setFilterPicker,
  warningStrip,
} from '../components/ui.js';
import { refresh, selectPort, setIncludeUDP } from '../actions.js';
import { inspector } from './inspector.js';

// Anchor of the tune menu, recorded on click; the menu is rendered in the same
// screen.
let tuneAnchor = null;

setFilterPicker((close, key, value) => {
  close();
  if (key === 'includeUdp') {
    patch({ includeUdp: value });
    setIncludeUDP(value);
    return;
  }
  patch({ [key]: value });
});

export function portsView() {
  const snap = state.snapshot;
  const list = visibleEntries();
  const searching = state.search.trim() !== '';
  const filtersOn = state.includeUdp || state.sort !== 'port';
  // The selected entry expands in the right pane; once a refresh drops it (the
  // process ended) the pane closes on its own.
  const selected = entryByKey(state.detailKey);

  return html`
    ${pageHead({
      title: S.ports.title,
      sub: S.ports.updated(updatedAgo(), snap?.listeningCount ?? 0),
      actions: refreshButton(refresh, state.refreshing),
    })}
    ${state.banner ? bannerView(state.banner) : nothing}
    ${state.offline ? warningStrip(S.common.offline) : nothing}
    ${snap?.warning ? warningStrip(snap.warning) : nothing}
    ${searchBar({
      value: state.search,
      onInput: (ev) => patch({ search: ev.target.value }),
      onFilter: (ev) => {
        tuneAnchor = ev.currentTarget;
        patch({ menu: state.menu === 'tune' ? null : 'tune' });
      },
      filterActive: filtersOn,
    })}
    <div class="split">
      <div class="split__main scroll">
        ${list.length === 0
          ? emptyState(
              searching || filtersOn
                ? {
                    iconName: 'search_off',
                    title: S.ports.searchEmptyTitle,
                    body: S.ports.searchEmptyBody,
                  }
                : { iconName: 'lan', title: S.ports.emptyTitle, body: S.ports.emptyBody },
            )
          : html`<div class="list">
              ${list.map((entry) =>
                portRow({
                  entry,
                  active: entry.key === state.detailKey,
                  onClick: () => selectPort(entry.key),
                }),
              )}
            </div>`}
        ${snap?.truncated
          ? html`<p class="page-sub">${S.ports.truncated(300)}</p>`
          : nothing}
      </div>
      ${selected ? inspector(selected) : nothing}
    </div>
    ${filterMenu({
      open: state.menu === 'tune',
      anchor: tuneAnchor,
      onClose: () => patch({ menu: null }),
    })}
  `;
}
