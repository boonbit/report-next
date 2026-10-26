import { tx, useLang } from "../../i18n";

export function StatusBar({ message, hasReport }: { message: string; hasReport: boolean }) {
  const lang = useLang();
  const t = (key: string): string => tx(lang, key);
  return (
    <div className="statusbar">
      <div className="status-cell">
        {hasReport ? (
          <>
            <span className="status-dot ok" />
            <strong>{t("status.loaded")}</strong>
          </>
        ) : (
          <>
            <span className="status-dot" />
            <span>{t("status.noReport")}</span>
          </>
        )}
      </div>
      <div className="status-cell status-message" title={message}>
        {message}
      </div>
      <div className="status-cell mono" title={t("status.contractTitle")}>
        {t("status.contract")}
      </div>
    </div>
  );
}
