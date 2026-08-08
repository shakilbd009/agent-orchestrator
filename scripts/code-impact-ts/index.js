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
const path = require("path");

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
    for (const dep of importersOf(file, head)) {
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
  // prototype §5.3: per-changed-route sibling/parent context + per-changed-file
  // import targets, exposed in the same JSON contract for the roots view.
  const routes = [];
  const imports = [];
  const changedSet = new Set(files);
  for (const file of files) {
    if (!isRoutePageFile(file)) continue;
    const parentLayout = findAncestorLayout(path.posix.dirname(file), base, file);
    const siblings = parentLayout ? routeSiblings(parentLayout, base, file) : [];
    routes.push({ file, parentLayout, siblings });
  }
  for (const file of files) {
    const targets = importTargets(file, head, changedSet);
    if (targets.length) imports.push({ from: file, targets });
  }
  process.stdout.write(JSON.stringify({ nodes, edges, note: "", routes, imports }));
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
function importersOf(file, head) {
  const lib = file.replace(/^frontend\/src\//, "").replace(/\.[tj]sx?$/, "").replace(/\.svelte$/, "");
  const base = lib.split("/").pop();
  let out;
  try {
    // ponytail: spec §2.2 suggests `^import .* from`, but (a) Svelte <script>
    // imports are indented and (b) git grep -E is POSIX ERE with no `\s`. Use an
    // unanchored pattern to find candidate files; the per-file module check below
    // does the precise matching.
    // M2: ref-qualify to HEAD_REF — `git grep <rev>` searches that rev's tree, not
    // the working tree (Phase 3 CI checks out `main`, pr-head is a ref only).
    out = execFileSync(
      "git",
      ["grep", "-l", "-E", "import .* from", head, "--", "frontend"],
      { encoding: "utf8" }
    );
  } catch (e) {
    return [];
  }
  const deps = new Set();
  const prefix = head + ":"; // rev-mode output is `<head>:path`
  for (const raw of out.split("\n")) {
    if (!raw) continue;
    const line = raw.startsWith(prefix) ? raw.slice(prefix.length) : raw;
    if (line === file) continue; // self
    if (line.startsWith("scripts/")) continue;
    if (!/\.[tj]sx?$|\.svelte$/.test(line)) continue;
    // read the import lines of the candidate and see if they mention this module
    let src;
    try {
      src = gitShow(head, line) || readDisk(line);
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

// ---------------- roots-view context (prototype §5.2.2 / §5.3) ----------------
// Faithful per §5.4: only real files at BASE_REF (siblings/parent layouts) or
// real import targets of a changed file. No speculative "related modules."

function isRoutePageFile(file) {
  return file.startsWith("frontend/src/routes/") && /\/\+page\.svelte$/.test(file);
}

function isTestFileName(file) {
  return /\.test\.[tj]sx?$|\.spec\.[tj]sx?$|\.test\.svelte$/.test(file);
}

function blobExists(ref, file) {
  return !!gitShow(ref, file);
}

function lsDir(ref, dir) {
  try {
    const out = execFileSync("git", ["ls-tree", "--name-only", ref, "--", dir + "/"], { encoding: "utf8" });
    return out.split("\n").map((s) => s.trim()).filter(Boolean);
  } catch {
    return [];
  }
}

// nearest ancestor +layout.svelte at BASE that is NOT the file itself (§5.2.2).
function findAncestorLayout(dir, base, selfFile) {
  const root = "frontend/src/routes";
  let d = dir;
  while (true) {
    const cand = d + "/+layout.svelte";
    if (cand !== selfFile && blobExists(base, cand)) return cand;
    if (d === root) break;
    const i = d.lastIndexOf("/");
    if (i < root.length) break;
    d = d.slice(0, i);
  }
  return "";
}

// sibling routes sharing parentLayout at BASE: the layout's own +page plus its
// subdirectory routes (e.g. board/+page.svelte under orchestration/+layout.svelte).
// ls-tree returns full repo paths, so derive the entry name by stripping the dir.
function routeSiblings(parentLayout, base, selfFile) {
  const dir = path.posix.dirname(parentLayout);
  const out = [];
  for (const full of lsDir(base, dir)) {
    if (full === selfFile) continue;
    const e = full.startsWith(dir + "/") ? full.slice(dir.length + 1) : full;
    if (/^\+page\.(svelte|ts|js)$/.test(e)) {
      if (!isTestFileName(full)) out.push(full);
    } else if (!e.includes(".")) {
      const sub = full + "/+page.svelte";
      if (blobExists(base, sub) && !isTestFileName(sub)) out.push(sub);
    }
  }
  return out.sort();
}

// resolved import targets of `file` that are real repo modules NOT in the
// changed set (§5.2.2: the reused existing roots, e.g. event-feed.ts → api/client.ts).
function importTargets(file, head, changedSet) {
  const src = gitShow(head, file) || readDisk(file);
  if (!src) return [];
  const out = [];
  const seen = new Set();
  const re = /import\s+(?:[\s\S]*?\s+from\s+)?['"]([^'"]+)['"]/g;
  let m;
  while ((m = re.exec(src)) !== null) {
    const resolved = resolveModule(file, m[1], head);
    if (!resolved || changedSet.has(resolved) || seen.has(resolved)) continue;
    seen.add(resolved);
    const exps = [...parseExports(gitShow(head, resolved) || readDisk(resolved))].slice(0, 3);
    out.push({ file: resolved, exports: exps });
  }
  return out;
}

function resolveModule(importer, spec, head) {
  let base;
  if (spec.startsWith("$lib/")) {
    base = "frontend/src/lib/" + spec.slice("$lib/".length);
  } else if (spec.startsWith("./") || spec.startsWith("../")) {
    base = path.posix.join(path.posix.dirname(importer), spec);
  } else {
    return ""; // bare/3rd-party (svelte, $app/*) — not a repo module
  }
  for (const cand of [base + ".ts", base + ".svelte", base + ".js", base + "/index.ts", base + "/index.js"]) {
    if (blobExists(head, cand)) return cand;
  }
  return "";
}

main();
