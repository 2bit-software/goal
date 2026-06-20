# goal Playground

An interactive, browser-based playground with **one live example per language
feature**. Each feature shows a `.goal` program and the plain Go it transpiles to;
edit the source and hit **Run** to re-transpile live.

Everything here is **auto-derived from [`../docs/by-example.md`](../docs/by-example.md)**
— the same feature reference whose goal→Go pairs are copied from the verified
golden files. The playground therefore can't drift from the language: a new
by-example section becomes a new playground feature on the next build.

## How it works

- **The transpiler runs entirely in the browser via WebAssembly.** goal is
  written in Go, so [`cmd/goal-wasm`](../cmd/goal-wasm) compiles the real
  `internal/pipeline` package to `goal.wasm` and exposes a single
  `goalTranspile(source)` function. There is **no backend and no CDN** — on the
  first **Run**, the page loads Go's `wasm_exec.js` glue and instantiates
  `goal.wasm`, and every subsequent transpile is a synchronous in-page call.
- **[`cmd/build-playground`](../cmd/build-playground)** parses
  `docs/by-example.md` and emits **`features.json`** — the per-feature manifest
  (title, prose, the `.goal` source, and the seed Go output). Because that tool
  imports the real pipeline, it **re-transpiles every example at generate time and
  asserts the result matches the Go block locked in the doc**, so the manifest and
  the doc cannot drift from the transpiler.
- Doctest features (e.g. the `Doctests` example) additionally surface the generated
  `_test.go` sidecar; the result pane offers a tab to switch between the transpiled
  Go and the doctest output.

A live Run reproduces the seed output byte-for-byte — the same `internal/pipeline`
code produced both (the seed via the host build, the live result via WASM).

## Build & serve

```bash
./scripts/build-site.sh                 # builds goal.wasm + wasm_exec.js + features.json
(cd site && python3 -m http.server 8000) # serve at http://localhost:8000
node playground-e2e/run.mjs              # headless gate: wasm in Node vs. every seed
```

Any static file server works, as long as it serves `.wasm` as `application/wasm`
(GitHub Pages does; the app falls back to a buffered fetch for servers that don't).

## Tracked vs generated

Tracked in git: the source assets — `index.html`, `app.js`, `styles.css`, this
README.

Generated (gitignored, rebuild with `scripts/build-site.sh`): `goal.wasm`,
`wasm_exec.js` (copied from the active Go toolchain so it always matches the
compiler), and `features.json`.

## Deployment

[`.github/workflows/pages.yml`](../.github/workflows/pages.yml) builds the site on
every push to `main` that touches the transpiler, the doc, or the site, gates it on
the Node smoke test, and deploys the `site/` directory to GitHub Pages. Enable it
once under **Settings → Pages → Build and deployment → Source: GitHub Actions**.
