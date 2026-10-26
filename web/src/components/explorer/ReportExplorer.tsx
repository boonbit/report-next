import { useState } from 'react';
import type { Report, Sel } from '../../ir';
import { addBand, addObject, deleteBand, deleteObject, duplicateObject, type NewObjectKind } from '../../ir/mutate';
import { ConfirmDialog } from '../dialogs/ConfirmDialog';

// Tree view of the report: pages > bands > objects.
// Sel with obj === -1 addresses the band itself.
// Small per-row buttons add/delete/duplicate nodes; deletes ask first.
export interface ReportExplorerProps {
  rep: Report | null;
  sel: Sel | null;
  onSelect: (s: Sel | null) => void;
  onMutate: (next: Report, sel: Sel | null) => void;
  issues?: Array<{ sel: { page: number; band: number; obj: number }; message: string }>;
}

function isSel(sel: Sel | null, page: number, band: number, obj: number): boolean {
  return sel !== null && sel.page === page && sel.band === band && sel.obj === obj;
}

export function ReportExplorer({ rep, sel, onSelect, onMutate, issues = [] }: ReportExplorerProps) {
  const [open, setOpen] = useState<Record<string, boolean>>({});
  const [pending, setPending] = useState<{ what: string; sel: Sel } | null>(null);
  const [query, setQuery] = useState('');
  const q = query.trim().toLowerCase();
  const filtering = q.length > 0;
  // While filtering, every node renders expanded; `open` is left untouched
  // so clearing the query restores the prior expand state.
  const isOpen = (key: string): boolean => filtering || open[key] !== false;
  const matches = (name: string): boolean => !filtering || name.toLowerCase().includes(q);
  const bandIssues = (pi: number, bi: number) =>
    issues.filter((iss) => iss.sel.page === pi && iss.sel.band === bi);
  const objIssues = (pi: number, bi: number, oi: number) =>
    issues.filter((iss) => iss.sel.page === pi && iss.sel.band === bi && iss.sel.obj === oi);

  if (!rep) {
    return (
      <div className="panel">
        <div className="panel-head">Report Explorer</div>
        <div className="empty">No report loaded — load the sample or import a .frx file.</div>
      </div>
    );
  }

  function toggle(key: string): void {
    setOpen((prev) => ({ ...prev, [key]: !prev[key] }));
  }

  function doAddObject(at: Sel, kind: NewObjectKind): void {
    const { next, sel: ns } = addObject(rep as Report, at, kind);
    onMutate(next, ns);
  }

  function confirmDelete(): void {
    if (!pending || !rep) return;
    const { next, sel: ns } =
      pending.what === 'band' ? deleteBand(rep, pending.sel) : deleteObject(rep, pending.sel);
    setPending(null);
    onMutate(next, ns);
  }

  return (
    <div className="panel">
      <div className="panel-head">
        Report Explorer
        <input
          className="input"
          placeholder="Filter bands and objects"
          aria-label="Filter explorer"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
      </div>
      <div className="panel-scroll tree">
        {rep.pages.map((page, pi) => {
          const pkey = `p${pi}`;
          return (
            <div key={pkey}>
              <div className="tree-item" onClick={() => toggle(pkey)}>
                <span className="tree-toggle">{isOpen(pkey) ? '▾' : '▸'}</span>
                <span>Page: {page.name}</span>
                <button
                  type="button"
                  className="btn small tree-btn"
                  title={`Add a band to ${page.name}`}
                  aria-label={`Add band to ${page.name}`}
                  onClick={(e) => {
                    e.stopPropagation();
                    const { next, sel: ns } = addBand(rep, pi);
                    onMutate(next, ns);
                  }}
                >
                  + Band
                </button>
              </div>
              {isOpen(pkey) &&
                page.bands.map((band, bi) => {
                  const bkey = `p${pi}b${bi}`;
                  const at = { page: pi, band: bi, obj: -1 };
                  const bandSel = isSel(sel, pi, bi, -1);
                  if (!matches(band.name) && !band.objects.some((o) => matches(o.name))) return null;
                  const bandFlags = bandIssues(pi, bi);
                  return (
                    <div key={bkey} className="tree-kids">
                      <div
                        className={bandSel ? 'tree-item sel' : 'tree-item'}
                        onClick={() => onSelect(bandSel ? null : at)}
                      >
                        <span
                          className="tree-toggle"
                          onClick={(e) => { e.stopPropagation(); toggle(bkey); }}
                        >
                          {isOpen(bkey) ? '▾' : '▸'}
                        </span>
                        <span>Band: {band.name}</span>
                        {bandFlags.map((iss, k) => (
                          <span key={k} className="chip bad" title={iss.message}>!</span>
                        ))}
                        <span className="tree-btns" onClick={(e) => e.stopPropagation()}>
                          <button type="button" className="btn small tree-btn" title={`Add a text object to ${band.name}`} aria-label={`Add text to ${band.name}`} onClick={() => doAddObject(at, 'TextObject')}>+T</button>
                          <button type="button" className="btn small tree-btn" title={`Add a barcode to ${band.name}`} aria-label={`Add barcode to ${band.name}`} onClick={() => doAddObject(at, 'BarcodeObject')}>+B</button>
                          <button type="button" className="btn small tree-btn danger" title={`Delete band ${band.name}`} aria-label={`Delete band ${band.name}`} onClick={() => setPending({ what: 'band', sel: at })}>✕</button>
                        </span>
                      </div>
                      {isOpen(bkey) &&
                        band.objects.map((obj, oi) => {
                          const oat = { page: pi, band: bi, obj: oi };
                          const objSel = isSel(sel, pi, bi, oi);
                          if (!matches(obj.name)) return null;
                          const objFlags = objIssues(pi, bi, oi);
                          return (
                            <div
                              key={oi}
                              className={objSel ? 'tree-item sel' : 'tree-item'}
                              style={{ paddingLeft: 22 }}
                              onClick={() => onSelect(objSel ? null : oat)}
                            >
                              {obj.kind}: {obj.name}
                              {objFlags.map((iss, k) => (
                                <span key={k} className="chip bad" title={iss.message}>!</span>
                              ))}
                              <span className="tree-btns" onClick={(e) => e.stopPropagation()}>
                                <button
                                  type="button"
                                  className="btn small tree-btn"
                                  title={`Duplicate ${obj.name}`}
                                  aria-label={`Duplicate ${obj.name}`}
                                  onClick={() => {
                                    const { next, sel: ns } = duplicateObject(rep, oat);
                                    onMutate(next, ns);
                                  }}
                                >
                                  ⧉
                                </button>
                                <button
                                  type="button"
                                  className="btn small tree-btn danger"
                                  title={`Delete ${obj.name}`}
                                  aria-label={`Delete ${obj.name}`}
                                  onClick={() => setPending({ what: 'object', sel: oat })}
                                >
                                  ✕
                                </button>
                              </span>
                            </div>
                          );
                        })}
                    </div>
                  );
                })}
            </div>
          );
        })}
      </div>
      {pending ? (
        <ConfirmDialog
          title={pending.what === 'band' ? 'Delete band?' : 'Delete object?'}
          message={pending.what === 'band'
            ? 'The band and all its objects will be removed.'
            : 'The object will be removed.'}
          onConfirm={confirmDelete}
          onCancel={() => setPending(null)}
        />
      ) : null}
    </div>
  );
}
