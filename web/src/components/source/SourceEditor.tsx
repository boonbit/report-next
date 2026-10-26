import { useState } from "react";
import { clone, type Report } from "../../ir";
import { tx, useLang } from "../../i18n";

// Raw JSON view of the report IR. Load applies the edited JSON to the
// app; Validate only checks it. Invalid JSON shows an error, never a crash.
export interface SourceEditorProps {
  rep: Report | null;
  onLoad: (r: Report) => void;
}

export function SourceEditor({ rep, onLoad }: SourceEditorProps) {
  const [text, setText] = useState(() => (rep ? JSON.stringify(rep, null, 2) : ""));
  const [error, setError] = useState<string | null>(null);
  const lang = useLang();
  const t = (key: string): string => tx(lang, key);

  function parse(): Report | null {
    try {
      const r = JSON.parse(text) as Report;
      setError(null);
      return r;
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      return null;
    }
  }

  function handleLoad(): void {
    const r = parse();
    if (r) onLoad(clone(r));
  }

  function handleValidate(): void {
    parse();
  }

  function handleFormat(): void {
    try {
      setText(JSON.stringify(JSON.parse(text), null, 2));
      setError(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  return (
    <div className="panel">
      <div className="panel-head">
        {t("src.title")}
        <span className="panel-actions">
          <button type="button" className="btn small" onClick={handleLoad}>
            {t("src.load")}
          </button>
          <button type="button" className="btn small" onClick={handleValidate}>
            {t("src.validate")}
          </button>
          <button type="button" className="btn small" onClick={handleFormat} title={t("src.formatTitle")}>
            {t("src.format")}
          </button>
          {error ? (
            <span className="chip bad" role="alert">
              {t("src.invalid")}
            </span>
          ) : (
            <span className="chip ok">{t("src.valid")}</span>
          )}
        </span>
      </div>
      <div className="source-wrap">
        <div className="source">
          {error ? (
            <div className="source-bar">
              <span className="error">{error}</span>
            </div>
          ) : null}
          <textarea
            className="source-editor"
            value={text}
            onChange={(e) => setText(e.target.value)}
            spellCheck={false}
            aria-label={t("src.aria")}
          />
        </div>
      </div>
    </div>
  );
}
