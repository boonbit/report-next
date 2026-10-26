// Client-side report templates and recent-reports storage (U6).
// Templates are built locally without a server round-trip; every template
// is a valid report/v1 document with no empty bands and unique object names
// so computeIssues reports zero issues for them.
import type { Report, RptObject } from "../../ir";

export type TemplateKind = "blank" | "table" | "invoice";

export interface RecentEntry {
  label: string;
  when: string;
  ir: Report;
}

const RECENT_KEY = "reportnext.recent.v1";
const MAX_RECENTS = 5;
const MAX_STORED_CHARS = 200000;

const BAND_WIDTH = "718.2";

function text(
  name: string,
  left: string,
  top: string,
  width: string,
  height: string,
  value: string,
): RptObject {
  return {
    kind: "TextObject",
    name,
    attrs: [
      { name: "Left", value: left },
      { name: "Top", value: top },
      { name: "Width", value: width },
      { name: "Height", value: height },
      { name: "Text", value },
    ],
  };
}

function titleBand(title: string, extra: RptObject[] = []): Report["pages"][number]["bands"][number] {
  return {
    kind: "ReportTitleBand",
    name: "ReportTitle1",
    attrs: [
      { name: "Width", value: BAND_WIDTH },
      { name: "Height", value: "80" },
    ],
    objects: [text("Title1", "0", "9.45", BAND_WIDTH, "28.35", title), ...extra],
  };
}

function buildBlank(): Report {
  return {
    version: "report/v1",
    pages: [
      {
        name: "Page1",
        bands: [
          {
            kind: "DataBand",
            name: "Data1",
            attrs: [
              { name: "Width", value: BAND_WIDTH },
              { name: "Height", value: "100" },
            ],
            objects: [text("Text1", "9.45", "9.45", "200", "28.35", "New text")],
          },
        ],
      },
    ],
  };
}

function buildTable(): Report {
  return {
    version: "report/v1",
    pages: [
      {
        name: "Page1",
        bands: [
          titleBand("Sales by product"),
          {
            kind: "DataBand",
            name: "Data1",
            attrs: [
              { name: "Width", value: BAND_WIDTH },
              { name: "Height", value: "100" },
            ],
            objects: [
              text("HeaderProduct", "0", "9.45", "239.4", "28.35", "Product"),
              text("HeaderQty", "239.4", "9.45", "239.4", "28.35", "Qty"),
              text("HeaderPrice", "478.8", "9.45", "239.4", "28.35", "Price"),
              text("TextProduct", "0", "47.25", "239.4", "28.35", "Sample item"),
              text("TextQty", "239.4", "47.25", "239.4", "28.35", "1"),
              text("TextPrice", "478.8", "47.25", "239.4", "28.35", "0.00"),
            ],
          },
        ],
      },
    ],
  };
}

function buildInvoice(): Report {
  return {
    version: "report/v1",
    pages: [
      {
        name: "Page1",
        bands: [
          titleBand("Invoice", [
            text("TextNo", "0", "47.25", "359.1", "28.35", "Invoice no. INV-0001"),
            text("TextDate", "359.1", "47.25", "359.1", "28.35", "Date: 2026-01-01"),
          ]),
          {
            kind: "DataBand",
            name: "Data1",
            attrs: [
              { name: "Width", value: BAND_WIDTH },
              { name: "Height", value: "100" },
            ],
            objects: [
              text("TextBillTo", "0", "9.45", "359.1", "28.35", "Bill to: Sample customer"),
              text("TextTotal", "359.1", "9.45", "359.1", "28.35", "Total: 0.00"),
              text("TextThanks", "0", "47.25", BAND_WIDTH, "28.35", "Thank you for your business."),
            ],
          },
        ],
      },
    ],
  };
}

export function buildTemplate(kind: TemplateKind): Report {
  switch (kind) {
    case "table":
      return buildTable();
    case "invoice":
      return buildInvoice();
    case "blank":
    default:
      return buildBlank();
  }
}

function isEntry(v: unknown): v is RecentEntry {
  if (typeof v !== "object" || v === null) return false;
  const e = v as Record<string, unknown>;
  return (
    typeof e.label === "string" &&
    typeof e.when === "string" &&
    typeof e.ir === "object" &&
    e.ir !== null
  );
}

export function loadRecents(): RecentEntry[] {
  try {
    const raw = localStorage.getItem(RECENT_KEY);
    if (!raw) return [];
    const parsed: unknown = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed.filter(isEntry).slice(0, MAX_RECENTS);
  } catch {
    return [];
  }
}

export function saveRecent(ir: Report, label: string): void {
  try {
    const next: RecentEntry[] = [{ label, when: new Date().toISOString(), ir }, ...loadRecents()];
    const raw = JSON.stringify(next.slice(0, MAX_RECENTS));
    if (raw.length > MAX_STORED_CHARS) return;
    localStorage.setItem(RECENT_KEY, raw);
  } catch {
    // Quota or storage errors are ignored: recents are a convenience only.
  }
}
