# แผนปรับปรุง web designer — ใช้ง่ายกว่าต้นฉบับ (Usability, ไม่ใช่ parity)

ที่มา: parity ครบแล้ว (shell diff-identical, e2e 34 เคส/88 checks `SMOKE-OK`)
เอกสารนี้ไม่ลอกต้นฉบับเพิ่ม แต่เติมสิ่งที่ต้นฉบับไม่มี/ทำไว้ยาก
เพื่อให้มือใหม่เปิดมาแล้วออกแบบรีพอร์ตได้โดยไม่ต้องอ่านคู่มือ

หลักการ: IR schema `report/v1` ไม่เปลี่ยน, ทุกแก้ผ่าน `commit`/`onChange`
เดิม (`App.tsx`), ทุก phase จบด้วย rebuild + `SMOKE-OK` + `no page errors`
คำสั่งปิดงานทุก phase: `tsc --noEmit` → `vite build` →
`docker compose up -d --build app` →
`python3 web/e2e/smoke.py --url http://192.168.1.19:8080/`
แล้วอัปเดต `web/e2e/REPORT.md` + `web/README.md`

## U1 — Object palette (แถบเครื่องมือวาง object)

ปัญหา: เพิ่ม object ได้เฉพาะปุ่ม `+T` ใน explorer มือใหม่หาไม่เจอ
งาน: rail ซ้ายของ canvas มีปุ่มชนิด object (Text, Barcode, Matrix, Gauge…)
คลิก = เพิ่มลง band ที่เลือก, ลาก = วางตำแหน่งบน canvas (reuse
`onMutate`/`commit` เดิม, undo ได้ทันที)
e2e: เคส `palette add` — คลิก Text แล้ว IR มี object เพิ่ม + ภาพ `21-palette.png`
ไม่ทำ: drag ข้าม panel อื่น, custom object

## U2 — คีย์บอร์ด + จัดแนว (แบ่ง 2 ขั้น)

U2a — Delete ลบ selection (ใช้ `ConfirmDialog` เดิม), Ctrl+D duplicate,
ลูกศร nudge ครั้งละ 1 (Shift = 10), ทุกอย่างผ่าน `commit` (undo ได้)
e2e: เคส `keyboard edit` — nudge แล้ว Left เปลี่ยน, Delete แล้วหาย +
Undo กลับมา
U2b — Multi-select (Shift+คลิก, Esc ล้าง) + แถบจัดแนว
(ซ้าย/กลาง/ขวา/บน/กลางแนวตั้ง/เท่ากว้าง/เท่าสูง/กระจายระยะ)
e2e: เคส `align` — เลือก 2 ชิ้น กด align-left แล้ว Left เท่ากัน
ไม่ทำ: rubber-band ลากคลุม (ใช้ Shift+คลิกพอ), rotate

## U3 — Data panel (ช่องว่างฟังก์ชันชิ้นเดียวที่เหลือ)

ปัญหา: engine มี SQL ครบ (SQLite ผ่าน `modernc.org/sqlite` pure-Go,
บวก PG/MSSQL ใน `sql_dialects.go`) แต่ designer ไม่มี UI ตั้งค่าเลย
งาน web: panel `Data` ใน rail ขวา — เพิ่ม source (SQLite path/ไฟล์,
CSV/JSON/XML อัปโหลด, PG/MSSQL connection string), แสดง
tables/columns, ผูก `DataSource` ให้ band, ดับเบิลคลิกคอลัมน์ =
สร้าง Text object `[table.column]`
งาน Go (เล็ก, จุดเดียวที่แตะ backend ในแผนนี้): endpoint
`POST /api/schema` รับ `{kind, dsn|file}` แล้วคืน dictionary
โดย reuse `sqlOpenDB`/`sqlInspectSchema`/`sqlInspectPostgres`/
`sqlInspectMSSQL`/`DataLoadCSV`/`DataLoadJSON`/`DataLoadXML` เดิม
+ `go test` ใหม่ свій SQLite ด้วย `testdata` ที่มีอยู่
e2e: เคส `data panel` — เปิด panel, เพิ่ม SQLite `testdata` (ไฟล์จริง),
เห็น table/column, ผูก band ได้ + ภาพ `22-data.png`
(กรณี PG/MSSQL ไม่มี server ให้ทดสอบแค่ validation message)
ไม่ทำ: query builder, relation editor, preview ข้อมูลดิบทั้งตาราง

## U4 — Expression helper

ปัญหา: expression ใน inspector พิมพ์ดิบ พิมพ์ชื่อ field ผิดแล้วรู้ตอน render
งาน: ปุ่ม `…` ข้าง field Text content เปิด popup: รายชื่อ field
จาก dictionary (ถ้ามี) + ฟังก์ชันจาก `expr.go`
(UpperCase/LowerCase/Trim/Substring/ToString/Iif/Abs/Round)
คลิก = แทรกลงตำแหน่ง cursor; ผิด syntax ขึ้นเตือนก่อน commit
e2e: เคส `expression helper` — แทรก `[field]` แล้ว IR เปลี่ยน
ไม่ทำ: syntax highlight เต็มรูปแบบ, custom function

## U5 — Zoom control + ไม้บรรทัด

ปัญหา: zoom มีแต่ Ctrl+wheel, pan มีแต่ Space+drag — ไม่มีใครค้นพบ
งาน: dropdown บน canvas (50/75/100/125/150% + Fit width) +
ตัวเลข % ใน StatusBar + ไม้บรรทัดบน/ซ้ายหน่วย px +
tooltip บอก shortcut บน canvas; logic zoom เดิม (ไม่แตะ IR)
e2e: เคส `zoom control` — เลือก 50% แล้ว canvas ย่อ, reset 100%
(แทนการหมุน wheel ในเคสเดิมบางส่วน)
ไม่ทำ: minimap, pinch touch

## U6 — Start tab: recent + template

งาน: Recent (เก็บ IR ล่าสุด ≤5 ใน localStorage, เปิดซ้ำได้ทันที) +
template 3 แบบ (Blank / Table / Invoice) สร้างจาก constructor
ฝั่ง client ไม่ต้องรอ server; ปุ่ม `New blank report` เดิมยังอยู่
e2e: เคส `recent+template` — สร้างจาก Table template แล้วแท็บ
Start มี recent ขึ้น + ภาพ `23-start.png`
ไม่ทำ: save ขึ้น server, แชร์ template

## U7 — ปุ่มสลับภาษา ไทย/อังกฤษ

เหตุผล: user ใช้ภาษาไทย, chrome ปัจจุบันอังกฤษล้วน
งาน: dictionary ข้อความ MenuBar/Toolbar/TabStrip/StartTab/Confirm/About
2 ภาษา, toggle ใน Help menu + จำใน localStorage, default อังกฤษ
(ไม่เปลี่ยน snapshot e2e เดิม)
e2e: เคส `th toggle` — สลับแล้วเมนู File เป็น `ไฟล์`, สลับกลับเป็น File
ไม่ทำ: แปล field/inspector ทั้งหมด (chrome ก่อน), RTL

## U8 — Export ที่เหลือ (แบ่ง web / Go)

U8a (web-only): ปุ่ม Download HTML (เอาผล `renderPreview` เซฟเป็นไฟล์,
ชื่อตาม report), แก้ชื่อไฟล์ export ตามชื่อ report
(ปัจจุบัน `report.pdf`/`report.docx` ตายตัว)
U8b (Go, เล็ก): endpoint ระดับรีพอร์ต `POST /api/csv` reuse
`srvExportCSV` เดิม + `POST /api/xlsx` ระดับรีพอร์ต
(ของเดิม `xlsxExportSource` export ได้ทีละ source — ต้องเขียน
`xlsxExportReport` ใหม่ + golden test) แล้วค่อย wire ปุ่มใน web
e2e: U8a เคส `html download` (magic `<html`); U8b เคส `csv/xlsx`
ตามลำดับ (ทำ U8b ก่อน U8a ไม่ได้)
ไม่ทำ: XLSX สไตล์ครบแบบ FastReport desktop, scheduler/smtp

## ลำดับทำ (คุณค่า/ความเสี่ยง)

U1 → U2a → U5 → U4 → U6 → U7 → U8a → U2b → U3 → U8b
(U3/U8b แตะ Go ไว้หลังสุด; แต่ละ phase ส่งของได้เอง)

## สถานะ (2026-10-26) — ครบทั้ง 10 phase

เสร็จแล้วรอบแรก: **U1** (Palette click-to-add), **U2a** (Delete/Ctrl+D/
ลูกศร nudge, ไม่ใช้ Confirm — undo ได้), **U4** (fx popup + bracket
check), **U5** (ปุ่ม preset 50–150% + Fit + rulers; ไม่แตะ IR),
**U6** (templates blank/table/invoice + recents localStorage),
**U8a** (Download HTML + ชื่อไฟล์ตาม Page1)
เสร็จแล้วรอบสอง (4 subagent ขนาน + parent รวมงาน):
**U2b** (Shift+click multi-select ใน canvas + แถบ Align 9 ปุ่ม, commit
เดียว, Esc ล้าง), **U3** (`POST /api/schema` รองรับ sqlite/pg/mssql/
csv/json/xml + DataPanel tab ข้าง Inspector: inspect → คลิกคอลัมน์แทรก
`[table.column]` → Bind ตั้ง `DataSource` ที่ band), **U7** (สลับไทย/
อังกฤษจาก Help menu, default อังกฤษ, จำใน localStorage — status/log
คงอังกฤษ), **U8b** (`POST /api/csv` + `/api/xlsx-report` ระดับรีพอร์ต
+ ปุ่มใน Preview, ไม่แตะ `/api/xlsx ?source=` เดิม)
e2e: **SMOKE-OK 117/117 (46 เคส)** รันซ้ำ 2 รอบ, `tsc` + `vite build` +
`go vet` + `go test ./internal/report/` เขียว, container rebuild แล้ว
บทเรียน e2e รอบนี้:
- hello มี object ชื่อ Text12 ใน PageFooter1 — selector "Text1" ชน
  substring ต้องใช้ aria-label เต็ม
- row click ตอน selected อยู่จะ toggle ออก (selection ข้าม tab ไม่หาย)
- `.side-rail` เป็น flex row — tab header ที่เพิ่มเข้าไปต้องห่อด้วย
  column wrapper ไม่งั้น panel เบียดกันจนคลิกไม่ได้
- palette spawn object ทับตำแหน่งเดิม — e2e ต้องลากออกก่อนคลิกชิ้นล่าง

## Non-goals (เหมือนเดิม)

AppWizard, cell Build/Live, recent projects ของ designer-next,
script editor, dialog form, RDL import UI, commit/push,
จำ state ข้าม report (ยกเว้น recent+ภาษา), rubber-band, rotate,
minimap, touch pinch

## ความเสี่ยง

- U2b/U3 ใหญ่สุด: U2b เปลี่ยน `Sel` เป็น multi — แยก state ใหม่
  (`multi: Sel[]`) อย่าแก้ shape `Sel` เดิมเพราะ canvas/inspector/
  e2e เก่าพึ่งมัน; U3 ขึ้นกับ Go endpoint ใหม่ — เขียน `go test`
  ให้เขียวก่อนค่อยต่อ web
- Selector ซ้ำทำ e2e พัง (บทเรียน 5 รอบ): ทุกปุ่มใหม่ใส่
  `title`/`aria-label` unique ตั้งแต่ตอนเขียน, อ่าน count หลัง
  `wait_for` เสมอ
