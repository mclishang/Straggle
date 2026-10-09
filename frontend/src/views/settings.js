import { html, nothing } from 'lit';
import { repeat } from 'lit/directives/repeat.js';
import { currentLanguage, S } from '../strings.js';
import { state } from '../store.js';
import { lead, pageHead, stackRow } from '../components/ui.js';
import { openRepo, updateSetting } from '../actions.js';

// Rebuilt on every render: the labels live in the active catalogue, which can
// change while the application runs.
const rows = () => [
  { key: 'background', icon: 'notifications', ...S.settings.background },
  { key: 'onlyLocal', icon: 'lan', ...S.settings.onlyLocal },
  { key: 'notifyOrphans', icon: 'warning', ...S.settings.notifyOrphans },
  { key: 'confirmKill', icon: 'help', ...S.settings.confirmKill },
];

// The stored value is '' to follow the system language, but Material's select drops
// an empty `value` and then displays nothing, so the field carries a sentinel and
// maps it back to ''. Both languages are named in their own words.
const LANGUAGE_SYSTEM = 'system';
const LANGUAGES = [
  { value: LANGUAGE_SYSTEM, label: () => S.settings.languageSystem },
  { value: 'en', label: () => S.settings.languageEnglish },
  { value: 'zh-CN', label: () => S.settings.languageChinese },
];

// md-select copies the selected option's text into itself and only refreshes it
// when another option is picked, so a field built from the catalogue has to be
// rebuilt when the catalogue changes.
const keyedField = (template) => repeat([currentLanguage()], (lang) => lang, () => template);

export function settingsView() {
  const settings = state.snapshot?.settings ?? {};
  const info = state.appInfo;
  const interval = String(settings.intervalSec ?? 5);

  const switchNode = (key) => html`
    <md-switch
      ?selected=${!!settings[key]}
      @click=${(ev) => ev.stopPropagation()}
      @change=${(ev) => updateSetting(key, ev.target.selected)}
    ></md-switch>
  `;

  return html`
    ${pageHead({ title: S.settings.title })}
    <div class="content__body scroll">
      <div class="list">
        ${rows().map((row) =>
          stackRow({
            lead: lead(row.icon),
            title: row.title,
            sub: row.sub,
            end: switchNode(row.key),
            onClick: () => updateSetting(row.key, !settings[row.key]),
          }),
        )}
      </div>

      <div class="field-block">
        ${keyedField(html`
          <md-outlined-select
            label=${S.settings.interval}
            .value=${interval}
            @change=${(ev) => updateSetting('intervalSec', Number(ev.target.value))}
          >
            ${[1, 5, 10].map(
              (n) => html`
                <md-select-option value=${String(n)}>
                  <div slot="headline">${S.settings.seconds(n)}</div>
                </md-select-option>
              `,
            )}
            <div slot="supporting-text">${S.settings.intervalNote}</div>
          </md-outlined-select>
        `)}
      </div>

      <div class="field-block">
        ${keyedField(html`
          <md-outlined-select
            label=${S.settings.language}
            .value=${settings.language || LANGUAGE_SYSTEM}
            @change=${(ev) =>
              updateSetting('language', ev.target.value === LANGUAGE_SYSTEM ? '' : ev.target.value)}
          >
            ${LANGUAGES.map(
              (lang) => html`
                <md-select-option value=${lang.value}>
                  <div slot="headline">${lang.label()}</div>
                </md-select-option>
              `,
            )}
            <div slot="supporting-text">${S.settings.languageNote}</div>
          </md-outlined-select>
        `)}
      </div>

      ${info
        ? html`
            <div class="about">
              <a
                class="about__row about__row--link"
                href=${info.repoUrl}
                @click=${(ev) => {
                  ev.preventDefault();
                  openRepo();
                }}
              >
                <span>${S.settings.repo}</span><span>${info.repoUrl}</span>
              </a>
              <div class="about__row"><span>${S.settings.version}</span><span>${info.version}</span></div>
              <div class="about__row"><span>${S.settings.platform}</span><span>${info.os}</span></div>
              <div class="about__row">
                <span>${S.settings.dataDir}</span><span class="mono">${info.dataDir}</span>
              </div>
            </div>
          `
        : nothing}
    </div>
  `;
}
