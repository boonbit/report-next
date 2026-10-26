import { tx, useLang } from "../../i18n";

// Print button for the rendered HTML preview (P3).
// Opens a same-origin blank window, writes a minimal HTML shell
// around the given body content, and calls print().
export function PreviewPrint({ html, fileName }: { html: string; fileName: string }) {
  const lang = useLang();
  const t = (key: string): string => tx(lang, key);
  const onPrint = (): void => {
    if (!html) return;
    const win = window.open("", "_blank");
    if (!win) return;
    win.document.write(
      "<!DOCTYPE html><html><head><title>" +
        fileName +
        "</title></head><body>" +
        html +
        "</body></html>",
    );
    win.document.close();
    win.focus();
    win.print();
  };
  return (
    <button
      type="button"
      className="btn small"
      onClick={onPrint}
      disabled={!html}
      title={t("preview.printTitle")}
      aria-label={t("preview.printAria")}
    >
      {t("preview.print")}
    </button>
  );
}
