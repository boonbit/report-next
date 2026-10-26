# แผนเติม web designer ให้ทัดเทียมต้นฉบับ + e2e

ที่มา: ผลตรวจเทียบ `designer-next/web/src` กับ `report-next/web/src` — shell chrome พอร์ตครบแล้ว แต่เครื่องมือสร้าง/แก้รายงานยังขาด (explorer อ่านอย่างเดียว, inspector มีแค่ name + attrs ดิบ, ไม่มี undo/canvas/console/issues/wizard)

หลักการ: web-only ทั้งหมด (ไม่แตะ Go engine), ทุกการแก้วิ่งผ่าน `onChange(next)` ตัวเดิมใน `App.tsx`, เพิ่มเคส e2e ทุก phase, จบแต่ละ phase ด้วย `SMOKE-OK` บน container ที่ rebuild แล้ว

## P1 — Explorer เขียนได้ (add/delete/duplicate)

- ใหม่ `web/src/ir/mutate.ts`: `addBand(rep, pageIdx)`, `deleteBand`, `addObject(rep, sel, kind)`, `deleteObject`, `duplicateObject` (object ใหม่ default ตาม kind: Text มี Text/left/top/width/height, Barcode มี Symbology/Text)
- `ReportExplorer.tsx`: ปุ่ม +/🗑/⧉ ระดับ band/object (ใช้ Confirm dialog ของ P1 ก่อนลบ), เลือก node ใหม่หลัง add
- ใหม่ `components/dialogs/ConfirmDialog.tsx` (ลอก `Modal.tsx`): ใช้ตอนลบเท่านั้น
- สถานะ unsaved: `baseline` เดิมเทียบ JSON ได้อยู่แล้ว ไม่ต้องเพิ่ม
- ตรวจ: `tsc --noEmit`, `vite build`, e2e เคส `explorer add/delete/duplicate` (add band → จำนวน Band: เพิ่ม, delete → confirm → หาย, duplicate object → ชื่อซ้ำ `* copy`), negative: กดลบแล้ว cancel ต้องยังอยู่

## P2 — Typed inspector + Source format

- `Inspector.tsx`: field เฉพาะทางตาม kind — Text: content/left/top/width/height/font/size/bold/color; Barcode: symbology select + text; ทุก object: border/visible; band: name/height; ค่าเก็บกลับเป็น attrs เดิม (ไม่เปลี่ยน IR schema)
- `SourceEditor.tsx`: ปุ่ม Format (JSON.stringify indent 2 ลง textarea เดิม)
- ตรวจ: e2e เคส `typed inspector` (แก้ width 245.7 → 300 แล้ว render ใหม่ iframe ขยับ + IR Source มีค่าใหม่), เคส `source format` (JSON ยุบ/ขยายแล้วยัง valid)

## P3 — Undo/Redo

- `App.tsx`: history stack (past/future, cap 50) ครอบ `onChange` + `onLoad` ทุกทาง (explorer/inspector/source/import/sample); toolbar ปุ่ม Undo/Redo (icon มีแล้ว) + เมนู Edit (ใหม่: Undo/Redo) + Ctrl+Z/Ctrl+Y; ทำแล้วล้าง future
- ตรวจ: e2e เคส `undo/redo` (แก้ชื่อ band → Undo ชื่อกลับ → Redo ชื่อมา, ปุ่ม disabled ถูกจังหวะ), negative: Undo ตอน stack ว่างต้องไม่พัง

## P4 — Design canvas (ชิ้นใหญ่สุด)

- ใหม่ `components/design/ReportCanvas.tsx`: วาด band/object ตามสัดส่วนจาก attrs (left/top/width/height, ใช้สเกลเดียวกับ preview), คลิกเลือก ↔ explorer/inspector (ใช้ `sel` เดิม), ลากย้าย object (pointer events, snap 1px, อัปเดต attrs ผ่าน `onChange` แบบเดียวกับ inspector)
- แท็บ Design แบ่งซ้าย tree / กลาง canvas / ขวา inspector (Splitter เดิม); ไม่ทำ resize-handle ในรอบนี้ (คงขนาดผ่าน inspector P2)
- ตรวจ: e2e เคส `canvas select/move` (คลิก object บน canvas แล้ว inspector เปลี่ยนตาม, ลากแล้ว left/top ใน IR เปลี่ยน), screenshot `11-canvas.png` ดูด้วยตา

## P5 — Console + Issues + New-report wizard-lite

- `components/console/OutputConsole.tsx`: แผงล่าง (toggle จากเมนู View) เก็บ log render/export/import/sample พร้อมเวลา (แทน `status` บรรทัดเดียว)
- Issues: ปุ่ม Validate ทั้งรีพอร์ต (ใช้ `Validate` ใน `validate.go` ผ่าน `/api/import`? — ไม่เพิ่ม API: ตรวจ client-side เท่าที่ IR มี: band ว่าง, object นอกขอบ, attr ชื่อซ้ำ) แสดงรายการ คลิกแล้ว jump ไป node นั้น
- Wizard-lite (ไม่ลอก AppWizard ทั้งก้อนเพราะเป็นของสร้างแอป): dialog `New report…` (blank A4 / copy จาก 1 ใน 6 samples) — เติมช่องว่าง "สร้างรีพอร์ต" ให้ครบโดยไม่ scope-bloat
- ตรวจ: e2e เคส `console/issues/wizard` (render แล้ว console มีบรรทัดใหม่, สร้าง blank แล้ว explorer มี 1 page ว่าง)

## e2e รวม (ท้ายทุก phase + จบงาน)

- ขยาย `web/e2e/smoke.py` ตาม pattern เดิม (case function + `check`/`shot`, fail ได้ `failure.png`): P1 3 เคส, P2 2 เคส, P3 2 เคส (รวม negative), P4 2 เคส + ภาพ, P5 3 เคส
- คำสั่งปิดงานทุก phase: `tsc --noEmit` → `vite build` → `docker compose up -d --build app` → `python3 web/e2e/smoke.py --url http://192.168.1.19:8080/` ต้อง `SMOKE-OK` + `no page errors`
- รายงานละเอียดลง `web/e2e/REPORT.md` + ภาพใน `web/e2e/shots/` (ดูด้วยตาก่อนปิด phase)

## Non-goals

- ไม่เปลี่ยน Go engine/IR schema/API, ไม่ทำ live DB, ไม่ทำ pixel-diff, ไม่รัน Firefox/WebKit (มี cache แต่ไม่มี CI), ไม่ commit/push (กฎผู้ใช้)
- Canvas ไม่ทำ resize-handle/zoom/align-guides ในรอบนี้, wizard ไม่ลอก AppWizard ฝั่งสร้างแอปมา

## ความเสี่ยง

- P4 ใหญ่สุด: ถ้าลากแล้วไม่เสถียร ให้ตัดเหลือ click-select + ย้ายผ่าน inspector (P2) แล้วจบ phase ด้วย e2e เท่าที่มี
- ชื่อปุ่มซ้ำทำให้ selector กำกวม (เจอมาแล้ว 3 รอบ): ตั้ง `title`/`aria-label` ไม่ซ้ำตั้งแต่ตอนเขียน ไม่มาแก้ทีหลัง
