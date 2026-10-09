// The window buttons the frontend draws (macOS-style traffic lights).
//
// The window is frameless, so these replace the system title bar and its three
// buttons: red closes, yellow minimises, green maximises or restores. Clicking
// them runs the Go side, which is why closing still hides to the tray while
// "keep running" is on instead of quitting.
//
// The glyphs follow macOS: hidden until the group is hovered (see app.css).
import { html } from 'lit';
import { S } from '../strings.js';
import { windowControl } from '../actions.js';

const CLOSE = html`<svg viewBox="0 0 12 12" aria-hidden="true">
  <path d="M3.5 3.5 8.5 8.5M8.5 3.5 3.5 8.5" />
</svg>`;

const MINIMISE = html`<svg viewBox="0 0 12 12" aria-hidden="true">
  <path d="M3.5 6h5" />
</svg>`;

// The green button is two solid triangles (the Big Sur look) separated by a
// diagonal gap.
const ZOOM = html`<svg viewBox="0 0 12 12" aria-hidden="true">
  <path d="M3.5 8V3.5H8z" />
  <path d="M8.5 4V8.5H4z" />
</svg>`;

const dot = (kind, label, glyph) => html`
  <button
    type="button"
    class="win-btn win-btn--${kind}"
    title=${label}
    aria-label=${label}
    @click=${() => windowControl(kind)}
  >
    ${glyph}
  </button>
`;

export const windowControls = () => html`
  <div class="win-controls" role="group" aria-label=${S.window.group}>
    ${dot('close', S.window.close, CLOSE)} ${dot('minimise', S.window.minimise, MINIMISE)}
    ${dot('zoom', S.window.zoom, ZOOM)}
  </div>
`;
