# code-impact — per-PR fine-grained blast-radius tool

Per-PR, **function/module-granularity** impact visualization. Emits a Mermaid
blast-radius diagram plus a one-line color legend to stdout, wrapped in an
idempotent splice fence for PR-body embedding. Complements the coarse `ARCHITECTURE.md` +
architecture-impact baseline (this is the fine layer; do not duplicate the coarse
component graph).

Spec: `data/orchestrator-code-impact-spec/report.md` §1 (colors), §2.1 (Go),
§2.2 (TS/Svelte), §2.3 (zero deps), §3.2 (fence). > **Note:** the canonical
`report.md` was absent from all repo refs at build time; this tool was built from
the dispatch task's inline spec reproduction + the `fm/orchestrator-arch-baseline-init`
`ARCHITECTURE.md` color baseline.

## Layout

- `scripts/code-impact` — bash dispatcher. Resolves `BASE_REF`/`HEAD_REF`, builds
  the Go core with the repo's pinned Go (1.25.6), execs it.
- `scripts/code-impact-go/` — Go core + renderer (stdlib only: `go/parser`,
  `go/ast`, `go/token`, `git`). Own module, zero external deps.
- `scripts/code-impact-ts/index.js` — frontend slice (Node, zero new npm; regex
  export detection + git-grep import scan).

## Usage

```bash
scripts/code-impact                              # BASE_REF=origin/main HEAD_REF=HEAD; --view auto
BASE_REF=<ref> HEAD_REF=<ref> scripts/code-impact
scripts/code-impact --base <ref> --head <ref>
scripts/code-impact --view roots|blast|auto      # roots = root-system (default for additive PRs)
scripts/code-impact > body-block.md              # worker-embeddable: gh pr create --body-file
```

Output is a `<!--code-impact:start--> … <!--code-impact:end-->` block (spec §3.2)
that a worker splices into a PR body without clobbering the author's description.

## How it works (static analysis only)

- **Go (§2.1 Approach A):** `go/parser` on both refs → changed `FuncDecl`s by
  line-span intersection with diff hunks; classify add/mod/remove by presence in
  each ref; 1-hop callees via `*ast.CallExpr`; 1-hop dependents via `git grep`
  of callers (receiver-qualified labels). **Never** `go run`/`go build`s the
  analyzed code — so it diagrams WIP PRs that do not compile.
- **TS/Svelte (§2.2 Approach A):** a changed `.ts`/`.svelte` file is a node;
  changed exports via regex; 1-hop import dependents via import scan.

## Colors (spec §1 — match architecture-impact)

green=addition · amber=modification · red=removal · gray=existing context ·
dashed=unexpected (overflow / unresolved callee). Applied via Mermaid
`classDef`/`style`.

## Capped neighborhood (spec §2)

25-node budget: changed production symbols → direct dependents → callees.
Overflow callees summarized as `… +N more`. Test functions are **not** graph
nodes (they'd drown the cap). When a large PR truncates the graph, the capping
warning prints to **stderr** — the embeddable stdout block stays just the
diagram + legend.

## Views: `--view roots|blast|auto` (prototype §5)

Two render modes layered on the **same** static analysis; the analyzer is
untouched.

- **`blast`** — the original flat blast-radius (`flowchart LR`): changed symbols
  + 1-hop dependents/callees. Best for heavily-modifying PRs where "before" is
  everywhere.
- **`roots`** — the root-system view (`flowchart TD`): surfaces the ESTABLISHED
  gray roots an additive PR grafts onto — the parent `+layout.svelte`, sibling
  routes, and the reused import targets of changed files (e.g. `event-feed.ts →
  api/client.ts`) — marks the graft point amber, adds a route-containment edge
  (so a modified layout is no longer an orphan), and emits a legend. Faithful
  per §5.4: every gray node is a real file at BASE_REF or a real import target
  of a changed file — nothing speculative.
- **`auto` (default)** — `roots` when ≥~70% of changed nodes are additions AND
  honest established context exists; else `blast`. Override any time with
  `--view`.

The roots block replaces the blast block inside the same idempotent
`<!--code-impact-->` fence. Both views emit **only** the Mermaid diagram plus a
one-line color legend — no in-diagram legend subgraph, no test-impact table.
Both views reuse the spec §1 color convention (green=add, amber=mod, red=rem,
gray=ctx). Zero new dependencies (prototype §5.3).

## Known limitations (deliberate, marked `ponytail:` in source)

- **Interface-dispatch blind spot:** for `h.Spawn(ctx)` we record the bare
  `Spawn` and cannot resolve the concrete type behind an interface var — would
  need `x/tools`, excluded per §2.3.
- **Bare-name caller grep:** `\.MethodName(` is high-recall; a method name shared
  across two types can over-match (no type info).
- **Relative-import dependents (TS):** matches the `$lib/…` alias form; unusual
  relative imports may be missed.
- **Phase 4 stubbed:** `--cover-go`/`--cover-ts` (coverage wiring) are accepted
  but no-ops.
