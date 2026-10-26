import { Modal } from "./Modal";
import { tx, useLang } from "../../i18n";

// Delete confirmation. Cancel and the ✕/backdrop/Escape all cancel;
// only the primary button confirms.
export function ConfirmDialog({
  title,
  message,
  confirmLabel,
  onConfirm,
  onCancel,
}: {
  title: string;
  message: string;
  confirmLabel?: string;
  onConfirm: () => void;
  onCancel: () => void;
}) {
  const lang = useLang();
  const t = (key: string): string => tx(lang, key);
  return (
    <Modal
      title={title}
      onClose={onCancel}
      footer={
        <>
          <button type="button" className="btn" onClick={onCancel}>
            {t("confirm.cancel")}
          </button>
          <button type="button" className="btn primary" onClick={onConfirm}>
            {confirmLabel ?? t("confirm.delete")}
          </button>
        </>
      }
    >
      <p>{message}</p>
    </Modal>
  );
}
