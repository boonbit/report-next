import { useEffect, useRef, useState } from 'react';
import { clone, type Attr, type Band, type Page, type Report, type Sel } from '../../ir';

// Visual design surface: pages as sheets, bands as stacked blocks,
// objects as absolutely positioned boxes (1 unit = 1px at 100% zoom, the
// same scale as the HTML preview). Click selects (shared Sel with
// tree/inspector); drag moves the object and commits once on release (one
// undo entry); the south-east handle resizes Width/Height the same way.
// While dragging/resizing, a moving edge within 3 units of a band edge or
// a sibling edge snaps to it and flashes a dashed blue guide; guides clear
// on pointer-up. Ctrl+wheel zooms display scale only (committed attrs are
// zoom-independent); Space-drag pans the scroll container.

function num(attrs: Attr[] | undefined, name: string, fallback: number): number {
  const raw = attrs?.find((a) => a.name === name)?.value;
  const v = raw === undefined || raw === '' ? NaN : parseFloat(raw);
  return Number.isFinite(v) ? (v as number) : fallback;
}

interface Drag {
  sel: Sel;
  baseLeft: number;
  baseTop: number;
  startX: number;
  startY: number;
  dx: number;
  dy: number;
}

interface Resize {
  sel: Sel;
  baseW: number;
  baseH: number;
  startX: number;
  startY: number;
  dw: number;
  dh: number;
}

interface Guides {
  x: number[];
  y: number[];
}

interface Pan {
  startX: number;
  startY: number;
  scrollLeft: number;
  scrollTop: number;
}

const EMPTY_GUIDES: Guides = { x: [], y: [] };

const SNAP_TOLERANCE = 3;
const MIN_ZOOM = 0.25;
const MAX_ZOOM = 2;
const ZOOM_STEP = 0.25;

function clampZoom(z: number): number {
  const stepped = Math.round(z / ZOOM_STEP) * ZOOM_STEP;
  return Math.min(MAX_ZOOM, Math.max(MIN_ZOOM, Math.round(stepped * 100) / 100));
}

// Zoom presets offered by the floating zoom control (percent values).
const ZOOM_PRESETS = [50, 75, 100, 125, 150];
// Ruler ticks every 50 units, numeric labels every 100 units.
const RULER_STEP = 50;
const RULER_LABEL_STEP = 100;

function rulerTicks(max: number): number[] {
  const out: number[] = [];
  for (let v = 0; v <= max; v += RULER_STEP) out.push(v);
  return out;
}

function widestSheet(rep: Report): number {
  return Math.max(400, ...rep.pages.map((p) => sheetWidth(p)));
}

// Total vertical content extent in units: stacked band heights plus any
// object bottoms (via objBox) that reach further, plus page gaps/padding.
function contentHeight(rep: Report): number {
  let total = 0;
  rep.pages.forEach((page, pi) => {
    let y = 0;
    let pageBottom = 0;
    page.bands.forEach((band) => {
      const h = bandHeight(band);
      band.objects.forEach((o) => {
        const b = objBox(o);
        pageBottom = Math.max(pageBottom, y + b.top + b.h);
      });
      y += h;
      pageBottom = Math.max(pageBottom, y);
    });
    total += pageBottom;
    if (pi < rep.pages.length - 1) total += 24;
  });
  return total + 32;
}

function sheetWidth(page: Page): number {
  return Math.max(400, ...page.bands.map((b) => num(b.attrs, 'Width', 0)));
}

function bandWidth(page: Page, band: Band): number {
  const raw = num(band.attrs, 'Width', Number.NaN);
  return Number.isFinite(raw) && raw > 0 ? raw : sheetWidth(page);
}

function bandHeight(band: Band): number {
  return Math.max(24, num(band.attrs, 'Height', 40));
}

function objBox(obj: { attrs?: Attr[] }): { left: number; top: number; w: number; h: number } {
  return {
    left: num(obj.attrs, 'Left', 0),
    top: num(obj.attrs, 'Top', 0),
    w: Math.max(8, num(obj.attrs, 'Width', 120)),
    h: Math.max(8, num(obj.attrs, 'Height', 20)),
  };
}

function siblingEdges(band: Band, skip: number): { xs: number[]; ys: number[] } {
  const xs: number[] = [];
  const ys: number[] = [];
  band.objects.forEach((o, oi) => {
    if (oi === skip) return;
    const b = objBox(o);
    xs.push(b.left, b.left + b.w);
    ys.push(b.top, b.top + b.h);
  });
  return { xs, ys };
}

// Snap a moving span [rawMin, rawMin + size] so either edge pulls to the
// nearest target within tolerance. Returns the snapped min and the guide
// position (unit coords), or null when nothing is in range.
function snapAxis(rawMin: number, size: number, targets: number[]): { value: number; guide: number | null } {
  let value = rawMin;
  let guide: number | null = null;
  let best = SNAP_TOLERANCE + 1;
  for (const t of targets) {
    const dMin = Math.abs(rawMin - t);
    if (dMin <= SNAP_TOLERANCE && dMin < best) {
      best = dMin;
      value = t;
      guide = t;
    }
    const dMax = Math.abs(rawMin + size - t);
    if (dMax <= SNAP_TOLERANCE && dMax < best) {
      best = dMax;
      value = t - size;
      guide = t;
    }
  }
  return { value, guide };
}

// Snap a resizing span whose min edge is fixed: only the moving max edge
// (fixed + rawSize) pulls to a target.
function snapSize(fixed: number, rawSize: number, targets: number[]): { size: number; guide: number | null } {
  let size = rawSize;
  let guide: number | null = null;
  let best = SNAP_TOLERANCE + 1;
  for (const t of targets) {
    const d = Math.abs(fixed + rawSize - t);
    if (d <= SNAP_TOLERANCE && d < best) {
      best = d;
      size = t - fixed;
      guide = t;
    }
  }
  return { size, guide };
}

function guidesEqual(a: Guides, b: Guides): boolean {
  return (
    a.x.length === b.x.length &&
    a.y.length === b.y.length &&
    a.x.every((v, i) => v === b.x[i]) &&
    a.y.every((v, i) => v === b.y[i])
  );
}

function isEditableTarget(t: EventTarget | null): boolean {
  if (!(t instanceof HTMLElement)) return false;
  const tag = t.tagName;
  return tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT' || t.isContentEditable;
}

function sameSel(a: Sel, b: Sel): boolean {
  return a.page === b.page && a.band === b.band && a.obj === b.obj;
}

type AlignMode =
  | 'left'
  | 'center'
  | 'right'
  | 'top'
  | 'middle'
  | 'bottom'
  | 'sameWidth'
  | 'sameHeight'
  | 'distribute';

export function ReportCanvas({
  rep,
  sel,
  onSelect,
  onChange,
}: {
  rep: Report;
  sel: Sel | null;
  onSelect: (s: Sel | null) => void;
  onChange: (next: Report) => void;
}) {
  const [drag, setDrag] = useState<Drag | null>(null);
  const dragRef = useRef<Drag | null>(null);
  const [resize, setResize] = useState<Resize | null>(null);
  const resizeRef = useRef<Resize | null>(null);
  const [guides, setGuides] = useState<Guides>(EMPTY_GUIDES);
  const [zoom, setZoom] = useState(1);
  const [spaceHeld, setSpaceHeld] = useState(false);
  // Canvas-local multi-select set (object Sels only). Independent of the
  // shared single Sel: shift+click toggles, plain click resets to one.
  const [multi, setMulti] = useState<Sel[]>([]);
  const scrollRef = useRef<HTMLDivElement | null>(null);
  const panRef = useRef<Pan | null>(null);
  const panMovedRef = useRef(false);

  // Space-held modifier for panning. Ignored inside editable fields.
  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent): void => {
      if (e.code !== 'Space' || e.repeat || isEditableTarget(e.target)) return;
      e.preventDefault();
      setSpaceHeld(true);
    };
    const onKeyUp = (e: KeyboardEvent): void => {
      if (e.code !== 'Space') return;
      setSpaceHeld(false);
    };
    const onBlur = (): void => {
      setSpaceHeld(false);
      panRef.current = null;
    };
    window.addEventListener('keydown', onKeyDown);
    window.addEventListener('keyup', onKeyUp);
    window.addEventListener('blur', onBlur);
    return () => {
      window.removeEventListener('keydown', onKeyDown);
      window.removeEventListener('keyup', onKeyUp);
      window.removeEventListener('blur', onBlur);
    };
  }, []);

  // Escape clears the canvas-local multi-select set. Ignored in fields.
  useEffect(() => {
    const onKey = (e: KeyboardEvent): void => {
      if (e.key !== 'Escape' || isEditableTarget(e.target)) return;
      setMulti([]);
    };
    window.addEventListener('keydown', onKey);
    return () => {
      window.removeEventListener('keydown', onKey);
    };
  }, []);

  // Ctrl+wheel zoom on the scroll container. Native listener (passive:
  // false) so preventDefault stops the browser zoom/scroll gesture.
  useEffect(() => {
    const el = scrollRef.current;
    if (!el) return;
    const onWheel = (e: WheelEvent): void => {
      if (!e.ctrlKey) return;
      e.preventDefault();
      const dir = e.deltaY > 0 ? -1 : 1;
      setZoom((z) => clampZoom(z + dir * ZOOM_STEP));
    };
    el.addEventListener('wheel', onWheel, { passive: false });
    return () => {
      el.removeEventListener('wheel', onWheel);
    };
  }, []);

  function beginDrag(e: React.PointerEvent, at: Sel, baseLeft: number, baseTop: number): void {
    e.stopPropagation();
    onSelect(at);
    if (e.shiftKey) {
      setMulti((prev) =>
        prev.some((s) => sameSel(s, at)) ? prev.filter((s) => !sameSel(s, at)) : [...prev, at],
      );
    } else {
      setMulti([at]);
    }
    (e.target as HTMLElement).setPointerCapture?.(e.pointerId);
    const d: Drag = { sel: at, baseLeft, baseTop, startX: e.clientX, startY: e.clientY, dx: 0, dy: 0 };
    dragRef.current = d;
    setDrag(d);
  }

  function moveDrag(e: React.PointerEvent): void {
    const d = dragRef.current;
    if (!d) return;
    // Pointer deltas are screen px: divide by zoom to get canvas units so
    // the committed Left/Top are identical at any zoom.
    const dxRaw = Math.round((e.clientX - d.startX) / zoom);
    const dyRaw = Math.round((e.clientY - d.startY) / zoom);
    let dx = dxRaw;
    let dy = dyRaw;
    let nextGuides: Guides = EMPTY_GUIDES;
    const page = rep.pages[d.sel.page];
    const band = page?.bands[d.sel.band];
    const obj = band?.objects[d.sel.obj];
    if (page && band && obj) {
      const box = objBox(obj);
      const edges = siblingEdges(band, d.sel.obj);
      const sx = snapAxis(d.baseLeft + dxRaw, box.w, [0, bandWidth(page, band), ...edges.xs]);
      const sy = snapAxis(d.baseTop + dyRaw, box.h, [0, bandHeight(band), ...edges.ys]);
      dx = Math.round(sx.value - d.baseLeft);
      dy = Math.round(sy.value - d.baseTop);
      if (sx.guide !== null || sy.guide !== null) {
        nextGuides = {
          x: sx.guide !== null ? [sx.guide] : [],
          y: sy.guide !== null ? [sy.guide] : [],
        };
      }
    }
    if (dx !== d.dx || dy !== d.dy) {
      const nd = { ...d, dx, dy };
      dragRef.current = nd;
      setDrag(nd);
    }
    if (!guidesEqual(nextGuides, guides)) setGuides(nextGuides);
  }

  function endDrag(): void {
    const d = dragRef.current;
    dragRef.current = null;
    setDrag(null);
    setGuides(EMPTY_GUIDES);
    if (!d || (d.dx === 0 && d.dy === 0)) return;
    const next = clone(rep);
    const band = next.pages[d.sel.page]?.bands[d.sel.band];
    const obj = band?.objects[d.sel.obj];
    if (!band || !obj) return;
    const setAttr = (name: string, v: string) => {
      const rest = (obj.attrs ?? []).filter((a) => a.name !== name);
      obj.attrs = [...rest, { name, value: v }];
    };
    setAttr('Left', String(Math.max(0, Math.round(d.baseLeft + d.dx))));
    setAttr('Top', String(Math.max(0, Math.round(d.baseTop + d.dy))));
    onChange(next);
  }

  function beginResize(e: React.PointerEvent, at: Sel, baseW: number, baseH: number): void {
    e.stopPropagation();
    onSelect(at);
    (e.target as HTMLElement).setPointerCapture?.(e.pointerId);
    const r: Resize = { sel: at, baseW, baseH, startX: e.clientX, startY: e.clientY, dw: 0, dh: 0 };
    resizeRef.current = r;
    setResize(r);
  }

  function moveResize(e: React.PointerEvent): void {
    const r = resizeRef.current;
    if (!r) return;
    const dwRaw = Math.round((e.clientX - r.startX) / zoom);
    const dhRaw = Math.round((e.clientY - r.startY) / zoom);
    let dw = dwRaw;
    let dh = dhRaw;
    let nextGuides: Guides = EMPTY_GUIDES;
    const page = rep.pages[r.sel.page];
    const band = page?.bands[r.sel.band];
    const obj = band?.objects[r.sel.obj];
    if (page && band && obj) {
      const box = objBox(obj);
      const edges = siblingEdges(band, r.sel.obj);
      const sx = snapSize(box.left, r.baseW + dwRaw, [0, bandWidth(page, band), ...edges.xs]);
      const sy = snapSize(box.top, r.baseH + dhRaw, [0, bandHeight(band), ...edges.ys]);
      dw = Math.round(sx.size - r.baseW);
      dh = Math.round(sy.size - r.baseH);
      if (sx.guide !== null || sy.guide !== null) {
        nextGuides = {
          x: sx.guide !== null ? [sx.guide] : [],
          y: sy.guide !== null ? [sy.guide] : [],
        };
      }
    }
    if (dw !== r.dw || dh !== r.dh) {
      const nr = { ...r, dw, dh };
      resizeRef.current = nr;
      setResize(nr);
    }
    if (!guidesEqual(nextGuides, guides)) setGuides(nextGuides);
  }

  function endResize(): void {
    const r = resizeRef.current;
    resizeRef.current = null;
    setResize(null);
    setGuides(EMPTY_GUIDES);
    if (!r || (r.dw === 0 && r.dh === 0)) return;
    const next = clone(rep);
    const band = next.pages[r.sel.page]?.bands[r.sel.band];
    const obj = band?.objects[r.sel.obj];
    if (!band || !obj) return;
    const setAttr = (name: string, v: string) => {
      const rest = (obj.attrs ?? []).filter((a) => a.name !== name);
      obj.attrs = [...rest, { name, value: v }];
    };
    setAttr('Width', String(Math.max(8, Math.round(r.baseW + r.dw))));
    setAttr('Height', String(Math.max(8, Math.round(r.baseH + r.dh))));
    onChange(next);
  }

  // Align the canvas-local multi-select set. Every mode derives one
  // reference from objBox edges and applies a single onChange commit, so
  // the whole alignment is one undo entry. Distribute keeps the first and
  // last left edges fixed and spaces the gaps between boxes evenly.
  function alignSelected(mode: AlignMode): void {
    const items: { s: Sel; box: { left: number; top: number; w: number; h: number } }[] = [];
    for (const s of multi) {
      if (s.obj < 0) continue;
      const o = rep.pages[s.page]?.bands[s.band]?.objects[s.obj];
      if (!o) continue;
      items.push({ s, box: objBox(o) });
    }
    if (items.length < 2) return;
    const next = clone(rep);
    const writeAttr = (s: Sel, name: string, value: number): void => {
      const obj = next.pages[s.page]?.bands[s.band]?.objects[s.obj];
      if (!obj) return;
      const floor = name === 'Width' || name === 'Height' ? 8 : 0;
      const rest = (obj.attrs ?? []).filter((a) => a.name !== name);
      obj.attrs = [...rest, { name, value: String(Math.max(floor, Math.round(value))) }];
    };
    if (mode === 'distribute') {
      if (items.length < 3) return;
      const ordered = [...items].sort((a, b) => a.box.left - b.box.left);
      const firstLeft = ordered[0].box.left;
      const lastLeft = ordered[ordered.length - 1].box.left;
      const leadingWidths = ordered.slice(0, -1).reduce((acc, it) => acc + it.box.w, 0);
      const gap = (lastLeft - firstLeft - leadingWidths) / (ordered.length - 1);
      let x = firstLeft;
      ordered.forEach((it, i) => {
        if (i === 0) {
          x = firstLeft + it.box.w + gap;
          return;
        }
        if (i === ordered.length - 1) return;
        writeAttr(it.s, 'Left', x);
        x += it.box.w + gap;
      });
      onChange(next);
      return;
    }
    const minLeft = Math.min(...items.map((it) => it.box.left));
    const maxRight = Math.max(...items.map((it) => it.box.left + it.box.w));
    const minTop = Math.min(...items.map((it) => it.box.top));
    const maxBottom = Math.max(...items.map((it) => it.box.top + it.box.h));
    const maxW = Math.max(...items.map((it) => it.box.w));
    const maxH = Math.max(...items.map((it) => it.box.h));
    for (const it of items) {
      switch (mode) {
        case 'left':
          writeAttr(it.s, 'Left', minLeft);
          break;
        case 'center':
          writeAttr(it.s, 'Left', (minLeft + maxRight) / 2 - it.box.w / 2);
          break;
        case 'right':
          writeAttr(it.s, 'Left', maxRight - it.box.w);
          break;
        case 'top':
          writeAttr(it.s, 'Top', minTop);
          break;
        case 'middle':
          writeAttr(it.s, 'Top', (minTop + maxBottom) / 2 - it.box.h / 2);
          break;
        case 'bottom':
          writeAttr(it.s, 'Top', maxBottom - it.box.h);
          break;
        case 'sameWidth':
          writeAttr(it.s, 'Width', maxW);
          break;
        case 'sameHeight':
          writeAttr(it.s, 'Height', maxH);
          break;
      }
    }
    onChange(next);
  }

  function beginPan(e: React.PointerEvent<HTMLDivElement>): void {
    if (!spaceHeld) return;
    panMovedRef.current = false;
    panRef.current = {
      startX: e.clientX,
      startY: e.clientY,
      scrollLeft: e.currentTarget.scrollLeft,
      scrollTop: e.currentTarget.scrollTop,
    };
    e.currentTarget.setPointerCapture?.(e.pointerId);
  }

  function movePan(e: React.PointerEvent<HTMLDivElement>): void {
    const p = panRef.current;
    if (!p) return;
    const dx = e.clientX - p.startX;
    const dy = e.clientY - p.startY;
    if (Math.abs(dx) + Math.abs(dy) > 2) panMovedRef.current = true;
    e.currentTarget.scrollLeft = p.scrollLeft - dx;
    e.currentTarget.scrollTop = p.scrollTop - dy;
  }

  function endPan(): void {
    panRef.current = null;
  }

  // Display-only fit: scale so the widest sheet fills the scroll viewport.
  function fitWidth(): void {
    const el = scrollRef.current;
    if (!el) return;
    const avail = el.clientWidth - 32;
    if (avail <= 0) return;
    setZoom(clampZoom(avail / widestSheet(rep)));
  }

  const rulerW = widestSheet(rep);
  const rulerH = contentHeight(rep);
  const topTicks = rulerTicks(rulerW);
  const leftTicks = rulerTicks(rulerH);
  // Multi-selected objects that still exist in the report.
  const validMulti = multi.filter(
    (s) => s.obj >= 0 && rep.pages[s.page]?.bands[s.band]?.objects[s.obj] !== undefined,
  );

  return (
    <div
      className="panel"
      style={{ flex: 1 }}
      onPointerMove={(e) => {
        moveDrag(e);
        moveResize(e);
      }}
      onPointerUp={() => {
        endDrag();
        endResize();
      }}
    >
      <div className="panel-head">
        <span>Design canvas</span>
        <span aria-label="Canvas zoom">{`${Math.round(zoom * 100)}%`}</span>
        <button type="button" aria-label="Reset zoom" title="Reset zoom to 100%" onClick={() => setZoom(1)}>
          Reset
        </button>
      </div>
      <div
        ref={scrollRef}
        className="canvas-scroll"
        title="Ctrl+wheel zooms, Space+drag pans"
        style={spaceHeld ? { cursor: 'grab' } : undefined}
        onClick={() => {
          // A space-drag pan ends with a click: swallow it so the
          // selection is not cleared.
          if (panMovedRef.current) {
            panMovedRef.current = false;
            return;
          }
          onSelect(null);
          setMulti([]);
        }}
        onPointerDown={beginPan}
        onPointerMove={movePan}
        onPointerUp={endPan}
        onPointerCancel={endPan}
      >
        <div
          style={{
            position: 'sticky',
            top: 8,
            zIndex: 12,
            height: 0,
            overflow: 'visible',
            display: 'flex',
            justifyContent: 'flex-end',
            pointerEvents: 'none',
          }}
          onClick={(e) => e.stopPropagation()}
        >
          <div
            role="group"
            aria-label="Zoom presets"
            style={{
              pointerEvents: 'auto',
              display: 'flex',
              gap: 4,
              alignItems: 'center',
              background: 'var(--ide-panel, #f8f9fb)',
              border: '1px solid var(--ide-line, #dadce0)',
              borderRadius: 4,
              padding: '2px 4px',
            }}
          >
            {ZOOM_PRESETS.map((pct) => (
              <button
                key={pct}
                type="button"
                title={`Set zoom to ${pct}%`}
                aria-label={`Set zoom to ${pct}%`}
                onClick={() => setZoom(clampZoom(pct / 100))}
              >
                {pct}%
              </button>
            ))}
            <button type="button" title="Fit width to canvas" aria-label="Fit width" onClick={fitWidth}>
              Fit
            </button>
          </div>
        </div>
        {validMulti.length >= 2 ? (
          <div
            style={{
              position: 'sticky',
              top: 8,
              zIndex: 12,
              height: 0,
              overflow: 'visible',
              display: 'flex',
              justifyContent: 'flex-start',
              pointerEvents: 'none',
            }}
            onClick={(e) => e.stopPropagation()}
          >
            <div
              role="group"
              aria-label="Align selected objects"
              style={{
                pointerEvents: 'auto',
                display: 'flex',
                gap: 4,
                alignItems: 'center',
                background: 'var(--ide-panel, #f8f9fb)',
                border: '1px solid var(--ide-line, #dadce0)',
                borderRadius: 4,
                padding: '2px 4px',
              }}
            >
              <button type="button" title="Align left edges" aria-label="Align left edges" onClick={() => alignSelected('left')} style={{ fontSize: 12, padding: '1px 6px' }}>
                Left
              </button>
              <button type="button" title="Align horizontal centers" aria-label="Align horizontal centers" onClick={() => alignSelected('center')} style={{ fontSize: 12, padding: '1px 6px' }}>
                Center
              </button>
              <button type="button" title="Align right edges" aria-label="Align right edges" onClick={() => alignSelected('right')} style={{ fontSize: 12, padding: '1px 6px' }}>
                Right
              </button>
              <button type="button" title="Align top edges" aria-label="Align top edges" onClick={() => alignSelected('top')} style={{ fontSize: 12, padding: '1px 6px' }}>
                Top
              </button>
              <button type="button" title="Align vertical middles" aria-label="Align vertical middles" onClick={() => alignSelected('middle')} style={{ fontSize: 12, padding: '1px 6px' }}>
                Middle
              </button>
              <button type="button" title="Align bottom edges" aria-label="Align bottom edges" onClick={() => alignSelected('bottom')} style={{ fontSize: 12, padding: '1px 6px' }}>
                Bottom
              </button>
              <button type="button" title="Align same width" aria-label="Align same width" onClick={() => alignSelected('sameWidth')} style={{ fontSize: 12, padding: '1px 6px' }}>
                Same width
              </button>
              <button type="button" title="Align same height" aria-label="Align same height" onClick={() => alignSelected('sameHeight')} style={{ fontSize: 12, padding: '1px 6px' }}>
                Same height
              </button>
              <button type="button" title="Align distribute horizontally" aria-label="Align distribute horizontally" onClick={() => alignSelected('distribute')} style={{ fontSize: 12, padding: '1px 6px' }}>
                Distribute
              </button>
            </div>
          </div>
        ) : null}
        <div style={{ display: 'flex', alignItems: 'flex-start' }}>
          <div
            aria-label="Ruler corner"
            style={{
              position: 'sticky',
              left: 0,
              zIndex: 10,
              flex: 'none',
              width: 28,
              height: 20,
              background: 'var(--ide-panel, #f8f9fb)',
              borderRight: '1px solid var(--ide-line, #dadce0)',
              borderBottom: '1px solid var(--ide-line, #dadce0)',
            }}
          />
          <div
            aria-label="Horizontal ruler in pixels"
            title="Horizontal ruler (pixels)"
            style={{
              position: 'sticky',
              top: 0,
              zIndex: 10,
              flex: 'none',
              width: rulerW * zoom,
              height: 20,
              background: 'var(--ide-panel, #f8f9fb)',
              borderBottom: '1px solid var(--ide-line, #dadce0)',
            }}
          >
            {topTicks.map((v) => (
              <div
                key={v}
                style={{ position: 'absolute', left: v * zoom, top: 0, bottom: 0, borderLeft: '1px solid #9aa0a6' }}
              >
                {v % RULER_LABEL_STEP === 0 ? (
                  <span style={{ fontSize: 9, color: '#5f6368', paddingLeft: 2 }}>{v}</span>
                ) : null}
              </div>
            ))}
          </div>
        </div>
        <div style={{ display: 'flex', alignItems: 'flex-start' }}>
          <div
            aria-label="Vertical ruler in pixels"
            title="Vertical ruler (pixels)"
            style={{
              position: 'sticky',
              left: 0,
              zIndex: 9,
              flex: 'none',
              width: 28,
              height: rulerH * zoom,
              background: 'var(--ide-panel, #f8f9fb)',
              borderRight: '1px solid var(--ide-line, #dadce0)',
            }}
          >
            {leftTicks.map((v) => (
              <div
                key={v}
                style={{ position: 'absolute', top: v * zoom, left: 0, right: 0, borderTop: '1px solid #9aa0a6' }}
              >
                {v % RULER_LABEL_STEP === 0 ? (
                  <span style={{ fontSize: 9, color: '#5f6368', paddingLeft: 2 }}>{v}</span>
                ) : null}
              </div>
            ))}
          </div>
          <div style={{ flex: 1, minWidth: 0 }}>
        {rep.pages.map((page, pi) => {
          const sheetW = sheetWidth(page);
          return (
            <div
              key={pi}
              className="canvas-sheet"
              style={{ width: sheetW * zoom }}
              data-page={pi}
              onClick={(e) => e.stopPropagation()}
            >
              {page.bands.map((band, bi) => {
                const bandSel = sel !== null && sel.page === pi && sel.band === bi && sel.obj === -1;
                const h = bandHeight(band);
                const bandActive =
                  (drag !== null && drag.sel.page === pi && drag.sel.band === bi) ||
                  (resize !== null && resize.sel.page === pi && resize.sel.band === bi);
                return (
                  <div key={bi} className="canvas-band" style={{ height: h * zoom }} data-band={`${pi}-${bi}`}>
                    <div
                      className={bandSel ? 'canvas-band-head sel' : 'canvas-band-head'}
                      onClick={(e) => {
                        e.stopPropagation();
                        onSelect(bandSel ? null : { page: pi, band: bi, obj: -1 });
                        setMulti([]);
                      }}
                    >
                      {band.kind}: {band.name}
                    </div>
                    {band.objects.map((obj, oi) => {
                      const oSel = sel !== null && sel.page === pi && sel.band === bi && sel.obj === oi;
                      const inMulti = multi.some((s) => s.page === pi && s.band === bi && s.obj === oi);
                      const box = objBox(obj);
                      const isDragged =
                        drag !== null && drag.sel.page === pi && drag.sel.band === bi && drag.sel.obj === oi;
                      const isResized =
                        resize !== null &&
                        resize.sel.page === pi &&
                        resize.sel.band === bi &&
                        resize.sel.obj === oi;
                      const liveW = isResized && resize ? Math.max(8, Math.round(resize.baseW + resize.dw)) : box.w;
                      const liveH = isResized && resize ? Math.max(8, Math.round(resize.baseH + resize.dh)) : box.h;
                      return (
                        <div
                          key={oi}
                          className={oSel ? 'canvas-obj sel' : 'canvas-obj'}
                          data-obj={`${pi}-${bi}-${oi}`}
                          style={{
                            left: box.left * zoom,
                            top: box.top * zoom,
                            width: liveW * zoom,
                            height: liveH * zoom,
                            transform:
                              isDragged && drag ? `translate(${drag.dx * zoom}px, ${drag.dy * zoom}px)` : undefined,
                            ...(inMulti ? { outline: '2px dashed #1a73e8', outlineOffset: 2 } : null),
                          }}
                          onPointerDown={(e) => beginDrag(e, { page: pi, band: bi, obj: oi }, box.left, box.top)}
                        >
                          <span>
                            {obj.kind}: {obj.name}
                            {isResized ? ` ${liveW}x${liveH}` : ''}
                          </span>
                          {oSel ? (
                            <div
                              className="canvas-resize"
                              data-resize={`${pi}-${bi}-${oi}`}
                              aria-label="Resize object"
                              title="Resize object"
                              onPointerDown={(e) =>
                                beginResize(e, { page: pi, band: bi, obj: oi }, box.w, box.h)
                              }
                            />
                          ) : null}
                        </div>
                      );
                    })}
                    {bandActive ? (
                      <>
                        {guides.x.map((gx) => (
                          <div
                            key={`gx-${gx}`}
                            className="canvas-guide"
                            data-orient="v"
                            style={{ left: gx * zoom }}
                          />
                        ))}
                        {guides.y.map((gy) => (
                          <div
                            key={`gy-${gy}`}
                            className="canvas-guide"
                            data-orient="h"
                            style={{ top: gy * zoom }}
                          />
                        ))}
                      </>
                    ) : null}
                  </div>
                );
              })}
            </div>
          );
        })}
          </div>
        </div>
      </div>
    </div>
  );
}
