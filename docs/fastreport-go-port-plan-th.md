# แผนพอร์ต FastReport → Go + Web Designer

แนวคิดเดียวกับ designer-next: โมเดลรีพอร์ตเป็น IR (JSON, มีเวอร์ชัน, deterministic,
diff-friendly), Go API เสิร์ฟ designer + preview, frontend เป็น React

## Goal

พอร์ต FastReport Open Source (737 ไฟล์ C# ใน `FastReport.Base`,
`FastReport.OpenSource`, `FastReport.Core.Web`, `Extras`) เป็น Go module ใหม่
ชื่อ `report-next` พร้อม web designer ที่ reuse เชลล์ของ designer-next
(`MenuBar`, `Toolbar`, `TabStrip`, Explorer, inspector, console, `StatusBar`,
Design Preview, Live Preview ผ่าน SSE)

## Success Criteria

- อ่าน/เขียน `.frx` ของตัวอย่างใน [โฟลเดอร์ Demos/Reports](/Volumes/APACER/data/tools/FastReport/Demos/Reports) ได้แบบ round-trip ไม่เสียข้อมูล
- render รีพอร์ตตัวอย่างหลัก (ตาราง + group + barcode + matrix) เป็น HTML ตรงกับ
  `HTMLExport` เดิมในระดับโครงสร้าง
- preview แบบหน้า (prepared pages + page cache) และ export ภาพ + PDF พื้นฐานได้
- web designer สร้าง/แก้/บันทึก IR ของรีพอร์ต และ preview ได้โดยไม่ต้อง build
- `go test ./...` ผ่านทั้งโมดูล

## Approach

ตัดสินใจหลัก 5 ข้อ:

| ข้อ | ทางเลือกที่แนะ | เหตุผล / ที่มา |
|---|---|---|
| โมเดลรีพอร์ต | IR ใหม่ (`report/v1` JSON) + ตัวแปลง `.frx` แยกชั้น | ตามรอย [model.go](/Volumes/APACER/data/tools/designer-next/internal/ir/model.go) กับ [validate.go](/Volumes/APACER/data/tools/designer-next/internal/ir/validate.go); designer/export/generator คุยกันผ่าน IR เท่านั้น ไม่แชร์ implementation |
| เครื่องยนต์จัดหน้า | พอร์ต `ReportEngine` ทีละไฟล์ (bands → databands → groups → pages → keep/break) | [โฟลเดอร์ Engine](/Volumes/APACER/data/tools/FastReport/FastReport.Base/Engine) แยกไฟล์ตามเรื่องอยู่แล้ว 19 ไฟล์ พอร์ตทีละเรื่องแล้วล็อกด้วย golden test ได้ |
| สคริปต์ C#/VB | รองรับแค่ subset: expression `[Field]`, parameter, total, function มาตรฐาน | [CodeProvider](/Volumes/APACER/data/tools/FastReport/FastReport.Base/Code/CodeProvider.cs) ผูกคอมไพเลอร์ .NET พอร์ตตรงไม่ได้ งานนี้ใหญ่สุดถ้าจะเอาเต็ม |
| export ภาพ/PDF | พรีวิวเป็น HTML/SVG ก่อน, raster/PDF ประเมินใน Phase 0 spike | `ImageExport` พึ่ง GDI ไม่มีใน Go; [PdfSimple](/Volumes/APACER/data/tools/FastReport/Extras/OpenSource/FastReport.OpenSource.Export.PdfSimple) มีต้นแบบ PDF writer ~30 ไฟล์ให้ลอกโครงได้ |
| designer | ไม่แตะ desktop designer (closed source) สร้าง web designer ใหม่ reuse เชลล์ designer-next | [README ของ FastReport](/Volumes/APACER/data/tools/FastReport/README.md) ระบุชัดว่าไม่เปิดซอร์ส editor; backend ออนไลน์ดีไซเนอร์มีแค่ [WebReportDesigner](/Volumes/APACER/data/tools/FastReport/FastReport.Core.Web/Application/WebReportDesigner.cs) กับ 3 controller ให้ลอก API |

ของที่ reuse จาก designer-next ได้ทันที: `web/src/components/shell/*`
(MenuBar, Toolbar, TabStrip, StatusBar), `web/src/ir/codec.ts`,
`web/src/ir/mutate.ts` (undo/redo), `web/src/components/preview/LivePreview.tsx`
(SSE), `web/src/components/wizard/AppWizard.tsx`, `internal/api/server.go`,
import DDL/live DB (`internal/schema`), i18n (`web/src/i18n.ts`),
driver sqlite/postgres/mssql ใน `go.mod`

## Feature Inventory (ครบ) → Phase

### 1. Bands (13 ชนิด + โครง) → P1/P3

| ต้นฉบับ | Go |
|---|---|
| `BandBase`, `BandCollection`, `HeaderFooterBandBase`, `PageColumns`, `BandColumns` | `report/v1` band model + validate |
| ReportTitle, ReportSummary, PageHeader, PageFooter, ColumnHeader, ColumnFooter | P3: เรนเดอร์ตรง ไม่มี logic |
| DataHeader, DataFooter, Data, GroupHeader, GroupFooter, Child, Overlay | P3: ผูก engine (databands, groups, subreports, reprint) |
| `ReportPage`, `PageBase`, `PageCollection`, `ReportTitleBand`, `ReportSummaryBand`, `ChildBand`, `OverlayBand`, `SubreportObject` | P1 model, P3 engine |
| `Watermark`, `ReportInfo`, `ReportSettings`, `ReportSettings.Core` | P1 model |

### 2. Report objects → P4 (พื้นฐาน) / P5 (ซับซ้อน)

| ต้นฉบับ | Go |
|---|---|
| `TextObject(+Base)`, `CheckBoxObject`, `LineObject`, `ShapeObject`, `PolyLineObject`, `PolygonObject`, `Hyperlink`, `Border`, `Fills`, `TextOutline` | P4 |
| `PictureObject(+Base)`, `HtmlObject`, `CellularTextObject`, `ZipCodeObject`, `RFIDLabel` | P4 (picture) / P5 (ที่เหลือ) |
| Barcode 1D: 128, 39, 93, Codabar, EAN, UPC, 2of5, MSI, Plessey, PostNet, Pharmacode, IntelligentMail, GS1 | P4 (วาดเป็น vector เอง ไม่พึ่งฟอนต์บาร์โค้ด) |
| Barcode 2D: QR, DataMatrix, PDF417, Aztec, MaxiCode | P5 |
| `Table` (TableObject, TableCell, rows/cols, TableHelper, TableResult) | P5 |
| `Matrix` (10 ไฟล์: descriptor, cells, helper, stylesheet) | P5 |
| `CrossView` (cube link, cells, descriptor, helper) | P5 |
| `Gauge` (linear, radial, simple + label/pointer/scale) | P5 |
| หมายเหตุ: ไม่มี Chart object ใน OSS (ค้นทั้ง Base/OpenSource ไม่พบ) ใช้ chart ของ designer-next แทนได้ | P8 (ถ้าต้องการ) |

### 3. Data layer → P2

| ต้นฉบับ | Go |
|---|---|
| `Dictionary`, `TableDataSource`, `ViewDataSource`, `ProcedureDataSource`, `BusinessObjectDataSource`, `VirtualDataSource`, `Relation(Collection)`, `Parameter(Collection)`, `Total(Collection)`, `SystemVariables`, `DataSourceFilter`, `Sort(Collection)` | P2 core |
| CSV/XML/JSON connections (`CsvDataConnection`, `XmlDataConnection`, `JsonConnection`) | P2 core |
| 19 connectors ใน [โฟลเดอร์ Extras Data](/Volumes/APACER/data/tools/FastReport/Extras/Core/FastReport.Data): MsSql, MySql, Postgres, SQLite, Odbc, Oracle, Firebird, MongoDB, Couchbase, RavenDB, Cassandra, ClickHouse, ElasticSearch, Excel, Linter, Ignite, GoogleSheets, Greenplum (+Json) | P2: เฉพาะที่ใช้ `database/sql` ได้ก่อน (MsSql/MySql/Postgres/SQLite/Odbc); NoSQL/REST เป็น P7 |
| `CubeSourceBase`, `SliceCubeSource`, `CubeHelper` | P7 (แค่พื้นฐาน) |
| `DataConnectionBase`, `ConnectionCollection`, `CommandParameter`, builders | P2 core |

### 4. Engine → P3

`ReportEngine` + 17 ไฟล์แยกเรื่อง (`Bands`, `DataBands`, `Groups`, `Pages`,
`Break`, `Keep`, `KeepWithData`, `Outline`, `PageNumbers`, `ProcessAt`,
`Reprint`, `Subreports` + ไฟล์ `.Async` คู่กัน) → พอร์ตทีละเรื่องตามลำดับนี้
`Report.cs` (2896 บรรทัด: Load/Save/Prepare/Design/ReportDone) กระจายเป็น
`load.go` / `prepare.go` / `run.go`, `Report.Core.cs` + `ReportEngine.Core.cs`
+ `ReportEngine.OpenSource.cs` เป็นตัวตัดสินว่า OSS ตัดอะไรออก (ใช้อ้าง scope)

### 5. Scripting + expressions + functions → P6

| ต้นฉบับ | Go |
|---|---|
| `CodeHelper`, `CsCodeHelper`, `VbCodeHelper`, `MsCodeProvider`, `MsAssemblyDescriptor`, `ExpressionDescriptor`, `AssemblyDescriptor` | แทนด้วย evaluator จำกัด (field/param/total/function) ไม่คอมไพล์ C# |
| `StdFunctions` + `NumToWords` 12 ภาษา + `NumToLetters` + `Roman` | P6 (พอร์ตฟังก์ชันเป็นตาราง ไม่เอารันไทม์ .NET) |
| `Format` 10 ชนิด (General/Number/Currency/Date/Time/Percent/Boolean/Custom) | P6 |
| `HighlightCondition`, `ConditionCollection` | P6 |
| `DialogPage.Core.cs` + `ReportEngine.Dialogs.OpenSource.cs` (ฟอร์มกรอกก่อนรัน) | P8 (map เป็นหน้า input ใน web designer) |

### 6. Serialization + preview → P1/P3

| ต้นฉบับ | Go |
|---|---|
| `FRReader`, `FRWriter`, `IFRSerializable`, `Base`, `ComponentBase`, `ObjectCollection`, `ReportComponentBase`, `Variant`, `Units`, `Converter` | P1: `.frx` round-trip (XML) + IR JSON |
| `PreparedPage(s)`, `PageCache`, `SourcePages`, `PreparedPagePostprocessor`, `Bookmarks`, `Outline`, `Preview/Dictionary` | P3: หน้าผลลัพธ์แบบแบ่งหน้า + cache |
| `Style`, `StyleBase`, `StyleSheet`, `StyleCollection`, `MatrixStyleSheet`, `TableStyleCollection` | P1 model |

### 7. Export → P7

| ต้นฉบับ | Go |
|---|---|
| `ExportBase`, `ExportUtils`, `ExportsOptions`, HTML 6 ไฟล์ (draw/layers/styles/templates/utils) | P7: HTML export ก่อน (เทียบ byte กับของเดิมได้) |
| `ImageExport` (BMP/PNG/JPEG/GIF/TIFF/EMF) | P7: PNG/JPEG ก่อน (EMF ตัด, ผูกผล spike P0) |
| [PdfSimple plugin](/Volumes/APACER/data/tools/FastReport/Extras/OpenSource/FastReport.OpenSource.Export.PdfSimple) (PdfCore 12 ไฟล์ + PdfObjects 9 ไฟล์) | P7: ลอกโครง writer แล้วเติมฟอนต์/รูป |
| commercial-only (PDF เต็ม, XLSX, DOCX, PPTX, RTF, ODT, SVG, XML, email) | ไม่ทำ (non-goal) แต่เก็บ `ExportsHelper`/`ReportExporter` interface ไว้ต่อ |

### 8. Import converters → P8

`ImportBase`, `ComponentsFactory` + 5 ตัว (RDL, JasperReports, StimulSoft,
DevExpress, ListAndLabel) → P8 ทำ RDL ก่อน (XML คล้าย `.frx` สุด) ที่เหลือทำตาม
ตัวอย่างไฟล์จริงที่มี

### 9. Web + designer backend → P8

| ต้นฉบับ | Go |
|---|---|
| `WebReport` (+Backend/Exports/Tabs/Html), toolbar/tabs/export-menu/cache (`Memory/Distributed/Legacy`), `WebUtils`, `ExportsHelper` | P8: Go API เสิร์ฟ preview/export แทน ASP.NET |
| `DesignerReportController`, `UtilsController`, `ConnectionsController`, `ReportDesignerService`, `DesignerUtilsService`, `DesignerOptions/Settings`, `FastReportMiddleware`, service-collection extensions | P8: ลอก API surface เป็น REST ตาม [server.go](/Volumes/APACER/data/tools/designer-next/internal/api/server.go) |
| `wwwroot` JS + 12 ไฟล์ localization ของ export settings | P8: ใช้ React แทน |
| `FastReport.Compat` (WinForms shims: TypeConverters, ListBox/ComboBox) | ตัดทิ้ง ไม่ต้องพอร์ต |
| 28 ไฟล์ [Localization](/Volumes/APACER/data/tools/FastReport/Localization) (`.frl` รวม `Thai.frl`) | P8: ย้ายเป็น message catalog แบบ `web/src/i18n.ts` |
| `ReportBuilder` (fluent API) + unit test | P7: Go builder API (`report.New().Data(...).Column(...)`) |
| `FastReport.Plugins.WebP`, `Transport` plugin | P7/P8 (ถ้าต้องใช้) |
| `Tools/FastReport.Tests.OpenSource` (BaseTests, TextObject, JsonParser) | แปลงเป็น Go tests ใน P1–P7 |
| Demos (console, MVC, SPA React/Angular, `_Shared`) | แทนด้วย `examples/` + Live Preview (มี `northwind.sql` อยู่แล้ว) |

## Steps

- P0 spike: `.frx` 1 ไฟล์ round-trip, render 1 band ลง HTML, ประเมิน raster/PDF (เลือกทางก่อนลงมือ) → ผล: round-trip + HTML ผ่าน (`internal/report`, golden `testdata/hello.frx`); stdlib `image` วาดตัวอักษรไม่ได้ ตัดสิน raster/PDF ตอน P7
- P1 model: IR `report/v1` + validate + `.frx` read/write + styles/settings/watermark
- P2 data: dictionary, datasource 6 ชนิด, relation/param/total/sysvar, CSV/XML/JSON, SQL 4 ฐาน reuse driver เดิม
- P3 engine: pagination → databands → groups → header/footer → keep/break → outline/bookmarks → prepared pages + cache
- P4 objects พื้นฐาน: text/picture/line/shape/checkbox/barcode-1D
- P5 objects ซับซ้อน: table/matrix/crosstab/gauge/barcode-2D/html/zip/rfid/subreport
- P6 expressions: evaluator + StdFunctions/NumToWords/Formats/highlight
- P7 output: HTML export, image export, PDF-simple, Go builder API, tests แปลงจากของเดิม
- P8 designer: REST API (ลอก controller), web designer reuse shell, wizard, `.frx`/RDL import, dialog-page เป็น input form, l10n, examples

## Validation Plan

- ทุก phase: `go test ./...` + golden test ด้วย `.frx` จริงจาก Demos (เริ่มจาก
  `Sales Statistics.frx`, `Barcode.frx`, `Advanced Matrix.frx`)
- P1: round-trip `.frx` → IR → `.frx` แล้ว `diff` ต้องว่าง (ยกเว้น whitespace)
- P3/P7: HTML ที่ render ต้องตรงโครงสร้าง `HTMLExport` เดิม (normalize แล้วเทียบ)
- P8: เปิด designer (`go run ./cmd/designer serve`) สร้างรีพอร์ตจาก wizard,
  preview ผ่านเบราว์เซอร์จริง, บันทึก IR แล้ว generate ซ้ำได้
- จุดเสี่ยงสูงสุดที่ต้องตรวจก่อน: P0 spike (ถ้า raster/PDF ไม่มีทางที่รับได้
  ต้องปรับ scope P7 เป็น HTML/SVG-only)

## Risks / Open Questions

- วัดข้อความ/ฟอนต์ให้ตรง GDI ทำไม่ได้ 100% ต้องยอมให้คลาดเคลื่อนระดับพิกเซล (สมมติฐาน)
- สคริปต์ C# เต็มรูปแบบไม่มีทางพอร์ต รีพอร์ตที่ใช้ event code ซับซ้อนต้องแก้ใหม่ (non-goal)
- EMF/GDI/WinForms dialogs ตัดทั้งหมด ไม่มีผลกับ web
- NoSQL/REST connectors มาทีหลัง ไม่ขวาง core
- Open questions: ไม่มี (scope ตัดจากหลักฐาน `*.OpenSource.cs` และ README ที่ระบุว่า editor ไม่เปิดซอร์ส)

## Non-goals

Desktop designer (closed source), คอมไพล์สคริปต์ C#/VB เต็มรูป, export เชิงพาณิชย์
(PDF ฝังฟอนต์/เข้ารหัส/เซ็น, XLSX/DOCX เต็ม), EMF, WinForms Compat, OLAP เต็มรูป

## Status (2026-10-26) — ทำแล้ว

- P0/P1: IR `report/v1` + `.frx` round-trip + validate + JSON + render band เดียว
- P2: Dictionary/Relation/Parameter/Total + loader CSV/JSON/XML (SQL live เลื่อน — ไม่มีเน็ตดึง driver)
- P4: Line/Shape/CheckBox/Picture-placeholder/Code39-SVG + fallback 2D
- P6: expression `[x]`/`[[x]]` + ฟังก์ชัน + Format + Go builder API
- P3: engine ผูก DataBand/Group/Total + แบ่งหน้า + `TestEngPrepareGroupsTotalsPagination`
- P5-part: TableObject เต็ม (nested cell render), Subreport placeholder
- P7-part: `srvExportCSV` + HTML ทั้งเอกสาร; PDF เต็มยังไม่ทำ
- P8-part: `cmd/report-designer` + REST (`/api/render|import|sample`) + UI static + `TestSrvRenderUsesEngine` พิสูจน์ render ผ่าน engine
- Library ใหม่ (stdlib-only): `internal/fontmetrics` (ตาราง base-14 + วัดความกว้าง, layout-grade) + `internal/pdf` (writer PDF 1.4 + xref ตรวจด้วยเทส) + `pdfExportReport` (text ASCII เท่านั้น, ข้ามไทยแล้วนับ) + `POST /api/pdf` + ปุ่ม Export PDF ใน designer
- เลื่อน: Matrix/CrossView/Gauge เต็มรูป, barcode 2D, PDF writer, SQL live, React designer เต็ม (ใช้ static UI + IR JSON ไปก่อน)

## Status (2026-10-26, รอบสอง) — ปิดของค้าง

- `internal/qr`: encoder byte-mode EC L/M/Q/H + mask ครบ (เทสโครงสร้าง 10 ชุด; ไม่มี oracle ภายนอกแบบ offline) ต่อ `Barcode="QR Code"` เข้า render แล้ว ส่วน 2D อื่นยัง fallback
- Matrix/Gauge: render ตาราง matrix + gauge SVG (linear/radial/simple); pivot เต็มรูปแบบเลื่อน
- SQLite live: `sqlOpenDB`/`sqlLoadTable`/`sqlInspectSchema` ผ่าน `modernc.org/sqlite` (pin offline จาก module cache); Postgres/MSSQL ไม่มี driver ใน cache เลื่อน
- ไทยใน PDF: `internal/pdffont` ฝัง subset ฟอนต์ TTF ของผู้ใช้ (`REPORT_FONT_TTF`), `TestFntPDFExportThai` + `TestFntSystemFont` ผ่านด้วย Ayuthaya.ttf; ไม่มีฟอนต์แจกใน repo (ลิขสิทธิ์)
- คงเหลือจริง: pivot เต็ม, barcode 2D ที่ไม่ใช่ QR, export เต็ม (XLSX/DOCX), React designer เต็ม, Postgres/MSSQL live

## Status (2026-10-26, รอบสี่) — DOCX + pivot เต็ม

- `internal/docx` (stdlib ล้วน: paragraphs + tables + styles) + `docxExportReport` (Text/Table, UTF-8 ผ่านหมดรวมไทย) + `POST /api/docx`
- Pivot เต็ม: `vizPivotGrid` (sum/count + row/col/grand totals) → สังเคราะห์เป็น TableObject ใน engine (`engPivotObject` ครอบทั้ง flow bands และ title/header/footer ที่ render ตรง) + เทส hand-computed (Alice 100+150=250, Bob 200+75=275, grand 525)
- DataMatrix/PDF417/Aztec/MaxiCode: ไม่ทำ — ไม่มี oracle แบบ offline และตาราง placement/codeword ต้องลอกจากสเปกด้วยมือ (พิมพ์ผิด = บาร์โค้ดสแกนไม่ติดโดยไม่รู้ตัว) แย่กว่า fallback ที่มีอยู่
- React designer เต็ม: ติด — npm cache ไม่มี react/react-dom tarball (ตรวจด้วย `npm cache ls` แล้ว) ต้องใช้เน็ต `npm install` หรือยก `node_modules` ของ designer-next มาทั้งก้อน (งาน port เชลล์ชิ้นใหญ่ชิ้นต่อไป)
- Postgres/MSSQL live: ติด — ไม่มี driver ใน module cache (ตรวจแล้วทั้ง extracted + download cache)

## Status (2026-10-26, รอบสาม) — XLSX export

- `internal/xlsx` (stdlib ล้วน: archive/zip + encoding/xml, inline strings, numeric cells): เทสแกะ zip ตรวจ XML จริง
- `xlsxExportSource` + `POST /api/xlsx?source=Name` (+ เทส route, content-type ถูกต้อง)
- ยืนยันบล็อก: ไม่มี pgx v4+/v5 และไม่มี microsoft driver ใน module cache (ตรวจแล้ว) → Postgres/MSSQL live ทำไม่ได้แบบ offline

## Status (2026-10-26, รอบห้า) — React designer เต็ม

- `web/` React 19 + Vite 6 (Explorer/Inspector/Preview/Source + MenuBar/TabStrip/StatusBar) ใช้ `node_modules` ที่ copy จาก designer-next (78M) — `vite build` ผ่านได้ offline หลังแก้ binary 2 ตัว: `@rollup/rollup-darwin-arm64@4.62.2` (rollup 4.65 โหลดได้) และ `@esbuild/darwin-arm64@0.25.10` + ดาวน์เกรด esbuild JS เป็น 0.25.10 (host/binary ต้องตรงกันเป๊ะ) — วิธีทำอยู่ใน `web/README.md`
- `vite build` → `web/dist` (38 modules, JS 231KB) + `TestSrvDistBundle` (serve dist + ตรวจ assets ref)
- Go server เสิร์ฟ `web/dist` เมื่อมี (fallback เป็น static UI เดิม)
- ตรวจไม่ผ่าน/ข้าม: เปิดเบราว์เซอร์จริงไม่ได้ (ไม่มี chromium), เปิด socket จริงใน sandbox ไม่ได้ (`bind: operation not permitted`) — เทส TCP ถูกถอด เหลือ handler-level; user เปิดเองที่ http://localhost:8080
- คงเหลือจริง: pivot เต็ม (เสร็จรอบนี้), 2D ที่ไม่ใช่ QR (บล็อก oracle), export XLSX/DOCX (เสร็จ), Postgres/MSSQL live (บล็อก driver)

## Status (2026-10-26, รอบหก) — SQL dialects + QR oracle + DataMatrix

- PG/MSSQL: เปิดเน็ตผ่าน proxy ได้ (`GOMODCACHE=/tmp/gomodcache`) → เพิ่ม `pgx/v5`, `go-mssqldb` + mock tests (`sql_dialects.go`, `go-sqlmock`); live server ยังไม่มีให้เทส
- QR oracle: defect EC byte-order เจอด้วย `gozxing` ชั่วคราว (low-first vs high-first) แก้แล้ว ตรวจ matrix ตรง ZXing เท่า byte + ล็อก golden ถาวร (`TestQrGoldenMatrix`); ลบ dep ชั่วคราวแล้ว (`go mod tidy`)
- DataMatrix ECC200 (`internal/datamatrix`, square-only, ASCII encodation): พอร์ตอัลกอริทึม ZXing ตรง (symbol table, GF 0x12D, factor polynomials, Utah+corners, finder) — EC byte-order บั๊กแบบเดียวกับ QR (reverse output) เจอด้วย oracle ชั่วคราว แก้แล้ว ตรวจ factor ครบ 16 แถว + matrix ตรง ZXing สำหรับเคส non-C40 (`AB` 10×10 ล็อก golden ถาวรใน `TestDmGoldenAB`)
- ต่อ render: `Barcode="Datamatrix"` + `Barcode.SymbolSize` (`Size24x24` → edge 24, ผิดรูป/เกิน → labeled fallback เดิม) ผ่าน `objBarcodeHTML`; เทส render + fallback (PDF417 ยัง fallback)
- ยอมรับโดยบันทึก: ZXing ใช้ C40 บางเคสทำให้ขนาด symbol ต่างกัน (decode ผ่านทั้งคู่) — ของเรา ASCII-only อย่างเดียว
- สถานะตรวจ: `gofmt` สะอาด, `go vet` ผ่าน, `go test -count=1 ./...` เขียว 8 แพ็กเกจ; temp oracle (`oracle_tmp/`, `oracle_dm/`, `*_temp_test.go`) ลบหมดแล้ว
- คงเหลือจริง: PDF417/Aztec/MaxiCode (ยัง fallback, ไม่มี oracle), Postgres/MSSQL live (มี driver แล้ว, ขาด server จริง), browser smoke (user เปิดเอง)

## Status (2026-10-26, รอบเจ็ด) — PDF export วาดบาร์โค้ด

- `pdfBarcode` ใน `internal/report/pdfexport.go`: วาด `BarcodeObject` เป็นสี่เหลี่ยม vector ลง PDF ตรงกับ HTML preview — QR/DataMatrix ใช้ encoder ที่ผ่าน oracle แล้ว, type อื่นใช้แถบ Code39, 2D ที่ไม่รองรับ (PDF417/Aztec/MaxiCode) ไม่วาด; caption ASCII วาดใต้สัญลักษณ์ตาม `ShowText`, caption non-ASCII นับเป็น skipped อย่างเดียวกับข้อความ
- แยก helper `objQrLevel` ใช้ร่วมกันระหว่าง HTML กับ PDF (dispatch เดิมไม่เปลี่ยน)
- เทสใหม่ 5 ชุด (`pdfbarcode_test.go`): นับ `re f` ตรงจำนวน dark module ของ encoder จริง (DM `"AB"`, QR `http://fast-report.com`), นับแท่ง Code39 ตรง pattern, fallback ไม่วาด, caption ไทยนับ skipped=1 แต่สัญลักษณ์ยังวาด
- สถานะตรวจ: `gofmt` สะอาด, `go vet` ผ่าน, `go test -count=1 ./...` เขียว 8 แพ็กเกจ (งาน QR/DataMatrix เดิมไม่แตะ)
- คงเหลือจริง: PDF417/Aztec/MaxiCode (ยัง fallback ทั้ง HTML/PDF — encoder ต้องใช้ตาราง/สเปกภายนอก ไม่มี oracle), Postgres/MSSQL live (มี driver + mock แล้ว, ขาด server จริง), ตาราง/รูปทรงใน PDF (ยังไม่วาด), browser smoke (user เปิดเอง)

## Status (2026-10-26, รอบแปด) — PDF export วาดตาราง + รูปทรง

- ขยาย `internal/pdf`: path ops ใหม่ (`FillRectRGB`, `StrokeRect`, `StrokeLine`, `Poly`, `Ellipse` แบบ bezier 4 เส้น) + เทส serialize (`pdfdraw_test.go`); rect เก่าเปลี่ยนเป็น `rg` (ไม่มีเทสใดผูกกับ op เดิม, `re f` ยังเหมือนเดิม)
- ใหม่ `internal/report/pdfobjects.go`: `pdfVectorObject` วาด `TableObject` (fill/outline/ข้อความ + HorzAlign/VAlign + ColSpan/RowSpan + nested Text/CheckBox, คอลัมน์ไร้ความกว้างแชร์ส่วนที่เหลือ, แถวไร้ความสูงแชร์ส่วนที่เหลือ) + `LineObject` (นอน/ตั้ง/ทแยง) + `ShapeObject` (Rectangle/RoundRectangle/Ellipse/Diamond/Triangle) + `CheckBoxObject` (Cross/Plus/Fill/tick) + `PictureObject` (กรอบ + label ASCII); ต่อทั้ง legacy และ embedded path; ข้อความ non-ASCII นับ skipped อย่างเดียวกับ TextObject
- ประเด็น fixture จริงที่เจอ: `Table1/Row1` ใน `table.frx` มี 6 logical columns บน 4 declared columns — PDF วาด overflow ต่อด้วยความกว้างเฉลี่ยเหมือน HTML; `Table2/Table3` อยู่ใน `ChildBand` ซึ่งทั้ง HTML และ PDF ถือเป็น opaque labeled box (เทสล็อกพฤติกรรมนี้ไว้)
- ประมาณที่บันทึก: มุม RoundRectangle ตัดเป็นเหลี่ยม, เส้นขอบตารางวาดเต็มกรอบเมื่อ `Border.Lines` ตั้งไว้ (ไม่แยกด้าน), baseline ข้อความใน cell เทียบเท่า TextObject
- สถานะตรวจ: `gofmt` สะอาด, `go vet` ผ่าน, `go test -count=1 ./...` เขียว 8 แพ็กเกจ (งาน QR/DataMatrix/PDF-barcode เดิมไม่แตะ)
- คงเหลือจริง: PDF417/Aztec/MaxiCode (ยัง fallback ทั้ง HTML/PDF), Postgres/MSSQL live (มี driver + mock แล้ว ขาด server จริง), gauge/matrix/subreport ใน PDF, browser smoke (user เปิดเอง)

## Status (2026-10-26, รอบเก้า) — PDF export วาด gauge + matrix + subreport

- ใหม่ `internal/report/pdfviz.go`: `pdfGaugeRect` วาด bar gauge (track + fill ตาม `vizGaugeFraction` + 5 ticks + label กลาง) และ radial gauge (dial arc หนา 24 chords + เข็มที่มุม pi*(1-frac) + hub + label) ตาม geometry `vizGaugeBarSVG`/`vizGaugeRadialSVG`; `pdfMatrix` วาด template grid ผ่าน `pdfGrid` ร่วมกับ TableObject; `pdfSubreport` วาดกรอบเทา + label ASCII — ต่อทั้ง legacy และ embedded path; label non-ASCII นับ skipped อย่างเดียวกับข้อความ
- ปรับ `internal/report/pdfobjects.go`: แยก `pdfGrid` (Table + Matrix ใช้ร่วมกัน), แยก `pdfShapeRect`/`pdfPictureRect`, cell รองรับ nested Shape/Picture/Gauge ที่ offset สัมพัทธ์
- ประเด็น fixture จริง: gauge ใน `Gauge.frx` ไม่มี Width/Height (ใช้ Pointer dims) — HTML ก็วาด 0-size เท่ากัน จึงคง parity (ไม่วาด) + เทสล็อกไว้; `Simple Matrix.frx`/`Subreport.frx` vendor เข้า `testdata/` แล้ววาดเต็ม (Employee/Total, WhiteSmoke fill, placeholder Subreport1)
- เทสใหม่ 10 ชุด (`pdfviz_test.go`): bar/radial/non-ASCII label/fallback/zero-size, matrix synthetic + no-grid + fixture, subreport synthetic + fixture
- สถานะตรวจ: `gofmt` สะอาด, `go vet` ผ่าน, `go test -count=1 ./...` เขียว 8 แพ็กเกจ (งาน QR/DataMatrix/PDF-barcode/table-shapes เดิมไม่แตะ)
- คงเหลือจริง: PDF417/Aztec/MaxiCode (ยัง fallback ทั้ง HTML/PDF), Postgres/MSSQL live (มี driver + mock แล้ว ขาด server จริง), browser smoke (user เปิดเอง)

## Status (2026-10-26, รอบสิบ) — live-source verification harness

- ใหม่ `internal/report/sql_live_test.go`: เทส live แบบ env-gated (`REPORT_PG_DSN`, `REPORT_MSSQL_DSN`) — สร้างตารางชั่วคราว, insert 2 แถวรวม NULL, round-trip ผ่าน `sqlOpenPostgres`/`sqlOpenMSSQL` + `sqlLoadTableWith` (quote ถูก dialect), ตรวจ type mapping (`int`→Int32, `varchar/nvarchar`→String, `numeric/decimal`→Decimal, `bool/bit`→Boolean, `datetime2`→DateTime), ตรวจ `sqlInspectPostgres`/`sqlInspectMSSQL` เจอตาราง+ครบ column, DROP ทิ้ง; ไม่มี env จะ skip
- ตรวจ mapping ล่วงหน้ากับ driver จริงใน module cache: pgx ส่ง INT4→int64, NUMERIC→string, BOOL→bool, TIMESTAMPTZ→time.Time (`DecodeDatabaseSQLValue`) — assertion ในเทสตรงกับพฤติกรรม driver แล้ว
- สภาพแวดล้อมที่นี่: Docker daemon ต่อไม่ได้ (permission denied ที่ socket), MySQL ไม่รัน — จึงรัน live จริงไม่ได้ ได้แค่ harness + mock; วิธีรันจริง: ตั้ง DSN แล้ว `go test -run TestSQLLive ./internal/report/`
- สถานะตรวจ: `gofmt` สะอาด, `go vet` ผ่าน, `go test -count=1 ./...` เขียว 8 แพ็กเกจ (live tests skip อย่างถูกต้อง)
- คงเหลือจริง: PDF417/Aztec/MaxiCode (ยัง fallback ทั้ง HTML/PDF — encoder ต้องใช้ตารางสเปกภายนอก ไม่มี oracle), live DB จริง (รอ server + DSN), browser smoke (user เปิดเองที่ `http://localhost:8080`)

## Status (2026-10-26, รอบสิบเอ็ด) — PDF417 + Aztec encoders

- ใหม่ `internal/pdf417`: high-level encoder (text/numeric/binary submodes) + RS GF(929) + ประกอบแถวด้วย cluster tables (3×929) + start/stop/indicator; ใหม่ `internal/aztec`: high-level state lattice (Upper/Lower/Digit/Mixed/Punct/binary-shift) + bit stuffing + RS หลาย GF (4/6/8/10/12 บิต) + mode message/bulls-eye/alignment (compact 11+4L, full 14+4L + alignment)
- แหล่งข้อมูล: เครือข่ายใช้ได้ (`GOPROXY=https://proxy.golang.org`) จึงดึง `boombuler/barcode` (MIT) เป็น oracle ชั่วคราว — ตารางตัวเลข (codewords 2787 ค่า, correction factors) copy ตรงพร้อม attribution แล้วตรวจ checksum ครบทุกค่า; ตรรกะ encoder เขียนใหม่ตามอัลกอริทึมที่อ่านมา
- ยืนยัน: matrix เท่า byte กับ upstream writer (PDF417 7 เคส levels 0/1/2/5/8, Aztec 9 เคส) ก่อนลบ oracle + `go mod tidy` (repo ไม่มี dep เพิ่ม); ล็อก golden ถาวร (`TestPdfGoldenAB` 3×4, `TestAzGoldenAB` 15×15) + structural (start/stop bits, indicator decode, ขนาด grid ถูกสูตร, bulls-eye core)
- ต่อ render: `Barcode="PDF417"` (`ErrorCorrection` 0..8/`Auto`=2) และ `Barcode="Aztec"` (ECC percent/`Auto`=33) ใน HTML (SVG + caption) และ PDF (vector rects + caption); level/percent ผิดและข้อมูลเกิน → labeled fallback เดิม (เทสย้ายไป MaxiCode ซึ่งเหลือเป็น 2D เดียวที่ยัง fallback)
- สถานะตรวจ: `gofmt` สะอาด, `go vet` ผ่าน, `go test -count=1 ./...` เขียว 10 แพ็กเกจ
- คงเหลือจริง: MaxiCode (2D สุดท้ายที่ fallback — ไม่มี encoder ใน boombuler/gozxing, ไม่มี oracle), live DB จริง (รอ server + DSN), browser smoke (user เปิดเอง)

## Status (2026-10-26, รอบสิบสอง) — MaxiCode encoder (2D ครบทุกตัว)

- ใหม่ `internal/maxicode`: structured primary (mode 2/3: postcode/country/service class) + secondary Appendix-A (shift/latch/number compression) + RS GF(2^6) + วาง modules ตาม ISO sequence + orientation marks; `Encode(mode 2..6, data)`; finder เป็น overlay ตอน render
- แหล่งข้อมูล: เจอ `ingridhq/maxicode` (MIT, pure Go) ผ่าน proxy — ใช้เป็น oracle ชั่วคราวแล้วลบ + `go mod tidy` (repo ไม่มี dep เพิ่ม); ตาราง placement/charset (990+256+256 ค่า) copy ตรงพร้อม attribution แล้วตรวจ checksum ครบ
- ยืนยัน: matrix เท่า byte กับ upstream writer (8 เคส: modes 2/3/4/5/6) ก่อนลบ oracle; ล็อก golden ถาวร (`TestMxGoldenAB` 30×33 mode 4) + structural (orientation marks 13 จุด, SVG มี finder circles, error paths mode/oversize/header)
- ประเด็นที่เจอ: เทส PNG ของ upstream เองตกใน env นี้ (raster drift ของ gg — ไม่เกี่ยวกับ grid logic ที่เราเทียบ); port RS ตอนแรกใช้ top-bit ผิด (32 แทน 64) แล้วแก้ — oracle จับได้ก่อนล็อก golden
- ต่อ render: `Barcode="MaxiCode"` (`Barcode.Mode` 2..6, default 4) ใน HTML (SVG + finder circles + caption) และ PDF (vector rects + finder core/ring + caption); PDF ปรับ grid แชร์เป็น nw×nh (MaxiCode 30×33 ไม่ใช่จัตุรัส — symbol จัตุรัสเดิมไม่เปลี่ยน); ค่าผิด/ข้อมูลเกิน → fallback เดิม
- สถานะตรวจ: `gofmt` สะอาด, `go vet` ผ่าน, `go test -count=1 ./...` เขียว 11 แพ็กเกจ
- คงเหลือจริง: live DB จริง (รอ server + DSN), browser smoke (user เปิดเองที่ `http://localhost:8080`)

## Status (2026-10-26, รอบสิบสาม) — ลองปิด live DB + browser smoke (ยังติด env)

- Live PG/MSSQL: ลองแล้ววันนี้ — ไม่มี DSN ใน env (`REPORT_PG_DSN`/`REPORT_MSSQL_DSN` ว่าง), Docker daemon ต่อไม่ได้ (`permission denied` ที่ `~/.docker/run/docker.sock`), ไม่มี postgres/mssql binary ใน homebrew (มีแค่ sqlite), `pip download pgserver` ล้ม (PyPI ต่อผ่าน proxy ไม่ได้ — proxy ใช้ได้เฉพาะ Go module proxy), mysqld ใน anaconda ไม่ช่วย (port นี้ไม่มี MySQL driver และแผนระบุ PG/MSSQL)
- หลักฐาน: `go test -v -run TestSQLLive ./internal/report/` → SKIP ทั้งสองเทส (`REPORT_PG_DSN not set`, `REPORT_MSSQL_DSN not set`), suite เขียว; mapping ตรวจกับ driver จริงใน module cache แล้ว (รอบสิบ)
- วิธีปิดโดย user: ตั้ง DSN แล้วรันคำสั่งเดียว — `REPORT_PG_DSN='postgres://user:pass@localhost:5432/reportnext?sslmode=disable' REPORT_MSSQL_DSN='sqlserver://sa:Pass@localhost:1433?database=master' GOMODCACHE=/tmp/gomodcache GOPROXY=https://proxy.golang.org GOSUMDB=off GOFLAGS=-mod=mod GOCACHE=/tmp/gocache-report go test -count=1 -run TestSQLLive ./internal/report/` (เทสสร้าง/ลบตารางชั่วคราวเอง ไม่กระทบข้อมูลจริง)
- Browser smoke: ลองแล้ววันนี้ — sandbox bind socket ไม่ได้ (`python socket.bind 127.0.0.1:8080/18080` → `Operation not permitted`) จึงรัน server/เปิด browser (มี Chrome/Safari เป็น .app แต่เปิด GUI ไม่ได้) จากที่นี่ไม่ได้; `TestSrvDistBundle` (handler-level) เขียวอยู่
- วิธีปิดโดย user: `cd report-next && go run ./cmd/report-designer` แล้วเปิด `http://localhost:8080` — ตรวจ Explorer/Inspector/Preview/Source + MenuBar/TabStrip/StatusBar ตาม `web/README.md`
- สถานะตรวจ: `go test -count=1 ./...` เขียว 11 แพ็กเกจ (ไม่มีโค้ดเปลี่ยนรอบนี้ นอกจาก docs)

## Status (2026-10-26, รอบสิบสี่) — web designer ใช้ shell เดียวกับ designer-next

- ปัญหา: `report-next/web` เดิมเป็นโครงบาง ๆ (7 ไฟล์, `styles.css` 16 บรรทัดพื้นเทาเข้ม) ไม่ใช่รูปลักษณ์ designer-next — ปรับใหม่ให้ใช้ IDE chrome ชุดเดียวกัน: copy `tokens.css` + `shell.css` + `Icon`/`Splitter`/`Modal` มาตรง ๆ พร้อม attribution, พอร์ต `MenuBar`/`Toolbar`/`TabStrip`/`StatusBar` เป็น report flavor (File/View/Help, Sample/Import/Render/PDF/DOCX, Start/Design/Preview/IR Source, status + report/v1), เพิ่ม `StartTab` (hero + capabilities) + `AboutDialog`, แถบข้างปรับขนาดได้ด้วย Splitter, shortcuts Ctrl+S (โหลด IR JSON) / F5 (render) / F1, เตือนก่อนปิดเมื่อมีงานที่ยังไม่เซฟ
- การตรวจหน้าจอ: ขอ Playwright มาตรง ๆ ไม่ได้ — npm registry ต่อไม่ติด, Chrome (headless-shell/full) crash ตั้งแต่ start ใน sandbox นี้; ใช้วิธีแทน: `tsc --noEmit` สะอาด, `vite build` สำเร็จ (46 modules), SSR `renderToString(<App/>)` ผ่าน 14 จุด (menubar/toolbar/tabs/hero/statusbar ครบ, ไม่ crash), `go test ./...` เขียว 11 แพ็กเกจรวม `TestSrvDistBundle` กับ dist ใหม่
- วิธีเปิดดูจริง: `cd report-next && go run ./cmd/report-designer` แล้วเปิด `http://localhost:8080/` (หรือ `http://IP-เครื่อง:8080`)

## Status (2026-10-26, รอบสิบห้า) — Playwright smoke test (รันที่เครื่อง user)

- คำถามว่า Playwright ใช้ Chromium หรือไม่: ใช่ — Playwright ขับ Chromium ของตัวเองผ่าน CDP (เครื่องนี้มี Python Playwright 1.60.0 + Chromium cache 1148→1243, headless shell, firefox, webkit ครบ)
- ลองรันจริงใน sandbox แล้วทั้ง 4 เอนจิน (chromium เต็ม/headless-shell/firefox/webkit): โปรเซสถูกฆ่าหมด (`SIGABRT` ตอน start, `TargetClosedError`, `kill EPERM`) — sandbox นี้รันเบราว์เซอร์ไม่ได้เลย ไม่ใช่ปัญหาของสคริปต์
- สิ่งที่ส่งแทน: `web/e2e/smoke.py` (stdlib + playwright เท่านั้น) — เช็ค server, shell ครบ (menu/toolbar/tabs/hero/status), เปิด File menu, โหลด sample ผ่าน toolbar, render preview ลง iframe, ตรวจ IR source, เปิด inspector, fail เมื่อมี page error (ยกเว้น favicon 404), เก็บ screenshot 4 ใบที่ `web/e2e/shots/`; ตรวจ syntax + fail path ที่นี่แล้ว (`SMOKE-FAIL: server unreachable; start it with go run ...`)
- วิธีรัน: `pip install playwright && playwright install chromium`, start server, `python3 web/e2e/smoke.py` (มี `--url`, `--headed`); วิธีรันอยู่ใน `web/README.md` แล้ว

## Status (2026-10-26, รอบสิบหก) — Docker Compose deploy

- ใหม่: `report-next/Dockerfile` (multi-stage: node build web → go build server → alpine runtime, `CGO_ENABLED=0`, healthcheck `GET /`), `report-next/docker-compose.yml` (service `app` port 8080 + profile `live` มี `pg` 17-alpine กับ `mssql` 2022 + healthchecks + volumes), `report-next/.dockerignore`
- ตรวจที่นี่ได้แค่ `docker compose config` (ผ่านทั้ง default และ `--profile live`) — daemon ต่อจาก sandbox ไม่ได้ (`permission denied`) เลย build/run ไม่ได้ ต้องรันที่เครื่อง user (มี Docker Desktop อยู่แล้ว)
- วิธีใช้: `docker compose up -d --build` แล้วเปิด `http://localhost:8080/` (compose แก้ปัญหา "เปิดไม่ขึ้น" เพราะไม่ต้อง start server เอง); live DB: `docker compose --profile live up -d` แล้วรัน `TestSQLLive` ด้วย DSN ใน `web/README.md`
- สถานะตรวจ: `docker compose config` ผ่าน; `go test ./...` ไม่แตะ (ไม่มีโค้ด Go เปลี่ยนรอบนี้)

## Status (2026-10-26, รอบสิบเจ็ด) — ปิด render fail + Playwright smoke เขียว

- ต้นเหตุ render ไม่สำเร็จ: image ที่รันอยู่เสิร์ฟ UI เก่า (`static/index.html` 1,098 bytes) ไม่ใช่ React designer — `Dockerfile` stage สุดท้าย copy แค่ binary ลืม `web/dist` ทั้งที่ `server.go` เขียนรอไว้แล้ว (`os.Stat("web/dist/index.html")` → เสิร์ฟ React ก่อน); แก้ 1 บรรทัด: `COPY --from=web /web/dist ./web/dist`
- rebuild จริงที่นี่ (ขออนุมัติ escalated เพื่อแตะ docker socket): `docker compose up -d --build app` ผ่าน, `GET /` เปลี่ยนเป็น React bundle (index + `/assets/*.js/*.css`)
- Playwright รันได้แล้วที่นี่ (หลัง escalated): ติดตั้ง `chromium_headless_shell-1223` เพิ่ม (`playwright install chromium`, cache เดิมมีถึง 1243 แต่ 1.60.0 เรียก 1223); แก้ selector ใน `web/e2e/smoke.py` 2 จุด (ปุ่ม Sample/Render ชื่อซ้ำกับ StartTab/preview pane — ใช้ `exact=True` กับ `button[title=...]`)
- ผล: `SMOKE-OK` 23/23 checks, `no page errors`, screenshots 4 ใบใน `web/e2e/shots/` (ดู `03-preview.png` แล้ว — preview แสดง "The report goes here..." ถูกต้อง); เปิด `http://192.168.1.19:8080/` ได้เลย
- คงเหลือจริง: live DB จริง (รอ DSN), `go test ./...` รอบนี้ไม่รันซ้ำ (แตะแค่ Dockerfile + smoke.py นอกแพ็กเกจ Go)

## Status (2026-10-26, รอบสิบแปด) — Playwright e2e ชุดใหม่ 13 เคส

- เขียน `web/e2e/smoke.py` ใหม่เป็นเคสฟังก์ชัน (พฤติกรรมเดิมครบ + เคสใหม่): import `.frx` จริง (`table.frx` ผ่าน `#rz-import`), render วัตถุหนัก (`Simple Matrix.frx`, `Gauge.frx`), ดาวน์โหลด PDF/DOCX ตรวจ magic bytes (`%PDF`/`PK`) + status รายไฟล์, JSON เสียใน Source → `role=alert` "invalid JSON" ไม่ crash, F5 กลับมาแท็บ Preview, About dialog เปิด/ปิดผ่าน Help menu, gate `no page errors` เหมือนเดิม
- ผลรันจริงกับ container: `SMOKE-OK` ครบทุกเช็ค (`no page errors (got [])`), screenshots `01-start` … `09-about` (ดู `09-about.png` แล้ว dialog ถูกต้อง); negative check: แก้ selector ผิดชั่วคราวได้ `SMOKE-FAIL` + `failure.png` แล้ว revert (ไฟล์กลับเท่าเดิม, `py_compile` ผ่าน)
- ระหว่างทำเจอ selector ซ้ำ 3 จุด (Sample/Render/Close มี 2 elements) — แก้ด้วย `exact=True` / `button[title=...]` / `.modal button.primary` แล้ว
- `web/README.md` อัปเดตรายการ 13 เคส + screenshots + options `--url`/`--headed` แล้ว

## Status (2026-10-26, รอบสิบเก้า) — Preview auto-render (แก้ "Load sample แล้ว Preview ว่าง")

- อาการ: กด `Load sample report` → `openReport` ล้าง `html` แล้วไปแท็บ Design; เปิดแท็บ Preview ต่อแต่ไม่มีอะไรสั่ง render → เจอข้อความ "Press Render to preview" ต้องกด Render เองอีกที (server ปกติ — `/api/render` ได้ 200 ตลอด)
- แก้: `web/src/App.tsx` เพิ่ม `useEffect` — เข้าแท็บ Preview ครั้งแรกโดยมี report แต่ `html === ""` ให้ `renderNow()` เอง (guard `busy !== "render"` กันยิงซ้ำ); `tsc --noEmit` + `vite build` ผ่าน, `web/dist` ใหม่, rebuild container แล้ว
- e2e: เพิ่มเคส `preview auto-renders` ใน `smoke.py` (import แล้วเปิด Preview ตรง ๆ ไมกด Render) — `SMOKE-OK` ครบทุกเช็ครวมเคสใหม่, `no page errors`
- พฤติกรรมที่เหลือ: แก้ IR หลัง render แล้วกลับมาแท็บ Preview จะยังเห็นภาพเก่าจนกว่าจะกด Render (ตั้งใจคงไว้ ไม่ auto ทุกครั้งเพื่อไม่ยิง server บ่อย)

## Status (2026-10-26, รอบยี่สิบ) — sample reports เพิ่มเป็น 6 ตัว + เลือกได้

- เดิมมี sample เดียว (`hello.frx` ผ่าน `/api/sample`); เพิ่มให้ครบ 6 ไฟล์ที่มีใน `testdata/`: `table`, `box`, `Gauge`, `Simple Matrix`, `Subreport`
- Server (`internal/report/server.go`): embed `testdata/*.frx` เป็น `srvSamples`, ใหม่ `GET /api/samples` → `["Gauge","Simple Matrix","Subreport","box","hello","table"]`, `GET /api/sample?name=table` (default `hello`; ชื่อมี `/\..` → 400, ไม่มีไฟล์ → 404); `go vet` + Go test ใหม่ `TestSrvSamplesList`/`TestSrvSampleByName` ผ่าน
- Web: `api.ts` มี `fetchSamples()` + `fetchSample(name)`; แท็บ Start แสดงปุ่มราย sample (ดึงจาก `/api/samples`, fallback `hello`); toolbar Sample + File menu ยังโหลด `hello` เหมือนเดิม; status ระบุชื่อ sample (`Sample loaded (table): …`)
- e2e: เคสใหม่ `sample picker loads table` (+screenshot `10-samples.png`) — รันจริงกับ container ที่ rebuild แล้ว `SMOKE-OK` 40/40, `no page errors`; `web/README.md` + `web/e2e/REPORT.md` อัปเดตแล้ว

## Status (2026-10-26, รอบยี่สิบเอ็ด) — parity plan P1–P5 ครบ (web designer เติมเต็ม)

- ทำตาม [web-designer-parity-plan-th.md](web-designer-parity-plan-th.md) ครบ 5 phase, web-only ทั้งหมด (ไม่แตะ Go engine/IR/API)
- P1 explorer เขียนได้: `ir/mutate.ts` (add/delete/duplicate band/object) + ปุ่มราย node + `ConfirmDialog` ก่อนลบ; P2 typed inspector (Text/geometry/font/Barcode symbology/ShowText/border + raw table เฉพาะ attr ที่เหลือ) + ปุ่ม Source Format; P3 undo/redo (history cap 50, coalesce 1.5s, toolbar + เมนู Edit + Ctrl+Z/Y, ข้าม native undo ใน input); P4 `ReportCanvas` (sheet/band/object 1:1 preview scale, คลิกเลือก, ลากย้าย commit ครั้งเดียว) เสียบกลางแท็บ Design; P5 `BottomDock` (Output/Issues + jump) + `NewReportDialog` (blank/copy sample) + เมนู File/View
- ระหว่างทำเจอและแก้: selector กำกวมเพิ่ม (canvas head vs tree → scope `.tree`), dock ไปอยู่ rail ขวาเพราะ shell เป็น row flex (override `flexDirection: column` ฝั่ง App บรรทัดเดียว ไม่แตะ shell.css ร่วม), test เขียนสมมุติผิด 1 จุด (console ไม่ว่างเพราะ log มาก่อน)
- ตรวจปิดงาน: `tsc --noEmit` + `vite build` ผ่านทุก phase, rebuild container, e2e `SMOKE-OK` **66/66 checks (25 เคส)** `no page errors`, ดูภาพ `14-canvas.png`/`15-dock.png` ด้วยตาแล้ว, `web/e2e/REPORT.md` + `web/README.md` อัปเดตเลขปัจจุบันแล้ว
- เบี่ยงจากแผนที่บันทึกไว้: Visible checkbox ไม่ทำ (renderer ไม่รองรับ field นี้ — ใส่ไปจะเป็น field หลอก)

## Status (2026-10-26, รอบยี่สิบสอง) — parity รอบ 2 ครบ (Q1–Q5 ด้วย 4 subagents)

- แบ่งงานขนาน 4 สายตามไฟล์ที่ไม่ทับกัน (Inspector / Explorer / Canvas+CSS / validate+TabStrip+Dock) ตรวจ contract แล้วประกอบ `App.tsx` เอง (issues→explorer, errorCount→TabStrip, log level) — `tsc` + `vite build` ผ่านรอบเดียว
- Q1 color picker (ยืนยันชื่อ attr จาก Go ก่อนเขียน: Fill/TextFill/Border.Color), Q2 search + badge, Q3 severity + error chip, Q4 resize handle (commit ครั้งเดียว), Q5 log level + filter + import-ไฟล์ขยะเป็น negative จริง
- ระหว่างทำเจอและแก้: เทสเลือกคำค้นทับ substring 2 รอบ (Text1/Text12 → ใช้ชื่อ band), chip นับเฉพาะ error (band ว่างเป็น warning ตามแผน — เทสเลยสร้าง duplicate ให้เป็น error), failure.png ค้างจากรันก่อน (timeout ไม่เขียนภาพใหม่), dock บัง log เพราะค้างแท็บ Issues, console.error จาก 400 ที่ตั้งใจยิง (ยกเว้นรายการนี้ใน gate พร้อมคอมเมนต์)
- ตรวจปิดงาน: rebuild container, e2e `SMOKE-OK` **79/79 checks (30 เคส)** `no page errors`, ดูภาพ `16-color.png`/`17-resize.png` ด้วยตาแล้ว, `web/e2e/REPORT.md` + `web/README.md` อัปเดตแล้ว

## Status (2026-10-26, รอบยี่สิบสาม) — parity รอบ 3 ครบ (R1–R3 ด้วย 2 subagents)

- แบ่งงานขนาน 2 สายตามไฟล์ที่ไม่ทับกัน (Inspector / Canvas+CSS) ตรวจ contract แล้วประกอบเอง — `tsc` + `vite` build ผ่านรอบเดียว
- R1 ReadonlyField (Objects/Total width/Characters/Engine), R2 snap (threshold 3px + เส้นไกด์, commit ครั้งเดียว), R3 zoom (Ctrl+wheel 0.25–2x + Reset, pan native scroll + space-drag, zoom ไม่แตะ IR)
- ระหว่างทำเจอและแก้: เทสอ่าน count ทันทีหลังคลิกโดยไม่รอ re-render — ไล่ใส่ `wait_for` ให้ครบทุจุดที่ assert หลัง mutation (บทเรียน: Playwright `wait_for` ก็ strict mode เหมือน `click`, selector ต้อง unique); เคส 31 นับ index ข้าม band — เปลี่ยนเป็นหาด้วยข้อความแทน; Ctrl+wheel ต้อง hover canvas ก่อน; `Sample loaded` status-sync แทนรอ `Page:` ที่ stale ได้
- ตรวจปิดงาน: rebuild container, e2e `SMOKE-OK` **86/86 checks (33 เคส)** `no page errors`, `web/e2e/REPORT.md` + `web/README.md` อัปเดตแล้ว

## Status (2026-10-26, รอบยี่สิบสี่) — parity รอบ 4 ครบ (S1 ด้วย 2 subagents)

- แบ่งงานขนาน 2 สายตาม contract ที่ล็อกไว้ก่อน (Inspector Section markup / CSS classes) ตรวจ contract แล้วประกอบเอง — `tsc` + `vite` build ผ่านรอบเดียว
- S1: `Section` พับได้ (title + count + actions) จัดกรุ๊ป General/Text/Barcode/Layout/Attributes ตาม kind + CSS ลูกศรพับกาง
- ตรวจปิดงาน: rebuild container, e2e `SMOKE-OK` **88/88 checks (34 เคส)** `no page errors`, ดูภาพ `20-sections.png` ด้วยตาแล้ว (General (2), Text (4), Layout (8), Attributes), `web/e2e/REPORT.md` + `web/README.md` อัปเดตแล้ว
