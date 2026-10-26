import type { ReportActions } from "./actions";
import { Icon } from "./Icon";
import type { Counts, ReportTab } from "../../App";
import { tx, useLang } from "../../i18n";

const TABS: { id: ReportTab; key: string; icon: "app" | "image" | "code" }[] = [
  { id: "start", key: "tabs.start", icon: "app" },
  { id: "design", key: "tabs.design", icon: "image" },
  { id: "preview", key: "tabs.preview", icon: "image" },
  { id: "source", key: "tabs.source", icon: "code" },
];

export function TabStrip({
  actions,
  tab,
  hasReport,
  dirty,
  counts,
  errorCount,
}: {
  actions: ReportActions;
  tab: ReportTab;
  hasReport: boolean;
  dirty: boolean;
  counts: Counts | null;
  errorCount: number;
}) {
  const lang = useLang();
  const t = (key: string): string => tx(lang, key);
  return (
    <div className="tabstrip" role="tablist">
      {TABS.map((tb) => {
        const disabled = tb.id !== "start" && !hasReport;
        return (
          <button
            key={tb.id}
            role="tab"
            aria-selected={tab === tb.id}
            className={tab === tb.id ? "tab active" : "tab"}
            disabled={disabled}
            onClick={() => actions.showTab(tb.id)}
            title={t(tb.key)}
          >
            <Icon name={tb.icon} size={13} />
            {t(tb.key)}
            {tb.id === "design" && dirty ? (
              <span className="tab-dot" title={t("tabs.unsaved")} />
            ) : null}
            {tb.id === "design" && errorCount > 0 ? (
              <span className="chip bad">
                <Icon name="warn" size={11} />
                {errorCount}
              </span>
            ) : null}
          </button>
        );
      })}
      <div className="tab-sub">
        {counts ? (
          <>
            <span className="mono">{t("tabs.project")}</span>
            <span>·</span>
            <span>
              {counts.pages} {t("tabs.pages")}, {counts.bands} {t("tabs.bands")}, {counts.objects}{" "}
              {t("tabs.objects")}
            </span>
          </>
        ) : (
          <span>{t("tabs.hint")}</span>
        )}
      </div>
    </div>
  );
}
