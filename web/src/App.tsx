import { useEffect, useMemo, useRef, useState } from "react";
import { exportDoc, exportReportFile, fetchOutline, fetchSample, fetchSamples, importFrx, renderPreview, type OutlineItem } from "./api";
import { BottomDock, type LogLine } from "./components/console/BottomDock";
import { AboutDialog } from "./components/dialogs/AboutDialog";
import { NewReportDialog } from "./components/dialogs/NewReportDialog";
import { DataPanel } from "./components/data/DataPanel";
import { Palette } from "./components/design/Palette";
import { ReportCanvas } from "./components/design/ReportCanvas";
import { ReportExplorer } from "./components/explorer/ReportExplorer";
import { Inspector } from "./components/inspector/Inspector";
import { PreviewPane } from "./components/preview/PreviewPane";
import type { ReportActions } from "./components/shell/actions";
import { MenuBar } from "./components/shell/MenuBar";
import { Splitter } from "./components/shell/Splitter";
import { StatusBar } from "./components/shell/StatusBar";
import { TabStrip } from "./components/shell/TabStrip";
import { Toolbar } from "./components/shell/Toolbar";
import { SourceEditor } from "./components/source/SourceEditor";
import { StartTab } from "./components/start/StartTab";
import { buildTemplate, saveRecent, type TemplateKind } from "./components/start/templates";
import { clone, getObject, type Report, type Sel } from "./ir";
import { addObject, deleteObject, duplicateObject, type NewObjectKind } from "./ir/mutate";
import { computeIssues } from "./ir/validate";
import "./styles/tokens.css";
import "./styles/shell.css";
import "./styles/report.css";

export type ReportTab = "start" | "design" | "preview" | "source";

export interface Counts {
  pages: number;
  bands: number;
  objects: number;
}

function countRep(rep: Report): Counts {
  let bands = 0;
  let objects = 0;
  for (const p of rep.pages) {
    bands += p.bands.length;
    for (const b of p.bands) objects += b.objects.length;
  }
  return { pages: rep.pages.length, bands, objects };
}

export default function App() {
  const [rep, setRep] = useState<Report | null>(null);
  const [baseline, setBaseline] = useState<string | null>(null);
  const [sel, setSel] = useState<Sel | null>(null);
  const [past, setPast] = useState<Report[]>([]);
  const [future, setFuture] = useState<Report[]>([]);
  const lastPush = useRef(0);
  const canUndo = past.length > 0;
  const canRedo = future.length > 0;
  const [tab, setTab] = useState<ReportTab>("start");
  const [status, setStatus] = useState("Ready — load the sample or import a .frx file.");
  const [busy, setBusy] = useState<string | null>(null);
  const [html, setHtml] = useState("");
  const [previewError, setPreviewError] = useState<string | null>(null);
  const [pvPage, setPvPage] = useState(1);
  const [pvTotal, setPvTotal] = useState(1);
  const [pvZoom, setPvZoom] = useState(1);
  const [outline, setOutline] = useState<OutlineItem[]>([]);
  const [outlineOpen, setOutlineOpen] = useState(false);
  const [dialog, setDialog] = useState<null | "about" | "new">(null);
  const [logs, setLogs] = useState<LogLine[]>([]);
  const [dockOpen, setDockOpen] = useState(false);
  const [railTab, setRailTab] = useState<"inspector" | "data">("inspector");
  const [samples, setSamples] = useState<string[]>(["hello"]);
  const issues = useMemo(() => (rep ? computeIssues(rep) : []), [rep]);

  function log(msg: string, level: LogLine["level"] = "info"): void {
    const t = new Date().toLocaleTimeString();
    setLogs((l) => [...l.slice(-199), { t, msg, level }]);
  }
  const [explorerWidth, setExplorerWidth] = useState(238);
  const [inspectorWidth, setInspectorWidth] = useState(310);
  const stateRef = useRef({ rep, tab, sel, pvPage });
  stateRef.current = { rep, tab, sel, pvPage };
  const pvTotalRef = useRef(pvTotal);
  pvTotalRef.current = pvTotal;
  const htmlRef = useRef(html);
  htmlRef.current = html;
  const dirty = rep !== null && baseline !== null && JSON.stringify(rep) !== baseline;
  const counts = rep ? countRep(rep) : null;

  // Every report change routes through commit so undo/redo sees it.
  // Bursts of keystrokes coalesce into one entry (1.5s window).
  function commit(next: Report, s: Sel | null, message: string): void {
    const cur = stateRef.current.rep;
    if (cur && JSON.stringify(cur) !== JSON.stringify(next)) {
      const now = Date.now();
      if (past.length === 0 || now - lastPush.current >= 1500) {
        lastPush.current = now;
        const snapshot = clone(cur);
        setPast((p) => [...p.slice(-49), snapshot]);
      }
    }
    setFuture([]);
    setRep(clone(next));
    setSel(s);
    setStatus(message);
  }

  function openReport(r: Report, message: string, recentLabel?: string) {
    commit(r, null, message);
    if (recentLabel) saveRecent(r, recentLabel);
    setBaseline(JSON.stringify(r));
    setHtml("");
    setPreviewError(null);
    setPvPage(1);
    setPvTotal(1);
    setOutline([]);
    setOutlineOpen(false);
    setTab("design");
  }

  async function loadSample(name = "hello") {
    setBusy("sample");
    try {
      const r = await fetchSample(name);
      openReport(r, `Sample loaded (${name}): ${r.pages.length} page(s).`, `sample:${name}`);
      log(`sample: loaded ${name} (${r.pages.length} page(s))`);
    } catch (e) {
      setStatus(String(e));
    } finally {
      setBusy(null);
    }
  }

  function importFile(f: File) {
    setBusy("import");
    f.arrayBuffer()
      .then((buf) => importFrx(buf))
      .then((r) => {
        openReport(r, `Imported ${f.name}: ${r.pages.length} page(s).`, f.name);
        log(`import: ${f.name} (${r.pages.length} page(s))`);
      })
      .catch((e: unknown) => {
        setStatus(String(e));
        log(`import: FAILED ${String(e)}`, "error");
      })
      .finally(() => setBusy(null));
  }

  function newBlank(): void {
    const blank: Report = {
      version: "report/v1",
      pages: [
        {
          name: "Page1",
          bands: [
            {
              kind: "DataBand",
              name: "Data1",
              attrs: [
                { name: "Width", value: "718.2" },
                { name: "Height", value: "100" },
              ],
              objects: [],
            },
          ],
        },
      ],
    };
    setDialog(null);
    openReport(blank, "New blank report.", "blank report");
    log("new: blank report");
  }

  async function renderNow(p?: number) {
    const r = stateRef.current.rep;
    if (!r) return;
    const pg = p ?? stateRef.current.pvPage;
    setBusy("render");
    setPreviewError(null);
    try {
      const out = await renderPreview(r, pg);
      setHtml(out.html);
      setPvTotal(out.total);
      setPvPage(Math.min(Math.max(pg, 1), Math.max(out.total, 1)));
      setTab("preview");
      setStatus("Rendered — preview uses the current IR.");
      log("render: ok");
      try {
        setOutline(await fetchOutline(r));
      } catch {
        setOutline([]);
      }
    } catch (e) {
      setPreviewError(e instanceof Error ? e.message : String(e));
      setTab("preview");
      setStatus("Render failed — see the preview pane.");
      log(`render: FAILED ${e instanceof Error ? e.message : String(e)}`, "error");
    } finally {
      setBusy(null);
    }
  }

  // P1 pager: clamp into range, skip no-op re-renders.
  function goPage(p: number): void {
    const r = stateRef.current.rep;
    if (!r) return;
    const total = pvTotalRef.current;
    const next = Math.min(Math.max(p, 1), Math.max(total, 1));
    if (next === stateRef.current.pvPage && htmlRef.current !== "") return;
    void renderNow(next);
  }

  async function exportNow(kind: "pdf" | "docx") {
    const r = stateRef.current.rep;
    if (!r) return;
    try {
      const blob = await exportDoc(kind, r);
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `${fileBase()}.${kind}`;
      document.body.appendChild(a);
      a.click();
      a.remove();
      setTimeout(() => URL.revokeObjectURL(url), 1000);
      setStatus(`Exported ${a.download}.`);
      log(`export: ${a.download}`);
    } catch (e) {
      setStatus(String(e));
      log(`export: FAILED ${String(e)}`, "error");
    }
  }

  function downloadJson() {
    const r = stateRef.current.rep;
    if (!r) return;
    const text = JSON.stringify(r, null, 2);
    const url = URL.createObjectURL(new Blob([text], { type: "application/json" }));
    const a = document.createElement("a");
    a.href = url;
    a.download = "report.json";
    document.body.appendChild(a);
    a.click();
    a.remove();
    setTimeout(() => URL.revokeObjectURL(url), 1000);
    setBaseline(text);
    setStatus("Downloaded report.json.");
  }

  function onChange(next: Report) {
    commit(next, sel, "Edited — preview and exports use the current IR.");
  }

  function onMutate(next: Report, s: Sel | null) {
    commit(next, s, "Edited — preview and exports use the current IR.");
  }

  // File base for downloads: first page name, sanitized.
  function fileBase(): string {
    const r = stateRef.current.rep;
    const raw = r?.pages[0]?.name || "report";
    const clean = raw.replace(/[^A-Za-z0-9-_]+/g, "-").replace(/^-+|-+$/g, "");
    return clean || "report";
  }

  // U1 palette: add an object to the selected band (or the first band).
  function addFromPalette(kind: NewObjectKind): void {
    const r = stateRef.current.rep;
    if (!r) return;
    const at = stateRef.current.sel ?? { page: 0, band: 0, obj: -1 };
    const { next, sel: ns } = addObject(r, { ...at, obj: -1 }, kind);
    onMutate(next, ns);
    log(`palette: added ${kind}`);
  }

  // U2a keyboard: nudge the selected object by dx/dy units.
  function nudge(dx: number, dy: number): void {
    const r = stateRef.current.rep;
    const s = stateRef.current.sel;
    if (!r || !s || !getObject(r, s)) return;
    const next = clone(r);
    const obj = getObject(next, s);
    if (!obj) return;
    for (const a of obj.attrs ?? []) {
      if (a.name === "Left") a.value = String((parseFloat(a.value) || 0) + dx);
      if (a.name === "Top") a.value = String((parseFloat(a.value) || 0) + dy);
    }
    onChange(next);
  }

  // U8b: report-level CSV/XLSX download (text rows of the current IR).
  async function exportReport(kind: "csv" | "xlsx"): Promise<void> {
    const r = stateRef.current.rep;
    if (!r) return;
    try {
      const blob = await exportReportFile(kind, r);
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `${fileBase()}.${kind === "csv" ? "csv" : "xlsx"}`;
      document.body.appendChild(a);
      a.click();
      a.remove();
      setTimeout(() => URL.revokeObjectURL(url), 1000);
      setStatus(`Exported ${a.download}.`);
      log(`export: ${a.download}`);
    } catch (e) {
      setStatus(String(e));
      log(`export: FAILED ${String(e)}`, "error");
    }
  }

  // U8a: download the rendered HTML without switching tabs.
  async function downloadHtml(): Promise<void> {
    const r = stateRef.current.rep;
    if (!r) return;
    setBusy("html");
    try {
      const text = (await renderPreview(r)).html;
      const url = URL.createObjectURL(new Blob([text], { type: "text/html" }));
      const a = document.createElement("a");
      a.href = url;
      a.download = `${fileBase()}.html`;
      document.body.appendChild(a);
      a.click();
      a.remove();
      setTimeout(() => URL.revokeObjectURL(url), 1000);
      setStatus(`Exported ${a.download}.`);
      log(`export: ${a.download}`);
    } catch (e) {
      setStatus(String(e));
      log(`export: FAILED ${String(e)}`, "error");
    } finally {
      setBusy(null);
    }
  }

  function undo(): void {
    const cur = stateRef.current.rep;
    if (!cur || past.length === 0) return;
    const prev = past[past.length - 1];
    setPast(past.slice(0, -1));
    setFuture((f) => [clone(cur), ...f].slice(0, 50));
    setRep(clone(prev));
    setSel(null);
    setStatus("Undone.");
  }

  function redo(): void {
    const cur = stateRef.current.rep;
    if (!cur || future.length === 0) return;
    const [nxt, ...rest] = future;
    setPast((p) => [...p.slice(-49), clone(cur)]);
    setFuture(rest);
    setRep(clone(nxt));
    setSel(null);
    setStatus("Redone.");
  }

  const actions: ReportActions = {
    loadSample: () => {
      void loadSample();
    },
    importFile,
    renderPreview: () => {
      void renderNow();
    },
    exportDoc: (kind) => {
      void exportNow(kind);
    },
    downloadJson,
    showTab: (t) => {
      if (t !== "start" && !stateRef.current.rep) return;
      setTab(t);
    },
    about: () => setDialog("about"),
    undo: () => undo(),
    redo: () => redo(),
    newReport: () => {
      setDialog("new");
      fetchSamples()
        .then((s) => setSamples(s))
        .catch(() => setSamples(["hello"]));
    },
    toggleConsole: () => setDockOpen((v) => !v),
  };

  // Shortcuts: Ctrl+S downloads IR JSON, F5 renders the preview.
  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      const target = event.target as HTMLElement | null;
      const typing =
        target !== null &&
        (target.tagName === "INPUT" || target.tagName === "TEXTAREA" || target.isContentEditable);
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === "s") {
        event.preventDefault();
        downloadJson();
        return;
      }
      // Undo/redo skip text fields so native input undo keeps working
      // (it feeds history through onChange).
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === "z" && !event.shiftKey) {
        if (typing) return;
        event.preventDefault();
        undo();
        return;
      }
      if (
        (event.ctrlKey || event.metaKey) &&
        (event.key.toLowerCase() === "y" || (event.key.toLowerCase() === "z" && event.shiftKey))
      ) {
        if (typing) return;
        event.preventDefault();
        redo();
        return;
      }
      if (event.key === "F5") {
        event.preventDefault();
        void renderNow();
        return;
      }
      if (event.key === "F1") {
        event.preventDefault();
        setDialog("about");
      }
      // U2a object keys: Delete removes, Ctrl+D duplicates, arrows nudge
      // (Shift = 10 units). All undoable; skipped while typing.
      if (typing) return;
      if (event.key === "Delete") {
        const cur = stateRef.current;
        if (cur.rep && cur.sel && getObject(cur.rep, cur.sel)) {
          event.preventDefault();
          const { next, sel: ns } = deleteObject(cur.rep, cur.sel);
          onMutate(next, ns);
          return;
        }
      }
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === "d") {
        const cur = stateRef.current;
        if (cur.rep && cur.sel && getObject(cur.rep, cur.sel)) {
          event.preventDefault();
          const { next, sel: ns } = duplicateObject(cur.rep, cur.sel);
          onMutate(next, ns);
          return;
        }
      }
      const step = event.shiftKey ? 10 : 1;
      const cur = stateRef.current;
      const hasObjSel = !!cur.rep && !!cur.sel && !!getObject(cur.rep, cur.sel);
      if (!hasObjSel) return;
      if (event.key === "ArrowLeft") {
        event.preventDefault();
        nudge(-step, 0);
      } else if (event.key === "ArrowRight") {
        event.preventDefault();
        nudge(step, 0);
      } else if (event.key === "ArrowUp") {
        event.preventDefault();
        nudge(0, -step);
      } else if (event.key === "ArrowDown") {
        event.preventDefault();
        nudge(0, step);
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  });

  // Auto-render the first time the preview tab is shown for a report,
  // so Load sample -> Preview shows content without an extra click.
  useEffect(() => {
    if (tab === "preview" && rep !== null && html === "" && busy !== "render") {
      void renderNow();
    }
  }, [tab, rep, html, busy]);

  // Warn before leaving with unsaved edits.
  useEffect(() => {
    if (!dirty) return;
    const onBeforeUnload = (event: BeforeUnloadEvent) => {
      event.preventDefault();
      event.returnValue = "";
    };
    window.addEventListener("beforeunload", onBeforeUnload);
    return () => window.removeEventListener("beforeunload", onBeforeUnload);
  }, [dirty]);

  const clampExplorer = (delta: number) =>
    setExplorerWidth((width) => Math.min(430, Math.max(180, width + delta)));
  const clampInspector = (delta: number) =>
    setInspectorWidth((width) => Math.min(540, Math.max(250, width - delta)));

  return (
    <div className="studio">
      <MenuBar actions={actions} tab={tab} hasReport={rep !== null} canUndo={canUndo} canRedo={canRedo} consoleOpen={dockOpen} />
      <Toolbar actions={actions} hasReport={rep !== null} dirty={dirty} busy={busy} counts={counts} canUndo={canUndo} canRedo={canRedo} />
      <TabStrip actions={actions} tab={tab} hasReport={rep !== null} dirty={dirty} counts={counts} errorCount={issues.filter((i) => i.severity === "error").length} />
      <input
        id="rz-import"
        className="hidden-import"
        type="file"
        accept=".frx"
        tabIndex={-1}
        aria-hidden="true"
        onChange={(e) => {
          const f = e.target.files?.[0];
          if (f) importFile(f);
          e.target.value = "";
        }}
      />

      {/* Column so the console dock spans the full width below the work row. */}
      <div className="studio-body" style={{ flexDirection: "column" }}>
        {tab === "start" ? (
          <StartTab
            onSample={(name) => void loadSample(name)}
            onTemplate={(kind: TemplateKind) =>
              openReport(buildTemplate(kind), `New ${kind} report from template.`, `template:${kind}`)
            }
            onRecent={(r) => openReport(clone(r), "Recent report reopened.")}
          />
        ) : !rep ? (
          <div className="empty" style={{ flex: 1 }}>
            <p>Open a report to start designing.</p>
          </div>
        ) : (
          <div className="work-row">
            {tab === "design" ? (
              <>
                <div className="side-rail" style={{ width: explorerWidth }}>
                  <ReportExplorer rep={rep} sel={sel} onSelect={setSel} onMutate={onMutate} issues={issues} />
                </div>
                <Splitter orientation="vertical" onDelta={clampExplorer} label="Resize explorer" />
                <main style={{ flex: 1, display: "flex", minWidth: 0 }}>
                  <Palette onAdd={addFromPalette} />
                  <ReportCanvas rep={rep} sel={sel} onSelect={setSel} onChange={onChange} />
                </main>
                <Splitter orientation="vertical" onDelta={clampInspector} label="Resize inspector" />
                <div className="side-rail" style={{ width: inspectorWidth }}>
                  <div style={{ display: "flex", flexDirection: "column", width: "100%", minHeight: 0 }}>
                  <div className="panel-head">
                    <button
                      type="button"
                      className={railTab === "inspector" ? "btn small primary" : "btn small"}
                      title="Show the object inspector"
                      aria-label="Show inspector panel"
                      aria-pressed={railTab === "inspector"}
                      onClick={() => setRailTab("inspector")}
                    >
                      Inspector
                    </button>
                    <button
                      type="button"
                      className={railTab === "data" ? "btn small primary" : "btn small"}
                      title="Show the data-source panel"
                      aria-label="Show data panel"
                      aria-pressed={railTab === "data"}
                      onClick={() => setRailTab("data")}
                    >
                      Data
                    </button>
                  </div>
                  <div style={{ flex: 1, minHeight: 0, overflow: "auto" }}>
                  {railTab === "inspector" ? (
                    <Inspector rep={rep} sel={sel} onChange={onChange} />
                  ) : (
                    <DataPanel rep={rep} sel={sel ?? { page: 0, band: 0, obj: -1 }} onChange={onChange} />
                  )}
                  </div>
                  </div>
                </div>
              </>
            ) : null}
            {tab === "preview" ? (
              <PreviewPane
                rep={rep}
                html={html}
                loading={busy === "render"}
                error={previewError}
                page={pvPage}
                total={pvTotal}
                onPage={goPage}
                zoom={pvZoom}
                onZoom={setPvZoom}
                outline={outline}
                outlineOpen={outlineOpen}
                onToggleOutline={() => setOutlineOpen((v) => !v)}
                onRender={() => void renderNow()}
                onExport={(kind) => void exportNow(kind)}
                onDownloadHtml={() => void downloadHtml()}
                onExportReport={(kind) => void exportReport(kind)}
                fileName={fileBase()}
              />
            ) : null}
            {tab === "source" ? (
              <SourceEditor
                rep={rep}
                onLoad={(r) => {
                  openReport(r, "IR loaded from source.", "IR source");
                }}
              />
            ) : null}
          </div>
        )}
        {dockOpen ? (
          <BottomDock
            logs={logs}
            issues={issues}
            onJump={(s) => {
              setSel(s);
              setTab("design");
            }}
            onClear={() => setLogs([])}
            onClose={() => setDockOpen(false)}
          />
        ) : null}
      </div>

      <StatusBar message={status} hasReport={rep !== null} />
      {dialog === "about" ? <AboutDialog onClose={() => setDialog(null)} /> : null}
      {dialog === "new" ? (
        <NewReportDialog
          samples={samples}
          onCreateBlank={newBlank}
          onCreateSample={(name) => {
            setDialog(null);
            void loadSample(name);
          }}
          onCancel={() => setDialog(null)}
        />
      ) : null}
    </div>
  );
}
