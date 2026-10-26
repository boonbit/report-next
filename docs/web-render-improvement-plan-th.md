# แผนปรับปรุง render — เท่าหรือดีกว่าต้นฉบับ

ที่มา: verify ละเอียด (`verify render`) — render ตรงในเคสมาตรฐาน
แต่ขาด layout ขั้นสูง, binding บางส่วน, 1D barcode, expression
และ export รักษาตำแหน่ง แผนนี้เติมตามลำดับคุ้มค่าโดยคงจุดที่เรา
ดีกว่าไว้ (IR อ่านง่าย, engine pure-Go ทดสอบด้วย golden)

หลักการ: IR schema `report/v1` ไม่เปลี่ยน, เติม engine/exporter
เดิม ไม่รื้อโครง, ทุก phase จบด้วย `go vet` + `go test -count=1
./internal/report/` เขียว + rebuild + `SMOKE-OK` + `no page errors`
คำสั่งปิดงานทุก phase: `go vet ./internal/report/` →
`go test -count=1 ./internal/report/` → `tsc --noEmit` →
`vite build` → `docker compose up -d --build app` →
`python3 web/e2e/smoke.py --url http://localhost:8080/`
แล้วอัปเดต `web/e2e/REPORT.md` + `web/README.md`

## R1 — CanGrow / KeepTogether / ShiftMode (Go, engine)

ปัญหาใหญ่สุด: ข้อความยาวล้นกรอบแล้วถูกตัด ไม่ยืด band ตาม
(`ReportComponentBase.cs:307-337` มี แต่เราอ่าน attr ทิ้ง)
งาน: engine อ่าน `CanGrow`/`CanShrink`/`ShiftMode`/`KeepTogether`
- TextObject `CanGrow=true` → วัดความสูงข้อความด้วย `fontmetrics`
  ที่มีอยู่แล้ว ขยาย band instance นั้น ดัน object ข้างล่างลง
  (ShiftMode) แทนการตัด
- Band `KeepTogether=true` → ถ้า instance ไม่พอในหน้าปัจจุบัน
  ยกทั้งก้อนไปหน้าใหม่ (reuse `engSplitSheets` ตรวจล่วงหน้า)
- `go test` ใหม่: golden วัดความสูง + `data-page` ขยับตามคาด
e2e: fixture `grow.frx` (Text ยาว + KeepTogether) → เนื้อความครบ
ไม่ถูกตัด + ขึ้นหน้าใหม่ทั้งก้อน → ภาพ `30-grow.png`
ไม่ทำ: GrowToBottom, Widow/Orphan ละเอียด, hyphenation

## R2 — Picture / CheckBox data binding (Go, engine)

โครง dict มีแล้ว (`engParams`) เหลือแค่ต่อสาย (`objects.go:184`
ยอมรับเองว่ายังไม่ผูก)
งาน: PictureObject อ่าน `DataColumn` จาก dict → ปล่อย `<img
src="data:…;base64,…">` (HTML) + วาด raster ลง PDF (ขยาย `pdfobjects`
+ XLSX/DOCX embed รูป); CheckBoxObject อ่าน `DataColumn`/`CheckState`
ก่อนตกไป `Checked=` คงที่
- `go test` ใหม่: golden base64 + checkbox true/false จาก dict
e2e: fixture `bind.frx` (รูป + checkbox ผูก dict ผ่าน `?source=` หรือ
CSV inline ถ้า render path รองรับ — ถ้ายัง ให้ e2e ตรวจ HTML มี
`data:image` หลัง render ด้วย dict ตัวอย่าง) → ภาพ `31-bind.png`
ไม่ทำ: SVG/รูปภายนอก URL, cache รูปข้ามรีพอร์ต

## R3 — บาร์โค้ด 1D ยอดนิยม (Go, libs + renderers)

เรามี 6 แบบ ต้นฉบับ ~19 (`FastReport.Base/Barcode/`)
งาน: เพิ่ม Code128 + EAN-13 + UPC-A (+ MSI ถ้าเหลือเวลา) เป็น package
ใหม่ใต้ `internal/` ตาม pattern เดิม (`qr`/`datamatrix`: encoder +
golden + oracle cross-check) แล้ว wire เข้า `objBarcodeHTML` +
`pdfobjects` + XLSX/DOCX (วาดเป็น block/rect ปัจจุบัน)
- `go test` ใหม่: golden + oracle cross-check ทุก symbology
e2e: fixture `barcode1d.frx` render ผ่าน + export PDF มี vector
(ตรวจ `%PDF` + ขนาดไฟล์โตกว่ารุ่นไม่มีบาร์โค้ด) → ไม่ต้องมีภาพใหม่
หรือ reuse `07-export.png`
ไม่ทำ: 2D นอกเหนือที่มี, Pharmacode/Plessey/PostNet (ใช้น้อย)

## R4 — Expression stdlib (Go, expr)

เรามี 8 ฟังก์ชัน ต้นฉบับ 141 (`StdFunctions.cs`)
งาน: เติมเป็นชุดด้วย Go stdlib — string (`Contains/Replace/
Remove/PadLeft/PadRight/Insert/Length/Chr/Asc/TitleCase`),
date (`AddDays/Months/Years/Day/Month/Year/DayOfWeek/
DaysInMonth/DateSerial`), math (`Maximum/Minimum`) + `Switch`
- ทุกฟังก์ชันมี unit test ตรง + golden ผ่าน engine (`[fname(…)]`)
e2e: fixture `expr.frx` ใช้ฟังก์ชันใหม่หลายตัว render ถูก →
ตรวจข้อความใน iframe (ไม่ต้องมีภาพใหม่)
ไม่ทำ: NumToWords ทุกภาษา (ยกเว้น En ถ้าง่าย), VB-style ฟังก์ชัน
แปลก ๆ, custom function ของ user

## R5 — ReportSummary / OverlayBand routing (Go, engine เล็ก)

งาน: `ReportSummaryBand` → render หลัง flow จบ (ไม่ใช่ปนกลางเรื่อง),
`OverlayBand` → ฝังซ้ำทุก prepared page (watermark ง่าย ๆ ได้ฟรี)
- `go test` ใหม่: summary อยู่ท้าย + overlay ปรากฏทุก `data-page`
e2e: fixture เดิม `outline.frx` ต่อท้ายด้วยสอง band นี้ → ตรวจจำนวน
`data-page` + ข้อความ overlay ซ้ำ → ไม่ต้องมีภาพใหม่
ไม่ทำ: Watermark เต็มรูปแบบ (รูปพื้นหลัง/โปร่งแสง), PrintOnOddEven

## R6 — Export รักษาตำแหน่ง (Go, exporters — งานใหญ่สุด แบ่ง 2 ขั้น)

XLSX/DOCX/CSV ปัจจุบันเป็นแค่แถวข้อความ (`reportExportRows`)
งานขั้น A: XLSX/DOCX วาง TextObject ตามตำแหน่ง (col/row ประมาณจาก
Left/Top + merge cell คร่าว ๆ) + Shape/Line เป็น drawing พื้นฐาน
งานขั้น B: embed รูป (ต่อจาก R2) + barcode vector ใน PDF มีแล้วให้คง
- `go test` ใหม่: เปิดไฟล์ผลลัพธ์ด้วย lib เดิมอ่านกลับ ตรวจตำแหน่ง/
  ข้อความ (round-trip แทน golden ไบนารีที่เปราะ)
e2e: export แล้วตรวจขนาด/ข้อความคงอยู่ (ทำได้แล้วบางส่วนในเคสเดิม)
ไม่ทำ: pixel-perfect กับ FastReport desktop, สไตล์ XLSX ครบ,
ฟอนต์ embed เต็มรูป (คงพฤติกรรม subset ปัจจุบัน)

## ลำดับทำ

R1 → R2 → R5 → R3 → R4 → R6A → R6B
(R1 กระทบทุกหน้าเลยก่อน; R6 ไว้หลังสุดเพราะใหญ่และพึ่ง R2)

## สถานะ (2026-10-26) — ครบ R1–R6A (เหลือ R6B embed รูป)

ทำด้วย 5 subagent ขนาน (layout/binding/encoders/expr/positioned-export)
แล้ว parent รวมงาน (wire 128/EAN/UPC ลง HTML+PDF, ต่อ DataColumn/
CheckState เข้า `engEvalBand`, fixtures + e2e ใหม่ 6 เคส)
- **R1** CanGrow (วัด text + ขยับ sibling ด้วย ShiftMode + KeepTogether)
  + **R5** Summary ท้ายเรื่อง + Overlay ทุกหน้า
- **R2** CheckBox/Picture ผูก DataColumn/CheckState (HTML แสดงผลจริง;
  PDF ยังเป็นกล่องเพราะ `internal/pdf` ไม่มี image primitive)
- **R3** Code128/EAN-13/UPC-A (oracle: reportlab + Wikipedia + hand-calc)
  wire ลง HTML (`<svg>`) + PDF (filled rect) แล้ว
- **R4** string/date/math/Switch (~20 ฟังก์ชัน, ~100 subtests)
- **R6A** XLSX/DOCX วางตามตำแหน่ง + merge (legacy flat path คงเดิม)
e2e: **SMOKE-OK 148/148 (60 เคส)** รันซ้ำหลายรอบ, `go vet` + `go test
./internal/...` เขียวทุก package, container rebuild แล้ว (+seed CSV
เข้า image จน `seeded=true`)
งาน parent เพิ่มเติมระหว่างรวมงาน (bug จริงที่เจอ):
- server วัดข้อความเป็น point แต่ browser render เป็น px → กล่องสั้น
  กว่า ~1/3: แก้ `engGrowInstance` คูณ 96/72 + `line-height:115%`
  ใน text ที่ CanGrow + fixture ใช้ `\n` จริง (สองฝั่งตัดบรรทัดตรงกัน)
- unbound DataBand ไม่ปล่อย instance เลย → fixture ใช้ ChildBand
- container ไม่มี seed CSV → COPY เข้า image ใน Dockerfile
บทเรียน e2e: fixture ใหม่ต้องขึ้นต้นชื่อ object ไม่ชนของเดิม;
assert ข้อความใน iframe ใช้ inner_text; xlsx lib ใช้ inline string
(ไม่มี sharedStrings.xml)

## Non-goals

Chart/Map/Sparkline/RichText/ZipCode/Cellular/HtmlObject (object ใหม่
ทั้งก้อน — เป็นแผนอีกฉบับ), script/double-pass/drill-down,
multi-column, RDL/script/dialog, แก้ IR schema, pixel-perfect export

## ความเสี่ยง

- วัดความสูงข้อความต้องตรงกับ browser/PDF — ใช้ `fontmetrics` ตัวเดียว
  กันทั้งสามทาง ห้าม hardcode คนละสูตร
- golden ไบนารีเปราะ — export ใช้ round-trip read-back แทน
- fixture ใหม่ต้องผ่าน `Validate` + ไม่กระทบเคสเก่า (ชื่อ object
  unique ระวัง Text12-style collision ตามบทเรียนเดิม)
- encoder 1D ใหม่ต้องมี oracle cross-check (ยืมวิธีเดิมจากรอบ QR)
