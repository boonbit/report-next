// HTTP API client for the report designer backend (see SrvMux in
// internal/report/server.go). All calls are same-origin; vite proxies
// /api to the Go server during dev.
import type { Report } from './ir';

async function fail(res: Response, what: string): Promise<never> {
  const text = await res.text().catch(() => '');
  throw new Error(`${what} failed: ${res.status} ${text}`);
}

export async function fetchSample(name?: string): Promise<Report> {
  const res = await fetch('/api/sample' + (name ? `?name=${encodeURIComponent(name)}` : ''));
  if (!res.ok) await fail(res, 'sample');
  return (await res.json()) as Report;
}

export async function fetchSamples(): Promise<string[]> {
  const res = await fetch('/api/samples');
  if (!res.ok) await fail(res, 'samples');
  return (await res.json()) as string[];
}

export async function importFrx(buf: ArrayBuffer): Promise<Report> {
  const res = await fetch('/api/import', { method: 'POST', body: buf });
  if (!res.ok) await fail(res, 'import');
  return (await res.json()) as Report;
}

export interface RenderedPage {
  html: string;
  total: number;
}

export async function renderPreview(rep: Report, page?: number): Promise<RenderedPage> {
  const res = await fetch('/api/render' + (page ? `?page=${page}` : ''), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(rep),
  });
  if (!res.ok) await fail(res, 'render');
  return { html: await res.text(), total: Number(res.headers.get('X-Total-Pages')) || 1 };
}

export interface OutlineItem {
  title: string;
  page: number;
}

export async function fetchOutline(rep: Report): Promise<OutlineItem[]> {
  const res = await fetch('/api/outline', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(rep),
  });
  if (!res.ok) await fail(res, 'outline');
  const body = (await res.json()) as { outline: OutlineItem[] };
  return body.outline ?? [];
}

export async function exportDoc(kind: 'pdf' | 'docx', rep: Report): Promise<Blob> {
  const res = await fetch(`/api/${kind}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(rep),
  });
  if (!res.ok) await fail(res, kind);
  return res.blob();
}

// U8b: report-level text-row exports (distinct from /api/xlsx ?source=).
export async function exportReportFile(kind: 'csv' | 'xlsx', rep: Report): Promise<Blob> {
  const route = kind === 'csv' ? '/api/csv' : '/api/xlsx-report';
  const res = await fetch(route, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(rep),
  });
  if (!res.ok) await fail(res, kind);
  return res.blob();
}

export interface SchemaTable {
  name: string;
  columns: { name: string; type: string }[];
}

export async function fetchSchema(req: {
  kind: string;
  dsn: string;
  content: string;
}): Promise<{ tables: SchemaTable[] }> {
  const res = await fetch('/api/schema', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  });
  if (!res.ok) await fail(res, 'schema');
  return (await res.json()) as { tables: SchemaTable[] };
}
