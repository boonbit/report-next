# report-next web designer (React)

Vanilla React 19 + Vite 6 designer for report IR. No `npm install` needed or
possible here: `node_modules` is a copy of designer-next's tree, and the two
platform binaries below were extracted from the local npm cache because the
sandbox has no network.

## One-time setup (already done in this checkout)

```sh
cp -r ../../designer-next/web/node_modules ./node_modules
```

Then install the two native binaries from the npm cache (exact versions the
tree requires — esbuild refuses to run on mismatch):

- `@rollup/rollup-darwin-arm64@4.62.2` into
  `node_modules/@rollup/rollup-darwin-arm64/` (rollup 4.65.0 loads it fine)
- `@esbuild/darwin-arm64@0.25.10` into
  `node_modules/@esbuild/darwin-arm64/` **plus** downgrade
  `node_modules/esbuild/{package.json,lib/main.js,bin/esbuild}` to the
  cached `esbuild-0.25.10.tgz` contents (host 0.25.12 rejects binary 0.25.10)

Extract any cached tarball by URL key with node and npm's bundled libs:

```sh
node /tmp/extract.js \
  https://registry.npmjs.org/@rollup/rollup-darwin-arm64/-/rollup-darwin-arm64-4.62.2.tgz \
  /tmp/rollup-darwin
```

## Build and run

```sh
node node_modules/vite/bin/vite.js build   # -> web/dist
cd .. && go run ./cmd/report-designer       # serves dist at /, API at /api
```

## Browser smoke test (Playwright, on your machine)

The sandbox cannot launch browsers, so run this where the server runs:

```sh
pip install playwright && playwright install chromium
cd report-next && go run ./cmd/report-designer   # keep it running
python3 web/e2e/smoke.py                          # needs the server at :8080
```

`smoke.py` covers 60 cases (148 checks): shell, File menu, sample load, preview
render (+auto-render, now paged), IR source, inspector, `.frx` import,
heavy-object render, PDF/DOCX/CSV/XLSX/HTML downloads (magic bytes),
invalid-JSON path, F5, About, sample picker, explorer add/delete/duplicate
(+Confirm, +cancel-keeps), undo/redo (+Edit menu), typed inspector, source
format, canvas click/drag, console dock, wizard + issues jump, color picker,
explorer search, issue badge + error chip, canvas resize, console error
filter (garbage import), readonly rows, canvas snap, canvas zoom
(IR-preserving), inspector sections (collapse/expand), palette add (U1),
keyboard edit (U2a), expression helper (U4), zoom presets + rulers (U5),
templates + recents (U6), HTML download (U8a), multi-select align (U2b),
data panel (U3), Thai toggle (U7), report CSV/XLSX (U8b), preview pager
(P1), preview zoom + find (P2), print popup (P3), hyperlinks + outline (P4),
CanGrow (R1), bound checkbox (R2), 1D barcodes (R3), stdlib expressions
(R4), summary/overlay (R5), positioned export (R6A), and a final
no-page-errors gate. Screenshots go to
`web/e2e/shots/` (`01-start` … `32-summary`, plus `failure.png` on failure)
for a manual look. Options: `--url` (default `http://localhost:8080/`),
`--headed`.

## Deploy with Docker Compose

Needs Docker Desktop running (one-time image pulls):

```sh
cd report-next
docker compose up -d --build   # app only -> http://localhost:8080/
docker compose logs -f app      # watch startup; Ctrl+C to detach
docker compose down             # stop
```

With live databases (Postgres + SQL Server, for `TestSQLLive`):

```sh
docker compose --profile live up -d
REPORT_PG_DSN='postgres://report:report@localhost:5432/reportnext?sslmode=disable' \
REPORT_MSSQL_DSN='sqlserver://sa:Report123!@localhost:1433?database=master' \
  go test -count=1 -run TestSQLLive ./internal/report/
```

`node_modules/.bin` has no exec bit on this volume, so always run vite
through `node`. The Go server serves `web/dist` when present and falls back
to the embedded static UI otherwise.
