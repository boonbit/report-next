import { Modal } from "./Modal";
import { tx, useLang } from "../../i18n";

export function AboutDialog({ onClose }: { onClose: () => void }) {
  const lang = useLang();
  const t = (key: string): string => tx(lang, key);
  return (
    <Modal
      title={t("about.title")}
      onClose={onClose}
      footer={
        <>
          <span className="spacer" />
          <button type="button" className="btn primary" onClick={onClose}>
            {t("about.close")}
          </button>
        </>
      }
    >
      <p>
        <strong>{t("about.name")}</strong> {t("about.body")}
      </p>
      <div className="capability-list">
        <div className="capability">
          <span className="cap-name">{t("about.backend")}</span>
          <span className="cap-version">{t("about.backendV")}</span>
        </div>
        <div className="capability">
          <span className="cap-name">{t("about.frontend")}</span>
          <span className="cap-version">{t("about.frontendV")}</span>
        </div>
        <div className="capability">
          <span className="cap-name">{t("about.shell")}</span>
          <span className="cap-version">{t("about.shellV")}</span>
        </div>
      </div>
    </Modal>
  );
}
