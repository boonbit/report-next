import { useState } from "react";
import { clone, type Report, type Sel } from "../../ir";
import { addObject } from "../../ir/mutate";
import { fetchSchema, type SchemaTable } from "../../api";

// Data panel (U3): inspect a data source schema and wire fields into the
// report. Inspect posts {kind, dsn, content} to /api/schema and shows the
// returned tables/columns as a tree. Clicking a column inserts a Text
// object with [table.column] into the selected band (or the first band);
// Bind band sets the picked table as the band DataSource attribute.
// Every report change flows through onChange; errors show inline.
export interface DataPanelProps {
  rep: Report;
  sel: Sel;
  onChange: (next: Report) => void;
}

type SchemaKind = "sqlite" | "postgres" | "mssql" | "csv" | "json" | "xml";

const KINDS: SchemaKind[] = ["sqlite", "postgres", "mssql", "csv", "json", "xml"];

function isFileKind(kind: SchemaKind): boolean {
  return kind === "csv" || kind === "json" || kind === "xml";
}

// Resolve the band that receives inserts and bindings: the selected band
// when it exists, otherwise the first band of the first page.
function targetBand(rep: Report, sel: Sel): Sel {
  if (rep.pages[sel.page]?.bands[sel.band]) return { page: sel.page, band: sel.band, obj: -1 };
  return { page: 0, band: 0, obj: -1 };
}

export function DataPanel({ rep, sel, onChange }: DataPanelProps) {
  const [kind, setKind] = useState<SchemaKind>("sqlite");
  const [dsn, setDsn] = useState("");
  const [content, setContent] = useState("");
  const [tables, setTables] = useState<SchemaTable[]>([]);
  const [picked, setPicked] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function handleInspect(): Promise<void> {
    setBusy(true);
    setError(null);
    try {
      const out = await fetchSchema({ kind, dsn, content });
      setTables(out.tables ?? []);
      setPicked(out.tables?.[0]?.name ?? null);
    } catch (e) {
      setTables([]);
      setPicked(null);
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  function handleInsertColumn(table: string, column: string): void {
    const at = targetBand(rep, sel);
    const band = rep.pages[at.page]?.bands[at.band];
    if (!band) {
      setError("No band to add the field to.");
      return;
    }
    const { next } = addObject(rep, at, "TextObject");
    if (next === rep) {
      setError("No band to add the field to.");
      return;
    }
    const target = next.pages[at.page]?.bands[at.band];
    const obj = target?.objects[target.objects.length - 1];
    if (!obj) {
      setError("No band to add the field to.");
      return;
    }
    obj.attrs = [...(obj.attrs ?? []).filter((a) => a.name !== "Text"), { name: "Text", value: `[${table}.${column}]` }];
    setError(null);
    onChange(next);
  }

  function handleBindBand(): void {
    if (!picked) {
      setError("Pick a table first.");
      return;
    }
    const at = targetBand(rep, sel);
    const next = clone(rep);
    const band = next.pages[at.page]?.bands[at.band];
    if (!band) {
      setError("No band to bind.");
      return;
    }
    band.attrs = [...(band.attrs ?? []).filter((a) => a.name !== "DataSource"), { name: "DataSource", value: picked }];
    setError(null);
    onChange(next);
  }

  return (
    <div className="panel">
      <div className="panel-head">
        Data
        <span className="panel-actions">
          <button
            type="button"
            className="btn small"
            title="Inspect the data source schema"
            aria-label="Inspect data source schema"
            onClick={handleInspect}
            disabled={busy}
          >
            {busy ? "Inspecting…" : "Inspect"}
          </button>
          <button
            type="button"
            className="btn small"
            title="Bind the selected band to the picked table"
            aria-label="Bind band to picked data table"
            onClick={handleBindBand}
            disabled={!picked}
          >
            Bind band
          </button>
        </span>
      </div>
      <div className="panel-scroll">
        <label className="field-row">
          Kind
          <select
            className="input"
            title="Data source kind for schema inspection"
            aria-label="Data source kind"
            value={kind}
            onChange={(e) => setKind(e.target.value as SchemaKind)}
          >
            {KINDS.map((k) => (
              <option key={k} value={k}>
                {k}
              </option>
            ))}
          </select>
        </label>
        {isFileKind(kind) ? (
          <label className="field-row">
            Content
            <textarea
              className="input"
              title="File content for CSV, JSON, or XML inspection"
              aria-label="File content for inspection"
              value={content}
              onChange={(e) => setContent(e.target.value)}
              rows={4}
              spellCheck={false}
            />
          </label>
        ) : (
          <label className="field-row">
            DSN
            <input
              className="input"
              title="Database path or connection string"
              aria-label="Database path or connection string"
              value={dsn}
              onChange={(e) => setDsn(e.target.value)}
              placeholder={kind === "sqlite" ? "path/to/db.sqlite" : "connection string"}
              spellCheck={false}
            />
          </label>
        )}
        {error ? (
          <div className="source-bar">
            <span className="error" role="alert">
              {error}
            </span>
          </div>
        ) : null}
        {tables.length === 0 ? (
          <div className="empty">No schema loaded — pick a kind and press Inspect.</div>
        ) : (
          <div className="tree">
            {tables.map((t) => (
              <div key={t.name}>
                <div className={picked === t.name ? "tree-item sel" : "tree-item"}>
                  <button
                    type="button"
                    className="btn small tree-btn"
                    title={`Pick data table ${t.name}`}
                    aria-label={`Pick data table ${t.name}`}
                    onClick={() => setPicked(t.name)}
                  >
                    {t.name}
                  </button>
                </div>
                <div className="tree-kids">
                  {t.columns.map((c) => (
                    <button
                      key={c.name}
                      type="button"
                      className="tree-item"
                      style={{ width: "100%", textAlign: "left" }}
                      title={`Insert field [${t.name}.${c.name}] as text`}
                      aria-label={`Insert field ${t.name} ${c.name} as text`}
                      onClick={() => handleInsertColumn(t.name, c.name)}
                    >
                      <span>
                        {c.name} <span className="faint">({c.type})</span>
                      </span>
                    </button>
                  ))}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
