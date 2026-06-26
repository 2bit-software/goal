// Loads the Goal TextMate grammar with the same engine VS Code uses
// (vscode-textmate + vscode-oniguruma) and asserts that representative
// tokens receive the expected scopes. Run: node test/tokenize.test.mjs
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import { createRequire } from "node:module";
import onigPkg from "vscode-oniguruma";
import vsctmPkg from "vscode-textmate";

const oniguruma = onigPkg.loadWASM ? onigPkg : onigPkg.default;
const vsctm = vsctmPkg.Registry ? vsctmPkg : vsctmPkg.default;
const require = createRequire(import.meta.url);
const here = path.dirname(fileURLToPath(import.meta.url));
const grammarPath = path.join(here, "..", "syntaxes", "goal.tmLanguage.json");

const wasmPath = require.resolve("vscode-oniguruma/release/onig.wasm");
const onigLib = oniguruma.loadWASM(fs.readFileSync(wasmPath).buffer).then(() => ({
  createOnigScanner: (p) => new oniguruma.OnigScanner(p),
  createOnigString: (s) => new oniguruma.OnigString(s),
}));

const registry = new vsctm.Registry({
  onigLib,
  loadGrammar: async (scope) =>
    scope === "source.goal"
      ? vsctm.parseRawGrammar(fs.readFileSync(grammarPath, "utf8"), grammarPath)
      : null,
});

const grammar = await registry.loadGrammar("source.goal");

function tokenize(source) {
  const tokens = [];
  let ruleStack = vsctm.INITIAL;
  for (const line of source.split("\n")) {
    const r = grammar.tokenizeLine(line, ruleStack);
    for (const t of r.tokens) {
      tokens.push({ text: line.slice(t.startIndex, t.endIndex), scopes: t.scopes });
    }
    ruleStack = r.ruleStack;
  }
  return tokens;
}

const sample = fs.readFileSync(path.join(here, "..", "examples", "sample.goal"), "utf8");
const tokens = tokenize(sample);

let failures = 0;
function expectScope(text, scope) {
  const hit = tokens.find((t) => t.text.trim() === text && t.scopes.some((s) => s === scope));
  if (hit) {
    console.log(`  ok   "${text}" -> ${scope}`);
  } else {
    failures++;
    const found = tokens.filter((t) => t.text.trim() === text).map((t) => t.scopes.join(" "));
    console.error(`  FAIL "${text}" expected scope ${scope}; got: ${found.join(" | ") || "(token not found)"}`);
  }
}

console.log("Goal grammar scope assertions:");
expectScope("enum", "storage.type.goal");
expectScope("sealed", "storage.type.goal");
expectScope("from", "storage.type.goal");
expectScope("match", "keyword.control.goal");
expectScope("assert", "keyword.control.goal");
expectScope("implements", "storage.modifier.goal");
expectScope("=>", "keyword.operator.arrow.goal");
expectScope("?", "keyword.operator.unwrap.goal");
expectScope("...defaults", "keyword.operator.spread.goal");
expectScope("Result", "support.type.goal");
expectScope("Option", "support.type.goal");
expectScope("Ok", "variable.other.enummember.goal");
expectScope("None", "variable.other.enummember.goal");
expectScope("State", "entity.name.type.goal");
expectScope("Idle", "variable.other.enummember.goal");
expectScope(">>>", "keyword.control.doctest.goal");
expectScope("func", "storage.type.goal");
expectScope("int", "support.type.builtin.goal");
expectScope("return", "keyword.control.goal");

if (failures > 0) {
  console.error(`\n${failures} assertion(s) failed.`);
  process.exit(1);
}
console.log(`\nAll ${tokens.length ? "" : ""}assertions passed.`);
