# code-impact — per-PR fine-grained blast-radius tool

Per-PR, **function/module-granularity** impact visualization. Emits a Mermaid
blast-radius graph + test-impact table to stdout, wrapped in an idempotent splice
fence for PR-body embedding. Complements the coarse `ARCHITECTURE.md` +
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
scripts/code-impact                              # BASE_REF=origin/main HEAD_REF=HEAD
BASE_REF=<ref> HEAD_REF=<ref> scripts/code-impact
scripts/code-impact --base <ref> --head <ref>
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
nodes (they'd drown the cap); they surface in the test-impact table's
"Tests likely impacted" column.

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
