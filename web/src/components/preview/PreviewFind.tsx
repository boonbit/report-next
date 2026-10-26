import { tx, useLang } from "../../i18n";

// Text search box with match count and search/next/prev actions.
// Pure presentational: query value, match count, and actions arrive via props.
export function PreviewFind({
  query,
  count,
  onQuery,
  onSearch,
  onNext,
  onPrev,
}: {
  query: string;
  count: number | null;
  onQuery: (q: string) => void;
  onSearch: () => void;
  onNext: () => void;
  onPrev: () => void;
}) {
  const lang = useLang();
  const t = (key: string): string => tx(lang, key);
  const countText =
    count === null
      ? ""
      : count === 0
        ? t("preview.findNone")
        : t("preview.findCount").replace("{n}", String(count));
  return (
    <div className="preview-find" style={{ display: "flex", alignItems: "center", gap: 4 }}>
      <input
        type="search"
        className="input"
        value={query}
        placeholder={t("preview.findPlaceholder")}
        title={t("preview.findTitle")}
        aria-label={t("preview.findAria")}
        onChange={(e) => onQuery(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === "Enter") onSearch();
        }}
      />
      {count !== null && (
        <span className="small" role="status">
          {countText}
        </span>
      )}
      <button
        type="button"
        className="btn small"
        onClick={onSearch}
        title={t("preview.findSearchTitle")}
        aria-label={t("preview.findSearchAria")}
      >
        {t("preview.findSearch")}
      </button>
      <button
        type="button"
        className="btn small"
        onClick={onPrev}
        title={t("preview.findPrevTitle")}
        aria-label={t("preview.findPrevAria")}
      >
        {t("preview.findPrev")}
      </button>
      <button
        type="button"
        className="btn small"
        onClick={onNext}
        title={t("preview.findNextTitle")}
        aria-label={t("preview.findNextAria")}
      >
        {t("preview.findNext")}
      </button>
    </div>
  );
}
