// goal Playground — drives the SPA from the generated features.json and runs the
// real transpiler in-browser via WebAssembly (lazily, on the first Run).
//
// No build step: this is a plain ES module. The feature content (goal source +
// the seed transpiled Go) is generated from docs/by-example.md by
// cmd/build-playground, so the playground can never drift from the language —
// and goal.wasm is the same pipeline package the CLI uses.

const state = {
  manifest: null,
  byAnchor: new Map(),
  editor: null, // current feature's source <textarea>
  current: null,
  outputTab: "go", // which output the result pane is showing
};

// --------------------------------------------------------------------------- //
// boot
// --------------------------------------------------------------------------- //

async function boot() {
  let manifest;
  try {
    const res = await fetch("features.json", { cache: "no-cache" });
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    manifest = await res.json();
  } catch (err) {
    document.getElementById("content").innerHTML =
      `<div class="panel error"><h2>Could not load features.json</h2>` +
      `<p>Build the site (<code>scripts/build-site.sh</code>) to generate it, then serve this directory.</p>` +
      `<pre>${escapeHtml(String(err))}</pre></div>`;
    return;
  }
  state.manifest = manifest;
  for (const cat of manifest.categories) {
    for (const feat of cat.features) state.byAnchor.set(feat.anchor, feat);
  }
  renderSidebar(manifest);
  window.addEventListener("hashchange", route);
  route();
}

function route() {
  const anchor = location.hash.replace(/^#/, "");
  if (anchor === "" || anchor === "about") {
    renderIntro();
    highlightActive("about");
    return;
  }
  const feat = state.byAnchor.get(anchor);
  if (feat) {
    renderFeature(feat);
    highlightActive(feat.anchor);
  } else {
    renderIntro();
    highlightActive("about");
  }
}

// renderIntro shows the landing page — the README-style overview generated from
// docs/overview.md — as the full-width home view.
function renderIntro() {
  const main = document.getElementById("content");
  main.innerHTML = "";
  const intro = el("div", "intro");
  intro.innerHTML = state.manifest.introHtml || "<p>No overview available.</p>";
  highlightStaticBlocks(intro);
  main.appendChild(intro);
}

// --------------------------------------------------------------------------- //
// sidebar
// --------------------------------------------------------------------------- //

function renderSidebar(manifest) {
  const nav = document.getElementById("sidebar");
  nav.innerHTML = "";

  // Home link: the overview / "what is goal?" landing page.
  const home = el("div", "nav-group");
  const homeList = el("ul", "nav-list");
  const homeLi = el("li");
  const homeLink = el("a", "nav-link nav-home", "About goal");
  homeLink.href = "#about";
  homeLink.dataset.anchor = "about";
  homeLi.appendChild(homeLink);
  homeList.appendChild(homeLi);
  home.appendChild(homeList);
  nav.appendChild(home);

  for (const cat of manifest.categories) {
    const group = el("div", "nav-group");
    group.appendChild(el("h3", "nav-cat", cat.name));
    const list = el("ul", "nav-list");
    for (const feat of cat.features) {
      const li = el("li");
      const a = el("a", "nav-link");
      a.innerHTML = feat.titleHtml || escapeHtml(feat.title);
      a.href = `#${feat.anchor}`;
      a.dataset.anchor = feat.anchor;
      if (feat.outputKind === "test") a.appendChild(el("span", "badge", "doctest"));
      li.appendChild(a);
      list.appendChild(li);
    }
    group.appendChild(list);
    nav.appendChild(group);
  }
}

function highlightActive(anchor) {
  for (const a of document.querySelectorAll(".nav-link")) {
    a.classList.toggle("active", a.dataset.anchor === anchor);
  }
}

// --------------------------------------------------------------------------- //
// feature playground
// --------------------------------------------------------------------------- //

function renderFeature(feat) {
  state.current = feat;
  // An "error" feature has no Go/test pane — its output is a located compile error,
  // shown in the (single) result tab. Drive the pane off the "go" slot in that case.
  state.outputTab = feat.outputKind === "error" ? "go" : feat.outputKind;
  state.lastResult = null; // drop any previous feature's run output
  const main = document.getElementById("content");
  main.innerHTML = "";

  const head = el("div", "feature-head");
  const h2 = el("h2");
  h2.innerHTML = feat.titleHtml || escapeHtml(feat.title);
  head.appendChild(h2);
  const desc = el("div", "feature-desc");
  desc.innerHTML = feat.descriptionHtml || "";
  highlightStaticBlocks(desc);
  head.appendChild(desc);
  main.appendChild(head);

  const grid = el("div", "playground");

  // ---- source pane ----
  const srcPanel = el("div", "panel sources");
  const srcHead = el("div", "panel-head");
  srcHead.appendChild(el("span", "panel-title", feat.sourceName || "source.goal"));
  const actions = el("div", "actions");
  const resetBtn = el("button", "btn btn-ghost", "Reset");
  resetBtn.addEventListener("click", () => renderFeature(feat));
  const runBtn = el("button", "btn btn-run", "Run ▶");
  runBtn.addEventListener("click", () => runFeature(runBtn));
  actions.appendChild(resetBtn);
  actions.appendChild(runBtn);
  srcHead.appendChild(actions);
  srcPanel.appendChild(srcHead);

  const ed = makeEditor(feat.source.replace(/\n+$/, ""), runBtn);
  state.editor = ed;
  srcPanel.appendChild(ed);
  grid.appendChild(srcPanel);

  // ---- result pane ----
  const resPanel = el("div", "panel result");
  const resHead = el("div", "panel-head");
  resHead.appendChild(buildOutputTabs(feat));
  resPanel.appendChild(resHead);
  const out = el("pre", "output", "result-output");
  out.id = "result-output";
  renderOutput(out, {
    go: feat.outputKind === "go" ? feat.expected : "",
    test: feat.outputKind === "test" ? feat.expected : "",
    error: feat.outputKind === "error" ? feat.expected : "",
    seed: true,
  });
  resPanel.appendChild(out);
  grid.appendChild(resPanel);

  main.appendChild(grid);

  if (feat.loweringHtml) {
    const note = el("div", "lowering");
    note.innerHTML = feat.loweringHtml;
    highlightStaticBlocks(note);
    main.appendChild(note);
  }
}

// buildOutputTabs renders the result-pane header: a tab per available output
// (transpiled Go, generated _test.go). Tabs that have no content are disabled.
function buildOutputTabs(feat) {
  const wrap = el("div", "tabs");
  const last = state.lastResult;
  const tabFor = (kind, label) => {
    const t = el("button", "tab", label);
    t.dataset.kind = kind;
    if (kind === state.outputTab) t.classList.add("active");
    t.addEventListener("click", () => {
      state.outputTab = kind;
      for (const b of wrap.querySelectorAll(".tab")) b.classList.toggle("active", b.dataset.kind === kind);
      const data = state.lastResult || {
        go: feat.outputKind === "go" ? feat.expected : "",
        test: feat.outputKind === "test" ? feat.expected : "",
        error: feat.outputKind === "error" ? feat.expected : "",
        seed: !state.lastResult,
      };
      renderOutput(document.getElementById("result-output"), data);
    });
    return t;
  };
  wrap.appendChild(tabFor("go", feat.outputKind === "error" ? "rejected" : "transpiled Go"));
  // A doctest feature always has a _test.go; others get the tab once a run shows one.
  if (feat.outputKind === "test" || (last && last.test)) {
    wrap.appendChild(tabFor("test", "generated _test.go"));
  }
  return wrap;
}

// --------------------------------------------------------------------------- //
// running (WebAssembly)
// --------------------------------------------------------------------------- //

async function runFeature(btn) {
  const out = document.getElementById("result-output");
  const original = btn.textContent;
  btn.disabled = true;
  btn.textContent = "Running…";
  try {
    const transpile = await getTranspiler();
    // Pass the feature's filename so located checker diagnostics render the same
    // `name.goal:line:col:` prefix the doc shows.
    const sourceName = state.current?.sourceName || "source.goal";
    const result = transpile(withTrailingNewline(state.editor.value), sourceName);
    state.lastResult = result;
    // Re-render tabs so a freshly-produced _test.go becomes selectable.
    const head = out.parentElement.querySelector(".panel-head");
    head.replaceChild(buildOutputTabs(state.current), head.querySelector(".tabs"));
    renderOutput(out, result);
  } catch (err) {
    state.lastResult = null;
    renderOutput(out, { error: `Runtime error:\n${String(err)}` });
  } finally {
    btn.disabled = false;
    btn.textContent = original;
  }
}

function renderOutput(out, result) {
  out.classList.remove("muted", "ok", "bad", "seed");
  if (result.error) {
    out.classList.add("bad");
    out.textContent = result.error;
    return;
  }
  const body = state.outputTab === "test" ? result.test : result.go;
  if (!body) {
    out.classList.add("muted");
    out.textContent =
      state.outputTab === "test"
        ? "This program has no doctests, so no _test.go is generated."
        : "(no output)";
    return;
  }
  out.classList.add("ok");
  if (result.seed) out.classList.add("seed");
  setGo(out, body.replace(/\n$/, ""));
}

// setGo fills a node with Go source, syntax-highlighted via Prism when it's
// available. Both the "go" and "test" tabs are plain Go (.go / _test.go), so the
// same grammar applies. Falls back to plain text if Prism failed to load.
function setGo(node, code) {
  if (globalThis.Prism?.languages?.go) {
    node.innerHTML = globalThis.Prism.highlight(code, globalThis.Prism.languages.go, "go");
  } else {
    node.textContent = code;
  }
}

// Lazily instantiate goal.wasm exactly once and return the goalTranspile fn.
let transpilerPromise = null;
function getTranspiler() {
  if (transpilerPromise) return transpilerPromise;
  transpilerPromise = (async () => {
    setRuntimeStatus("loading", "wasm: loading…");
    await loadScript("wasm_exec.js"); // defines globalThis.Go
    const go = new globalThis.Go();
    const instance = await instantiate(go);
    // main() blocks on select{}; it registers goalTranspile before parking, so
    // do not await go.run (its promise never resolves).
    go.run(instance);
    if (typeof globalThis.goalTranspile !== "function") {
      throw new Error("goal.wasm did not register goalTranspile");
    }
    setRuntimeStatus("ready", "wasm: ready");
    return globalThis.goalTranspile;
  })().catch((err) => {
    setRuntimeStatus("error", "wasm: failed");
    transpilerPromise = null; // allow a retry on the next Run
    throw err;
  });
  return transpilerPromise;
}

// instantiate prefers streaming compilation, falling back to a buffered fetch
// when the server doesn't send application/wasm (e.g. some local static servers).
async function instantiate(go) {
  if (WebAssembly.instantiateStreaming) {
    try {
      const { instance } = await WebAssembly.instantiateStreaming(
        fetch("goal.wasm", { cache: "no-cache" }),
        go.importObject,
      );
      return instance;
    } catch {
      // fall through to the buffered path
    }
  }
  const bytes = await (await fetch("goal.wasm", { cache: "no-cache" })).arrayBuffer();
  const { instance } = await WebAssembly.instantiate(bytes, go.importObject);
  return instance;
}

// --------------------------------------------------------------------------- //
// helpers
// --------------------------------------------------------------------------- //

function setRuntimeStatus(stateName, text) {
  const node = document.getElementById("runtime-status");
  node.dataset.state = stateName;
  node.textContent = text;
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

// The transpiler expects newline-terminated source; editors strip trailing blank
// lines for display, so normalize to exactly one trailing newline before running.
function withTrailingNewline(s) {
  return s.replace(/\s*$/, "") + "\n";
}

function enableTab(ta) {
  ta.addEventListener("keydown", (e) => {
    if (e.key !== "Tab") return;
    e.preventDefault();
    const start = ta.selectionStart;
    const end = ta.selectionEnd;
    ta.value = ta.value.slice(0, start) + "    " + ta.value.slice(end);
    ta.selectionStart = ta.selectionEnd = start + 4;
  });
}

// setupGoalEditor registers the code-input template (Prism + a 4-space Indent
// plugin) exactly once. Returns false if the editor scripts or the goal grammar
// failed to load, so makeEditor can fall back to a plain textarea. Globals come
// from the classic scripts loaded before this module (code-input*.js, prism*.js).
function setupGoalEditor() {
  if (setupGoalEditor.ready) return true;
  if (!globalThis.codeInput || !globalThis.Prism?.languages?.goal) return false;
  codeInput.registerTemplate(
    "goal",
    codeInput.templates.prism(Prism, [new codeInput.plugins.Indent(true, 4)]),
  );
  setupGoalEditor.ready = true;
  return true;
}

// makeEditor builds the .goal source editor: a <code-input> (transparent textarea
// over a Prism-highlighted <pre>) when available, else a plain <textarea> with the
// same behavior. Both expose `.value`, so runFeature/Reset don't care which it is.
function makeEditor(source, runBtn) {
  const runOnCmdEnter = (e) => {
    if ((e.metaKey || e.ctrlKey) && e.key === "Enter") {
      e.preventDefault();
      runFeature(runBtn);
    }
  };
  if (setupGoalEditor()) {
    const ed = document.createElement("code-input");
    ed.className = "editor";
    ed.setAttribute("language", "goal");
    ed.setAttribute("template", "goal");
    ed.setAttribute("spellcheck", "false");
    // Seed via the `value` attribute, not inner text: code-input reads it on
    // connect, and as a DOM attribute the source's <, >, & stay literal (never
    // parsed as HTML). Tab is handled by the Indent plugin. Cmd/Ctrl+Enter
    // bubbles from the inner textarea to this host.
    ed.setAttribute("value", source);
    ed.addEventListener("keydown", runOnCmdEnter);
    return ed;
  }
  const ta = el("textarea", "editor");
  ta.spellcheck = false;
  ta.value = source;
  enableTab(ta);
  ta.addEventListener("keydown", runOnCmdEnter);
  return ta;
}

// highlightStaticBlocks colors the read-only .goal/.go snippets in generated doc
// HTML in place. The build emits <pre class="code lang-goal"><code>…</code></pre>
// (also lang-go, lang-error); we Prism-highlight goal and go, and leave anything
// without a grammar (lang-error) as plain text.
function highlightStaticBlocks(root) {
  if (!globalThis.Prism) return;
  for (const pre of root.querySelectorAll("pre.code[class*='lang-']")) {
    const code = pre.querySelector("code");
    const lang = (pre.className.match(/lang-(\w+)/) || [])[1];
    const grammar = lang === "goal" ? Prism.languages.goal : lang === "go" ? Prism.languages.go : null;
    if (code && grammar) code.innerHTML = Prism.highlight(code.textContent, grammar, lang);
  }
}

function el(tag, className, text) {
  const node = document.createElement(tag);
  if (className) node.className = className;
  if (text != null) node.textContent = text;
  return node;
}

function escapeHtml(s) {
  return s.replace(/[&<>]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;" })[c]);
}

boot();
