import { useRef, useState } from "react";
import type { Report } from "../../ir";
import { tx, useLang } from "../../i18n";
import type { OutlineItem } from "../../api";
import { PreviewFind } from "./PreviewFind";
import { PreviewOutline } from "./PreviewOutline";
import { PreviewPager } from "./PreviewPager";
import { PreviewPrint } from "./PreviewPrint";
import { PreviewZoom } from "./PreviewZoom";

// Rendered-HTML preview plus document exports for the current IR.
// The HTML comes from App (toolbar Render, one prepared page at a time);
// exports post the IR directly. Zoom, find, and the outline pane are
// display-only: the committed IR never changes from here.
export interface PreviewPaneProps {
  rep: Report | null;
  html: string;
  loading: boolean;
  error: string | null;
  page: number;
  total: number;
  onPage: (p: number) => void;
  zoom: number;
  onZoom: (z: number) => void;
  outline: OutlineItem[];
  outlineOpen: boolean;
  onToggleOutline: () => void;
  onRender: () => void;
  onExport: (kind: "pdf" | "docx") => void;
  onDownloadHtml: () => void;
  onExportReport: (kind: "csv" | "xlsx") => void;
  fileName: string;
}

export function PreviewPane({
  rep,
  html,
  loading,
  error,
  page,
  total,
  onPage,
  zoom,
  onZoom,
  outline,
  outlineOpen,
  onToggleOutline,
  onRender,
  onExport,
  onDownloadHtml,
  onExportReport,
  fileName,
}: PreviewPaneProps) {
  const lang = useLang();
  const t = (key: string): string => tx(lang, key);
  const frameRef = useRef<HTMLIFrameElement | null>(null);
  const [query, setQuery] = useState("");
  const [count, setCount] = useState<number | null>(null);

  // window.find is non-standard and missing from this TS lib; the
  // same-origin srcDoc iframe supports it in Chromium.
  type Finder = { find?: (q: string, cs?: boolean, back?: boolean, wrap?: boolean) => boolean };
  function frameFind(q: string, back: boolean): boolean {
    const w = frameRef.current?.contentWindow as (Window & Finder) | null;
    return w?.find?.(q, false, back, true) ?? false;
  }

  // Page-jump links rendered by the engine (<a data-goto-page>) have no
  // href, so hook them to the pager after every load (same-origin srcDoc).
  function hookLinks(): void {
    const doc = frameRef.current?.contentDocument;
    if (!doc) return;
    doc.querySelectorAll("a[data-goto-page]").forEach((a) => {
      a.addEventListener("click", (e) => {
        e.preventDefault();
        onPage(Number((a as HTMLElement).dataset.gotoPage) || 1);
      });
    });
  }

  function doSearch(q: string): void {
    const frame = frameRef.current;
    const doc = frame?.contentDocument;
    if (!doc || !q) {
      setCount(q ? 0 : null);
      return;
    }
    const text = doc.body?.innerText ?? "";
    const n = text.toLowerCase().split(q.toLowerCase()).length - 1;
    setCount(n);
    if (n > 0) frameFind(q, false);
  }

  function stepFind(back: boolean): void {
    if (query) frameFind(query, back);
  }

  if (!rep) {
    return (
      <div className="panel">
        <div className="panel-head">{t("preview.title")}</div>
        <div className="empty">{t("preview.empty")}</div>
      </div>
    );
  }
  return (
    <div className="panel">
      <div className="panel-head">
        {t("preview.title")}
        <span className="panel-actions">
          <button type="button" className="btn small" onClick={onRender} disabled={loading}>
            {loading ? t("preview.rendering") : t("preview.render")}
          </button>
          <button type="button" className="btn small" onClick={() => onExport("pdf")}>
            {t("preview.exportPdf")}
          </button>
          <button type="button" className="btn small" onClick={() => onExport("docx")}>
            {t("preview.exportDocx")}
          </button>
          <button
            type="button"
            className="btn small"
            onClick={() => onExportReport("csv")}
            title={t("preview.csvTitle")}
            aria-label={t("preview.csvAria")}
          >
            {t("preview.exportCsv")}
          </button>
          <button
            type="button"
            className="btn small"
            onClick={() => onExportReport("xlsx")}
            title={t("preview.xlsxTitle")}
            aria-label={t("preview.xlsxAria")}
          >
            {t("preview.exportXlsx")}
          </button>
          <button
            type="button"
            className="btn small"
            onClick={onDownloadHtml}
            title={t("preview.htmlTitle")}
            aria-label={t("preview.htmlAria")}
          >
            {t("preview.html")}
          </button>
        </span>
      </div>
      <div style={{ display: "flex", gap: 8, alignItems: "center", padding: "4px 8px", flexWrap: "wrap" }}>
        <PreviewPager
          page={page}
          total={total}
          onFirst={() => onPage(1)}
          onPrev={() => onPage(page - 1)}
          onNext={() => onPage(page + 1)}
          onLast={() => onPage(total)}
        />
        <PreviewZoom zoom={zoom} onZoom={onZoom} />
        <button
          type="button"
          className="btn small"
          onClick={onToggleOutline}
          title={t("preview.outlineTitle")}
          aria-label={t("preview.outlineTitle")}
          aria-pressed={outlineOpen}
        >
          {t("preview.outlineTitle")}
        </button>
        <PreviewPrint html={html} fileName={fileName} />
      </div>
      <div style={{ padding: "0 8px 4px" }}>
        <PreviewFind
          query={query}
          count={count}
          onQuery={setQuery}
          onSearch={() => doSearch(query)}
          onNext={() => stepFind(false)}
          onPrev={() => stepFind(true)}
        />
      </div>
      <div className="preview-pane">
        {error && (
          <div className="preview-bar" role="alert">
            <span className="chip bad">{t("preview.failed")}</span>
            <span className="small">{error}</span>
          </div>
        )}
        <div style={{ display: "flex", flex: 1, minHeight: 0 }}>
          {outlineOpen && outline.length > 0 ? (
            <div style={{ width: 220, flex: "none", overflow: "auto" }}>
              <PreviewOutline items={outline} selected={page} onJump={onPage} />
            </div>
          ) : null}
          <div className="preview-body" style={{ flex: 1 }}>
            {html && !loading ? (
              <iframe
                ref={frameRef}
                title={t("preview.frameTitle")}
                srcDoc={html}
                onLoad={hookLinks}
                style={{ width: `${Math.round(zoom * 100)}%` }}
              />
            ) : (
              <div className="empty">
                <p>{loading ? t("preview.rendering") : t("preview.press")}</p>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
