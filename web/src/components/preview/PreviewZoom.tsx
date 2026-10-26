import { tx, useLang } from "../../i18n";

// Zoom preset buttons. Pure presentational: zoom level and setter via props.
const PRESETS = [50, 75, 100, 150, 200];

export function PreviewZoom({ zoom, onZoom }: { zoom: number; onZoom: (z: number) => void }) {
  const lang = useLang();
  const t = (key: string): string => tx(lang, key);
  return (
    <div className="preview-zoom" style={{ display: "flex", alignItems: "center", gap: 4 }}>
      <span className="small">{t("preview.zoomLabel")}</span>
      {PRESETS.map((p) => {
        const active = Math.round(zoom * 100) === p;
        return (
          <button
            key={p}
            type="button"
            className="btn small"
            onClick={() => onZoom(Math.min(2, Math.max(0.5, p / 100)))}
            aria-pressed={active}
            title={t("preview.zoomPresetTitle").replace("{z}", String(p))}
            aria-label={t("preview.zoomPresetAria").replace("{z}", String(p))}
            style={active ? { fontWeight: "bold" } : undefined}
          >
            {p}%
          </button>
        );
      })}
    </div>
  );
}
