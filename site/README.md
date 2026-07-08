# goal Playground

An interactive, browser-based playground with **one live example per language
feature**, each shown in both a **valid** and a **broken** form. A feature shows a
`.goal` program and the plain Go it transpiles to (or, in the broken form, the
located checker diagnostic); edit the source and the output re-transpiles live.

Content is **auto-derived from [`../docs/by-example.md`](../docs/by-example.md)**
— the same feature reference whose goal→Go pairs are copied from the verified
golden files. The playground therefore can't drift from the language: a new
by-example section becomes a new playground feature on the next build.

## How it works

- **The transpiler runs entirely in the browser via WebAssembly.** goal is
  written in Go, so [`cmd/goal-wasm`](../cmd/goal-wasm) compiles the real
  `internal/pipeline` package to `goal.wasm` and exposes a single
  `goalTranspile(source, name)` function. There is **no backend and no CDN** —
  `goal.wasm` is **preloaded in the background on page load**, so the first
  transpile has no load pause; every transpile is then a synchronous in-page call.
- **[`cmd/build-playground`](../cmd/build-playground)** parses
  `docs/by-example.md` and emits **`features.json`** — the per-feature manifest
  (title, prose, the `.goal` source, and the seed Go output). Because that tool
  imports the real pipeline, it **re-transpiles every example at generate time and
  asserts the result matches the Go block locked in the doc**, so the manifest and
  the doc cannot drift from the transpiler.
- **`playground-map.json`** is the presentation layer: it groups features into
  categories, assigns accent colors and display numbers, and pairs each feature
  with its **valid → error** variants. It holds *only* presentation — all content
  still comes from `features.json`. `app.js` joins the two by anchor and shows an
  "out of sync" banner if they disagree; the same invariant is a build gate
  ([`cmd/build-playground/map_test.go`](../cmd/build-playground/map_test.go)),
  which fails if the map and `docs/by-example.md` fall out of correspondence — so
  a feature added, removed, or re-categorized in the doc can't silently desync the
  page.
- `app.js` is a dependency-free ES module: no editor or highlighter libraries —
  a plain textarea with a line-number gutter, and a small goal/Go highlighter and
  diagnostic formatter built in.

A live transpile reproduces the seed output byte-for-byte — the same
`internal/pipeline` code produced both (the seed via the host build, the live
result via WASM).

## Build & serve

```bash
./scripts/build-site.sh                 # builds goal.wasm + wasm_exec.js + features.json
(cd site && python3 -m http.server 8000) # serve at http://localhost:8000
node playground-e2e/run.mjs              # headless gate: wasm in Node vs. every seed
```

Any static file server works, as long as it serves `.wasm` as `application/wasm`
(GitHub Pages does; the app falls back to a buffered fetch for servers that don't).

## Tracked vs generated

Tracked in git: the source assets — `index.html`, `app.js`, `styles.css`,
`playground-map.json`, this README.

Generated (gitignored, rebuild with `scripts/build-site.sh`): `goal.wasm`,
`wasm_exec.js` (copied from the active Go toolchain so it always matches the
compiler), and `features.json`.

## Deployment

[`.github/workflows/pages.yml`](../.github/workflows/pages.yml) builds the site on
every push to `main` that touches the transpiler, the doc, or the site, gates it on
the Node smoke test, and deploys the `site/` directory to GitHub Pages. Enable it
once under **Settings → Pages → Build and deployment → Source: GitHub Actions**.
