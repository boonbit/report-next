# แผน parity รอบ 2 — ปิดช่องว่าง designer-next ที่เหลือ

ที่มา: ผล verify แบบละเอียด (diff shell ตรง 100%, ฟังก์ชันตรง ~80%) ช่องว่างที่เหลือมี 5 ชิ้น เอกสารนี้ปิดทั้งหมดโดยไม่แตะ Go engine/IR/API และไม่ลอกของที่เป็น domain สร้างแอป

หลักการเหมือนเดิม: web-only, ทุกการแก้ผ่าน `onChange`/`commit` ตัวเดิม (`App.tsx`), เพิ่มเคส e2e ทุก phase, จบแต่ละ phase ด้วย `SMOKE-OK` บน container ที่ rebuild แล้ว

## Q1 — Inspector color picker (เล็ก)

- `Inspector.tsx`: เพิ่ม `ColorField` (`<input type="color">` + ช่อง hex) ให้ attr สีที่ renderer รู้จัก: `TextFill.Color`, `Fill.Color`, `Border.Color` (ยืนยันชื่อจาก `objects.go`/`html.go` ก่อนเขียน ห้ามเดา — เจอชื่อไหนใช้ชื่อนั้น)
- แปลงค่า: color input ต้องการ `#rrggbb` ส่วน attr เดิมอาจเป็น `110, 145, 190` — เขียน `parseCssColor`/`toCssColor` แปลงไป-กลับ (rgb tuple ↔ hex), ค่าที่แปลงไม่ได้ให้ fallback เป็น text field เดิม
- ตรวจ: e2e เคส `color picker` (เลือก Text1 → ตั้ง TextFill.Color เป็น `#ff0000` ผ่าน `fill` → IR มีค่า hex → render ผ่าน)

## Q2 — Explorer search + issue badges (กลาง)

- ช่อง search ในหัว explorer: filter substring ชื่อ band/object (case-insensitive), ขณะ filter ให้ expand ทั้งหมดและซ่อน node ที่ไม่ตรง, ล้างแล้วกลับสู่ state เดิม
- badge ราย node: สร้าง index จาก `computeIssues` (`Map<"pi/bi/oi", n>`) รวมระดับ band (issue ของ object ใต้ band นับรวมที่ band), แสดงจุดสีแดง/เหลืองท้ายแถว tree (reuse class `chip bad`/`warn` ของ shell)
- ตรวจ: e2e เคส `explorer search` (พิมพ์ `Text1` แล้ว `Text12` หาย, ล้างแล้วกลับมา) + เคส `issue badge` (blank report มี badge ที่ band ว่าง)

## Q3 — TabStrip error chip + severity (เล็ก)

- `ir/validate.ts`: เพิ่ม `severity: "error" | "warning"` ให้ `Issue` — overflow/ชื่อซ้ำ/attr ว่าง = error, band ว่าง = warning
- `TabStrip.tsx`: chip เตือนบนแท็บ Design แบบต้นฉบับ (`<Icon name="warn">` + จำนวน error, มีเฉพาะตอน > 0) — นับจาก `issues` ที่ `App` มีอยู่แล้ว ไม่ต้องเพิ่ม state
- `BottomDock.tsx`: แถว issue ใช้ class ตาม severity (ไม่เปลี่ยน behavior การ jump)
- ตรวจ: e2e เคส `error chip` (blank report → chip ขึ้นเลข 1 ที่แท็บ Design; เติม text object → chip หาย)

## Q4 — Canvas resize handles (ใหญ่สุดรอบนี้)

- `ReportCanvas.tsx`: handle มุมขวาล่างของ object ที่ถูกเลือก ลากแล้วเปลี่ยน Width/Height (snap 1px, ขั้นต่ำ 8, clamp ไม่ติดลบ, commit ครั้งเดียวตอนปล่อยแบบเดียวกับ drag-move)
- ขณะ resize แสดงขนาดเป็นตัวเลขบน object ที่ลาก (reuse `span` เดิม)
- ไม่ทำ: pan/zoom/align-guides/rotate (คงสเกล 1:1 กับ preview ตามเดิม)
- ตรวจ: e2e เคส `canvas resize` (ลาก handle ของ Text1 แล้ว Width/Height ใน IR เปลี่ยน, render ผ่าน) + ภาพ `16-resize.png`

## Q5 — Console level filter + import-failure log (เล็ก)

- `App.tsx` `log(msg, level = "info")`: render/export/import fail = error นอกนั้น info; `BottomDock` แท็บ Output มีปุ่ม All/Errors (reuse ปุ่ม `btn small`)
- e2e เคส `console error filter` (negative จริง): เลือกไฟล์ขยะผ่าน Import (`.txt` เนื้อความมั่ว) → `/api/import` 400 → log `import: FAILED …` ระดับ error → สลับ filter Errors แล้วบรรทัดยังอยู่ สลับกลับ All ก็อยู่
- ตรวจ: e2e เคสนี้ + `SMOKE-OK` เต็มชุด

## e2e รวม + ปิดงาน

- ขยาย `web/e2e/smoke.py` ตาม pattern เดิม: Q1 1 เคส, Q2 2 เคส, Q3 1 เคส, Q4 1 เคส + ภาพ, Q5 1 เคส (negative จริง)
- คำสั่งปิดงานทุก phase: `tsc --noEmit` → `vite build` → `docker compose up -d --build app` → `python3 web/e2e/smoke.py --url http://192.168.1.19:8080/` ต้อง `SMOKE-OK` + `no page errors`
- อัปเดต `web/e2e/REPORT.md` + `web/README.md` + เติม Status รอบใหม่ใน `docs/fastreport-go-port-plan-th.md` ตอนจบ

## Non-goals (ตั้งใจไม่ทำ — domain สร้างแอป ไม่ใช่สร้างรีพอร์ต)

- AppWizard, cell Build/Live ใน StatusBar, DB tools (Add Table/Column/Relation), recent projects, canvas pan/zoom/rotate, script editor, RDL import UI, commit/push

## ความเสี่ยง

- ชื่อปุ่ม/field ซ้ำทำให้ selector กำกวม (เจอมา 4 รอบ): ตั้ง `title`/`aria-label` ไม่ซ้ำตั้งแต่ตอนเขียน
- Q4 ใหญ่สุดรอบนี้: ถ้าลาก resize ไม่เสถียร ให้ตัดเหลือ handle แสดงขนาดอย่างเดียว + ปรับขนาดผ่าน inspector (P2) แล้วจบ phase ด้วย e2e เท่าที่มี
- แปลงค่าสี rgb tuple ↔ hex: เทสด้วย attr จริงจาก fixture (`Fill.Color="110, 145, 190"`) ไม่ใช่ค่าที่แต่งเอง
