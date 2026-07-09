// goal Playground — a dependency-free SPA built from two inputs:
//
//   features.json        generated from docs/by-example.md by cmd/build-playground.
//                        The source of truth for content (source, transpiled Go,
//                        diagnostics, prose) — verified against the live transpiler
//                        at build time, so it cannot drift from the language.
//   playground-map.json  the presentation layer for this redesign: category
//                        grouping, accent colors, display numbers, and the
//                        valid<->error pairing. Holds ONLY presentation.
//
// The map is joined to features.json by anchor. Any disagreement (an anchor the
// map references that features.json lacks, or a feature the map never covers)
// raises a visible "out of sync" banner — the browser-side twin of the Go drift
// test in cmd/build-playground/map_test.go.
//
// The real transpiler (goal.wasm — the same pipeline package the CLI uses) is
// preloaded in the background on boot, so the first Run has no load pause. Output
// is auto-transpiled live whenever you open a feature, flip valid/error, or edit.

const state = {
  manifest: null,
  map: null,
  cards: [], // primaries in map order, each joined to its feature + error features
  byId: new Map(), // anchor -> card
  drift: [],

  view: "about", // "feature" | "playground" | "overview" | "about"
  currentId: null, // active idea (primary) anchor
  mode: "valid", // "valid" | "error"
  exampleIdx: 0, // which example, when an idea has more than one
  editorValue: "",
  scratch: "", // the free-playground ("Playground" tab) source
  collapsed: false, // sidebar rail mode

  runSeq: 0, // guards against a stale live-run overwriting a newer one
  scratchSeq: 0,
  editTimer: null,
  scratchTimer: null,
};

const GH = "https://github.com/2bit-software/goal";
const GH_RELEASES = "https://github.com/2bit-software/goal/releases/latest";

// Seed for the free-playground scratchpad: a small program touching several
// features at once, so the pane isn't empty on first open.
const DEFAULT_SCRATCH = `package main

enum Shape {
    Circle { r: float64 }
    Rect   { w: float64, h: float64 }
}

func area(s Shape) float64 {
    return match s {
        Shape.Circle(c) => 3.14159 * c.r * c.r
        Shape.Rect(r)   => r.w * r.h
    }
}
`;

// --------------------------------------------------------------------------- //
// boot
// --------------------------------------------------------------------------- //

async function boot() {
  let manifest, map;
  try {
    [manifest, map] = await Promise.all([
      fetchJSON("features.json"),
      fetchJSON("playground-map.json"),
    ]);
  } catch (err) {
    document.getElementById("app").innerHTML =
      `<p class="boot">Could not load the playground data (${escapeHtml(String(err))}). ` +
      `Build the site with <code>task site</code>, then serve this directory.</p>`;
    return;
  }
  state.manifest = manifest;
  state.map = map;
  join(manifest, map);
  restore();
  preloadWasm(); // fire-and-forget: ready before the first Run
  render();
  window.addEventListener("keydown", onKey);
}

// join builds the card list (map order) and records any drift between the map and
// the generated manifest, so presentation and content can never silently diverge.
function join(manifest, map) {
  const byAnchor = new Map();
  for (const cat of manifest.categories) {
    for (const feat of cat.features) byAnchor.set(feat.anchor, feat);
  }
  const covered = new Set();
  const drift = [];
  for (const cat of map.categories) {
    for (const mf of cat.features) {
      const primary = byAnchor.get(mf.anchor);
      if (!primary) {
        drift.push(`map references "${mf.anchor}", which features.json does not contain`);
        continue;
      }
      covered.add(mf.anchor);
      // Each example links a valid anchor to an optional error anchor. Resolve both
      // to features.json entries; a missing anchor is drift.
      const examples = [];
      for (const me of mf.examples || []) {
        const valid = byAnchor.get(me.valid);
        if (!valid) {
          drift.push(`map references valid "${me.valid}", which features.json does not contain`);
          continue;
        }
        covered.add(me.valid);
        let error = null;
        if (me.error) {
          error = byAnchor.get(me.error);
          if (!error) {
            drift.push(`map references error "${me.error}", which features.json does not contain`);
          } else {
            covered.add(me.error);
          }
        }
        examples.push({ label: me.label, valid, error });
      }
      if (!examples.length) drift.push(`idea "${mf.anchor}" has no examples`);
      const card = {
        anchor: mf.anchor,
        num: mf.num,
        accent: cat.accent,
        catName: cat.name,
        primary,
        examples,
      };
      state.cards.push(card);
      state.byId.set(mf.anchor, card);
    }
  }
  for (const [anchor] of byAnchor) {
    if (!covered.has(anchor)) drift.push(`features.json has "${anchor}", which the map never places`);
  }
  state.drift = drift;
  if (state.cards.length && !state.byId.has(state.currentId)) {
    state.currentId = state.cards[0].anchor;
  }
}

// --------------------------------------------------------------------------- //
// persistence
// --------------------------------------------------------------------------- //

function restore() {
  let saved = {};
  try {
    saved = JSON.parse(localStorage.getItem("goal_pg") || "{}");
  } catch {
    saved = {};
  }
  if (saved.id && state.byId.has(saved.id)) state.currentId = saved.id;
  else if (state.cards.length) state.currentId = state.cards[0].anchor;
  if (saved.mode === "valid" || saved.mode === "error") state.mode = saved.mode;
  if (["feature", "playground", "overview", "about"].includes(saved.view)) state.view = saved.view;
  if (Number.isInteger(saved.ex)) state.exampleIdx = saved.ex;
  state.scratch = typeof saved.scratch === "string" ? saved.scratch : DEFAULT_SCRATCH;
  state.collapsed = saved.collapsed === true;
  normalizeMode();
}

function persist() {
  try {
    localStorage.setItem(
      "goal_pg",
      JSON.stringify({
        id: state.currentId,
        mode: state.mode,
        view: state.view,
        ex: state.exampleIdx,
        scratch: state.scratch,
        collapsed: state.collapsed,
      }),
    );
  } catch {
    /* private mode / disabled storage — non-fatal */
  }
}

// normalizeMode clamps the example index and drops back to "valid" when the current
// example has no error variant.
function normalizeMode() {
  const card = state.byId.get(state.currentId);
  const count = card ? card.examples.length : 0;
  if (state.exampleIdx >= count || state.exampleIdx < 0) state.exampleIdx = 0;
  if (!currentExample() || !currentExample().error) state.mode = "valid";
}

// currentExample is the selected example of the active idea.
function currentExample() {
  const card = state.byId.get(state.currentId);
  if (!card || !card.examples.length) return null;
  return card.examples[Math.min(state.exampleIdx, card.examples.length - 1)];
}

// --------------------------------------------------------------------------- //
// state transitions
// --------------------------------------------------------------------------- //

function openFeature(id) {
  state.currentId = id;
  state.view = "feature";
  state.exampleIdx = 0;
  normalizeMode();
  seedEditor();
  persist();
  render();
}

function setView(view) {
  state.view = view;
  persist();
  render();
}

function toggleCollapse() {
  state.collapsed = !state.collapsed;
  persist();
  render();
}

function resetScratch() {
  state.scratch = DEFAULT_SCRATCH;
  persist();
  render();
}

function setMode(mode) {
  const ex = currentExample();
  if (mode === "error" && (!ex || !ex.error)) return;
  state.mode = mode;
  seedEditor();
  persist();
  render();
}

function setExample(i) {
  state.exampleIdx = i;
  normalizeMode();
  seedEditor();
  persist();
  render();
}

function move(delta) {
  const i = state.cards.findIndex((c) => c.anchor === state.currentId);
  if (i < 0) return;
  const j = (i + delta + state.cards.length) % state.cards.length;
  openFeature(state.cards[j].anchor);
}

// activeFeature returns the feature whose source the editor is showing: the current
// example's valid feature in valid mode, its error feature in error mode.
function activeFeature() {
  const ex = currentExample();
  if (!ex) return null;
  return state.mode === "error" && ex.error ? ex.error : ex.valid;
}

function seedEditor() {
  const feat = activeFeature();
  state.editorValue = feat ? feat.source.replace(/\n+$/, "") : "";
}

function reset() {
  seedEditor();
  render();
}

function onKey(e) {
  if (e.target && /TEXTAREA|INPUT/.test(e.target.tagName)) return;
  // ↑↓/j/k/e are feature-navigation keys; leave the scratchpad and doc views alone.
  if (state.view !== "feature") return;
  if (e.key === "ArrowDown" || e.key === "j") {
    e.preventDefault();
    move(1);
  } else if (e.key === "ArrowUp" || e.key === "k") {
    e.preventDefault();
    move(-1);
  } else if (e.key === "e") {
    setMode(state.mode === "valid" ? "error" : "valid");
  }
}

// --------------------------------------------------------------------------- //
// render
// --------------------------------------------------------------------------- //

function render() {
  if (!state.editorValue) seedEditor();
  const app = document.getElementById("app");
  const card = state.byId.get(state.currentId);
  app.style.setProperty("--accent", card ? card.accent : "#4dd3d8");
  app.dataset.collapsed = state.collapsed ? "true" : "false";

  app.innerHTML = "";
  if (state.drift.length) app.appendChild(driftBanner());
  app.appendChild(renderSidebar());
  app.appendChild(renderMain());

  refreshWasmStatus();
  if (state.view === "feature") {
    wireEditor();
    runCurrent(); // replace the seed output with a live transpile
  } else if (state.view === "playground") {
    wireScratch();
    runScratch();
  }
}

function driftBanner() {
  const d = el("div", "drift");
  d.innerHTML =
    `<b>Playground map out of sync.</b> ` +
    escapeHtml(state.drift.join("; ")) +
    `. Regenerate <code>features.json</code> (<code>task site</code>) or update <code>site/playground-map.json</code>.`;
  return d;
}

// --------------------------------------------------------------------------- //
// sidebar
// --------------------------------------------------------------------------- //

function renderSidebar() {
  const aside = el("aside", "sidebar");

  // Collapsed rail: brand dot + an expand button. Shown only in collapsed mode
  // (CSS); hovering the rail slides the full sidebar back in as an overlay.
  const rail = el("div", "side-rail");
  rail.appendChild(el("span", "brand-dot"));
  const expand = el("button", "iconbtn", "»");
  expand.title = "Expand sidebar";
  expand.addEventListener("click", () => toggleCollapse());
  rail.appendChild(expand);
  aside.appendChild(rail);

  const full = el("div", "side-full");

  const head = el("div", "side-head");
  const brand = el("div", "brand");
  brand.appendChild(el("span", "brand-dot"));
  brand.appendChild(el("span", "brand-name", "goal"));
  brand.appendChild(el("span", "brand-chip", `v0 · ${state.cards.length} features`));
  brand.appendChild(el("span", "brand-spacer"));
  const collapse = el("button", "collapse-btn", "«");
  collapse.title = "Collapse sidebar";
  collapse.addEventListener("click", () => toggleCollapse());
  brand.appendChild(collapse);
  head.appendChild(brand);
  head.appendChild(
    el(
      "div",
      "brand-sub",
      "A correctness-oriented Go dialect. One example per feature, valid and broken.",
    ),
  );
  full.appendChild(head);

  const tabs = el("div", "tabs");
  for (const [view, label] of [
    ["about", "About"],
    ["overview", "Overview"],
    ["playground", "Playground"],
  ]) {
    const t = el("button", "tab" + (state.view === view ? " active" : ""), label);
    t.addEventListener("click", () => setView(view));
    tabs.appendChild(t);
  }
  full.appendChild(tabs);

  const nav = el("nav", "nav");
  for (const cat of state.map.categories) {
    const cards = cat.features.map((f) => state.byId.get(f.anchor)).filter(Boolean);
    if (!cards.length) continue;
    const group = el("div", "nav-group");
    group.appendChild(el("div", "nav-cat", cat.name));
    for (const card of cards) {
      const active = card.anchor === state.currentId && state.view === "feature";
      const item = el("button", "nav-item" + (active ? " active" : ""));
      item.style.setProperty("--accent", card.accent);
      item.appendChild(el("span", "nav-num", card.num));
      const label = el("span", "nav-label");
      label.innerHTML = card.primary.titleHtml || escapeHtml(card.primary.title);
      item.appendChild(label);
      item.appendChild(el("span", "nav-chev", "›"));
      item.addEventListener("click", () => openFeature(card.anchor));
      group.appendChild(item);
    }
    nav.appendChild(group);
  }
  full.appendChild(nav);

  const foot = el("div", "side-foot");
  const src = el("a");
  src.href = GH;
  src.target = "_blank";
  src.rel = "noopener";
  src.appendChild(el("span", null, "source ↗"));
  foot.appendChild(src);
  const status = el("span", "wasm-status");
  status.id = "wasm-status";
  status.appendChild(el("span", "dot"));
  status.appendChild(el("span", "to-go", "transpiles → Go"));
  foot.appendChild(status);
  full.appendChild(foot);

  aside.appendChild(full);
  return aside;
}

// --------------------------------------------------------------------------- //
// main
// --------------------------------------------------------------------------- //

function renderMain() {
  const main = el("main", "main");
  main.appendChild(renderTopbar());
  // The scratchpad ("Playground") manages its own full-height layout; the other
  // views share the scrolling, max-width content column.
  if (state.view === "playground") {
    main.appendChild(renderScratchpad());
    return main;
  }
  const scroll = el("div", "scroll");
  const content = el("div", "content");
  if (state.view === "feature") content.appendChild(renderFeature());
  else if (state.view === "overview") content.appendChild(renderOverview());
  else content.appendChild(renderAbout());
  scroll.appendChild(content);
  main.appendChild(scroll);
  return main;
}

function renderTopbar() {
  const bar = el("div", "topbar");
  const crumbs = el("div", "crumbs");
  const card = state.byId.get(state.currentId);
  const [a, b] =
    state.view === "feature"
      ? ["feature", card ? card.primary.title : ""]
      : state.view === "playground"
        ? ["playground", "scratchpad"]
        : state.view === "overview"
          ? ["overview", "all features"]
          : ["about", "the language goals"];
  crumbs.appendChild(el("span", null, a));
  crumbs.appendChild(el("span", "sep", "/"));
  const cb = el("span", "crumb-b");
  cb.innerHTML = state.view === "feature" && card ? card.primary.titleHtml || escapeHtml(b) : escapeHtml(b);
  crumbs.appendChild(cb);
  bar.appendChild(crumbs);

  const right = el("div", "topbar-right");
  if (state.view === "feature" && card) {
    const idx = state.cards.findIndex((c) => c.anchor === state.currentId);
    right.appendChild(el("span", "count", `${idx + 1} / ${state.cards.length}`));
    const btns = el("div", "navbtns");
    const up = el("button", "iconbtn", "↑");
    up.title = "Previous (↑)";
    up.addEventListener("click", () => move(-1));
    const down = el("button", "iconbtn", "↓");
    down.title = "Next (↓)";
    down.addEventListener("click", () => move(1));
    btns.appendChild(up);
    btns.appendChild(down);
    right.appendChild(btns);
    right.appendChild(el("span", "divider"));
  }
  const gh = el("a", "gh-link", "GitHub ↗");
  gh.href = GH;
  gh.target = "_blank";
  gh.rel = "noopener";
  right.appendChild(gh);
  const dl = el("a", "dl-btn", "↓ Download");
  dl.href = GH_RELEASES;
  dl.target = "_blank";
  dl.rel = "noopener";
  dl.title = "Download the latest release";
  right.appendChild(dl);
  bar.appendChild(right);
  return bar;
}

// --------------------------------------------------------------------------- //
// feature view (a single documented feature, its examples, editor + output)
// --------------------------------------------------------------------------- //

function renderFeature() {
  const card = state.byId.get(state.currentId);
  const wrap = el("div");
  if (!card) return wrap;

  // header
  const head = el("div", "feat-head");
  head.appendChild(el("span", "feat-num", card.num));
  const htext = el("div");
  htext.style.flex = "1";
  htext.style.minWidth = "0";
  const h1 = el("h1", "feat-title");
  h1.innerHTML = card.primary.titleHtml || escapeHtml(card.primary.title);
  htext.appendChild(h1);
  const blurb = el("div", "feat-blurb");
  blurb.innerHTML = card.primary.descriptionHtml || "";
  highlightStaticBlocks(blurb);
  htext.appendChild(blurb);
  head.appendChild(htext);
  wrap.appendChild(head);

  // lowers-to
  if (card.primary.loweringHtml) {
    const low = el("div", "lowers");
    low.appendChild(el("span", "tag", "lowers to"));
    low.appendChild(el("span", "arrow", "→"));
    const body = el("div");
    body.innerHTML = card.primary.loweringHtml;
    highlightStaticBlocks(body);
    low.appendChild(body);
    wrap.appendChild(low);
  }

  if (card.examples.length > 1) wrap.appendChild(renderExampleTabs(card));
  wrap.appendChild(renderToolbar(card));
  wrap.appendChild(renderGrid(card));
  wrap.appendChild(renderStatus(card));

  const hint = el("div", "hintline");
  hint.innerHTML =
    "↑ ↓ to move between features · <code>e</code> to toggle valid/error · output is transpiled live by goal.wasm.";
  wrap.appendChild(hint);
  return wrap;
}

function renderExampleTabs(card) {
  const row = el("div", "ex-tabs");
  card.examples.forEach((ex, i) => {
    const t = el("button", "ex-tab" + (i === state.exampleIdx ? " active" : ""), ex.label || `Example ${i + 1}`);
    t.addEventListener("click", () => setExample(i));
    row.appendChild(t);
  });
  return row;
}

function renderToolbar() {
  const feat = activeFeature();
  const ex = currentExample();
  const bar = el("div", "toolbar");
  const fn = el("div", "filename");
  fn.appendChild(el("span", "sq"));
  fn.appendChild(el("span", "name", feat.sourceName || "source.goal"));
  bar.appendChild(fn);

  const right = el("div", "toolbar-right");
  right.appendChild(el("span", "lbl", "show as"));
  const toggle = el("div", "toggle");
  const valid = el("button", state.mode === "valid" ? "on-valid" : "", "✓ Valid");
  valid.addEventListener("click", () => setMode("valid"));
  const error = el("button", state.mode === "error" ? "on-error" : "", "⚠ Error");
  if (!ex || !ex.error) error.disabled = true;
  error.addEventListener("click", () => setMode("error"));
  toggle.appendChild(valid);
  toggle.appendChild(error);
  right.appendChild(toggle);
  bar.appendChild(right);
  return bar;
}

function renderGrid(card) {
  const grid = el("div", "grid");
  grid.appendChild(renderEditorPane());
  // The output is wrapped so it can hover-expand over the page for wide Go output
  // without disturbing the two-column layout (see .out-wrap in styles.css).
  const wrap = el("div", "out-wrap");
  wrap.appendChild(renderOutputPane(card));
  grid.appendChild(wrap);
  return grid;
}

function renderEditorPane() {
  const pane = el("section", "pane");
  const head = el("div", "pane-head");
  head.appendChild(el("span", "cap", "editor · .goal"));
  const right = el("div", "right");
  const run = el("button", "runbtn", "Run ▶");
  run.id = "run-btn";
  run.addEventListener("click", () => runCurrent(true));
  const rst = el("button", "resetbtn", "reset");
  rst.title = "Reset to example";
  rst.addEventListener("click", () => reset());
  right.appendChild(rst);
  right.appendChild(run);
  head.appendChild(right);
  pane.appendChild(head);

  const row = el("div", "editor-row");
  const gutter = el("div", "gutter");
  gutter.id = "editor-gutter";
  // A transparent textarea over a Prism-free highlighted <pre>: the pre shows the
  // colored .goal source, the textarea provides editing + the caret. Both share the
  // exact font/metrics so the layers line up. (See .editor-wrap in styles.css.)
  const wrap = el("div", "editor-wrap");
  const hl = el("pre", "editor-hl");
  hl.id = "editor-hl";
  hl.setAttribute("aria-hidden", "true");
  const ta = el("textarea", "editor");
  ta.id = "editor";
  ta.spellcheck = false;
  ta.value = state.editorValue;
  wrap.appendChild(hl);
  wrap.appendChild(ta);
  row.appendChild(gutter);
  row.appendChild(wrap);
  pane.appendChild(row);
  return pane;
}

function renderOutputPane(card) {
  const isError = state.mode === "error";
  const pane = el("section", "pane" + (isError ? " bad" : ""));
  const head = el("div", "pane-head");
  const feat = activeFeature();
  const cap = el("span", "cap" + (isError ? " bad" : ""));
  cap.textContent = outputCaption(feat, isError);
  head.appendChild(cap);
  const badge = el("span", "badge" + (isError ? " bad" : ""), isError ? "exit 1" : "preview");
  head.appendChild(badge);
  pane.appendChild(head);

  const body = el("div", "out-body" + (isError ? " bad" : ""));
  const gutter = el("div", "gutter");
  gutter.id = "output-gutter";
  const pre = el("pre", "output");
  pre.id = "output";
  body.appendChild(gutter);
  body.appendChild(pre);
  pane.appendChild(body);
  // Seed immediately from the verified locked output; a live run replaces it.
  paintOutput(seedResult(feat, isError));
  return pane;
}

function outputCaption(feat, isError) {
  if (!isError) return feat.outputKind === "test" ? "generated · _test.go" : "transpiled · .go";
  return feat.outputKind === "doctest-failure" ? "go test · failure" : "goal check · diagnostics";
}

// seedResult is the verified, locked output for a feature — shown instantly so the
// pane is never blank while goal.wasm transpiles the (possibly edited) source live.
function seedResult(feat, isError) {
  if (isError) {
    return feat.outputKind === "doctest-failure"
      ? { kind: "testfail", text: feat.expected }
      : { kind: "diag", text: feat.expected };
  }
  return { kind: feat.outputKind === "test" ? "test" : "go", text: feat.expected };
}

function renderStatus(card) {
  const isError = state.mode === "error";
  const feat = activeFeature();
  const bar = el("div", "statusbar" + (isError ? " bad" : ""));
  bar.id = "statusbar";
  bar.appendChild(el("span", "dot"));
  bar.appendChild(el("span", "text", isError ? "goal check: 1 error" : "goal check: ok"));
  bar.appendChild(
    el(
      "span",
      "sub",
      isError ? "located, machine-checkable feedback" : "compiles clean · transpiles to idiomatic Go",
    ),
  );
  const cmd = isError ? `goal check ${feat.sourceName}` : `goalc ${feat.sourceName}`;
  bar.appendChild(el("span", "cmd", "$ " + cmd));
  return bar;
}

// --------------------------------------------------------------------------- //
// scratchpad view (the "Playground" tab — a free editor over the live transpiler)
// --------------------------------------------------------------------------- //

function renderScratchpad() {
  const wrap = el("div", "scratch");

  const head = el("div", "scratch-head");
  const heading = el("div");
  heading.appendChild(el("h1", "scratch-title", "Scratchpad"));
  heading.appendChild(
    el("p", "scratch-sub", "Write any goal you like and watch it lower to Go. Ambiguity welcome."),
  );
  head.appendChild(heading);

  const controls = el("div", "scratch-controls");
  const status = el("span", "scratch-status");
  status.id = "scratch-status";
  status.appendChild(el("span", "dot"));
  status.appendChild(el("span", "txt", "transpiler loading…"));
  controls.appendChild(status);
  const rst = el("button", "resetbtn", "reset");
  rst.addEventListener("click", () => resetScratch());
  controls.appendChild(rst);
  head.appendChild(controls);
  wrap.appendChild(head);

  const grid = el("div", "scratch-grid");

  // editor pane (fills height, internal scroll)
  const edPane = el("section", "pane scratch-pane");
  const edHead = el("div", "pane-head");
  edHead.appendChild(el("span", "cap", "editor · scratch.goal"));
  edHead.appendChild(el("span", "hint", "editable"));
  edPane.appendChild(edHead);
  const edRow = el("div", "editor-row fill");
  const edGutter = el("div", "gutter");
  edGutter.id = "scratch-gutter";
  const edWrap = el("div", "editor-wrap fill");
  const hl = el("pre", "editor-hl");
  hl.id = "scratch-hl";
  hl.setAttribute("aria-hidden", "true");
  const ta = el("textarea", "editor");
  ta.id = "scratch";
  ta.spellcheck = false;
  ta.value = state.scratch;
  edWrap.appendChild(hl);
  edWrap.appendChild(ta);
  edRow.appendChild(edGutter);
  edRow.appendChild(edWrap);
  edPane.appendChild(edRow);
  grid.appendChild(edPane);

  // output pane (fills height, internal scroll)
  const outPane = el("section", "pane scratch-pane");
  const outHead = el("div", "pane-head");
  outHead.appendChild(el("span", "cap", "transpiled · .go"));
  outHead.appendChild(el("span", "badge", "preview"));
  outPane.appendChild(outHead);
  const outBody = el("div", "out-body fill");
  const outGutter = el("div", "gutter");
  outGutter.id = "scratch-out-gutter";
  const pre = el("pre", "output");
  pre.id = "scratch-output";
  outBody.appendChild(outGutter);
  outBody.appendChild(pre);
  outPane.appendChild(outBody);
  grid.appendChild(outPane);

  wrap.appendChild(grid);
  return wrap;
}

function wireScratch() {
  const ta = document.getElementById("scratch");
  if (!ta) return;
  const hl = document.getElementById("scratch-hl");
  const paint = () => {
    if (hl) hl.innerHTML = highlight(ta.value) + "\n";
    syncGutter("scratch-gutter", ta.value);
  };
  const syncScroll = () => {
    if (hl) {
      hl.scrollTop = ta.scrollTop;
      hl.scrollLeft = ta.scrollLeft;
    }
    const g = document.getElementById("scratch-gutter");
    if (g) g.scrollTop = ta.scrollTop;
  };
  paint();
  ta.addEventListener("input", () => {
    state.scratch = ta.value;
    paint();
    persist();
    clearTimeout(state.scratchTimer);
    state.scratchTimer = setTimeout(() => runScratch(), 300);
  });
  ta.addEventListener("scroll", syncScroll);
  ta.addEventListener("keydown", (e) => {
    if (e.key === "Tab") {
      e.preventDefault();
      const s = ta.selectionStart,
        en = ta.selectionEnd;
      ta.value = ta.value.slice(0, s) + "    " + ta.value.slice(en);
      ta.selectionStart = ta.selectionEnd = s + 4;
      state.scratch = ta.value;
      paint();
    }
  });
}

// runScratch transpiles the scratchpad source live and paints the output pane,
// coloring Go on success and formatting a diagnostic on rejection.
async function runScratch() {
  if (state.view !== "playground") return;
  const seq = ++state.scratchSeq;
  setScratchStatus("loading");
  try {
    const transpile = await getTranspiler();
    const result = transpile(withTrailingNewline(state.scratch), "scratch.goal");
    if (seq !== state.scratchSeq) return;
    if (result.error) {
      paintScratch({ kind: "diag", text: result.error }, true);
      setScratchStatus("error");
    } else {
      paintScratch({ kind: "go", text: result.go || "" }, false);
      setScratchStatus("ok");
    }
  } catch (err) {
    if (seq === state.scratchSeq) {
      paintScratch({ kind: "diag", text: `Runtime error:\n${String(err)}` }, true);
      setScratchStatus("error");
    }
  }
}

function paintScratch(result, bad) {
  const pre = document.getElementById("scratch-output");
  if (!pre) return;
  const body = pre.closest(".out-body");
  if (body) body.classList.toggle("bad", bad);
  const pane = pre.closest(".pane");
  if (pane) pane.classList.toggle("bad", bad);
  const text = (result.text || "").replace(/\n$/, "");
  pre.classList.remove("muted");
  if (!text) {
    pre.classList.add("muted");
    pre.textContent = "(no output)";
    syncGutter("scratch-out-gutter", "");
    return;
  }
  pre.innerHTML = result.kind === "diag" ? formatDiag(text) : highlight(text);
  syncGutter("scratch-out-gutter", text);
}

function setScratchStatus(kind) {
  const node = document.getElementById("scratch-status");
  if (!node) return;
  const map = {
    loading: ["loading", "transpiler loading…"],
    ok: ["ok", "transpiler connected"],
    error: ["bad", "goal check: rejected"],
  };
  const [cls, txt] = map[kind] || map.loading;
  node.dataset.state = cls;
  node.querySelector(".txt").textContent = txt;
}

// --------------------------------------------------------------------------- //
// editor wiring (plain textarea + synced line-number gutter)
// --------------------------------------------------------------------------- //

function wireEditor() {
  const ta = document.getElementById("editor");
  if (!ta) return;
  const hl = document.getElementById("editor-hl");
  // paint syncs the three layers to the textarea's current value: the highlight
  // <pre>, the line-number gutter, and the textarea's row count (so it grows with
  // content and the layers stay the same height).
  const paint = () => {
    if (hl) hl.innerHTML = highlight(ta.value) + "\n";
    syncGutter("editor-gutter", ta.value);
    ta.rows = Math.max(ta.value.split("\n").length, 8);
  };
  const syncScroll = () => {
    if (hl) {
      hl.scrollTop = ta.scrollTop;
      hl.scrollLeft = ta.scrollLeft;
    }
    const g = document.getElementById("editor-gutter");
    if (g) g.scrollTop = ta.scrollTop;
  };
  paint();
  ta.addEventListener("input", () => {
    state.editorValue = ta.value;
    paint();
    clearTimeout(state.editTimer);
    state.editTimer = setTimeout(() => runCurrent(), 350);
  });
  ta.addEventListener("scroll", syncScroll);
  ta.addEventListener("keydown", (e) => {
    if (e.key === "Tab") {
      e.preventDefault();
      const s = ta.selectionStart,
        en = ta.selectionEnd;
      ta.value = ta.value.slice(0, s) + "    " + ta.value.slice(en);
      ta.selectionStart = ta.selectionEnd = s + 4;
      state.editorValue = ta.value;
      paint();
    } else if ((e.metaKey || e.ctrlKey) && e.key === "Enter") {
      e.preventDefault();
      runCurrent(true);
    }
  });
}

function syncGutter(id, text) {
  const g = document.getElementById(id);
  if (!g) return;
  const n = Math.max(text.split("\n").length, 1);
  g.textContent = Array.from({ length: n }, (_, i) => i + 1).join("\n");
}

// --------------------------------------------------------------------------- //
// live transpile
// --------------------------------------------------------------------------- //

async function runCurrent(explicit) {
  if (state.view !== "feature") return;
  const feat = activeFeature();
  if (!feat) return;
  // A doctest-failure is proven by running `go test`, which the browser can't do —
  // its verified failure text is the locked seed, so there is nothing to run live.
  if (state.mode === "error" && feat.outputKind === "doctest-failure") return;

  const seq = ++state.runSeq;
  const btn = document.getElementById("run-btn");
  if (explicit && btn) {
    btn.disabled = true;
    btn.textContent = "Running…";
  }
  try {
    const transpile = await getTranspiler();
    const result = transpile(withTrailingNewline(state.editorValue), feat.sourceName || "source.goal");
    if (seq !== state.runSeq) return; // superseded by a newer run
    applyLiveResult(feat, result);
  } catch (err) {
    if (seq === state.runSeq) paintOutput({ kind: "diag", text: `Runtime error:\n${String(err)}` }, true);
  } finally {
    if (explicit && btn) {
      btn.disabled = false;
      btn.textContent = "Run ▶";
    }
  }
}

// applyLiveResult renders the transpiler's real output and reconciles the status
// bar with what actually happened (a program the checker rejects flips it to error).
function applyLiveResult(feat, result) {
  const isError = state.mode === "error";
  if (result.error) {
    paintOutput({ kind: "diag", text: result.error }, true);
    setStatus(true, feat);
    return;
  }
  const body = feat.outputKind === "test" && !isError ? result.test : result.go;
  paintOutput({ kind: feat.outputKind === "test" && !isError ? "test" : "go", text: body || "" }, false);
  setStatus(false, feat);
}

// paintOutput fills the output pane + its gutter with either highlighted Go, a
// formatted diagnostic, or a raw test-failure block.
function paintOutput(result, forceBad) {
  const pre = document.getElementById("output");
  if (!pre) return;
  const body = document.querySelector(".out-body");
  const bad = forceBad || result.kind === "diag" || result.kind === "testfail";
  pre.classList.remove("muted");
  if (body) body.classList.toggle("bad", bad);
  const pane = pre.closest(".pane");
  if (pane) pane.classList.toggle("bad", bad);

  const text = (result.text || "").replace(/\n$/, "");
  if (!text) {
    pre.classList.add("muted");
    pre.textContent = "(no output)";
    syncGutter("output-gutter", "");
    document.getElementById("output-gutter").textContent = "";
    return;
  }
  if (result.kind === "go" || result.kind === "test") pre.innerHTML = highlight(text);
  else if (result.kind === "diag") pre.innerHTML = formatDiag(text);
  else pre.textContent = text;
  syncGutter("output-gutter", text);
}

function setStatus(isError, feat) {
  const bar = document.getElementById("statusbar");
  if (!bar) return;
  bar.classList.toggle("bad", isError);
  bar.querySelector(".text").textContent = isError ? "goal check: 1 error" : "goal check: ok";
  bar.querySelector(".sub").textContent = isError
    ? "located, machine-checkable feedback"
    : "compiles clean · transpiles to idiomatic Go";
  bar.querySelector(".cmd").textContent =
    "$ " + (isError ? `goal check ${feat.sourceName}` : `goalc ${feat.sourceName}`);
}

// --------------------------------------------------------------------------- //
// overview view
// --------------------------------------------------------------------------- //

function renderOverview() {
  const wrap = el("div");
  const head = el("div", "ov-head");
  head.appendChild(el("h1", null, `All ${state.cards.length} features`));
  head.appendChild(
    el(
      "p",
      null,
      "Each transpiles to idiomatic Go; the guarantee behind it is enforced by the checker. Pick any row to open it in the playground.",
    ),
  );
  wrap.appendChild(head);

  for (const cat of state.map.categories) {
    const cards = cat.features.map((f) => state.byId.get(f.anchor)).filter(Boolean);
    if (!cards.length) continue;
    const group = el("div", "ov-group");
    group.style.setProperty("--accent", cat.accent);
    const gh = el("div", "ov-group-head");
    gh.appendChild(el("span", "sq"));
    gh.appendChild(el("span", "name", cat.name));
    gh.appendChild(el("span", "rule"));
    group.appendChild(gh);
    const rows = el("div", "ov-rows");
    for (const card of cards) {
      const row = el("button", "ov-row");
      row.style.setProperty("--accent", card.accent);
      row.appendChild(el("span", "num", card.num));
      const mid = el("span");
      const name = el("span", "rname");
      name.innerHTML = card.primary.titleHtml || escapeHtml(card.primary.title);
      mid.appendChild(name);
      mid.appendChild(el("span", "rguard", firstSentence(card.primary.descriptionHtml)));
      row.appendChild(mid);
      row.appendChild(el("code", "rlower", "→ " + firstSentence(card.primary.loweringHtml)));
      row.addEventListener("click", () => openFeature(card.anchor));
      rows.appendChild(row);
    }
    group.appendChild(rows);
    wrap.appendChild(group);
  }
  return wrap;
}

// --------------------------------------------------------------------------- //
// about view
// --------------------------------------------------------------------------- //

function renderAbout() {
  const wrap = el("div", "about");
  wrap.appendChild(el("div", "kicker", "About goal"));
  const h1 = el("h1");
  h1.innerHTML = `The Go <span class="em">Augmented</span> Language`;
  wrap.appendChild(h1);

  const intro = el("div", "intro");
  intro.innerHTML = state.manifest.introHtml || "<p>No overview available.</p>";
  highlightStaticBlocks(intro);
  wrap.appendChild(intro);

  const actions = el("div", "actions");
  const browse = el("button", "btn-primary", "Browse the features →");
  browse.addEventListener("click", () => setView("overview"));
  const play = el("button", "btn-ghost", "Open the playground");
  play.addEventListener("click", () => setView("playground"));
  const src = el("a", "btn-ghost", "source ↗");
  src.href = GH;
  src.target = "_blank";
  src.rel = "noopener";
  src.style.display = "inline-flex";
  src.style.alignItems = "center";
  actions.appendChild(browse);
  actions.appendChild(play);
  actions.appendChild(src);
  wrap.appendChild(actions);
  return wrap;
}

// --------------------------------------------------------------------------- //
// WebAssembly transpiler (preloaded on boot)
// --------------------------------------------------------------------------- //

let transpilerPromise = null;

function preloadWasm() {
  getTranspiler().catch(() => {
    /* surfaced via the runtime status dot; a Run retries */
  });
}

function getTranspiler() {
  if (transpilerPromise) return transpilerPromise;
  transpilerPromise = (async () => {
    setWasmState("loading");
    await loadScript("wasm_exec.js"); // defines globalThis.Go
    const go = new globalThis.Go();
    const instance = await instantiate(go);
    // main() blocks on select{}; it registers goalTranspile before parking, so we
    // must not await go.run (its promise never resolves).
    go.run(instance);
    if (typeof globalThis.goalTranspile !== "function") {
      throw new Error("goal.wasm did not register goalTranspile");
    }
    setWasmState("ready");
    return globalThis.goalTranspile;
  })().catch((err) => {
    setWasmState("error");
    transpilerPromise = null; // allow a retry on the next Run
    throw err;
  });
  return transpilerPromise;
}

async function instantiate(go) {
  if (WebAssembly.instantiateStreaming) {
    try {
      const { instance } = await WebAssembly.instantiateStreaming(
        fetch("goal.wasm", { cache: "no-cache" }),
        go.importObject,
      );
      return instance;
    } catch {
      // fall through to the buffered path (server didn't send application/wasm)
    }
  }
  const bytes = await (await fetch("goal.wasm", { cache: "no-cache" })).arrayBuffer();
  const { instance } = await WebAssembly.instantiate(bytes, go.importObject);
  return instance;
}

let wasmState = "idle";
function setWasmState(s) {
  wasmState = s;
  refreshWasmStatus();
}
function refreshWasmStatus() {
  const node = document.getElementById("wasm-status");
  if (!node) return;
  node.dataset.state = wasmState;
  const label = { idle: "wasm: idle", loading: "wasm: loading…", ready: "transpiles → Go", error: "wasm: failed" };
  const go = node.querySelector(".to-go");
  if (go) go.textContent = label[wasmState] || "transpiles → Go";
}

// --------------------------------------------------------------------------- //
// syntax highlight (goal + Go) — ported from the redesign, emitting CSS classes
// --------------------------------------------------------------------------- //

const KW = new Set([
  "package", "import", "func", "return", "type", "struct", "implements", "nozero",
  "enum", "match", "from", "derive", "assert", "if", "else", "for", "range", "var",
  "const", "true", "false", "nil", "switch", "case", "default",
]);
const TYP = new Set([
  "Result", "Option", "int", "float64", "float32", "string", "bool", "byte", "error",
  "uint", "any", "rune",
]);
const CTOR = new Set(["Ok", "Err", "Some", "None"]);

function highlight(src) {
  let i = 0;
  const n = src.length;
  const out = [];
  const push = (t, cls) => out.push(cls ? `<span class="${cls}">${escapeHtml(t)}</span>` : escapeHtml(t));
  while (i < n) {
    const rest = src.slice(i);
    let m;
    if ((m = /^\/\/\/.*/.exec(rest))) {
      push(m[0], "tok-doc");
    } else if ((m = /^\/\/.*/.exec(rest))) {
      push(m[0], "tok-com");
    } else if ((m = /^"(?:[^"\\]|\\.)*"/.exec(rest))) {
      push(m[0], "tok-str");
    } else if ((m = /^(=>|\?|\.\.\.|>>>)/.exec(rest))) {
      push(m[0], "tok-op");
    } else if ((m = /^[0-9]+(\.[0-9]+)?/.exec(rest))) {
      push(m[0], "tok-num");
    } else if ((m = /^[A-Za-z_][A-Za-z0-9_]*/.exec(rest))) {
      const w = m[0];
      let cls = null;
      if (KW.has(w)) cls = "tok-kw";
      else if (TYP.has(w)) cls = "tok-type";
      else if (CTOR.has(w)) cls = "tok-ctor";
      else if (/^[A-Z]/.test(w)) cls = "tok-type";
      else if (/^\s*\(/.test(src.slice(i + w.length))) cls = "tok-fn";
      push(w, cls);
    } else {
      push(src[i]);
      i += 1;
      continue;
    }
    i += m[0].length;
  }
  return out.join("");
}

function formatDiag(text) {
  return text
    .split("\n")
    .map((line) => {
      const m = line.match(/^(.+?:\d+:\d+:)\s+(error|warning):\s+(\[[^\]]+\])\s+(.*)$/);
      if (m) {
        return (
          `<span class="d-loc">${escapeHtml(m[1])}</span> ` +
          `<span class="d-sev">${escapeHtml(m[2])}:</span> ` +
          `<span class="d-code">${escapeHtml(m[3])}</span> ` +
          `<span class="d-msg">${escapeHtml(m[4])}</span>`
        );
      }
      if (/^(=== RUN)/.test(line)) return `<span class="d-run">${escapeHtml(line)}</span>`;
      if (/^\s*--- FAIL/.test(line) || /^FAIL/.test(line) || /^ok\b/.test(line) || /^exit status/.test(line) || /^\d+ (error|warning)/.test(line))
        return `<span class="d-fail">${escapeHtml(line)}</span>`;
      if (/^\s+/.test(line)) return `<span class="d-indent">${escapeHtml(line)}</span>`;
      return `<span class="d-msg">${escapeHtml(line)}</span>`;
    })
    .join("\n");
}

// highlightStaticBlocks colors the read-only .goal/.go snippets embedded in
// generated doc HTML (blurbs, lowerings, the about overview).
function highlightStaticBlocks(root) {
  for (const pre of root.querySelectorAll("pre.code[class*='lang-']")) {
    const code = pre.querySelector("code");
    const lang = (pre.className.match(/lang-(\w+)/) || [])[1];
    if (code && (lang === "goal" || lang === "go")) code.innerHTML = highlight(code.textContent);
  }
}

// --------------------------------------------------------------------------- //
// helpers
// --------------------------------------------------------------------------- //

async function fetchJSON(url) {
  const res = await fetch(url, { cache: "no-cache" });
  if (!res.ok) throw new Error(`${url}: HTTP ${res.status}`);
  return res.json();
}

function loadScript(src) {
  return new Promise((resolve, reject) => {
    if (document.querySelector(`script[src="${src}"]`)) return resolve();
    const s = document.createElement("script");
    s.src = src;
    s.onload = () => resolve();
    s.onerror = () => reject(new Error(`failed to load ${src}`));
    document.head.appendChild(s);
  });
}

function withTrailingNewline(s) {
  return s.replace(/\s*$/, "") + "\n";
}

// firstSentence strips tags from generated HTML, decodes entities, and returns its
// first sentence — a compact one-liner for the overview's guard/lowering columns.
function firstSentence(html) {
  const text = decodeEntities((html || "").replace(/<[^>]+>/g, ""))
    .replace(/\s+/g, " ")
    .trim();
  const dot = text.indexOf(". ");
  return dot > 0 ? text.slice(0, dot + 1) : text;
}

function decodeEntities(s) {
  return s
    .replace(/&lt;/g, "<")
    .replace(/&gt;/g, ">")
    .replace(/&quot;/g, '"')
    .replace(/&#39;/g, "'")
    .replace(/&amp;/g, "&");
}

function el(tag, className, text) {
  const node = document.createElement(tag);
  if (className) node.className = className;
  if (text != null) node.textContent = text;
  return node;
}

function escapeHtml(s) {
  return String(s).replace(/[&<>]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;" })[c]);
}

boot();
