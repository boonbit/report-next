import { tx, useLang } from "../../i18n";

// Page navigation buttons plus a "Page X of Y" counter.
// Pure presentational: page position and actions arrive via props.
export function PreviewPager({
  page,
  total,
  onFirst,
  onPrev,
  onNext,
  onLast,
}: {
  page: number;
  total: number;
  onFirst: () => void;
  onPrev: () => void;
  onNext: () => void;
  onLast: () => void;
}) {
  const lang = useLang();
  const t = (key: string): string => tx(lang, key);
  const atFirst = page <= 1;
  const atLast = page >= total;
  const counter = t("preview.pagerCounter")
    .replace("{page}", String(page))
    .replace("{total}", String(total));
  return (
    <div className="preview-pager" style={{ display: "flex", alignItems: "center", gap: 4 }}>
      <button
        type="button"
        className="btn small"
        onClick={onFirst}
        disabled={atFirst}
        title={t("preview.pagerFirstTitle")}
        aria-label={t("preview.pagerFirstAria")}
      >
        {t("preview.pagerFirst")}
      </button>
      <button
        type="button"
        className="btn small"
        onClick={onPrev}
        disabled={atFirst}
        title={t("preview.pagerPrevTitle")}
        aria-label={t("preview.pagerPrevAria")}
      >
        {t("preview.pagerPrev")}
      </button>
      <span className="small" role="status">
        {counter}
      </span>
      <button
        type="button"
        className="btn small"
        onClick={onNext}
        disabled={atLast}
        title={t("preview.pagerNextTitle")}
        aria-label={t("preview.pagerNextAria")}
      >
        {t("preview.pagerNext")}
      </button>
      <button
        type="button"
        className="btn small"
        onClick={onLast}
        disabled={atLast}
        title={t("preview.pagerLastTitle")}
        aria-label={t("preview.pagerLastAria")}
      >
        {t("preview.pagerLast")}
      </button>
    </div>
  );
}
