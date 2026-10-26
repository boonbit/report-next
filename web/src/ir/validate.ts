import type { Report, Sel } from '../ir';

// Client-side report checks (no server round-trip). Each issue carries
// the node to jump to when clicked.

export interface Issue {
  sel: Sel;
  message: string;
  severity: 'error' | 'warning';
}

function num(v: string | undefined, fallback: number): number {
  if (v === undefined || v === '') return fallback;
  const n = parseFloat(v);
  return Number.isFinite(n) ? n : fallback;
}

function attr(attrs: { name: string; value: string }[] | undefined, name: string): string {
  return attrs?.find((a) => a.name === name)?.value ?? '';
}

export function computeIssues(rep: Report): Issue[] {
  const out: Issue[] = [];
  rep.pages.forEach((page, pi) => {
    page.bands.forEach((band, bi) => {
      const at = { page: pi, band: bi, obj: -1 };
      if (band.objects.length === 0) {
        out.push({ sel: at, message: `Band ${band.name} is empty`, severity: 'warning' });
      }
      const bandW = num(attr(band.attrs, 'Width') || undefined, 0);
      const seen = new Set<string>();
      band.objects.forEach((obj, oi) => {
        const oat = { page: pi, band: bi, obj: oi };
        if (seen.has(obj.name)) {
          out.push({ sel: oat, message: `Duplicate object name ${obj.name}`, severity: 'error' });
        } else {
          seen.add(obj.name);
        }
        if (bandW > 0) {
          const right = num(attr(obj.attrs, 'Left') || undefined, 0) + num(attr(obj.attrs, 'Width') || undefined, 0);
          if (right > bandW + 0.01) {
            out.push({ sel: oat, message: `Object ${obj.name} overflows band width`, severity: 'error' });
          }
        }
        for (const a of obj.attrs ?? []) {
          if (a.name.trim() === '') {
            out.push({ sel: oat, message: `Object ${obj.name} has an unnamed attribute`, severity: 'error' });
            break;
          }
        }
      });
    });
  });
  return out;
}
