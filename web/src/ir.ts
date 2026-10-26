// Shared report-IR types for the web designer. Mirrors the Go IR in
// internal/report/ir.go (report/v1). Sel addresses one node: obj === -1
// means the band itself is selected.
export interface Attr {
  name: string;
  value: string;
}

export interface RptObject {
  kind: string;
  name: string;
  attrs?: Attr[];
  children?: unknown[];
}

export interface Band {
  kind: string;
  name: string;
  attrs?: Attr[];
  objects: RptObject[];
}

export interface Page {
  name: string;
  attrs?: Attr[];
  bands: Band[];
}

export interface Report {
  version: string;
  scriptLanguage?: string;
  info?: Record<string, string>;
  styles?: { name: string; attrs?: Attr[] }[];
  pages: Page[];
}

export interface Sel {
  page: number;
  band: number;
  obj: number; // -1 = band selected
}

export function getBand(rep: Report, sel: Sel): Band | null {
  const p = rep.pages[sel.page];
  if (!p) return null;
  return p.bands[sel.band] ?? null;
}

export function getObject(rep: Report, sel: Sel): RptObject | null {
  const b = getBand(rep, sel);
  if (!b || sel.obj < 0) return null;
  return b.objects[sel.obj] ?? null;
}

export function clone<T>(v: T): T {
  return JSON.parse(JSON.stringify(v)) as T;
}
