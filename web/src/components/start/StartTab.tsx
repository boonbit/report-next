import { useEffect, useState } from "react";
import { fetchSamples } from "../../api";
import type { Report } from "../../ir";
import { Icon } from "../shell/Icon";
import { tx, useLang } from "../../i18n";
import { buildTemplate, loadRecents, type TemplateKind } from "./templates";

export function StartTab({
  onSample,
  onTemplate,
  onRecent,
}: {
  onSample: (name: string) => void;
  onTemplate: (kind: TemplateKind) => void;
  onRecent: (rep: Report) => void;
}) {
  // Bundled samples from /api/samples; fall back to hello offline.
  const [samples, setSamples] = useState<string[] | null>(null);
  // Recents are read once per mount; the tab remounts when shown again.
  const [recents] = useState(() => loadRecents());
  const lang = useLang();
  const t = (key: string): string => tx(lang, key);
  useEffect(() => {
    let live = true;
    fetchSamples()
      .then((s) => {
        if (live) setSamples(s);
      })
      .catch(() => {
        if (live) setSamples(["hello"]);
      });
    return () => {
      live = false;
    };
  }, []);
  return (
    <div className="start">
      <div className="start-hero">
        <div className="start-mark">R</div>
        <div>
          <h1>{t("start.heroTitle")}</h1>
          <p>{t("start.heroBody")}</p>
        </div>
      </div>
      <div className="start-grid">
        <div className="start-card">
          <div className="card-head">
            <Icon name="open" size={14} /> {t("start.open")}
          </div>
          <div className="card-body">
            <div className="start-actions">
              {(samples ?? ["hello"]).map((s, i) => (
                <button
                  key={s}
                  type="button"
                  className={i === 0 ? "btn primary" : "btn"}
                  onClick={() => onSample(s)}
                  title={t("start.loadSampleTitle").replace("{s}", s)}
                >
                  {s === "hello" ? t("start.loadSample") : s}
                </button>
              ))}
              <button
                type="button"
                className="btn"
                onClick={() => document.getElementById("rz-import")?.click()}
              >
                {t("start.importFrx")}
              </button>
            </div>
            <p className="small muted">
              {t("start.samplesPre")}
              <span className="mono">/api/sample?name=…</span>
              {t("start.samplesMid")}
              <span className="mono">/api/import</span>
              {t("start.samplesPost")}
            </p>
          </div>
        </div>
        <div className="start-card">
          <div className="card-head">
            <Icon name="newpage" size={14} /> {t("start.templates")}
          </div>
          <div className="card-body">
            <div className="start-actions">
              <button
                type="button"
                className="btn"
                onClick={() => onTemplate("blank")}
                title={t("start.blankTitle")}
                aria-label={t("start.blankTitle")}
              >
                {t("start.blank")}
              </button>
              <button
                type="button"
                className="btn"
                onClick={() => onTemplate("table")}
                title={t("start.tableTitle")}
                aria-label={t("start.tableTitle")}
              >
                {t("start.table")}
              </button>
              <button
                type="button"
                className="btn"
                onClick={() => onTemplate("invoice")}
                title={t("start.invoiceTitle")}
                aria-label={t("start.invoiceTitle")}
              >
                {t("start.invoice")}
              </button>
            </div>
            <p className="small muted">
              {t("start.templatesPre")}
              <span className="mono">buildTemplate</span>
              {t("start.templatesPost")}
            </p>
          </div>
        </div>
        <div className="start-card">
          <div className="card-head">
            <Icon name="save" size={14} /> {t("start.recents")}
          </div>
          <div className="card-body">
            {recents.length === 0 ? (
              <p className="small muted">{t("start.noRecents")}</p>
            ) : (
              <ul className="start-list">
                {recents.map((e, i) => (
                  <li key={`${e.when}-${i}`}>
                    <button
                      type="button"
                      onClick={() => onRecent(e.ir)}
                      title={t("start.recentTitle")
                        .replace("{i}", String(i + 1))
                        .replace("{label}", e.label)}
                      aria-label={t("start.recentAria")
                        .replace("{i}", String(i + 1))
                        .replace("{label}", e.label)}
                    >
                      <span className="proj-name">{e.label}</span>
                      <span className="proj-meta">{formatWhen(e.when)}</span>
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </div>
        <div className="start-card">
          <div className="card-head">
            <Icon name="check" size={14} /> {t("start.caps")}
          </div>
          <div className="card-body">
            <div className="capability-list">
              <div className="capability">
                <span className="cap-name">{t("start.capFrx")}</span>
                <span className="cap-version">{t("start.capFrxV")}</span>
              </div>
              <div className="capability">
                <span className="cap-name">{t("start.capEngine")}</span>
                <span className="cap-version">{t("start.capEngineV")}</span>
              </div>
              <div className="capability">
                <span className="cap-name">{t("start.capExports")}</span>
                <span className="cap-version">{t("start.capExportsV")}</span>
              </div>
              <div className="capability">
                <span className="cap-name">{t("start.capBarcodes")}</span>
                <span className="cap-version">{t("start.capBarcodesV")}</span>
              </div>
            </div>
            <p className="start-note">
              {t("start.shortcutsPre")}
              <span className="kbd">Ctrl</span> + <span className="kbd">S</span>
              {t("start.shortcutsMid")}
              {t("start.shortcutsMid2")}
              <span className="kbd">F5</span>
              {t("start.shortcutsPost")}
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}

function formatWhen(when: string): string {
  const t = Date.parse(when);
  if (Number.isNaN(t)) return when;
  return new Date(t).toLocaleString();
}

// Re-exported so tests and the parent can build without importing twice.
export { buildTemplate };
export type { TemplateKind };
