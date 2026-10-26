# รายงานทดสอบ e2e — report-next designer

- วันที่: 2026-10-26 (รันสุดท้ายหลัง render R1–R6A)
- คำสั่ง: `python3 web/e2e/smoke.py --url http://localhost:8080/` (รันซ้ำ 2 รอบเขียวติด)
- ผล: **SMOKE-OK — ผ่าน 148/148 checks, ล้ม 0, exit code 0**
- สคริปต์: `web/e2e/smoke.py` (Playwright sync API + stdlib, ไม่มี dependency เพิ่ม)

## สภาพแวดล้อม

- Server: container `report-next-app-1` (image build จาก `Dockerfile` ปัจจุบัน), `GET /` 200
- Browser: Chromium headless-shell 1223 (Playwright 1.60.0, Python 3.12), viewport 1440x900
- Gate สุดท้าย: `no page errors (got [])` — ไม่มี pageerror/console.error หลุดมาเลย

## ผลรายเคส (60 เคส / 148 checks)

1. **Shell** — เมนู File/Edit/View/Help, toolbar Sample/Import/Undo/Redo/Render/PDF/DOCX, แท็บ Start/Design/Preview/IR Source, hero, statusbar → `01-start.png`
2. **File menu** — เปิดเห็นเมนู กด Escape ปิด
3. **Sample load** — explorer เจอ Band, status ยืนยันจำนวนหน้า → `02-design.png`
4. **Preview render** — iframe มีเนื้อความ → `03-preview.png`
5. **IR Source** — editor มองเห็น มี `"pages"` (เก็บข้อความ valid ไว้ใช้เคสหลัง)
6. **Inspector** — คลิก Band แล้วเจอช่อง Band name → `04-inspector.png`
7. **Import .frx** — `table.frx` มีจริง status "Imported table.frx" → `05-import.png`
8. **Preview auto-render** — เปิดแท็บ Preview ตรง ๆ ไม่กด Render ก็มีภาพ
9. **Heavy objects** — `Simple Matrix.frx` + `Gauge.frx` import และ render ได้ → `06-objects.png`
10. **Export downloads** — PDF ขึ้นต้น `%PDF` + status, DOCX ขึ้นต้น `PK` + status → `07-export.png`
11. **Invalid JSON** — alert "invalid JSON" ไม่ crash แล้วโหลดค่าถูกต้องกลับ → `08-invalid.png`
12. **F5** — กลับมาแท็บ Preview status ขึ้น Rendered
13. **About** — เปิด/ปิด dialog ได้ → `09-about.png`
14. **Sample picker** — แท็บ Start แสดง samples จาก `/api/samples`, กด `table` แล้ว status "Sample loaded (table)" → `10-samples.png`
15. **Explorer add band** — เพิ่ม band ได้, ลบขึ้น Confirm, Cancel แล้ว band ยังอยู่ → `11-explorer.png`
16. **Explorer add/duplicate/delete object** — +T ได้ `Text1`, duplicate ได้ `Text1 copy`, ลบหายจริง
17. **Explorer delete band** — confirm Delete แล้วจำนวน band กลับเท่าเดิม
18. **Undo/redo** — โหลดใหม่ stack ว่างปุ่ม disabled, rename → Undo กลับ → Redo มาใหม่ + status Undone, เปิดเมนู Edit ได้ → `13-undo.png`
19. **Typed inspector** — Width 245.7 → 300 ผ่าน field, IR มีค่าใหม่, render ใหม่มีเนื้อความ → `12-inspector.png`
20. **Source Format** — JSON ยุบแล้วขยาย indent 2 ถูกต้อง
21. **Canvas click** — คลิก object บน canvas แล้ว inspector เปลี่ยนตาม → `14-canvas.png`
22. **Canvas drag** — ลากแล้ว Left 0→50, Top 0→20 ใน IR (commit ครั้งเดียวต่อครั้งลาก)
23. **Console dock** — เปิดจาก View menu, มีบรรทัด sample, render แล้วมี `render: ok`
24. **Wizard + Issues** — New blank report ได้, Issues (1) คลิก jump
25. **Color picker** — ตั้ง TextFill `#ff0000` ลง IR + render ผ่าน → `16-color.png`
26. **Explorer search** — พิมพ์แล้วแถวไม่ตรงหาย ล้างแล้วกลับมา
27. **Issue badge + error chip** — band ว่างมี badge, ชื่อซ้ำ chip ขึ้นเลขที่แท็บ Design แก้แล้วหาย
28. **Canvas resize** — ลาก handle แล้ว Width 245.7→286 Height 28.35→48 → `17-resize.png`
29. **Console error filter** — import ไฟล์ขยะได้ log error, filter Errors ยังเห็น ซ่อน info (400 ที่ตั้งใจยิงถูกยกเว้นใน gate สุดท้าย)
30. **Readonly rows** — เลือก Text1 เห็น Characters 23, เพิ่ม object แล้ว count ใน band เพิ่ม → `18-readonly.png`
31. **Canvas snap** — ลากชิดขอบซ้ายมีเส้นไกด์ Left เป็น 0 พอดี
32. **Canvas zoom** — Ctrl+wheel เปลี่ยน % (ไม่แตะ IR: ลากที่ zoom แล้ว delta เท่า 1x), reset กลับ 100% → `19-zoom.png`
33. **Inspector sections** — section "Text (4)" กดยุบ (field หาย) กดกางกลับ (field มา) → `20-sections.png`
24. **Wizard + Issues** — New report… สร้าง blank ได้, Issues (1) มี `Band Data1 is empty`, คลิก jump ไป band → `15-dock.png`
35. **Palette add (U1)** — ปุ่ม Add text object เพิ่ม Text2 ลง band แรก → `21-palette.png`
36. **Keyboard edit (U2a)** — ArrowRight ดัน Left 0→1, Delete ลบ + Undo กลับ (เลี่ยง substring Text12 ด้วย aria-label) → `22-keyboard.png`
37. **Expression helper (U4)** — popup แทรก UpperCase skeleton, วงเล็บไม่ครบขึ้น alert ไม่ commit → `23-expr.png`
38. **Zoom presets + rulers (U5)** — ปุ่ม 50%/100% เปลี่ยน % (display-only), ไม้บรรทัด px บน/ซ้ายแสดง
39. **Templates + recents (U6)** — table template โหลด, Start tab reopen จาก recent ได้ → `24-start.png`
40. **HTML download (U8a)** — ปุ่ม Download HTML ได้ไฟล์ขึ้นต้น `<`, status `Exported Page1.html.`
42. **Multi-select align (U2b)** — Shift+คลิก 2 object แล้ว Align left (Left เท่ากันจริงใน IR), Esc ล้างชุด → `25-align.png`
43. **Data panel (U3)** — Inspect CSV inline ได้ตาราง `data`, คลิกคอลัมน์แทรก `[data.name]`, Bind ตั้ง `DataSource` ที่ band → `26-data.png`
44. **Thai toggle (U7)** — Help → Switch to Thai แล้ว toolbar/เมนูเป็นไทย, สลับกลับอังกฤษได้ → `27-thai.png`
45. **CSV/XLSX exports (U8b)** — CSV มี header, XLSX ขึ้นต้น `PK`, status ตามชื่อไฟล์
47. **Preview pager (P1)** — sample `outline` 2 หน้า, Next/Prev/Last/First เปลี่ยนเนื้อหาตาม, counter ถูก → `28-pager.png`
48. **Preview zoom (P2)** — preset 150% ขยาย iframe (display-only), กลับ 100% หน้าคงเดิม
49. **Preview find (P2)** — เจอ "Chapter" นับ matches, คำมั่วขึ้น "No matches"
50. **Preview print (P3)** — popup มีเนื้อรายงาน, ไม่มี page error (dialog กดต่อไม่ได้)
51. **Preview hyperlinks (P4)** — ลิงก์ URL เปิดแท็บ vendor, `Hyperlink.Value="2"` กระโดดไปหน้า 2
52. **Preview outline (P4)** — pane แสดง Cover/Chapter One/Chapter Two, คลิกกระโดดตามหน้า → `29-outline.png`
54. **CanGrow (R1)** — fixture `grow.frx` ข้อความยาวเกินกรอบยืดเป็น 46px (จาก 20px) + ShiftMode ดัน object ข้างล่าง → `30-grow.png`
55. **Bound checkbox (R2)** — `bind.frx` ผูก `Customers.Country` ติ๊กถูก 3 แถว, static ไม่ติ๊ก → `31-bind.png`
56. **1D barcode (R3)** — `barcode1d.frx` Code128/EAN13/UPC-A ขึ้นสัญลักษณ์ (EAN grid 95 โมดูล)
57. **Stdlib expressions (R4)** — `expr.frx` TitleCase/PadLeft/AddDays/DaysInMonth/Switch/Maximum render ถูก
58. **Summary/overlay (R5)** — `summary.frx` Summary อยู่ท้าย + Overlay ปรากฏบนหน้า → `32-summary.png`
59. **Positioned export (R6A)** — XLSX มีข้อความ + `mergeCell`, DOCX มีข้อความ (round-trip ผ่าน Go test)
60. **No page errors** — gate สุดท้าย (ยกเว้น favicon 404 + 400 ตั้งใจของเคส garbage import)

## Screenshots (`web/e2e/shots/`)

| ไฟล์ | เคส | ขนาด |
|------|------|------|
| 01-start.png | shell | 112,412 |
| 02-design.png | sample | 73,557 |
| 03-preview.png | preview | 50,991 |
| 04-inspector.png | inspector | 81,754 |
| 05-import.png | import | 100,288 |
| 06-objects.png | matrix/gauge | 50,787 |
| 07-export.png | download | 72,158 |
| 08-invalid.png | invalid JSON | 52,096 |
| 09-about.png | about | 84,798 |
| 10-samples.png | sample picker | 100,673 |
| 11-explorer.png | add band | 110,252 |
| 12-inspector.png | typed inspector | 51,697 |
| 13-undo.png | undo/redo | 70,625 |
| 14-canvas.png | canvas | 88,471 |
| 15-dock.png | dock/wizard | 65,210 |
| 16-color.png | color picker | 59,548 |
| 17-resize.png | canvas resize | 121,916 |
| 18-readonly.png | readonly rows | 131,843 |
| 19-zoom.png | canvas zoom | 152,535 |
| 20-sections.png | inspector sections | 138,097 |
| 21-palette.png | palette add (U1) | 153,584 |
| 22-keyboard.png | keyboard edit (U2a) | 127,042 |
| 23-expr.png | expression helper (U4) | 152,452 |
| 24-start.png | templates + recents (U6) | 154,841 |
| 25-align.png | multi-select align (U2b) | 152,957 |
| 26-data.png | data panel (U3) | 152,450 |
| 27-thai.png | Thai toggle (U7) | 153,620 |
| 28-pager.png | preview pager (P1) | 101,487 |
| 29-outline.png | outline pane + page 2 (P4) | 100,245 |
| 30-grow.png | CanGrow + ShiftMode (R1) | 102,690 |
| 31-bind.png | bound checkbox (R2) | 100,500 |
| 32-summary.png | summary/overlay (R5) | 103,298 |
| 16-color.png | color picker | 59,548 |
| 17-resize.png | canvas resize | 121,916 |

ดูด้วยตาแล้ว: `14-canvas.png` (tree/canvas/inspector 3 ช่อง, Text1 ถูกเลือก), `15-dock.png` (dock เต็มความกว้างด้านล่าง, Issues (1) ชี้ไป band)

## หมายเหตุ

- Negative checks: selector ผิด → `SMOKE-FAIL` + `failure.png`; ลบแล้ว cancel → ของยังอยู่; Undo ตอน stack ว่าง → ปุ่ม disabled ไม่พัง
- รันซ้ำ: `python3 web/e2e/smoke.py [--url ...] [--headed]`
