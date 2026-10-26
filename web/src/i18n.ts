import { useSyncExternalStore } from "react";

// U7 Thai/English chrome dictionary. Every English string stays
// byte-identical to the pre-U7 literal so the default-language e2e
// snapshot keeps passing.
export type Lang = "en" | "th";

const LANG_KEY = "reportnext.lang";
const LANG_EVENT = "reportnext:lang";

function readStored(): Lang {
  try {
    return window.localStorage.getItem(LANG_KEY) === "th" ? "th" : "en";
  } catch {
    return "en";
  }
}

let current: Lang = typeof window === "undefined" ? "en" : readStored();

export function getLang(): Lang {
  return current;
}

export function setLang(lang: Lang): void {
  current = lang;
  try {
    window.localStorage.setItem(LANG_KEY, lang);
  } catch {
    // Blocked storage (private mode): keep the in-memory value.
  }
  window.dispatchEvent(new CustomEvent<Lang>(LANG_EVENT, { detail: lang }));
}

function subscribeLang(onChange: () => void): () => void {
  window.addEventListener(LANG_EVENT, onChange);
  return () => window.removeEventListener(LANG_EVENT, onChange);
}

export function useLang(): Lang {
  return useSyncExternalStore(subscribeLang, () => current, () => "en");
}

export const STRINGS: Record<string, { en: string; th: string }> = {
  // MenuBar chrome.
  "menu.file": { en: "File", th: "ไฟล์" },
  "menu.newReport": { en: "New report…", th: "รายงานใหม่…" },
  "menu.loadSample": { en: "Load sample report", th: "โหลดรายงานตัวอย่าง" },
  "menu.importFrx": { en: "Import .frx…", th: "นำเข้า .frx…" },
  "menu.downloadJson": { en: "Download IR JSON", th: "ดาวน์โหลด IR JSON" },
  "menu.exportPdf": { en: "Export PDF", th: "ส่งออก PDF" },
  "menu.exportDocx": { en: "Export DOCX", th: "ส่งออก DOCX" },
  "menu.about": { en: "About report-next…", th: "เกี่ยวกับ report-next…" },
  "menu.edit": { en: "Edit", th: "แก้ไข" },
  "menu.undo": { en: "Undo", th: "เลิกทำ" },
  "menu.redo": { en: "Redo", th: "ทำซ้ำ" },
  "menu.view": { en: "View", th: "มุมมอง" },
  "menu.tabStart": { en: "Start", th: "เริ่มต้น" },
  "menu.tabDesign": { en: "Design", th: "ออกแบบ" },
  "menu.tabPreview": { en: "Preview", th: "แสดงตัวอย่าง" },
  "menu.tabSource": { en: "IR Source", th: "ซอร์ส IR" },
  "menu.outputConsole": { en: "Output Console", th: "คอนโซลผลลัพธ์" },
  "menu.help": { en: "Help", th: "ช่วยเหลือ" },
  "menu.shortcuts": { en: "Keyboard Shortcuts", th: "แป้นพิมพ์ลัด" },
  "menu.useThai": { en: "Switch to Thai", th: "เปลี่ยนเป็นภาษาไทย" },
  "menu.useEnglish": { en: "Switch to English", th: "เปลี่ยนเป็นภาษาอังกฤษ" },
  "menu.langTitle": { en: "Switch UI language (EN/TH)", th: "สลับภาษาหน้าจอ (EN/TH)" },
  // Toolbar chrome.
  "tb.sample": { en: "Sample", th: "ตัวอย่าง" },
  "tb.import": { en: "Import", th: "นำเข้า" },
  "tb.undo": { en: "Undo", th: "เลิกทำ" },
  "tb.redo": { en: "Redo", th: "ทำซ้ำ" },
  "tb.undoTitle": { en: "Undo (Ctrl+Z)", th: "เลิกทำ (Ctrl+Z)" },
  "tb.redoTitle": { en: "Redo (Ctrl+Y)", th: "ทำซ้ำ (Ctrl+Y)" },
  "tb.render": { en: "Render", th: "เรนเดอร์" },
  "tb.rendering": { en: "Rendering…", th: "กำลังเรนเดอร์…" },
  "tb.renderTitle": {
    en: "Render the current IR to HTML preview",
    th: "เรนเดอร์ IR ปัจจุบันเป็นตัวอย่าง HTML",
  },
  "tb.pdfTitle": { en: "Export the current IR to PDF", th: "ส่งออก IR ปัจจุบันเป็น PDF" },
  "tb.docxTitle": { en: "Export the current IR to DOCX", th: "ส่งออก IR ปัจจุบันเป็น DOCX" },
  "tb.saveTitle": {
    en: "Download the report IR as JSON (Ctrl+S)",
    th: "ดาวน์โหลด IR รายงานเป็น JSON (Ctrl+S)",
  },
  "tb.loadSampleTitle": { en: "Load the bundled sample report", th: "โหลดรายงานตัวอย่างที่แถมมา" },
  "tb.importTitle": { en: "Import a .frx file", th: "นำเข้าไฟล์ .frx" },
  "tb.toolbarLabel": { en: "Report commands", th: "คำสั่งรายงาน" },
  "tb.pdf": { en: "PDF", th: "PDF" },
  "tb.docx": { en: "DOCX", th: "DOCX" },
  "tb.pages": { en: "pages", th: "หน้า" },
  "tb.bands": { en: "bands", th: "แบนด์" },
  "tb.objects": { en: "objects", th: "ออบเจกต์" },
  "tb.modified": { en: "modified", th: "มีการแก้ไข" },
  "tb.saved": { en: "saved", th: "บันทึกแล้ว" },
  "tb.noReport": { en: "No report loaded", th: "ยังไม่มีรายงาน" },
  "tb.project": { en: "report/v1", th: "report/v1" },
  // TabStrip chrome.
  "tabs.start": { en: "Start", th: "เริ่มต้น" },
  "tabs.design": { en: "Design", th: "ออกแบบ" },
  "tabs.preview": { en: "Preview", th: "แสดงตัวอย่าง" },
  "tabs.source": { en: "IR Source", th: "ซอร์ส IR" },
  "tabs.unsaved": { en: "Unsaved changes", th: "มีการเปลี่ยนแปลงที่ยังไม่บันทึก" },
  "tabs.hint": { en: "Sample, import and IR source", th: "ตัวอย่าง นำเข้า และซอร์ส IR" },
  "tabs.pages": { en: "pages", th: "หน้า" },
  "tabs.bands": { en: "bands", th: "แบนด์" },
  "tabs.objects": { en: "objects", th: "ออบเจกต์" },
  "tabs.project": { en: "report/v1", th: "report/v1" },
  // StatusBar chrome (the live message itself stays untouched).
  "status.loaded": { en: "report loaded", th: "โหลดรายงานแล้ว" },
  "status.noReport": { en: "No report", th: "ไม่มีรายงาน" },
  "status.contract": { en: "report/v1", th: "report/v1" },
  "status.contractTitle": { en: "IR contract version", th: "เวอร์ชันสัญญา IR" },
  // StartTab chrome.
  "start.heroTitle": { en: "report-next designer", th: "report-next designer" },
  "start.heroBody": {
    en: "Web report designer for FastReport-compatible reports. Load the sample, import a .frx file, edit bands and objects in the explorer, and render or export from the preview.",
    th: "เว็บดีไซเนอร์รายงานสำหรับรายงานที่เข้ากันได้กับ FastReport โหลดตัวอย่าง นำเข้าไฟล์ .frx แก้ไขแบนด์และออบเจกต์ในเอกซ์พลอเรอร์ แล้วเรนเดอร์หรือส่งออกจากตัวอย่าง",
  },
  "start.open": { en: "Open a report", th: "เปิดรายงาน" },
  "start.loadSample": { en: "Load sample report", th: "โหลดรายงานตัวอย่าง" },
  "start.loadSampleTitle": { en: "Load the bundled {s} sample", th: "โหลดตัวอย่าง {s} ที่แถมมา" },
  "start.importFrx": { en: "Import .frx…", th: "นำเข้า .frx…" },
  "start.samplesPre": { en: "Samples come from ", th: "ตัวอย่างมาจาก " },
  "start.samplesMid": { en: "; imports post .frx bytes to ", th: "; การนำเข้าจะส่งไบต์ .frx ไปที่ " },
  "start.samplesPost": { en: " and open as report IR.", th: " แล้วเปิดเป็น IR รายงาน" },
  "start.templates": { en: "Report templates", th: "เทมเพลตรายงาน" },
  "start.blank": { en: "Blank template", th: "เทมเพลตว่าง" },
  "start.table": { en: "Table template", th: "เทมเพลตตาราง" },
  "start.invoice": { en: "Invoice template", th: "เทมเพลตใบแจ้งหนี้" },
  "start.blankTitle": {
    en: "Create a new report from the blank template",
    th: "สร้างรายงานใหม่จากเทมเพลตว่าง",
  },
  "start.tableTitle": {
    en: "Create a new report from the table template",
    th: "สร้างรายงานใหม่จากเทมเพลตตาราง",
  },
  "start.invoiceTitle": {
    en: "Create a new report from the invoice template",
    th: "สร้างรายงานใหม่จากเทมเพลตใบแจ้งหนี้",
  },
  "start.templatesPre": {
    en: "Templates build a report on the client via ",
    th: "เทมเพลตสร้างรายงานบนฝั่งไคลเอนต์ผ่าน ",
  },
  "start.templatesPost": { en: ", no server round-trip needed.", th: " โดยไม่ต้องเรียกเซิร์ฟเวอร์" },
  "start.recents": { en: "Recent reports", th: "รายงานล่าสุด" },
  "start.noRecents": { en: "No recent reports yet.", th: "ยังไม่มีรายงานล่าสุด" },
  "start.recentTitle": { en: "Reopen recent report {i}: {label}", th: "เปิดรายงานล่าสุด {i} อีกครั้ง: {label}" },
  "start.recentAria": { en: "Reopen recent report {i}: {label}", th: "เปิดรายงานล่าสุด {i} อีกครั้ง: {label}" },
  "start.caps": { en: "Capabilities", th: "ความสามารถ" },
  "start.capFrx": { en: ".frx round-trip", th: ".frx ไป-กลับ" },
  "start.capFrxV": { en: "parse + write", th: "อ่าน + เขียน" },
  "start.capEngine": { en: "Engine preview", th: "ตัวอย่างจากเอนจิน" },
  "start.capEngineV": { en: "bands · groups · totals", th: "แบนด์ · กลุ่ม · ยอดรวม" },
  "start.capExports": { en: "Exports", th: "ส่งออก" },
  "start.capExportsV": { en: "HTML · PDF · DOCX", th: "HTML · PDF · DOCX" },
  "start.capBarcodes": { en: "Barcodes", th: "บาร์โค้ด" },
  "start.capBarcodesV": { en: "QR · DM · PDF417 · Aztec · MaxiCode", th: "QR · DM · PDF417 · Aztec · MaxiCode" },
  "start.shortcutsPre": { en: "Shortcuts: ", th: "ทางลัด: " },
  "start.shortcutsMid": { en: " downloads", th: " ดาวน์โหลด" },
  "start.shortcutsMid2": { en: " the IR JSON, ", th: " IR JSON, " },
  "start.shortcutsPost": { en: " renders the preview.", th: " เรนเดอร์ตัวอย่าง" },
  // AboutDialog chrome.
  "about.title": { en: "About report-next", th: "เกี่ยวกับ report-next" },
  "about.close": { en: "Close", th: "ปิด" },
  "about.name": { en: "report-next designer", th: "report-next designer" },
  "about.body": {
    en: "— a web report designer for FastReport-compatible reports, with the same IDE shell as designer-next.",
    th: "— เว็บดีไซเนอร์รายงานสำหรับรายงานที่เข้ากันได้กับ FastReport ใช้เชลล์ IDE เดียวกับ designer-next",
  },
  "about.backend": { en: "Backend", th: "แบ็กเอนด์" },
  "about.backendV": { en: "Go · report/v1 IR", th: "Go · report/v1 IR" },
  "about.frontend": { en: "Frontend", th: "ฟรอนต์เอนด์" },
  "about.frontendV": { en: "React 19 · Vite 6", th: "React 19 · Vite 6" },
  "about.shell": { en: "Shell", th: "เชลล์" },
  "about.shellV": { en: "designer-next IDE chrome", th: "designer-next IDE chrome" },
  // ConfirmDialog chrome (caller title/message stay untouched).
  "confirm.cancel": { en: "Cancel", th: "ยกเลิก" },
  "confirm.delete": { en: "Delete", th: "ลบ" },
  // NewReportDialog chrome.
  "new.title": { en: "New report", th: "รายงานใหม่" },
  "new.cancel": { en: "Cancel", th: "ยกเลิก" },
  "new.copy": { en: "Copy sample", th: "คัดลอกตัวอย่าง" },
  "new.create": { en: "Create blank", th: "สร้างรายงานว่าง" },
  "new.createAria": { en: "Create blank report", th: "สร้างรายงานว่าง" },
  "new.sample": { en: "Sample", th: "ตัวอย่าง" },
  "new.sampleAria": { en: "Sample to copy", th: "ตัวอย่างที่จะคัดลอก" },
  "new.hint": {
    en: "Blank starts with one empty A4-width band; copy starts from a bundled sample.",
    th: "แบบว่างเริ่มจากแบนด์เปล่าขนาด A4 หนึ่งแบนด์ แบบคัดลอกเริ่มจากตัวอย่างที่แถมมา",
  },
  // BottomDock chrome (log text and issue messages stay untouched).
  "dock.output": { en: "Output", th: "ผลลัพธ์" },
  "dock.issues": { en: "Issues", th: "ปัญหา" },
  "dock.all": { en: "All", th: "ทั้งหมด" },
  "dock.errors": { en: "Errors", th: "ข้อผิดพลาด" },
  "dock.clear": { en: "Clear", th: "ล้าง" },
  "dock.allAria": { en: "Show all logs", th: "แสดงล็อกทั้งหมด" },
  "dock.errorsAria": { en: "Show errors only", th: "แสดงเฉพาะข้อผิดพลาด" },
  "dock.closeAria": { en: "Close panel", th: "ปิดแผง" },
  "dock.emptyLog": {
    en: "No log lines yet — render or export a report.",
    th: "ยังไม่มีล็อก — เรนเดอร์หรือส่งออกรายงานก่อน",
  },
  "dock.emptyIssues": {
    en: "No issues — every band has objects that fit.",
    th: "ไม่มีปัญหา — ทุกแบนด์มีออบเจกต์ที่พอดี",
  },
  // PreviewPane chrome (backend error text stays untouched).
  "preview.title": { en: "Preview", th: "แสดงตัวอย่าง" },
  "preview.render": { en: "Render", th: "เรนเดอร์" },
  "preview.rendering": { en: "Rendering…", th: "กำลังเรนเดอร์…" },
  "preview.exportPdf": { en: "Export PDF", th: "ส่งออก PDF" },
  "preview.exportDocx": { en: "Export DOCX", th: "ส่งออก DOCX" },
  "preview.exportCsv": { en: "Export CSV", th: "ส่งออก CSV" },
  "preview.exportXlsx": { en: "Export XLSX", th: "ส่งออก XLSX" },
  "preview.csvTitle": { en: "Export the current IR to CSV", th: "ส่งออก IR ปัจจุบันเป็น CSV" },
  "preview.csvAria": { en: "Export CSV", th: "ส่งออก CSV" },
  "preview.xlsxTitle": { en: "Export the current IR to XLSX", th: "ส่งออก IR ปัจจุบันเป็น XLSX" },
  "preview.xlsxAria": { en: "Export XLSX", th: "ส่งออก XLSX" },
  "preview.html": { en: "HTML", th: "HTML" },
  "preview.htmlTitle": {
    en: "Download the rendered HTML preview as a file",
    th: "ดาวน์โหลดตัวอย่าง HTML เป็นไฟล์",
  },
  "preview.htmlAria": { en: "Download HTML", th: "ดาวน์โหลด HTML" },
  "preview.print": { en: "Print", th: "พิมพ์" },
  "preview.printTitle": {
    en: "Print the rendered preview",
    th: "พิมพ์ตัวอย่างที่เรนเดอร์แล้ว",
  },
  "preview.printAria": { en: "Print preview", th: "พิมพ์ตัวอย่าง" },
  "preview.failed": { en: "render failed", th: "เรนเดอร์ล้มเหลว" },
  "preview.empty": {
    en: "No report loaded — load the sample or import a .frx file.",
    th: "ยังไม่มีรายงาน — โหลดตัวอย่างหรือนำเข้าไฟล์ .frx",
  },
  "preview.press": {
    en: "Press Render to preview the current IR.",
    th: "กดเรนเดอร์เพื่อดูตัวอย่าง IR ปัจจุบัน",
  },
  "preview.frameTitle": { en: "Report preview", th: "ตัวอย่างรายงาน" },
  // SourceEditor chrome (parser error text stays untouched).
  "src.title": { en: "IR Source", th: "ซอร์ส IR" },
  "src.load": { en: "Load", th: "โหลด" },
  "src.validate": { en: "Validate", th: "ตรวจสอบ" },
  "src.format": { en: "Format", th: "จัดรูปแบบ" },
  "src.formatTitle": {
    en: "Reformat the JSON with 2-space indent",
    th: "จัดรูปแบบ JSON ใหม่ด้วยการเยื้อง 2 ช่อง",
  },
  "src.invalid": { en: "invalid JSON", th: "JSON ไม่ถูกต้อง" },
  "src.valid": { en: "valid JSON", th: "JSON ถูกต้อง" },
  "src.aria": { en: "Report JSON source", th: "ซอร์ส JSON ของรายงาน" },
  // Preview pager chrome.
  "preview.pagerFirst": { en: "First", th: "หน้าแรก" },
  "preview.pagerFirstTitle": { en: "Go to the first page", th: "ไปหน้าแรก" },
  "preview.pagerFirstAria": { en: "Go to the first page", th: "ไปหน้าแรก" },
  "preview.pagerPrev": { en: "Prev", th: "ก่อนหน้า" },
  "preview.pagerPrevTitle": { en: "Go to the previous page", th: "ไปหน้าก่อนหน้า" },
  "preview.pagerPrevAria": { en: "Go to the previous page", th: "ไปหน้าก่อนหน้า" },
  "preview.pagerNext": { en: "Next", th: "หน้าถัดไป" },
  "preview.pagerNextTitle": { en: "Go to the next page", th: "ไปหน้าถัดไป" },
  "preview.pagerNextAria": { en: "Go to the next page", th: "ไปหน้าถัดไป" },
  "preview.pagerLast": { en: "Last", th: "หน้าสุดท้าย" },
  "preview.pagerLastTitle": { en: "Go to the last page", th: "ไปหน้าสุดท้าย" },
  "preview.pagerLastAria": { en: "Go to the last page", th: "ไปหน้าสุดท้าย" },
  "preview.pagerCounter": { en: "Page {page} of {total}", th: "หน้า {page} จาก {total}" },
  // Preview zoom chrome.
  "preview.zoomLabel": { en: "Zoom", th: "ซูม" },
  "preview.zoomPresetTitle": { en: "Set zoom to {z}%", th: "ตั้งค่าซูมเป็น {z}%" },
  "preview.zoomPresetAria": { en: "Set zoom to {z} percent", th: "ตั้งค่าซูมเป็นร้อยละ {z}" },
  // Preview find chrome.
  "preview.findPlaceholder": { en: "Find in preview…", th: "ค้นหาในตัวอย่าง…" },
  "preview.findTitle": { en: "Find text in the preview", th: "ค้นหาข้อความในตัวอย่าง" },
  "preview.findAria": { en: "Find text in preview", th: "ค้นหาข้อความในตัวอย่าง" },
  "preview.findSearch": { en: "Search", th: "ค้นหา" },
  "preview.findSearchTitle": { en: "Search for the text", th: "ค้นหาข้อความ" },
  "preview.findSearchAria": { en: "Search for the text", th: "ค้นหาข้อความ" },
  "preview.findPrev": { en: "Prev", th: "ก่อนหน้า" },
  "preview.findPrevTitle": { en: "Go to the previous match", th: "ไปรายการที่ตรงก่อนหน้า" },
  "preview.findPrevAria": { en: "Go to the previous match", th: "ไปรายการที่ตรงก่อนหน้า" },
  "preview.findNext": { en: "Next", th: "ถัดไป" },
  "preview.findNextTitle": { en: "Go to the next match", th: "ไปรายการที่ตรงถัดไป" },
  "preview.findNextAria": { en: "Go to the next match", th: "ไปรายการที่ตรงถัดไป" },
  "preview.findCount": { en: "{n} matches", th: "พบ {n} รายการ" },
  "preview.findNone": { en: "No matches", th: "ไม่พบรายการที่ตรง" },
  // Preview outline chrome.
  "preview.outlineTitle": { en: "Outline", th: "โครงร่าง" },
  "preview.outlineEmpty": { en: "No outline entries.", th: "ไม่มีรายการโครงร่าง" },
  "preview.outlineJumpTitle": { en: "Go to page {page}: {title}", th: "ไปที่หน้า {page}: {title}" },
  "preview.outlineJumpAria": { en: "Go to page {page}: {title}", th: "ไปที่หน้า {page}: {title}" },
};

// Translate a dictionary key for a language. Unknown keys fall back to
// the key itself so a missing entry never crashes the chrome.
export function tx(lang: Lang, key: string): string {
  const entry = STRINGS[key];
  if (!entry) return key;
  return lang === "th" ? entry.th : entry.en;
}
