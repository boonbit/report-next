import type { ReportActions } from "./actions";
import { Icon } from "./Icon";
import type { Counts } from "../../App";
import { tx, useLang } from "../../i18n";

export function Toolbar({
  actions,
  hasReport,
  dirty,
  busy,
  counts,
  canUndo,
  canRedo,
}: {
  actions: ReportActions;
  hasReport: boolean;
  dirty: boolean;
  busy: string | null;
  counts: Counts | null;
  canUndo: boolean;
  canRedo: boolean;
}) {
  const lang = useLang();
  const t = (key: string): string => tx(lang, key);
  return (
    <div className="toolbar" role="toolbar" aria-label={t("tb.toolbarLabel")}>
      <button className="btn" onClick={actions.loadSample} title={t("tb.loadSampleTitle")}>
        <Icon name="app" size={15} />
        <span className="label">{t("tb.sample")}</span>
      </button>
      <button
        className="btn"
        onClick={() => document.getElementById("rz-import")?.click()}
        title={t("tb.importTitle")}
      >
        <Icon name="open" size={15} />
        <span className="label">{t("tb.import")}</span>
      </button>
      <div className="divider-v" />
      <button
        type="button"
        className="btn icon-btn"
        onClick={actions.undo}
        disabled={!canUndo}
        title={t("tb.undoTitle")}
        aria-label={t("tb.undo")}
      >
        <Icon name="undo" size={15} />
      </button>
      <button
        type="button"
        className="btn icon-btn"
        onClick={actions.redo}
        disabled={!canRedo}
        title={t("tb.redoTitle")}
        aria-label={t("tb.redo")}
      >
        <Icon name="redo" size={15} />
      </button>
      <div className="divider-v" />
      <button
        className="btn primary"
        onClick={actions.renderPreview}
        disabled={!hasReport || busy === "render"}
        title={t("tb.renderTitle")}
      >
        <Icon name="play" size={14} />
        <span className="label">{busy === "render" ? t("tb.rendering") : t("tb.render")}</span>
      </button>
      <button
        className="btn"
        onClick={() => actions.exportDoc("pdf")}
        disabled={!hasReport}
        title={t("tb.pdfTitle")}
      >
        <Icon name="page" size={15} />
        <span className="label">{t("tb.pdf")}</span>
      </button>
      <button
        className="btn"
        onClick={() => actions.exportDoc("docx")}
        disabled={!hasReport}
        title={t("tb.docxTitle")}
      >
        <Icon name="doc" size={15} />
        <span className="label">{t("tb.docx")}</span>
      </button>
      <div className="divider-v" />
      <button
        className="btn icon-btn"
        onClick={actions.downloadJson}
        disabled={!hasReport}
        title={t("tb.saveTitle")}
      >
        <Icon name="save" size={15} />
      </button>
      <div className="toolbar-spacer" />
      <div className="toolbar-meta">
        {counts ? (
          <>
            <span className="toolbar-project">{t("tb.project")}</span>
            <span className="chip ok">
              {counts.pages} {t("tb.pages")}
            </span>
            <span className="chip">
              {counts.bands} {t("tb.bands")}
            </span>
            <span className="chip">
              {counts.objects} {t("tb.objects")}
            </span>
            <span className={dirty ? "chip warn" : "chip ok"}>
              {dirty ? t("tb.modified") : t("tb.saved")}
            </span>
          </>
        ) : (
          <span className="chip plain">{t("tb.noReport")}</span>
        )}
      </div>
    </div>
  );
}
