# แผน parity รอบ 4 — Inspector Section พับได้ (จุดสุดท้ายที่ต่าง)

ที่มา: verify รอบหลัง R1–R3 — เหลือจุดเดียวที่เทียบ domain เดียวกันแล้วยังต่าง: ต้นฉบับกรุ๊ป field เป็น `Section` (พับได้ มีตัวเลขนับ field กำกับ มีปุ่ม action) ส่วนของเราเป็นหัวข้อ `insp-head` ธรรมดายุบไม่ได้ เอกสารนี้ปิดจุดนั้นจุดเดียว ไม่แตะ Go engine/IR/API

หลักการเหมือนเดิม: web-only, ทุกการแก้ผ่าน `onChange`/`commit` ตัวเดิม (`App.tsx`), จบด้วย `SMOKE-OK` บน container ที่ rebuild แล้ว

## S1 — Collapsible Section ใน Inspector (เล็ก)

- `Inspector.tsx`: เพิ่ม `Section` (ลอก API ต้นฉบับ `components/inspector/controls.tsx`: `title` + `count?` + `actions?` + `open` default true, reuse class shell เดิม + CSS เพิ่มแค่ลูกศรพับ/กางใน `report.css`)
- จัดกรุ๊ปเดิมให้เข้า section: Band → "General" (name/width/height) + "Attributes" (ตารางดิบ); TextObject → "General" + "Text" (content/font/size/bold) + "Layout" (geometry/border) + "Attributes"; BarcodeObject → "General" + "Barcode" (symbology/text/show) + "Layout" + "Attributes"; object ชนิดอื่น → "General" + "Layout" + "Attributes"
- `count` = จำนวน field ในกรุ๊ป (นับเฉพาะ typed field ไม่นับแถวตารางดิบ), คง state เปิด/ปิดไว้ใน component (remount แล้วกลับค่า default — ยอมรับได้ ไม่ต้อง persist)
- ไม่ทำ: drag-reorder field ข้ามกรุ๊ป, จำ state พับไว้ข้าม report, Section ใน panel อื่น (Source/Explorer ไม่เกี่ยว)

## e2e + ปิดงาน

- ขยาย `web/e2e/smoke.py` 1 เคส: `inspector sections` (เลือก Text1 → เห็น section "Text (4)" → คลิกหัว section แล้วยุบ (field หาย) → คลิกอีกทีกางกลับ (field กลับมา)) + ภาพ `20-sections.png` (เห็นหัว section ทั้งหมด)
- คำสั่งปิดงาน: `tsc --noEmit` → `vite build` → `docker compose up -d --build app` → `python3 web/e2e/smoke.py --url http://192.168.1.19:8080/` ต้อง `SMOKE-OK` + `no page errors`
- อัปเดต `web/e2e/REPORT.md` + `web/README.md` + เติม Status รอบใหม่ใน `docs/fastreport-go-port-plan-th.md` ตอนจบ

## Non-goals (เหมือนเดิม)

- AppWizard, cell Build/Live, DB tools, recent projects, script editor, RDL import UI, commit/push, จำ state พับไว้ข้าม report

## ความเสี่ยง

- ต่ำ: งานย้าย JSX เข้ากรุ๊ปเป็นหลัก ไม่มี logic ใหม่; ถ้าชื่อ section ซ้ำกันหลาย panel ให้เติม `aria-label` ให้ unique ตั้งแต่ตอนเขียน (บทเรียน 5 รอบ: selector ซ้ำทำ e2e พัง)
