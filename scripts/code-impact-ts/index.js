#!/usr/bin/env node
// code-impact-ts — frontend slice of the per-PR code-impact tool (spec §2.2
// Approach A). Emits one JSON object to stdout: { nodes, edges, note }.
//
// ponytail: regex-based export detection + git-grep import scan, zero new npm
// deps. The task allows the installed `typescript` devDep (ts.createSourceFile)
// as an alternative parser; regex is chosen because it has no module-resolution
// failure mode on a minimal CI runner and needs nothing from node_modules.
// Upgrade path: swap parseExports() to ts.createSourceFile for edge-case accuracy.
//
// STATIC ANALYSIS ONLY: never imports or executes the analyzed frontend modules.
"use strict";

const { execFileSync } = require("child_process");
const fs = require("fs");

function main() {
  const args = parseArgs(process.argv.slice(2));
  const base = args.base || process.env.BASE_REF || "origin/main";
  const head = args.head || process.env.HEAD_REF || "HEAD";
  const files = args._.filter((f) => /\.[tj]sx?$|\.svelte$/.test(f));

  const nodes = [];
  const edges = [];
  for (const file of files) {
    const baseSrc = gitShow(base, file);
    const headSrc = gitShow(head, file) || readDisk(file); // WIP fallback
    const change = classifyFile(baseSrc, headSrc);
    const baseExp = parseExports(baseSrc);
    const headExp = parseExports(headSrc);
    const exp = exportDelta(baseExp, headExp); // changed export names
    if (change === "ctx") continue; // no diff detected (shouldn't happen for a listed file)
    const label = file.replace(/^frontend\/src\//, "");
    const isTest = /\.test\.[tj]sx?$|\.spec\.[tj]sx?$/.test(file);
    const node = {
      key: "ts::" + file,
      label,
      file,
      start: 1,
      lang: file.endsWith(".svelte") ? "svelte" : "ts",
      change,
      exports: exp,
      isTest,
    };
    nodes.push(node);
    for (const dep of importersOf(file)) {
      const isTest = /\.test\.[tj]sx?$|\.spec\.[tj]sx?$|\.test\.svelte$/.test(dep);
      edges.push({
        from: "ts::" + dep,
        to: node.key,
        label: "imports",
        kind: "imports",
      });
      nodes.push({
        key: "ts::" + dep,
        label: dep.replace(/^frontend\/src\//, ""),
        file: dep,
        start: 1,
        lang: dep.endsWith(".svelte") ? "svelte" : "ts",
        change: "ctx",
        exports: [],
        isTest,
      });
    }
  }
  process.stdout.write(JSON.stringify({ nodes, edges, note: "" }));
}

function parseArgs(argv) {
  const out = { _: [] };
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    if (a === "-base") out.base = argv[++i];
    else if (a === "-head") out.head = argv[++i];
    else out._.push(a);
  }
  return out;
}

function classifyFile(baseSrc, headSrc) {
  if (!baseSrc && headSrc) return "add";
  if (baseSrc && !headSrc) return "rem";
  if (baseSrc !== headSrc) return "mod";
  return "ctx";
}

// ponytail: regex export detection. Catches the export shapes used in this repo
// (named function/const, grouped `export { ... }`, default). Misses exotic
// forms; ts.createSourceFile is the upgrade (see header).
function parseExports(src) {
  const set = new Set();
  if (!src) return set;
  const re =
    /export\s+(?:async\s+)?(?:function|class|const|let|var)\s+([A-Za-z_$][\w$]*)|export\s+\{([^}]+)\}|export\s+default\s+(?:async\s+)?function\s+([A-Za-z_$][\w$]*)/g;
  let m;
  while ((m = re.exec(src)) !== null) {
    if (m[1]) set.add(m[1]);
    if (m[2]) m[2].split(",").map((s) => s.replace(/\bas\s+.*$/, "").trim()).filter(Boolean).forEach((n) => set.add(n));
    if (m[3]) set.add("default " + m[3]);
  }
  return set;
}

function exportDelta(baseSet, headSet) {
  const out = [];
  for (const e of headSet) if (!baseSet.has(e)) out.push(e);
  for (const e of baseSet) if (!headSet.has(e)) out.push(e);
  return out;
}

// importersOf: 1-hop dependents via import scan (spec §2.2).
// ponytail: matches the changed file's $lib alias path or basename. Relative
// import resolution (../../foo) is approximate — false-negative risk on unusual
// relative paths; acceptable for blast-radius viz, no type info needed.
function importersOf(file) {
  const lib = file.replace(/^frontend\/src\//, "").replace(/\.[tj]sx?$/, "").replace(/\.svelte$/, "");
  const base = lib.split("/").pop();
  let out;
  try {
    // ponytail: spec §2.2 suggests `^import .* from`, but (a) Svelte <script>
    // imports are indented and (b) git grep -E is POSIX ERE with no `\s`. Use an
    // unanchored pattern to find candidate files; the per-file module check below
    // does the precise matching.
    out = execFileSync(
      "git",
      ["grep", "-l", "-E", "import .* from", "--", "frontend"],
      { encoding: "utf8" }
    );
  } catch (e) {
    return [];
  }
  const deps = new Set();
  for (const line of out.split("\n")) {
    if (!line) continue;
    if (line === file) continue; // self
    if (line.startsWith("scripts/")) continue;
    if (!/\.[tj]sx?$|\.svelte$/.test(line)) continue;
    // read the import lines of the candidate and see if they mention this module
    let src;
    try {
      src = gitShow("HEAD", line) || readDisk(line);
    } catch {
      continue;
    }
    if (!src) continue;
    const wants =
      src.includes("$lib/" + lib) ||
      src.includes("$lib/" + lib + ".js") ||
      // basename fallback for relative imports of an index/barrel
      (base === "index" ? false : new RegExp(`from\\s+['"][^'"]*\\b${escapeRe(base)}['"]`).test(src));
    if (wants) deps.add(line);
  }
  return [...deps];
}

function escapeRe(s) {
  return s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

function gitShow(ref, file) {
  try {
    return execFileSync("git", ["show", `${ref}:${file}`], { encoding: "utf8" });
  } catch {
    return "";
  }
}

function readDisk(file) {
  try {
    return fs.readFileSync(file, "utf8");
  } catch {
    return "";
  }
}

main();
