import { tx, useLang } from "../../i18n";

// Document outline: section titles that jump to a page.
// Pure presentational: items, selection, and jump action arrive via props.
export function PreviewOutline({
  items,
  selected,
  onJump,
}: {
  items: { title: string; page: number }[];
  selected: number;
  onJump: (page: number) => void;
}) {
  const lang = useLang();
  const t = (key: string): string => tx(lang, key);
  if (items.length === 0) {
    return <div className="empty">{t("preview.outlineEmpty")}</div>;
  }
  return (
    <div className="preview-outline" style={{ display: "flex", flexDirection: "column", gap: 2 }}>
      <span className="small">{t("preview.outlineTitle")}</span>
      {items.map((item, i) => (
        <button
          key={`${item.page}-${i}`}
          type="button"
          className="btn small"
          onClick={() => onJump(item.page)}
          aria-current={i === selected ? "true" : undefined}
          title={t("preview.outlineJumpTitle")
            .replace("{page}", String(item.page))
            .replace("{title}", item.title)}
          aria-label={t("preview.outlineJumpAria")
            .replace("{page}", String(item.page))
            .replace("{title}", item.title)}
          style={i === selected ? { fontWeight: "bold" } : undefined}
        >
          {item.title}
        </button>
      ))}
    </div>
  );
}
