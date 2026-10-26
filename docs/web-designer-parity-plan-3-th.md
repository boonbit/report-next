# แผน parity รอบ 3 — ปิดช่องว่าง 3 อย่างสุดท้าย

ที่มา: verify รอบหลัง Q1–Q5 — จุดที่เทียบ domain เดียวกันได้ตรง ~95% เหลือ ReadOnly + snap (เล็ก) และ pan/zoom (ใหญ่) เอกสารนี้ปิดทั้ง 3 อย่างโดยไม่แตะ Go engine/IR/API

หลักการเหมือนเดิม: web-only, ทุกการแก้ผ่าน `onChange`/`commit` ตัวเดิม (`App.tsx`), เพิ่มเคส e2e ทุก phase, จบแต่ละ phase ด้วย `SMOKE-OK` บน container ที่ rebuild แล้ว

## R1 — Inspector ReadOnly rows (เล็ก)

- `Inspector.tsx`: เพิ่ม `ReadOnlyField` (label + ค่าข้อความธรรมดา ไม่ใช่ input) ใช้แสดงค่าที่คำนวณได้ ไม่ใช่ค่าที่แก้ได้:
  - Band: จำนวน object (`objects.length`), ความกว้างรวมของ object ทั้งหมด (ผลรวม Width)
  - TextObject: จำนวนตัวอักษรของ Text
  - BarcodeObject: ชนิดที่ renderer ตีความได้ (ข้อความดิบของ attr `Barcode` + วงเล็บว่าใช้เอนจินไหน เช่น `QR Code (qr)`)
- วางใต้ field แก้ไขได้ ก่อนหัว `ATTRIBUTES` (reuse class `field-row` + `insp-head`, ไม่เพิ่ม CSS)
- ตรวจ: e2e เคส `readonly rows` (เลือก Text1 → เห็นจำนวนตัวอักษรของ "The report goes here..." = 23, เพิ่ม object ใน band แล้วตัวเลขนับเพิ่ม)

## R2 — Canvas snap (เล็ก)

- `ReportCanvas.tsx`: ขณะ drag-move และ resize ให้ snap ค่า Left/Top/Width/Height เป็นจำนวนเต็มอยู่แล้ว (มี `Math.round`) — เพิ่ม snap เสริมแบบต้นฉบับ: แสดงเส้นไกด์ชั่วคราวเมื่อขอบ object ตรงกับขอบ band หรือขอบ object อื่นใน band เดียวกัน (threshold 3px, ดึงค่าเข้าหาเป้า)
- เส้นไกด์: div เส้นประสีฟ้า (`canvas-guide` ใน `report.css`, ลบออกตอนปล่อย) ไม่ commit ค่าอะไรเพิ่ม — ค่าสุดท้ายยังผ่าน `onChange` ครั้งเดียวเหมือนเดิม
- ไม่ทำ: snap ข้าม band, snap ระยะห่างเท่ากัน (spacing), rotate
- ตรวจ: e2e เคส `canvas snap` (ลาก Text1 ไปชิดขอบซ้าย band → Left เป็น 0 พอดี + มีเส้นไกด์โผล่ระหว่างลาก — เช็คผ่านตัวเลข Left ไม่ต้องอ่านพิกเซล)

## R3 — Canvas pan/zoom (ใหญ่สุดรอบนี้ — ทำแบบคุม scope)

- `ReportCanvas.tsx`: zoom ด้วย Ctrl+wheel (0.25x–2x, step 0.25, โชว์ % ที่หัว panel) และ pan ด้วยการลากพื้นที่ว่าง (space-drag) หรือ scroll ธรรมดา; zoom เปลี่ยนแค่สเกลแสดงผล (หาร/คูณ delta ของ pointer ด้วย zoom) **ไม่เปลี่ยนค่า attrs** — IR เท่าเดิมทุกระดับ zoom
- กฎเหล็ก: e2e ทุกเคสที่ลาก (drag/resize) ต้องผ่านที่ zoom 1x เหมือนเดิม + เคสใหม่ zoom 1.5x แล้วลาก ผลลัพธ์ใน IR ต้องเท่ากับลากที่ 1x (พิสูจน์ว่า zoom ไม่เพี้ยนค่า)
- ไม่ทำ: minimap, zoom-to-fit ปุ่ม (คงไว้แค่ Ctrl+wheel + ตัวเลข % + ปุ่ม reset 100%), touch pinch
- ตรวจ: e2e เคส `canvas zoom` (zoom 1.5x → ลาก Text1 เท่าเดิม → Left/Top เท่าเคส 1x, reset กลับ 100%) + ภาพ `18-zoom.png`

## e2e รวม + ปิดงาน

- ขยาย `web/e2e/smoke.py` ตาม pattern เดิม: R1 1 เคส, R2 1 เคส, R3 1 เคส + ภาพ
- คำสั่งปิดงานทุก phase: `tsc --noEmit` → `vite build` → `docker compose up -d --build app` → `python3 web/e2e/smoke.py --url http://192.168.1.19:8080/` ต้อง `SMOKE-OK` + `no page errors`
- อัปเดต `web/e2e/REPORT.md` + `web/README.md` + เติม Status รอบใหม่ใน `docs/fastreport-go-port-plan-th.md` ตอนจบ

## Non-goals (เหมือนเดิม — domain สร้างแอป ไม่ใช่สร้างรีพอร์ต)

- AppWizard, cell Build/Live ใน StatusBar, DB tools, recent projects, script editor, RDL import UI, commit/push
- Minimap, zoom-to-fit, touch pinch, rotate, snap ข้าม band / snap ระยะห่าง

## ความเสี่ยง

- R3 ใหญ่สุดรอบนี้: กฎ zoom-ไม่แตะ-IR ต้องถือให้มั่น — ถ้าลากที่ zoom แล้วค่าเพี้ยน ให้ตัดเหลือ zoom แสดงผลอย่างเดียว (ดึงค่ากลับด้วยสูตรเดิม) แล้วจบ phase ด้วย e2e เท่าที่มี
- ชื่อปุ่ม/field ซ้ำทำให้ selector กำกวม (เจอมา 5 รอบ): ตั้ง `title`/`aria-label` ไม่ซ้ำตั้งแต่ตอนเขียน
- ReadOnly คำนวณจาก attr ดิบ — ถ้า attr ไม่ใช่ตัวเลขให้แสดง `–` แทน NaN ไม่ crash
