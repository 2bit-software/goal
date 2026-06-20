// Playground gate: load the real goal.wasm in Node (Go WASM runs headless here,
// no browser needed) and assert that every feature's .goal source transpiles to
// the seed output baked into features.json. The seed is itself verified against
// the host transpiler at generate time, so this closes the loop: doc == host
// transpiler == WASM transpiler. A mismatch blocks the Pages deploy.

import { readFile } from "node:fs/promises";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

import "../site/wasm_exec.js"; // side effect: defines globalThis.Go

const siteDir = join(dirname(fileURLToPath(import.meta.url)), "..", "site");

const go = new globalThis.Go();
const wasm = await readFile(join(siteDir, "goal.wasm"));
const { instance } = await WebAssembly.instantiate(wasm, go.importObject);
// Do NOT await: main() blocks on select{} so go.run never resolves. It runs
// synchronously up to that block first, registering goalTranspile.
go.run(instance);

if (typeof globalThis.goalTranspile !== "function") {
  console.error("goalTranspile was not registered by the wasm module");
  process.exit(1);
}

const manifest = JSON.parse(await readFile(join(siteDir, "features.json"), "utf8"));

const trim = (s) => (s ?? "").replace(/\n+$/, "");
let total = 0;
let failed = 0;

for (const cat of manifest.categories) {
  for (const feat of cat.features) {
    total++;
    const res = globalThis.goalTranspile(feat.source);
    if (res.error) {
      failed++;
      console.error(`✗ ${feat.title} — transpile error:\n${res.error}\n`);
      continue;
    }
    const got = trim(feat.outputKind === "test" ? res.test : res.go);
    const want = trim(feat.expected);
    if (got !== want) {
      failed++;
      console.error(`✗ ${feat.title} — output does not match seed`);
      console.error(`  --- seed ---\n${indent(want)}\n  --- wasm ---\n${indent(got)}\n`);
      continue;
    }
    console.log(`✓ ${feat.title}`);
  }
}

console.log(`\n${total - failed}/${total} features reproduced their seed output.`);
process.exit(failed ? 1 : 0);

function indent(s) {
  return s
    .split("\n")
    .map((l) => "    " + l)
    .join("\n");
}
