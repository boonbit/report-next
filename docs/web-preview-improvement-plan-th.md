# แผนปรับปรุง Preview — เท่าหรือดีกว่าต้นฉบับ ใช้ง่ายกว่าเดิม

ที่มา: verify ละเอียด (`verify preview` รอบก่อน) — preview เราเป็น
"ภาพนิ่งเอกสารยาว" ขาด viewer มาตรฐานเกือบทั้งหมด ขณะที่ต้นฉบับ
(`ToolbarSettings.cs`, `WebReport.cs`, `Hyperlink.cs` ใน FastReport)
มีครบ แผนนี้เติมตามลำดับคุ้มค่า โดยคงจุดที่เราดีกว่าไว้
(IR Source ข้าง Preview, console log, Download HTML)

หลักการ: IR schema `report/v1` ไม่เปลี่ยน, ทุก phase จบด้วย rebuild +
`SMOKE-OK` + `no page errors`
คำสั่งปิดงานทุก phase: `tsc --noEmit` → `vite build` →
`docker compose up -d --build app` →
`python3 web/e2e/smoke.py --url http://localhost:8080/` (+
`go test -count=1 ./internal/report/` เมื่อแตะ Go)
แล้วอัปเดต `web/e2e/REPORT.md` + `web/README.md`

## P1 — Page navigation (Go เล็ก + web)

ต้นฉบับมี First/Prev/Next/Last + เลขหน้า (`CurrentPageIndex`/
`TotalPages` จาก `PreparedPages.Count`); engine เราแบ่งหน้าไว้แล้ว
(`<section class="page" data-page="N">` ใน `engRenderDocumentHTML`)
เหลือแค่ UI
งาน Go: `POST /api/render?page=N` (1-based; ไม่ส่ง = ทุกหน้าเหมือนเดิม)
reuse `engPrepare` แล้ว render เฉพาะหน้านั้น + ส่งยอดหน้ากลับใน
header `X-Total-Pages`
งาน web: แถบ pager ใน `PreviewPane` (First/Prev/Next/Last + "หน้า x
จาก y", ปุ่ม unique title/aria-label + key i18n `preview.*` th/en),
จำเลขหน้าต่อ report (เปลี่ยน report/reset)
e2e: เคส `preview pager` — import ไฟล์หลายหน้า, counter ขึ้น `1/…`,
กด Next เนื้อหาเปลี่ยนเลขเป็น 2, Last แล้ว First กลับ 1
ไม่ทำ: ช่องพิมพ์เลขหน้าโดยตรง (ปุ่มพอ), จำเลขหน้าข้าม report

## P2 — Zoom + ค้นหาใน preview (web-only)

ต้นฉบับมี `ShowZoomButton`/`ShowSearchButton`; ของเรา zoom มีเฉพาะ
canvas ฝั่ง design
งาน web: ปุ่ม zoom 50–200% (สเกล iframe ด้วย CSS display-only,
ไม่แตะ IR) + ช่องค้นหาใช้ `iframe.contentWindow.find()` (srcDoc
same-origin ใช้ได้) โชว์จำนวนที่เจอ + ปุ่ม Next/Prev ข้ามผล
e2e: เคส `preview zoom find` — zoom 150% ความกว้าง iframe เปลี่ยน,
ค้นคำที่มีในรายงานเจอ ≥1, ค้นคำมั่วเจอ 0 (Chromium-only ตาม e2e เดิม)
ไม่ทำ: replace (preview อ่านอย่างเดียว), highlight ถาวรข้ามหน้า

## P3 — พิมพ์ (web-only)

ต้นฉบับมี `PrintInHtml`/`PrintInPdf`; เราไม่มี
งาน web: ปุ่ม Print เปิดหน้าต่างใหม่เขียน HTML ปัจจุบันลงไปแล้วเรียก
`print()` ให้ user เลือกเครื่องพิมพ์เอง (CSS มี `page-break-after`
รองรับอยู่แล้วจาก `engine.go`)
e2e: เคส `preview print` — คลิกแล้วไม่มี page error + popup มีเนื้อ
รายงาน (dialog ของ browser กดต่อไม่ได้ — บอกข้อจำกัดนี้ไว้ตรง ๆ)
ไม่ทำ: ตั้งค่าหน้ากระดาษในแอป, silent print

## P4 — Hyperlink + outline (Go ก่อน แล้ว web)

ต้นฉบับ: `Hyperlink.Value="…"` เป็น attr ติด object (URL / เลขหน้า /
bookmark), `OutlineExpression="…"` สร้างสารบัญ (`Hyperlink.cs:302`,
fixture `Complex (Hyperlinks, Outline, TOC).frx`); IR เราเก็บ attr
พวกนี้ไว้แล้วแต่ engine ไม่อ่าน (`engine.go`/`html.go` ไม่มี `<a>` เลย)
งาน Go: engine อ่าน `Hyperlink.Value` → ปล่อย `<a href target=_blank
rel=noopener>` สำหรับ http(s)/mailto, `data-goto-page` สำหรับเลขหน้า;
อ่าน `OutlineExpression` ต่อ data row → endpoint ใหม่ `POST
/api/outline` คืน `[{title, page}]` (static text + `[field]` ผ่าน
expression เดิม); `go test` ใหม่ golden ทั้งลิงก์และ outline
งาน web: ลิงก์ URL คลิกเปิดแท็บใหม่, ลิงก์เลขหน้ากระโดดผ่าน pager P1,
ปุ่ม Outline เปิด pane ซ้ายรายการสารบัญ คลิกกระโดดหน้าจอ
fixture e2e: ตัด fixture ใหม่ลง `testdata/` (static outline +
hyperlink URL + `Hyperlink.Value="2"`) ไม่ดึงไฟล์ FastReport ตรง ๆ
e2e: เคส `preview links` (popup event ได้ URL) + `preview outline`
(เห็นหัวข้อ, คลิกแล้ว pager ไปหน้านั้น)
ไม่ทำ: drill-down re-query (render ซ้ำพร้อม param), dialog page,
ส่งออก outline เป็น TOC ใน PDF/DOCX

## ลำดับทำ

P1 → P2 → P3 → P4 (P4 ทำ Go ก่อน web ภายใน phase; แต่ละ phase ส่งของได้เอง)

## สถานะ (2026-10-26) — ครบทั้ง 4 phase

ทำด้วย 3 subagent ขนาน (Go ทั้งก้อน + presentational components + ปุ่มพิมพ์)
แล้ว parent รวมงาน (pager/zoom/find/print/outline ใน PreviewPane + App state)
- **P1** `POST /api/render?page=N` + header `X-Total-Pages` + pager
  First/Prev/Next/Last + counter (fixture ใหม่ `testdata/outline.frx` 2 หน้า)
- **P2** zoom 50–200% (display-only) + ช่องค้นหา (`iframe.find`, นับ matches)
- **P3** ปุ่ม Print เปิด popup เขียน HTML แล้วเรียก browser print
- **P4** engine อ่าน `Hyperlink.Value` (URL เปิดแท็บใหม่ / เลขหน้ากระโดด
  ผ่าน pager) + `POST /api/outline` (Cover/Chapter One/Chapter Two) +
  pane สารบัญคลิกกระโดด
e2e: **SMOKE-OK 130/130 (53 เคส)** รันซ้ำ 2 รอบ, `tsc` + `vite build` +
`go vet` + `go test ./internal/report/` เขียว, container rebuild แล้ว
ภาพ `28-pager.png`/`29-outline.png` ดูด้วยตาแล้วตรง
บทเรียน e2e รอบนี้:
- ปุ่มที่มี aria-label จะใช้ชื่อนั้นแทนข้อความในปุ่ม (Search/Print/Outline)
- แถว toolbar ที่เพิ่มเหนือ `.preview-pane` — assertion ต้องใช้ scope กว้าง
- `window.print()` ใน headless ไม่พัง ยืนยันแค่ popup มีเนื้อ
- hyperlink/table-cell นอก TextObject ยังไม่ปล่อย `<a>` (งานต่อถ้าต้องการ)

## Non-goals (เหมือนเดิม)

LivePreview แบบ designer-next (รันแอปทั้งระบบ — คนละ domain),
thumbnail strip, bottom toolbar แยก, export format ใหม่, แก้ IR
จาก preview, RDL/script/dialog

## ความเสี่ยง

- `iframe.find()` ต่าง browser — e2e เรา Chromium-only อยู่แล้ว รับได้
- dialog พิมพ์ทดสอบอัตโนมัติไม่ได้ — e2e ยืนยันแค่ "ไม่พัง + มีเนื้อ"
- `OutlineExpression` มี `[field]` ต้องมี dict — fixture ใช้ static text
  เป็นหลัก มี field ได้เฉพาะเมื่อผูก sample data สำเร็จ
- selector ซ้ำทำ e2e พัง (บทเรียนเดิม): ปุ่ม pager/zoom/find/print/
  outline ใส่ title/aria-label unique ตั้งแต่ตอนเขียน
