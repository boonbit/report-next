import { useRef, useState, type ReactNode } from 'react';
import { clone, getBand, getObject, type Attr, type Report, type RptObject, type Sel } from '../../ir';

// Property inspector for the selected band or object.
// Sel with obj === -1 addresses the band itself; otherwise it
// addresses one object. Known attributes get typed fields (numbers,
// text, selects, checkboxes); anything else stays in the raw table.
// Every edit emits a NEW cloned report.

function getAttr(attrs: Attr[], name: string): string {
  return attrs.find((a) => a.name === name)?.value ?? '';
}

function setAttrIn(attrs: Attr[], name: string, value: string): Attr[] {
  const next = attrs.filter((a) => a.name !== name);
  next.push({ name, value });
  return next;
}

function splitAttrs(attrs: Attr[], handled: Set<string>): { known: Attr[]; rest: Attr[] } {
  const known: Attr[] = [];
  const rest: Attr[] = [];
  for (const a of attrs) (handled.has(a.name) ? known : rest).push(a);
  return { known, rest };
}

function mergeAttrs(known: Attr[], rest: Attr[]): Attr[] {
  return [...known, ...rest];
}

function parseFont(value: string): { family: string; size: string; bold: boolean } {
  const parts = value.split(',').map((s) => s.trim());
  const size = (parts[1] ?? '').replace(/pt$/i, '');
  return { family: parts[0] ?? '', size, bold: /bold/i.test(parts[2] ?? '') };
}

function buildFont(family: string, size: string, bold: boolean): string {
  const fam = family.trim() === '' ? 'Segoe UI' : family.trim();
  const sz = size.trim() === '' ? '10' : size.trim();
  return `${fam}, ${sz}pt${bold ? ', style=Bold' : ''}`;
}

function NumField({
  label,
  aria,
  value,
  onCommit,
}: {
  label: string;
  aria: string;
  value: string;
  onCommit: (v: string) => void;
}) {
  return (
    <label className="field-row">
      {label}
      <input
        className="input"
        type="number"
        aria-label={aria}
        value={value}
        onChange={(e) => onCommit(e.target.value)}
      />
    </label>
  );
}

function TextField({
  label,
  aria,
  value,
  onCommit,
}: {
  label: string;
  aria: string;
  value: string;
  onCommit: (v: string) => void;
}) {
  return (
    <label className="field-row">
      {label}
      <input className="input" aria-label={aria} value={value} onChange={(e) => onCommit(e.target.value)} />
    </label>
  );
}

// Bracket balance for expression text: every [ needs its ].
function bracketsBalanced(s: string): boolean {
  let depth = 0;
  for (let i = 0; i < s.length; i++) {
    const ch: string = s.charAt(i);
    if (ch === '[') depth++;
    else if (ch === ']') {
      depth--;
      if (depth < 0) return false;
    }
  }
  return depth === 0;
}

interface ExprFunc {
  name: string;
  skeleton: string;
}

// Insert skeletons use the real expression syntax: [[...]] blocks with
// [table.column]-style field references (internal/report/expr.go).
const EXPR_FUNCTIONS: ExprFunc[] = [
  { name: 'UpperCase', skeleton: '[[UpperCase([table.column])]]' },
  { name: 'LowerCase', skeleton: '[[LowerCase([table.column])]]' },
  { name: 'Trim', skeleton: '[[Trim([table.column])]]' },
  { name: 'Substring', skeleton: '[[Substring([table.column], 0, 1)]]' },
  { name: 'ToString', skeleton: '[[ToString([table.column])]]' },
  { name: 'Iif', skeleton: '[[Iif([table.column], "yes", "no")]]' },
  { name: 'Abs', skeleton: '[[Abs([table.column])]]' },
  { name: 'Round', skeleton: '[[Round([table.column], 2)]]' },
];

// Text content field with an expression helper. It keeps the same label,
// aria-label and input className as TextField, and edits still flow only
// through onCommit. Unbalanced brackets show an inline error and are
// never committed.
function ExprTextField({
  label,
  aria,
  value,
  onCommit,
}: {
  label: string;
  aria: string;
  value: string;
  onCommit: (v: string) => void;
}) {
  const [draft, setDraft] = useState(value);
  const [error, setError] = useState('');
  const [open, setOpen] = useState(false);
  const inputRef = useRef<HTMLInputElement | null>(null);
  const caretRef = useRef<number | null>(null);
  const [lastValue, setLastValue] = useState(value);
  if (value !== lastValue) {
    setLastValue(value);
    setDraft(value);
    setError('');
    setOpen(false);
  }

  function commitIfBalanced(v: string): void {
    if (!bracketsBalanced(v)) {
      setError('Unbalanced brackets: every [ needs a ]');
      return;
    }
    setError('');
    onCommit(v);
  }

  function insertSkeleton(skeleton: string): void {
    const at: number = caretRef.current ?? draft.length;
    const next: string = draft.slice(0, at) + skeleton + draft.slice(at);
    const cursor: number = at + skeleton.length;
    setDraft(next);
    setError('');
    onCommit(next);
    setOpen(false);
    window.setTimeout(() => {
      const node: HTMLInputElement | null = inputRef.current;
      if (node !== null) {
        node.focus();
        node.setSelectionRange(cursor, cursor);
      }
    }, 0);
  }

  function closePopup(): void {
    setOpen(false);
    inputRef.current?.focus();
  }

  return (
    <label
      className="field-row"
      onKeyDown={(e) => {
        if (e.key === 'Escape' && open) closePopup();
      }}
    >
      {label}
      <input
        ref={inputRef}
        className="input"
        aria-label={aria}
        value={draft}
        onChange={(e) => {
          caretRef.current = e.target.selectionStart ?? e.target.value.length;
          setDraft(e.target.value);
          commitIfBalanced(e.target.value);
        }}
        onSelect={(e) => {
          caretRef.current = e.currentTarget.selectionStart;
        }}
      />
      <span className={open ? 'menu-root open' : 'menu-root'}>
        <button
          type="button"
          className="btn small"
          title="Insert expression"
          aria-label="Insert expression"
          aria-expanded={open}
          aria-haspopup="dialog"
          onClick={() => setOpen((v) => !v)}
        >
          fx
        </button>
        {open ? (
          <div className="menu-drop" role="dialog" title="Expression functions" aria-label="Expression functions">
            {EXPR_FUNCTIONS.map((f) => (
              <button
                key={f.name}
                type="button"
                className="menu-item"
                title={`Insert ${f.name} expression`}
                aria-label={`Insert ${f.name} expression`}
                onClick={() => insertSkeleton(f.skeleton)}
              >
                {f.name}
              </button>
            ))}
          </div>
        ) : null}
      </span>
      {error === '' ? null : <span role="alert">{error}</span>}
    </label>
  );
}

function CheckField({
  label,
  aria,
  checked,
  onCommit,
}: {
  label: string;
  aria: string;
  checked: boolean;
  onCommit: (v: boolean) => void;
}) {
  return (
    <label className="field-row">
      {label}
      <input
        type="checkbox"
        aria-label={aria}
        checked={checked}
        onChange={(e) => onCommit(e.target.checked)}
      />
    </label>
  );
}

function ReadOnlyField({
  label,
  aria,
  value,
}: {
  label: string;
  aria: string;
  value: string;
}) {
  return (
    <div className="field-row" aria-label={aria}>
      <span>{label}</span>
      <span>{value}</span>
    </div>
  );
}

// Summed Width attrs of a band's objects. Non-numeric or missing values
// contribute 0; when nothing parses as numeric the result is an en dash
// (never NaN).
function totalObjectWidth(objects: RptObject[]): string {
  let sum = 0;
  let any = false;
  for (const o of objects) {
    const raw: string = getAttr(o.attrs ?? [], 'Width').trim();
    if (raw === '') continue;
    const f: number = parseFloat(raw);
    if (Number.isNaN(f)) continue;
    sum += f;
    any = true;
  }
  return any ? String(sum) : '–';
}

// Mirrors the objBarcodeQRHTML dispatch in internal/report/objects.go:
// lowercase substring match on the raw Barcode attr text.
function barcodeEngine(raw: string): string {
  const l: string = raw.toLowerCase();
  if (l.includes('datamatrix') || l.includes('data matrix')) return 'datamatrix';
  if (l.includes('pdf417') || l.includes('pdf-417')) return 'pdf417';
  if (l.includes('aztec')) return 'aztec';
  if (l.includes('maxicode') || l.includes('maxi-code')) return 'maxicode';
  if (l.includes('qr')) return 'qr';
  return '1d-fallback';
}

function parseStoredColor(value: string): string | null {
  const v: string = value.trim();
  if (/^#[0-9a-fA-F]{6}$/.test(v)) return v.toLowerCase();
  const short: RegExpExecArray | null = /^#([0-9a-fA-F]{3})$/.exec(v);
  if (short !== null) {
    const h: string = short[1] as string;
    const hex: string = h.toLowerCase();
    return `#${hex[0]}${hex[0]}${hex[1]}${hex[1]}${hex[2]}${hex[2]}`;
  }
  const parts: string[] = v.split(',');
  if (parts.length === 3) {
    const nums: number[] = [];
    for (const p of parts) {
      const t: string = p.trim();
      if (!/^\d{1,3}$/.test(t)) return null;
      const n: number = parseInt(t, 10);
      if (n < 0 || n > 255) return null;
      nums.push(n);
    }
    const hex: string = nums.map((n: number) => n.toString(16).padStart(2, '0')).join('');
    return `#${hex}`;
  }
  return null;
}

function hexToStoredColor(hex: string, original: string): string {
  const norm: string = hex.toLowerCase();
  if (original.trim().startsWith('#')) return norm;
  const r: number = parseInt(norm.slice(1, 3), 16);
  const g: number = parseInt(norm.slice(3, 5), 16);
  const b: number = parseInt(norm.slice(5, 7), 16);
  return `${r}, ${g}, ${b}`;
}

function ColorField({
  label,
  aria,
  value,
  onCommit,
}: {
  label: string;
  aria: string;
  value: string;
  onCommit: (v: string) => void;
}) {
  const hex: string | null = parseStoredColor(value);
  if (hex === null) {
    return <TextField label={label} aria={aria} value={value} onCommit={onCommit} />;
  }
  return (
    <label className="field-row">
      {label}
      <input
        className="input"
        type="color"
        aria-label={aria}
        value={hex}
        onChange={(e) => onCommit(hexToStoredColor(e.target.value, value))}
      />
      <input
        className="input"
        aria-label={`${aria} hex`}
        value={hex}
        onChange={(e) => {
          const parsed: string | null = parseStoredColor(e.target.value);
          if (parsed !== null) onCommit(hexToStoredColor(parsed, value));
        }}
      />
    </label>
  );
}

function AttrRows({
  attrs,
  onAttrs,
}: {
  attrs: Attr[];
  onAttrs: (next: Attr[]) => void;
}) {
  function setAttr(i: number, patch: Partial<Attr>): void {
    const next = attrs.map((a, j) => (j === i ? { ...a, ...patch } : a));
    onAttrs(next);
  }

  function addAttr(): void {
    onAttrs([...attrs, { name: '', value: '' }]);
  }

  function removeAttr(i: number): void {
    onAttrs(attrs.filter((_, j) => j !== i));
  }

  return (
    <div>
      {attrs.map((a, i) => (
        <div key={i} className="attr-row">
          <input
            className="input"
            value={a.name}
            placeholder="name"
            aria-label={`Attribute ${i} name`}
            onChange={(e) => setAttr(i, { name: e.target.value })}
          />
          <input
            className="input"
            value={a.value}
            placeholder="value"
            aria-label={`Attribute ${i} value`}
            onChange={(e) => setAttr(i, { value: e.target.value })}
          />
          <button type="button" className="btn small" aria-label={`Remove attribute ${i}`} onClick={() => removeAttr(i)}>
            Remove
          </button>
        </div>
      ))}
      <button type="button" className="btn small" onClick={addAttr}>
        Add attribute
      </button>
    </div>
  );
}

function EmptySel() {
  return <div className="empty">Select a band or object in the explorer.</div>;
}

function PanelHead({ title }: { title: string }) {
  return <div className="panel-head">{title}</div>;
}

function Section({
  title,
  count,
  actions,
  children,
}: {
  title: string;
  count?: number;
  actions?: ReactNode;
  children: ReactNode;
}) {
  const [open, setOpen] = useState(true);
  return (
    <div className="insp-section">
      <button
        type="button"
        className="insp-section-head"
        aria-expanded={open}
        aria-label={`${title} section`}
        onClick={() => setOpen((v) => !v)}
      >
        <span className="insp-section-toggle">{open ? '▾' : '▸'}</span>
        <span>{title}</span>
        {count !== undefined ? <span className="chip">{count}</span> : null}
        {actions ?? null}
      </button>
      <div className="insp-section-body">{open ? children : null}</div>
    </div>
  );
}

const BAND_HANDLED = new Set(['Width', 'Height']);
const TEXT_HANDLED = new Set(['Left', 'Top', 'Width', 'Height', 'Text', 'Font', 'Border.Width', 'Fill.Color', 'TextFill.Color', 'Border.Color']);
const BARCODE_HANDLED = new Set(['Left', 'Top', 'Width', 'Height', 'Text', 'Barcode', 'ShowText', 'Border.Width', 'Fill.Color', 'TextFill.Color', 'Border.Color']);
const OBJ_HANDLED = new Set(['Left', 'Top', 'Width', 'Height', 'Border.Width', 'Fill.Color', 'TextFill.Color', 'Border.Color']);

const SYMBOLOGIES = ['QR Code', 'DataMatrix', 'PDF417', 'Aztec', 'MaxiCode', 'Code 39'];

function GeometryFields({
  attrs,
  prefix,
  onCommit,
}: {
  attrs: Attr[];
  prefix: string;
  onCommit: (name: string, v: string) => void;
}) {
  return (
    <>
      <NumField label="Left" aria={`${prefix} left`} value={getAttr(attrs, 'Left')} onCommit={(v) => onCommit('Left', v)} />
      <NumField label="Top" aria={`${prefix} top`} value={getAttr(attrs, 'Top')} onCommit={(v) => onCommit('Top', v)} />
      <NumField label="Width" aria={`${prefix} width`} value={getAttr(attrs, 'Width')} onCommit={(v) => onCommit('Width', v)} />
      <NumField label="Height" aria={`${prefix} height`} value={getAttr(attrs, 'Height')} onCommit={(v) => onCommit('Height', v)} />
      <NumField label="Border width" aria={`${prefix} border width`} value={getAttr(attrs, 'Border.Width')} onCommit={(v) => onCommit('Border.Width', v)} />
    </>
  );
}

export function Inspector({
  rep,
  sel,
  onChange,
}: {
  rep: Report | null;
  sel: Sel | null;
  onChange: (next: Report) => void;
}) {
  if (!rep || !sel) {
    return (
      <div className="panel">
        <PanelHead title="Properties" />
        <EmptySel />
      </div>
    );
  }
  const cur: Report = rep;
  const at: Sel = sel;

  if (at.obj === -1) {
    const band = getBand(cur, at);
    if (!band) {
      return (
        <div className="panel">
          <PanelHead title="Properties" />
          <EmptySel />
        </div>
      );
    }
    const { known, rest } = splitAttrs(band.attrs ?? [], BAND_HANDLED);
    function patchBand(patch: { name?: string; attrs?: Attr[] }): void {
      const next = clone(cur);
      const b = next.pages[at.page].bands[at.band];
      if (patch.name !== undefined) b.name = patch.name;
      if (patch.attrs !== undefined) b.attrs = patch.attrs;
      onChange(next);
    }
    const setKnown = (name: string, v: string) =>
      patchBand({ attrs: mergeAttrs(setAttrIn(known, name, v), rest) });
    return (
      <div className="panel">
        <PanelHead title={`Band: ${band.name}`} />
        <div className="panel-scroll">
          <Section title="General" count={5}>
            <label className="field-row">
              Name
              <input className="input" value={band.name} aria-label="Band name" onChange={(e) => patchBand({ name: e.target.value })} />
            </label>
            <NumField label="Width" aria="Band width" value={getAttr(known, 'Width')} onCommit={(v) => setKnown('Width', v)} />
            <NumField label="Height" aria="Band height" value={getAttr(known, 'Height')} onCommit={(v) => setKnown('Height', v)} />
            <ReadOnlyField label="Objects" aria="Readonly objects" value={String((band.objects ?? []).length)} />
            <ReadOnlyField label="Total width" aria="Readonly total width" value={totalObjectWidth(band.objects ?? [])} />
          </Section>
          <Section title="Attributes">
            <AttrRows attrs={rest} onAttrs={(r) => patchBand({ attrs: mergeAttrs(known, r) })} />
          </Section>
        </div>
      </div>
    );
  }

  const obj = getObject(cur, at);
  if (!obj) {
    return (
      <div className="panel">
        <PanelHead title="Properties" />
        <EmptySel />
      </div>
    );
  }
  const handled = obj.kind === 'TextObject' ? TEXT_HANDLED : obj.kind === 'BarcodeObject' ? BARCODE_HANDLED : OBJ_HANDLED;
  const { known, rest } = splitAttrs(obj.attrs ?? [], handled);
  function patchObj(patch: { name?: string; attrs?: Attr[] }): void {
    const next = clone(cur);
    const o = next.pages[at.page].bands[at.band].objects[at.obj];
    if (patch.name !== undefined) o.name = patch.name;
    if (patch.attrs !== undefined) o.attrs = patch.attrs;
    onChange(next);
  }
  const setKnown = (name: string, v: string) =>
    patchObj({ attrs: mergeAttrs(setAttrIn(known, name, v), rest) });
  const font = parseFont(getAttr(known, 'Font'));
  const setFont = (f: { family: string; size: string; bold: boolean }) =>
    setKnown('Font', buildFont(f.family, f.size, f.bold));
  const readonlyCharacters: string = String(getAttr(known, 'Text').length);
  const barcodeRaw: string = getAttr(known, 'Barcode');
  const engine: string = barcodeEngine(barcodeRaw);
  const readonlyEngine: string = barcodeRaw === '' ? engine : `${barcodeRaw} (${engine})`;

  const generalCount = obj.kind === 'TextObject' || obj.kind === 'BarcodeObject' ? 2 : 1;
  return (
    <div className="panel">
      <PanelHead title={`${obj.kind}: ${obj.name}`} />
      <div className="panel-scroll">
        <Section title="General" count={generalCount}>
          <label className="field-row">
            Name
            <input className="input" value={obj.name} aria-label="Object name" onChange={(e) => patchObj({ name: e.target.value })} />
          </label>
          {obj.kind === 'TextObject' ? (
            <ReadOnlyField label="Characters" aria="Readonly characters" value={readonlyCharacters} />
          ) : null}
          {obj.kind === 'BarcodeObject' ? (
            <ReadOnlyField label="Engine" aria="Readonly engine" value={readonlyEngine} />
          ) : null}
        </Section>
        {obj.kind === 'TextObject' ? (
          <Section title="Text" count={4}>
            <ExprTextField label="Text" aria="Text content" value={getAttr(known, 'Text')} onCommit={(v) => setKnown('Text', v)} />
            <TextField label="Font" aria="Font family" value={font.family} onCommit={(v) => setFont({ ...font, family: v })} />
            <NumField label="Font size" aria="Font size" value={font.size} onCommit={(v) => setFont({ ...font, size: v })} />
            <CheckField label="Bold" aria="Bold" checked={font.bold} onCommit={(v) => setFont({ ...font, bold: v })} />
          </Section>
        ) : null}
        {obj.kind === 'BarcodeObject' ? (
          <Section title="Barcode" count={3}>
            <label className="field-row">
              Symbology
              <select
                className="input"
                aria-label="Barcode symbology"
                value={getAttr(known, 'Barcode')}
                onChange={(e) => setKnown('Barcode', e.target.value)}
              >
                {Array.from(new Set([getAttr(known, 'Barcode'), ...SYMBOLOGIES])).map((s) => (
                  <option key={s} value={s}>{s === '' ? '(none)' : s}</option>
                ))}
              </select>
            </label>
            <TextField label="Text" aria="Barcode text" value={getAttr(known, 'Text')} onCommit={(v) => setKnown('Text', v)} />
            <CheckField
              label="Show text"
              aria="Show barcode text"
              checked={getAttr(known, 'ShowText').toLowerCase() !== 'false'}
              onCommit={(v) => setKnown('ShowText', v ? 'true' : 'false')}
            />
          </Section>
        ) : null}
        <Section title="Layout" count={8}>
          <GeometryFields attrs={known} prefix="Object" onCommit={setKnown} />
          <ColorField label="Fill color" aria="Color Fill" value={getAttr(known, 'Fill.Color')} onCommit={(v) => setKnown('Fill.Color', v)} />
          <ColorField label="Text fill" aria="Color TextFill" value={getAttr(known, 'TextFill.Color')} onCommit={(v) => setKnown('TextFill.Color', v)} />
          <ColorField label="Border color" aria="Color Border" value={getAttr(known, 'Border.Color')} onCommit={(v) => setKnown('Border.Color', v)} />
        </Section>
        <Section title="Attributes">
          <AttrRows attrs={rest} onAttrs={(r) => patchObj({ attrs: mergeAttrs(known, r) })} />
        </Section>
      </div>
    </div>
  );
}
