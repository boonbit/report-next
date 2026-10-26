import { clone, getBand, getObject, type Report, type RptObject, type Sel } from "../ir";

// Pure report mutations for the explorer. Every function clones and
// returns the next report plus the selection to show afterwards
// (Sel | null when nothing sensible remains selected).

function uniqueName(base: string, taken: Set<string>): string {
  if (!taken.has(base)) return base;
  let i = 2;
  while (taken.has(`${base} ${i}`)) i++;
  return `${base} ${i}`;
}

function textDefaults(name: string): RptObject {
  return {
    kind: "TextObject",
    name,
    attrs: [
      { name: "Left", value: "9.45" },
      { name: "Top", value: "9.45" },
      { name: "Width", value: "200" },
      { name: "Height", value: "28.35" },
      { name: "Text", value: "New text" },
    ],
  };
}

function barcodeDefaults(name: string): RptObject {
  return {
    kind: "BarcodeObject",
    name,
    attrs: [
      { name: "Left", value: "9.45" },
      { name: "Top", value: "9.45" },
      { name: "Width", value: "150" },
      { name: "Height", value: "75" },
      { name: "Barcode", value: "QR" },
      { name: "Text", value: "12345" },
    ],
  };
}

export type NewObjectKind = "TextObject" | "BarcodeObject";

export function addBand(rep: Report, page: number, kind = "DataBand"): { next: Report; sel: Sel } {
  const next = clone(rep);
  const p = next.pages[page];
  const fallback = { next: rep, sel: { page, band: 0, obj: -1 } };
  if (!p) return fallback;
  const taken = new Set(p.bands.map((b) => b.name));
  const name = uniqueName(kind.replace(/Band$/, "") + "1", taken);
  p.bands.push({ kind, name, attrs: [{ name: "Width", value: "718.2" }, { name: "Height", value: "37.8" }], objects: [] });
  return { next, sel: { page, band: p.bands.length - 1, obj: -1 } };
}

export function deleteBand(rep: Report, sel: Sel): { next: Report; sel: Sel | null } {
  const next = clone(rep);
  const p = next.pages[sel.page];
  if (!p || !p.bands[sel.band]) return { next: rep, sel };
  p.bands.splice(sel.band, 1);
  if (p.bands.length === 0) return { next, sel: null };
  const band = Math.min(sel.band, p.bands.length - 1);
  return { next, sel: { page: sel.page, band, obj: -1 } };
}

export function addObject(rep: Report, sel: Sel, kind: NewObjectKind): { next: Report; sel: Sel } {
  const next = clone(rep);
  const band = getBand(next, sel);
  if (!band) return { next: rep, sel };
  const taken = new Set(band.objects.map((o) => o.name));
  const base = kind === "TextObject" ? "Text" : "Barcode";
  let n = 1;
  while (taken.has(base + n)) n++;
  const name = base + n;
  const obj = kind === "TextObject" ? textDefaults(name) : barcodeDefaults(name);
  band.objects.push(obj);
  return { next, sel: { page: sel.page, band: sel.band, obj: band.objects.length - 1 } };
}

export function deleteObject(rep: Report, sel: Sel): { next: Report; sel: Sel | null } {
  const next = clone(rep);
  const band = getBand(next, sel);
  if (!band || sel.obj < 0 || !band.objects[sel.obj]) return { next: rep, sel };
  band.objects.splice(sel.obj, 1);
  if (band.objects.length === 0) return { next, sel: { page: sel.page, band: sel.band, obj: -1 } };
  return { next, sel: { page: sel.page, band: sel.band, obj: Math.min(sel.obj, band.objects.length - 1) } };
}

export function duplicateObject(rep: Report, sel: Sel): { next: Report; sel: Sel } {
  const next = clone(rep);
  const band = getBand(next, sel);
  const src = getObject(next, sel);
  if (!band || !src || sel.obj < 0) return { next: rep, sel };
  const taken = new Set(band.objects.map((o) => o.name));
  const copy = clone(src);
  copy.name = uniqueName(src.name + " copy", taken);
  for (const a of copy.attrs ?? []) {
    if (a.name === "Left") a.value = String((parseFloat(a.value) || 0) + 18.9);
    if (a.name === "Top") a.value = String((parseFloat(a.value) || 0) + 18.9);
  }
  band.objects.push(copy);
  return { next, sel: { page: sel.page, band: sel.band, obj: band.objects.length - 1 } };
}
