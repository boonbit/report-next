"""Browser e2e for the report-next designer (Playwright, sync API).

Runs on your machine, not in the sandbox: it needs the Go server and a
Playwright browser (Chromium default).

  1. cd report-next && docker compose up -d --build app   # or go run ./cmd/report-designer
  2. python3 web/e2e/smoke.py [--url http://localhost:8080/] [--headed]

Cases: shell renders, File menu, sample load, preview render, IR source,
inspector, .frx import, heavy-object render (matrix/gauge), PDF/DOCX
downloads (magic bytes), invalid-JSON error path, F5 shortcut, About
dialog, sample picker, explorer add/delete/duplicate, undo/redo, typed
inspector, source format, canvas click/drag, console dock, wizard+issues,
color picker, explorer search, issue badge+error chip, canvas resize,
error filter, readonly rows, snap, zoom, inspector sections, palette add
(U1), keyboard edit (U2a), expression helper (U4), zoom presets+rulers
(U5), templates+recents (U6), HTML download (U8a), multi-select align (U2b), data panel (U3),
Thai toggle (U7), CSV/XLSX exports (U8b), preview pager/zoom/find (P1/P2),
print popup (P3), hyperlinks + outline (P4), and a final
no-page-errors gate. Screenshots land in web/e2e/shots/.
"""

import argparse
import json
import sys
import urllib.request
from pathlib import Path

HERE = Path(__file__).resolve().parent
SHOTS = HERE / "shots"
# Repo root is two levels up from web/e2e/.
TESTDATA = HERE.parent.parent / "internal" / "report" / "testdata"


def fail(msg: str) -> "NoReturn":
    print("SMOKE-FAIL:", msg)
    sys.exit(1)


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("--url", default="http://localhost:8080/")
    ap.add_argument("--headed", action="store_true")
    args = ap.parse_args()

    try:
        from playwright.sync_api import sync_playwright
    except ImportError:
        fail("playwright not installed: pip install playwright && playwright install chromium")

    try:
        with urllib.request.urlopen(args.url, timeout=5) as res:
            if res.status != 200:
                fail(f"server at {args.url} returned {res.status}")
    except Exception as e:
        fail(f"server at {args.url} unreachable ({e}); start it with: go run ./cmd/report-designer")

    SHOTS.mkdir(exist_ok=True)
    errors: list[str] = []
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=not args.headed)
        page = browser.new_page(viewport={"width": 1440, "height": 900})
        page.on("pageerror", lambda e: errors.append(f"pageerror: {e}"))
        page.on("console", lambda m: errors.append(f"console.{m.type}: {m.text}") if m.type == "error" else None)
        page.goto(args.url, wait_until="networkidle")

        def shot(name: str) -> None:
            page.screenshot(path=str(SHOTS / name))

        def check(cond: bool, what: str) -> None:
            print(("PASS " if cond else "FAIL ") + what)
            if not cond:
                shot("failure.png")
                fail(what)

        def load_sample() -> None:
            page.get_by_role("button", name="Sample", exact=True).click()
            page.get_by_text("Page:").first.wait_for(timeout=15000)

        # 1. Shell renders.
        for label in ("File", "Edit", "View", "Help"):
            check(page.get_by_role("button", name=label, exact=True).count() >= 1, f"menu {label}")
        for label in ("Sample", "Import", "Render", "PDF", "DOCX"):
            check(page.get_by_role("button", name=label).count() >= 1, f"toolbar {label}")
        for label in ("Start", "Design", "Preview", "IR Source"):
            check(page.get_by_role("tab", name=label).count() >= 1, f"tab {label}")
        check(page.get_by_text("report-next designer").count() >= 1, "start hero")
        check("report/v1" in (page.text_content(".statusbar") or ""), "statusbar")
        shot("01-start.png")

        # 2. File menu opens and closes.
        page.get_by_role("button", name="File", exact=True).click()
        check(page.locator(".menu-drop").is_visible(), "file menu opens")
        page.keyboard.press("Escape")

        # 3. Load the sample through the toolbar.
        load_sample()
        check(page.get_by_text("Band:").count() >= 1, "explorer shows bands")
        check("page(s)" in (page.text_content(".statusbar") or ""), "status confirms load")
        shot("02-design.png")

        # 4. Preview tab renders the report into an iframe.
        page.get_by_role("tab", name="Preview").click()
        page.locator('button[title="Render the current IR to HTML preview"]').click()
        frame = page.frame_locator("iframe[title='Report preview']")
        frame.locator("body").wait_for(timeout=15000)
        body = frame.locator("body").inner_text()
        check(len(body.strip()) > 0, "preview iframe has content")
        shot("03-preview.png")

        # 5. Source tab shows editable IR JSON; keep a valid copy for later.
        page.get_by_role("tab", name="IR Source").click()
        editor = page.get_by_label("Report JSON source")
        check(editor.is_visible(), "source editor visible")
        valid_json = editor.input_value() or ""
        check('"pages"' in valid_json, "source holds IR JSON")

        # 6. Back to design; inspector edits without crashing.
        page.get_by_role("tab", name="Design").click()
        page.get_by_text("Band:").first.click()
        check(page.get_by_label("Band name").count() >= 1, "inspector shows band")
        shot("04-inspector.png")

        # 7. Import a real .frx file through the hidden file input.
        frx = TESTDATA / "table.frx"
        check(frx.is_file(), f"fixture {frx.name} exists")
        page.set_input_files("#rz-import", str(frx))
        page.get_by_text(f"Imported {frx.name}").wait_for(timeout=15000)
        check(page.get_by_text("Band:").count() >= 1, "import shows bands")
        shot("05-import.png")
        # Preview tab auto-renders on first open (no Render click needed).
        page.get_by_role("tab", name="Preview").click()
        auto = page.frame_locator("iframe[title='Report preview']")
        auto.locator("body").wait_for(timeout=15000)
        check(len((auto.locator("body").inner_text() or "").strip()) > 0, "preview auto-renders")
        page.get_by_role("tab", name="Design").click()

        # 8. Heavy objects (matrix, gauge) import and render without errors.
        for name in ("Simple Matrix.frx", "Gauge.frx"):
            fx = TESTDATA / name
            check(fx.is_file(), f"fixture {name} exists")
            page.set_input_files("#rz-import", str(fx))
            page.get_by_text(f"Imported {name}").wait_for(timeout=15000)
            page.locator('button[title="Render the current IR to HTML preview"]').click()
            frm = page.frame_locator("iframe[title='Report preview']")
            frm.locator("body").wait_for(timeout=15000)
            check(len((frm.locator("body").inner_text() or "").strip()) > 0, f"render {name}")
        shot("06-objects.png")

        # 9. PDF/DOCX toolbar buttons download real files (magic bytes).
        load_sample()
        with page.expect_download() as dl_pdf:
            page.locator('button[title="Export the current IR to PDF"]').click()
        pdf_bytes = Path(str(dl_pdf.value.path())).read_bytes()
        check(pdf_bytes.startswith(b"%PDF"), "pdf download is a PDF")
        check("Exported Page1.pdf." in (page.text_content(".statusbar") or ""), "status confirms pdf")
        with page.expect_download() as dl_docx:
            page.locator('button[title="Export the current IR to DOCX"]').click()
        docx_bytes = Path(str(dl_docx.value.path())).read_bytes()
        check(docx_bytes.startswith(b"PK"), "docx download is a zip")
        check("Exported Page1.docx." in (page.text_content(".statusbar") or ""), "status confirms docx")
        shot("07-export.png")

        # 10. Invalid JSON in Source shows an inline error, never a crash.
        page.get_by_role("tab", name="IR Source").click()
        editor = page.get_by_label("Report JSON source")
        editor.fill("{not valid json")
        page.get_by_role("button", name="Validate", exact=True).click()
        alert = page.get_by_role("alert")
        check(alert.is_visible(), "invalid JSON shows alert")
        check("invalid JSON" in (alert.inner_text() or ""), "alert names the problem")
        shot("08-invalid.png")
        # Restore the valid IR so later cases render.
        editor.fill(valid_json)
        page.get_by_role("button", name="Load", exact=True).click()
        page.get_by_text("Page:").first.wait_for(timeout=15000)

        # 11. F5 renders from anywhere and lands on the preview tab.
        page.keyboard.press("Escape")
        page.keyboard.press("F5")
        page.get_by_text("Rendered").first.wait_for(timeout=15000)
        check(page.get_by_role("tab", name="Preview").get_attribute("aria-selected") == "true",
              "F5 lands on preview")
        check("Rendered" in (page.text_content(".statusbar") or ""), "F5 status confirms render")

        # 12. Help menu opens the About dialog; Close dismisses it.
        page.get_by_role("button", name="Help", exact=True).click()
        page.get_by_text("About report-next…").click()
        check(page.get_by_role("dialog").is_visible(), "about dialog opens")
        shot("09-about.png")
        page.locator('.modal button.btn.primary:has-text("Close")').click()
        check(page.get_by_role("dialog").count() == 0, "about dialog closes")

        # 13. Start tab lists bundled samples; picking one loads it.
        page.get_by_role("tab", name="Start").click()
        page.get_by_role("button", name="table", exact=True).click()
        page.get_by_text("Sample loaded (table)").wait_for(timeout=15000)
        check(page.get_by_text("Band:").count() >= 1, "sample picker loads table")
        shot("10-samples.png")

        # 14. Explorer adds a band; deleting asks first (cancel keeps it).
        # Band rows live in the tree; the inspector head also says "Band:".
        def tree_bands() -> int:
            return page.locator(".tree").get_by_text("Band:").count()

        page.get_by_role("tab", name="Design").click()
        bands0 = tree_bands()
        page.locator('button[aria-label^="Add band to"]').first.click()
        page.locator(".tree").get_by_text("Band: Data1 2").wait_for(timeout=15000)
        check(tree_bands() == bands0 + 1, "explorer adds band")
        page.locator('button[aria-label^="Delete band"]').last.click()
        check(page.get_by_role("dialog").is_visible(), "delete asks first")
        page.get_by_role("button", name="Cancel", exact=True).click()
        check(tree_bands() == bands0 + 1, "cancel keeps band")
        shot("11-explorer.png")

        # 15. New band takes a text object; duplicate + delete object.
        page.locator('button[aria-label^="Add text to"]').last.click()
        page.locator(".panel-head").get_by_text("TextObject: Text1").wait_for(timeout=15000)
        check(page.get_by_text("TextObject: Text1").count() >= 1, "explorer adds text")
        page.get_by_role("button", name="Duplicate Text1", exact=True).click()
        page.locator(".tree").get_by_text("Text1 copy").wait_for(timeout=15000)
        check(page.get_by_text("Text1 copy").count() >= 1, "duplicate copies object")
        page.get_by_role("button", name="Delete Text1 copy", exact=True).click()
        page.locator('.modal button.btn.primary:has-text("Delete")').click()
        page.locator(".tree").get_by_text("Text1 copy").wait_for(state="detached", timeout=15000)
        check(page.get_by_text("Text1 copy").count() == 0, "delete removes object")

        # 16. Delete the added band for real.
        page.locator('button[aria-label^="Delete band"]').last.click()
        page.locator('.modal button.btn.primary:has-text("Delete")').click()
        page.locator(".tree").get_by_text("Band: Data1 2").wait_for(state="detached", timeout=15000)
        check(tree_bands() == bands0, "delete removes band")

        # 17. Undo/redo round-trip on a band rename (fresh load = empty stack).
        page.reload(wait_until="networkidle")
        undo_btn = page.locator('button[title="Undo (Ctrl+Z)"]')
        check(undo_btn.is_disabled(), "undo disabled on fresh load")
        page.get_by_role("button", name="Sample", exact=True).click()
        page.get_by_text("Page:").first.wait_for(timeout=15000)
        page.get_by_role("tab", name="Design").click()
        page.locator(".tree").get_by_text("Band:").first.click()
        bname = page.get_by_label("Band name")
        orig = bname.input_value()
        bname.fill("Renamed")
        page.locator(".tree").get_by_text("Band: Renamed").wait_for(timeout=15000)
        check(page.get_by_text("Band: Renamed").count() >= 1, "rename applies")
        undo_btn.click()
        page.locator(".tree").get_by_text(f"Band: {orig}").wait_for(timeout=15000)
        check(page.get_by_text(f"Band: {orig}").count() >= 1, "undo restores name")
        check("Undone." in (page.text_content(".statusbar") or ""), "undo status")
        page.locator('button[title="Redo (Ctrl+Y)"]').click()
        page.locator(".tree").get_by_text("Band: Renamed").wait_for(timeout=15000)
        check(page.get_by_text("Band: Renamed").count() >= 1, "redo reapplies name")
        page.get_by_role("button", name="Edit", exact=True).click()
        check(page.locator(".menu-drop").is_visible(), "edit menu opens")
        page.keyboard.press("Escape")
        shot("13-undo.png")

        # 18. Typed inspector: change a text width, render again.
        page.get_by_role("button", name="Sample", exact=True).click()
        page.get_by_text("Page:").first.wait_for(timeout=15000)
        page.get_by_role("tab", name="Design").click()
        page.locator(".tree").get_by_text("TextObject:").first.click()
        width = page.get_by_label("Object width")
        check(width.input_value() == "245.7", "inspector shows typed width")
        width.fill("300")
        page.get_by_role("tab", name="IR Source").click()
        src = page.get_by_label("Report JSON source")
        check('"value": "300"' in (src.input_value() or ""), "IR holds new width")
        page.locator('button[title="Render the current IR to HTML preview"]').click()
        frm2 = page.frame_locator("iframe[title='Report preview']")
        frm2.locator("body").wait_for(timeout=15000)
        check(len((frm2.locator("body").inner_text() or "").strip()) > 0, "render after typed edit")
        shot("12-inspector.png")

        # 19. Canvas click selects the object in the inspector.
        page.get_by_role("button", name="Sample", exact=True).click()
        page.get_by_text("Page:").first.wait_for(timeout=15000)
        page.get_by_role("tab", name="Design").click()
        page.locator('[data-obj="0-0-0"]').click()
        check(page.get_by_label("Object name").input_value() == "Text1", "canvas click selects")
        shot("14-canvas.png")

        # 20. Canvas drag moves the object (Left 0->50, Top 0->20).
        box = page.locator('[data-obj="0-0-0"]').bounding_box()
        check(box is not None, "canvas object has geometry")
        assert box is not None
        cx, cy = box["x"] + box["width"] / 2, box["y"] + box["height"] / 2
        page.mouse.move(cx, cy)
        page.mouse.down()
        page.mouse.move(cx + 50, cy + 20, steps=5)
        page.mouse.up()
        page.get_by_role("tab", name="IR Source").click()
        moved = page.get_by_label("Report JSON source").input_value() or ""
        check('"value": "50"' in moved, "drag sets Left")
        check('"value": "20"' in moved, "drag sets Top")

        # 21. Source Format reindents JSON without touching the report.
        page.get_by_role("tab", name="IR Source").click()
        ed = page.get_by_label("Report JSON source")
        ed.fill('{"a":1,"b":[2,3]}')
        page.get_by_role("button", name="Format", exact=True).click()
        check(ed.input_value() == '{\n  "a": 1,\n  "b": [\n    2,\n    3\n  ]\n}', "source format reindents")

        # 22. Console dock opens and logs a render.
        page.get_by_role("button", name="Sample", exact=True).click()
        page.get_by_text("Page:").first.wait_for(timeout=15000)
        page.get_by_role("button", name="View", exact=True).click()
        page.get_by_role("menuitem", name="Output Console").click()
        check(page.get_by_text("sample: loaded hello").count() >= 1, "console logs sample")
        page.locator('button[title="Render the current IR to HTML preview"]').click()
        page.get_by_text("render: ok").wait_for(timeout=15000)
        check(page.get_by_text("render: ok").count() >= 1, "console logs render")

        # 23. Wizard creates a blank report; its empty band is a clickable issue.
        page.get_by_role("button", name="File", exact=True).click()
        page.get_by_role("menuitem", name="New report…").click()
        check(page.get_by_role("dialog").is_visible(), "wizard opens")
        page.get_by_role("button", name="Create blank report", exact=True).click()
        page.locator(".tree").get_by_text("Band: Data1").wait_for(timeout=15000)
        page.get_by_role("button", name="Issues (1)", exact=True).click()
        page.get_by_text("Band Data1 is empty").click()
        check(page.get_by_label("Band name").input_value() == "Data1", "issue jumps to band")
        shot("15-dock.png")

        # 24. Color picker writes a hex color into the IR.
        page.get_by_role("button", name="Sample", exact=True).click()
        page.get_by_text("Page:").first.wait_for(timeout=15000)
        page.get_by_role("tab", name="Design").click()
        page.locator(".tree").get_by_text("TextObject:").first.click()
        page.get_by_label("Color TextFill").fill("#ff0000")
        page.get_by_role("tab", name="IR Source").click()
        check('"#ff0000"' in (page.get_by_label("Report JSON source").input_value() or ""), "IR holds hex color")
        page.locator('button[title="Render the current IR to HTML preview"]').click()
        frm3 = page.frame_locator("iframe[title='Report preview']")
        frm3.locator("body").wait_for(timeout=15000)
        check(len((frm3.locator("body").inner_text() or "").strip()) > 0, "render after color edit")
        shot("16-color.png")

        # 25. Explorer search filters rows; clearing restores them.
        page.get_by_role("tab", name="Design").click()
        page.get_by_label("Filter explorer").fill("PageFooter")
        check(page.locator(".tree").get_by_text("Band: ReportTitle1").count() == 0, "search hides non-match")
        check(page.locator(".tree").get_by_text("Band: PageFooter1").count() >= 1, "search keeps match")
        page.get_by_label("Filter explorer").fill("")
        check(page.locator(".tree").get_by_text("Band: ReportTitle1").count() >= 1, "clear restores rows")

        # 26. Empty band shows an issue badge in the tree.
        page.get_by_role("button", name="File", exact=True).click()
        page.get_by_role("menuitem", name="New report…").click()
        page.get_by_role("button", name="Create blank report", exact=True).click()
        page.locator(".tree").get_by_text("Band: Data1").wait_for(timeout=15000)
        check(page.locator(".tree .chip").count() >= 1, "tree shows issue badge")

        # 27. Error chip on the Design tab appears on duplicate names, clears when fixed.
        page.locator('button[aria-label^="Add text to"]').first.click()
        page.locator('button[aria-label^="Add text to"]').first.click()
        page.get_by_label("Object name").fill("Text1")
        page.locator(".tab .chip.bad").wait_for(timeout=15000)
        check(page.locator(".tab .chip.bad").count() >= 1, "design tab shows error chip")
        page.get_by_label("Object name").fill("Text2")
        check(page.locator(".tab .chip.bad").count() == 0, "chip clears when fixed")

        # 28. Canvas resize handle changes Width/Height.
        page.get_by_role("button", name="Sample", exact=True).click()
        page.get_by_text("Page:").first.wait_for(timeout=15000)
        page.get_by_role("tab", name="Design").click()
        page.locator('[data-obj="0-0-0"]').click()
        handle = page.locator('[data-resize="0-0-0"]')
        hbox = handle.bounding_box()
        check(hbox is not None, "resize handle visible")
        assert hbox is not None
        hx, hy = hbox["x"] + hbox["width"] / 2, hbox["y"] + hbox["height"] / 2
        page.mouse.move(hx, hy)
        page.mouse.down()
        page.mouse.move(hx + 40, hy + 20, steps=5)
        page.mouse.up()
        page.get_by_role("tab", name="IR Source").click()
        resized = page.get_by_label("Report JSON source").input_value() or ""
        check('"value": "286"' in resized, "resize sets Width")
        check('"value": "48"' in resized, "resize sets Height")
        shot("17-resize.png")

        # 29. Importing garbage logs an error the Errors filter keeps.
        import tempfile
        with tempfile.NamedTemporaryFile("w", suffix=".frx", delete=False) as f:
            f.write("this is not a report {{{")
            bad_path = f.name
        if page.locator(".dock").count() == 0:
            page.get_by_role("button", name="View", exact=True).click()
            page.get_by_role("menuitem", name="Output Console").click()
        page.get_by_role("button", name="Output", exact=True).click()
        page.set_input_files("#rz-import", bad_path)
        page.get_by_text("import: FAILED").wait_for(timeout=15000)
        page.get_by_role("button", name="Show errors only", exact=True).click()
        check(page.get_by_text("import: FAILED").count() >= 1, "error filter keeps errors")
        check(page.get_by_text("sample: loaded").count() == 0, "error filter hides info")
        page.get_by_role("button", name="Show all logs", exact=True).click()

        # 30. Inspector shows derived read-only rows.
        page.get_by_role("button", name="Sample", exact=True).click()
        page.get_by_text("Sample loaded (hello)").wait_for(timeout=15000)
        page.get_by_role("tab", name="Design").click()
        page.locator(".tree").get_by_text("TextObject:").first.click()
        check("23" in page.get_by_label("Readonly characters").inner_text(), "readonly char count")
        page.locator(".tree").get_by_text("Band: ReportTitle1", exact=True).click()
        page.get_by_label("Band name").wait_for(timeout=15000)
        before = page.get_by_label("Readonly objects").inner_text()
        page.locator('button[aria-label^="Add text to"]').first.click()
        page.locator(".panel-head").get_by_text("TextObject: Text2").wait_for(timeout=15000)
        # Adding selects the new object; go back to the band to read its rows.
        page.locator(".tree").get_by_text("Band: ReportTitle1", exact=True).click()
        page.get_by_label("Band name").wait_for(timeout=15000)
        check(page.get_by_label("Readonly objects").inner_text() != before, "readonly count grows")
        shot("18-readonly.png")

        # 31. Dragging to the band edge snaps (guide shows, Left lands 0).
        page.get_by_role("button", name="Sample", exact=True).click()
        page.get_by_text("Sample loaded (hello)").wait_for(timeout=15000)
        page.get_by_role("tab", name="Design").click()
        page.locator('button[aria-label^="Add text to"]').first.click()
        page.locator(".tree").get_by_text("TextObject: Text2").wait_for(timeout=15000)
        page.get_by_role("tab", name="Design").click()
        box = page.locator('.canvas-obj', has_text="TextObject: Text2").bounding_box()
        assert box is not None
        cx, cy = box["x"] + box["width"] / 2, box["y"] + box["height"] / 2
        page.mouse.move(cx, cy)
        page.mouse.down()
        page.mouse.move(cx - 100, cy, steps=5)
        check(page.locator(".canvas-guide").count() >= 1, "snap guide shows")
        page.mouse.up()
        page.get_by_role("tab", name="IR Source").click()
        check('"value": "0"' in (page.get_by_label("Report JSON source").input_value() or ""), "snap lands Left 0")

        # 32. Zoom changes display only: same drag, same IR delta.
        page.get_by_role("button", name="Sample", exact=True).click()
        page.get_by_text("Page:").first.wait_for(timeout=15000)
        page.get_by_role("tab", name="Design").click()
        page.locator(".canvas-scroll").hover()
        page.keyboard.down("Control")
        page.mouse.wheel(0, -240)
        page.keyboard.up("Control")
        zl = page.get_by_label("Canvas zoom").inner_text()
        check(zl != "100%", f"zoom changes display ({zl})")
        z = float(zl.strip("%")) / 100
        page.locator('[data-obj="0-0-0"]').click()
        box = page.locator('[data-obj="0-0-0"]').bounding_box()
        assert box is not None
        cx, cy = box["x"] + box["width"] / 2, box["y"] + box["height"] / 2
        page.mouse.move(cx, cy)
        page.mouse.down()
        page.mouse.move(cx + round(50 * z), cy, steps=5)
        page.mouse.up()
        page.get_by_label("Reset zoom").click()
        check(page.get_by_label("Canvas zoom").inner_text() == "100%", "zoom resets")
        page.get_by_role("tab", name="IR Source").click()
        moved2 = page.get_by_label("Report JSON source").input_value() or ""
        check('"value": "50"' in moved2, "zoom keeps IR delta")
        shot("19-zoom.png")

        # 33. Inspector sections collapse and expand.
        page.get_by_role("button", name="Sample", exact=True).click()
        page.get_by_text("Sample loaded (hello)").wait_for(timeout=15000)
        page.get_by_role("tab", name="Design").click()
        page.locator(".tree").get_by_text("TextObject:").first.click()
        page.get_by_role("button", name="Text section", exact=True).click()
        check(page.get_by_label("Font family").count() == 0, "section collapses")
        page.get_by_role("button", name="Text section", exact=True).click()
        check(page.get_by_label("Font family").count() >= 1, "section expands")
        shot("20-sections.png")

        # 35. U1 palette adds a text object to the first band.
        load_sample()
        page.get_by_text("Sample loaded (hello)").wait_for(timeout=15000)
        page.get_by_role("tab", name="Design").click()
        page.get_by_role("button", name="Add text object", exact=True).click()
        page.locator(".tree").get_by_text("TextObject: Text2").wait_for(timeout=15000)
        check(page.locator(".tree").get_by_text("TextObject: Text2").count() >= 1, "palette adds text")
        shot("21-palette.png")

        # 36. U2a keyboard: arrows nudge, Delete removes, Ctrl+Z restores.
        load_sample()
        page.get_by_text("Sample loaded (hello)").wait_for(timeout=15000)
        page.get_by_role("tab", name="Design").click()
        # Tree rows have no ids; Duplicate/Delete aria-labels are the unique hooks
        # (PageFooter1 already holds an object named Text12, so "Text1" is a substring).
        page.locator('button[aria-label="Duplicate Text1"]').locator("xpath=../..").click()
        page.keyboard.press("ArrowRight")
        page.get_by_role("tab", name="IR Source").click()
        nudged = page.get_by_label("Report JSON source").input_value() or ""
        check('"value": "1"' in nudged, "arrow nudges Left")
        page.get_by_role("tab", name="Design").click()
        # Selection survives tab switches, so Text1 is still selected here;
        # clicking again would toggle it off.
        page.keyboard.press("Delete")
        page.locator('button[aria-label="Duplicate Text1"]').wait_for(state="detached", timeout=15000)
        check(page.locator('button[aria-label="Duplicate Text1"]').count() == 0, "delete removes object")
        page.keyboard.press("Control+Z")
        page.locator('button[aria-label="Duplicate Text1"]').wait_for(timeout=15000)
        check(page.locator('button[aria-label="Duplicate Text1"]').count() >= 1, "undo restores object")
        shot("22-keyboard.png")

        # 37. U4 expression helper inserts a function skeleton, blocks unbalanced.
        page.locator('button[aria-label="Duplicate Text1"]').locator("xpath=../..").click()
        page.get_by_role("button", name="Insert expression", exact=True).click()
        check(page.get_by_role("dialog", name="Expression functions").is_visible(), "expr popup opens")
        page.get_by_role("button", name="Insert UpperCase expression", exact=True).click()
        check("UpperCase" in (page.get_by_label("Text content").input_value() or ""), "expr inserts skeleton")
        page.get_by_label("Text content").fill("[unclosed")
        page.keyboard.press("Enter")
        check(page.get_by_role("alert").is_visible(), "unbalanced expr warns")
        shot("23-expr.png")

        # 38. U5 zoom preset buttons rescale; rulers render in pixels.
        load_sample()
        page.get_by_text("Sample loaded (hello)").wait_for(timeout=15000)
        page.get_by_role("tab", name="Design").click()
        page.get_by_role("button", name="Set zoom to 50%", exact=True).click()
        check(page.get_by_label("Canvas zoom").inner_text() == "50%", "zoom preset applies")
        check(page.get_by_label("Horizontal ruler in pixels").count() >= 1, "horizontal ruler shows")
        check(page.get_by_label("Vertical ruler in pixels").count() >= 1, "vertical ruler shows")
        page.get_by_role("button", name="Set zoom to 100%", exact=True).click()
        check(page.get_by_label("Canvas zoom").inner_text() == "100%", "zoom preset resets")

        # 39. U6 table template loads; Start tab reopens it from recents.
        page.get_by_role("tab", name="Start").click()
        page.get_by_role("button", name="Create a new report from the table template", exact=True).click()
        page.locator(".tree").get_by_text("Band: Data1").wait_for(timeout=15000)
        check("table" in (page.text_content(".statusbar") or ""), "template loads with status")
        page.get_by_role("tab", name="Start").click()
        page.locator('button[aria-label^="Reopen recent report 1"]').click()
        page.locator(".tree").get_by_text("Band: Data1").wait_for(timeout=15000)
        check("Recent report reopened." in (page.text_content(".statusbar") or ""), "recent reopens report")
        shot("24-start.png")

        # 40. U8a preview HTML button downloads the rendered document.
        page.get_by_role("tab", name="Preview").click()
        with page.expect_download() as dl_html:
            page.get_by_role("button", name="Download HTML", exact=True).click()
        html_bytes = Path(str(dl_html.value.path())).read_bytes()
        check(html_bytes.lstrip().startswith(b"<"), "html download is markup")
        check("Exported Page1.html." in (page.text_content(".statusbar") or ""), "status confirms html")

        # 42. U2b multi-select: shift-click two objects, Align left equalizes.
        load_sample()
        page.get_by_text("Sample loaded (hello)").wait_for(timeout=15000)
        page.get_by_role("tab", name="Design").click()
        page.get_by_role("button", name="Add text object", exact=True).click()
        page.locator('[data-obj="0-0-1"]').wait_for(timeout=15000)
        # Palette spawns the new object exactly atop Text1; drag it clear first.
        box42 = page.locator('[data-obj="0-0-1"]').bounding_box()
        assert box42 is not None
        page.mouse.move(box42["x"] + box42["width"] / 2, box42["y"] + box42["height"] / 2)
        page.mouse.down()
        page.mouse.move(box42["x"] + box42["width"] / 2 + 400, box42["y"] + box42["height"] / 2, steps=5)
        page.mouse.up()
        page.locator('[data-obj="0-0-0"]').click()
        page.locator('[data-obj="0-0-1"]').click(modifiers=["Shift"])
        check(page.get_by_role("group", name="Align selected objects").is_visible(), "align toolbar shows")
        page.get_by_role("button", name="Align left edges", exact=True).click()
        page.get_by_role("tab", name="IR Source").click()
        ir = json.loads(page.get_by_label("Report JSON source").input_value() or "{}")
        objs = ir["pages"][0]["bands"][0]["objects"]
        lefts = {a["value"] for o in objs for a in o.get("attrs", []) if a.get("name") == "Left"}
        check(len(objs) == 2 and len(lefts) == 1, "align left equalizes")
        page.keyboard.press("Escape")
        check(page.get_by_role("group", name="Align selected objects").count() == 0, "escape clears multi")
        shot("25-align.png")

        # 43. U3 data panel: inspect inline CSV, insert a field, bind the band.
        page.get_by_role("tab", name="Design").click()
        page.get_by_role("button", name="Show data panel", exact=True).click()
        page.get_by_label("Data source kind").select_option("csv")
        page.get_by_label("File content for inspection").fill("name,age\nAnn,30")
        page.get_by_role("button", name="Inspect data source schema", exact=True).click()
        page.get_by_role("button", name="Pick data table data", exact=True).wait_for(timeout=15000)
        check(page.get_by_text("name").count() >= 1, "schema lists csv columns")
        page.get_by_role("button", name="Insert field data name as text", exact=True).click()
        page.locator(".tree").get_by_text("TextObject: Text3").wait_for(timeout=15000)
        check(page.locator(".tree").get_by_text("TextObject: Text3").count() >= 1, "field insert adds text")
        page.get_by_role("button", name="Bind band to picked data table", exact=True).click()
        page.get_by_role("tab", name="IR Source").click()
        src43 = page.get_by_label("Report JSON source").input_value() or ""
        check('"[data.name]"' in src43, "IR holds field text")
        check('"DataSource"' in src43 and '"data"' in src43, "bind sets DataSource")
        shot("26-data.png")

        # 44. U7 Thai toggle translates the chrome and back.
        page.get_by_role("button", name="Help", exact=True).click()
        page.get_by_role("menuitem", name="Switch to Thai").click()
        check(page.get_by_role("button", name="ตัวอย่าง", exact=True).count() >= 1, "toolbar speaks Thai")
        check(page.get_by_role("button", name="ไฟล์", exact=True).count() >= 1, "menu speaks Thai")
        shot("27-thai.png")
        page.get_by_role("button", name="ช่วยเหลือ", exact=True).click()
        page.get_by_role("menuitem", name="เปลี่ยนเป็นภาษาอังกฤษ").click()
        check(page.get_by_role("button", name="Sample", exact=True).count() >= 1, "english restored")

        # 45. U8b report exports: CSV has headers, XLSX is a zip.
        page.get_by_role("tab", name="Preview").click()
        with page.expect_download() as dl_csv:
            page.get_by_role("button", name="Export CSV", exact=True).click()
        csv_bytes = Path(str(dl_csv.value.path())).read_bytes()
        check(b"Text" in csv_bytes, "csv holds header")
        check("Exported Page1.csv." in (page.text_content(".statusbar") or ""), "status confirms csv")
        with page.expect_download() as dl_xlsx:
            page.get_by_role("button", name="Export XLSX", exact=True).click()
        xlsx_bytes = Path(str(dl_xlsx.value.path())).read_bytes()
        check(xlsx_bytes.startswith(b"PK"), "xlsx download is a zip")
        check("Exported Page1.xlsx." in (page.text_content(".statusbar") or ""), "status confirms xlsx")

        # 47. P1 pager navigates the prepared pages.
        page.get_by_role("tab", name="Start").click()
        page.get_by_role("button", name="outline", exact=True).click()
        page.get_by_text("Sample loaded (outline)").wait_for(timeout=15000)
        page.get_by_role("tab", name="Preview").click()
        page.get_by_text("Page 1 of 2").wait_for(timeout=15000)
        check(page.get_by_text("Page 1 of 2").count() >= 1, "pager shows counter")
        page.locator('button[aria-label="Go to the next page"]').click()
        page.get_by_text("Page 2 of 2").wait_for(timeout=15000)
        frm47 = page.frame_locator("iframe[title='Report preview']")
        check("Second chapter body" in (frm47.locator("body").inner_text() or ""), "next shows page 2")
        page.locator('button[aria-label="Go to the previous page"]').click()
        page.get_by_text("Page 1 of 2").wait_for(timeout=15000)
        page.locator('button[aria-label="Go to the last page"]').click()
        page.get_by_text("Page 2 of 2").wait_for(timeout=15000)
        page.locator('button[aria-label="Go to the first page"]').click()
        page.get_by_text("Page 1 of 2").wait_for(timeout=15000)
        check(page.get_by_text("Page 1 of 2").count() >= 1, "first returns to page 1")
        shot("28-pager.png")

        # 48. P2 zoom rescales the frame display-only.
        page.locator('button[aria-label="Set zoom to 150 percent"]').click()
        check(page.locator('button[aria-label="Set zoom to 150 percent"]').get_attribute("aria-pressed") == "true",
              "zoom preset active")
        check("150%" in (page.locator("iframe[title='Report preview']").get_attribute("style") or ""),
              "frame widens")
        page.locator('button[aria-label="Set zoom to 100 percent"]').click()
        check(page.get_by_text("Page 1 of 2").count() >= 1, "zoom keeps page")

        # 49. P2 find counts matches in the rendered page.
        page.get_by_label("Find text in preview").fill("Chapter")
        page.locator('button[aria-label="Search for the text"]').click()
        check(page.get_by_text("matches").count() >= 1, "find counts matches")
        page.get_by_label("Find text in preview").fill("zzz-no-such-text")
        page.locator('button[aria-label="Search for the text"]').click()
        check(page.get_by_text("No matches").count() >= 1, "find reports none")

        # 50. P3 print opens a printable popup holding the report.
        with page.expect_popup() as pop50:
            page.locator('button[aria-label="Print preview"]').click()
        popup = pop50.value
        popup.wait_for_load_state("domcontentloaded", timeout=15000)
        check("Outline demo" in (popup.content() or ""), "print popup holds report")
        popup.close()

        # 51. P4 hyperlinks: vendor URL opens a tab, page-jump uses the pager.
        frm51 = page.frame_locator("iframe[title='Report preview']")
        with page.expect_popup() as pop51:
            frm51.get_by_text("Visit example site").click()
        linkpop = pop51.value
        check("example.com" in (linkpop.url or ""), "url link opens example page")
        linkpop.close()
        frm51.get_by_text("Go to chapter two").click()
        page.get_by_text("Page 2 of 2").wait_for(timeout=15000)
        check(page.get_by_text("Page 2 of 2").count() >= 1, "page-jump link pages")

        # 52. P4 outline pane lists entries and jumps.
        page.locator('button[aria-label="Go to the first page"]').click()
        page.get_by_text("Page 1 of 2").wait_for(timeout=15000)
        page.get_by_role("button", name="Outline", exact=True).click()
        check(page.get_by_text("Chapter Two").count() >= 1, "outline lists entries")
        page.get_by_role("button", name="Go to page 2: Chapter Two", exact=True).click()
        page.get_by_text("Page 2 of 2").wait_for(timeout=15000)
        check(page.get_by_text("Page 2 of 2").count() >= 1, "outline jumps to page")
        shot("29-outline.png")

        # 54. R1 CanGrow: the long text object grows past its 20px design height.
        page.set_input_files("#rz-import", str(TESTDATA / "grow.frx"))
        page.get_by_text("Imported grow.frx").wait_for(timeout=15000)
        page.get_by_role("tab", name="Preview").click()
        frm54 = page.frame_locator("iframe[title='Report preview']")
        frm54.locator("body").wait_for(timeout=15000)
        check("GROWTAILMARKER" in (frm54.locator("body").inner_text() or ""), "grown text renders fully")
        long_style = frm54.locator("div.text").nth(1).get_attribute("style") or ""
        check("46." in long_style, "object grew past design height")
        shot("30-grow.png")

        # 55. R2 bound checkbox: dict Country renders checked, static stays clear.
        page.set_input_files("#rz-import", str(TESTDATA / "bind.frx"))
        page.get_by_text("Imported bind.frx").wait_for(timeout=15000)
        page.get_by_role("tab", name="Preview").click()
        frm55 = page.frame_locator("iframe[title='Report preview']")
        frm55.locator("body").wait_for(timeout=15000)
        body55 = frm55.locator("body").inner_text() or ""
        check("Alfreds" in body55, "bound row renders")
        check("✓" in body55, "bound checkbox checks")
        shot("31-bind.png")

        # 56. R3 1D barcodes render symbols (EAN grid is 95 modules wide).
        page.set_input_files("#rz-import", str(TESTDATA / "barcode1d.frx"))
        page.get_by_text("Imported barcode1d.frx").wait_for(timeout=15000)
        page.get_by_role("tab", name="Preview").click()
        frm56 = page.frame_locator("iframe[title='Report preview']")
        frm56.locator("body").wait_for(timeout=15000)
        check("HELLO128" in (frm56.locator("body").inner_text() or ""), "code128 caption renders")
        inner56 = frm56.locator("body").evaluate("el => el.innerHTML") or ""
        check('viewBox="0 0 95 1"' in inner56, "ean grid is 95 modules")

        # 57. R4 stdlib expressions evaluate in text objects.
        page.set_input_files("#rz-import", str(TESTDATA / "expr.frx"))
        page.get_by_text("Imported expr.frx").wait_for(timeout=15000)
        page.get_by_role("tab", name="Preview").click()
        frm57 = page.frame_locator("iframe[title='Report preview']")
        frm57.locator("body").wait_for(timeout=15000)
        body57 = frm57.locator("body").inner_text() or ""
        for want in ("Hello World", "007", "2026-01-02", "28", "one", "7"):
            check(want in body57, f"expr renders {want}")

        # 58. R5 summary renders last, overlay renders on the page.
        page.set_input_files("#rz-import", str(TESTDATA / "summary.frx"))
        page.get_by_text("Imported summary.frx").wait_for(timeout=15000)
        page.get_by_role("tab", name="Preview").click()
        frm58 = page.frame_locator("iframe[title='Report preview']")
        frm58.locator("body").wait_for(timeout=15000)
        body58 = frm58.locator("body").inner_text() or ""
        check("Summary tail" in body58, "summary renders")
        check("Overlay every page" in body58, "overlay renders")
        check(body58.index("Summary tail") > body58.index("body row"), "summary comes last")
        shot("32-summary.png")

        # 59. R6 positioned XLSX/DOCX keep content with merged layout cells.
        import zipfile
        page.get_by_role("tab", name="Start").click()
        page.get_by_role("button", name="outline", exact=True).click()
        page.get_by_text("Sample loaded (outline)").wait_for(timeout=15000)
        page.get_by_role("tab", name="Preview").click()
        with page.expect_download() as dl_xlsx59:
            page.get_by_role("button", name="Export XLSX", exact=True).click()
        zpath = Path(str(dl_xlsx59.value.path()))
        with zipfile.ZipFile(zpath) as zf:
            sheet = zf.read("xl/worksheets/sheet1.xml").decode("utf-8", "replace")
        check("Outline demo report" in sheet, "xlsx keeps title text")
        check("mergeCell" in sheet, "xlsx merges wide objects")
        with page.expect_download() as dl_docx59:
            page.get_by_role("button", name="Export DOCX", exact=True).click()
        with zipfile.ZipFile(Path(str(dl_docx59.value.path()))) as zf:
            docxml = zf.read("word/document.xml").decode("utf-8", "replace")
        check("Outline demo report" in docxml, "docx keeps title text")

        # 60. No page errors. Two noises are expected and excluded: the
        # favicon 404, and the single 400 the garbage-import case fires
        # on purpose (case 29 asserts on it directly above).
        real = [e for e in errors if "favicon.ico" not in e and "400 (Bad Request)" not in e]
        check(not real, f"no page errors (got {real!r})")

        browser.close()
    print("SMOKE-OK: screenshots in web/e2e/shots/")


if __name__ == "__main__":
    main()
