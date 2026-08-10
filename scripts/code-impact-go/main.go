// Command code-impact-go is the backend/renderer of the per-PR code-impact tool
// (spec §2.1 Approach A, §2.3). It performs STATIC ANALYSIS ONLY: go/parser +
// go/ast + git grep. It NEVER `go run`s, `go build`s, or otherwise executes the
// analyzed backend code — that is what lets it emit a diagram for a WIP PR that
// does not compile (spec §2, Phase 3 CI security model).
//
// It also shells out to scripts/code-impact-ts for the frontend slice (§2.2).
// Running our OWN analysis script is allowed; only executing the *analyzed*
// code is forbidden.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

// spec §1 color convention — MUST match architecture-impact baseline.
//
//	green=addition, amber=modification, red=removal, gray=existing context,
//	dashed=unexpected (overflow / unresolved target).
const (
	classAdd = "add" // green
	classMod = "mod" // amber
	classRem = "rem" // red
	classCtx = "ctx" // gray
)

// defaultCap is the capped-neighborhood budget (spec §2 "capped neighborhood").
const defaultCap = 25

func main() {
	var (
		baseRef  = envOr("BASE_REF", "origin/main")
		headRef  = envOr("HEAD_REF", "HEAD")
		tsScript = envOr("CODE_IMPACT_TS", "") // optional override; default resolved next to this binary
		view     = envOr("VIEW", "auto")       // §5.1: roots|blast|auto
		showHelp bool
	)
	flag.StringVar(&baseRef, "base", baseRef, "BASE_REF (default origin/main)")
	flag.StringVar(&headRef, "head", headRef, "HEAD_REF (default HEAD)")
	flag.StringVar(&view, "view", view, "view: roots|blast|auto (auto = roots for additive PRs with established context, else blast)")
	// ponytail: --cover-go/--cover-ts are Phase 4 (coverage wiring). Stubbed on purpose.
	flag.Bool("cover-go", false, "Phase 4 — stubbed (coverage not wired yet)")
	flag.Bool("cover-ts", false, "Phase 4 — stubbed (coverage not wired yet)")
	flag.BoolVar(&showHelp, "h", false, "show usage")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: code-impact-go [-base BASE_REF] [-head HEAD_REF] [-view roots|blast|auto]")
	}
	flag.Parse()
	if showHelp {
		flag.Usage()
		os.Exit(0)
	}

	if err := run(baseRef, headRef, tsScript, view); err != nil {
		// ponytail: never hide a failure behind exit 0; fail loud (AGENTS Rule 12).
		fmt.Fprintf(os.Stderr, "code-impact-go: %v\n", err)
		os.Exit(1)
	}
}

func run(baseRef, headRef, tsScript, viewReq string) error {
	if !gitExists() {
		return fmt.Errorf("git not found on PATH")
	}
	files, err := changedFiles(baseRef, headRef)
	if err != nil {
		return err
	}
	var goFiles, tsFiles []string
	for _, f := range files {
		switch {
		case strings.HasSuffix(f, ".go"):
			// NOTE: the tool's own main.go IS analyzable; the callersOf grep filter
			// separately excludes self-hits, so no special-casing needed here.
			goFiles = append(goFiles, f)
		case strings.HasSuffix(f, ".ts"), strings.HasSuffix(f, ".svelte"):
			if strings.HasSuffix(f, ".test.ts") || strings.HasSuffix(f, ".spec.ts") {
				continue // ponytail: test files are dependents, not primary changed nodes
			}
			tsFiles = append(tsFiles, f)
		}
	}

	graph := newGraph()
	ga := &goAnalyzer{base: baseRef, head: headRef, g: graph}
	for _, f := range goFiles {
		ga.analyzeFile(f)
	}
	// Frontend slice (§2.2): delegate to the Node script, parse its JSON.
	if len(tsFiles) > 0 {
		tsNodes, tsEdges, tsRoutes, tsImports, note, err := runTSScript(tsScript, baseRef, headRef, tsFiles)
		if err != nil {
			// ponytail: degrade gracefully — node may be absent on a minimal runner.
			// Backend diagram still ships; flag loudly in the output.
			graph.notes = append(graph.notes, "frontend analysis skipped: "+err.Error())
		} else {
			// roots-view context (prototype §5.2.2): route spine + reused import targets.
			graph.routes = tsRoutes
			graph.imports = tsImports
			for _, n := range tsNodes {
				graph.addNode(n)
			}
			graph.edges = append(graph.edges, tsEdges...)
			if note != "" {
				graph.notes = append(graph.notes, note)
			}
		}
	}

	// §5.1 view selection: explicit --view wins; auto picks roots for additive
	// PRs with honest established context, else blast (unchanged).
	chosen := chooseView(viewReq, graph)
	var out string
	if chosen == "roots" {
		out = graph.renderRoots(defaultCap)
	} else {
		out = graph.render(defaultCap)
	}
	// spec §3.2 idempotent splice fence — worker-embeddable. stdout is ONLY the
	// fenced block (Mermaid diagram + one-line legend); capping/skip warnings go
	// to stderr so they never pollute the embeddable PR-description block.
	fmt.Println("<!--code-impact:start-->")
	fmt.Print(out)
	fmt.Println("<!--code-impact:end-->")
	for _, n := range graph.notes {
		fmt.Fprintf(os.Stderr, "code-impact-go: %s\n", n)
	}
	return nil
}

// ---------------- graph model ----------------

type node struct {
	key     string
	label   string // human name (receiver-qualified for methods / file for ts)
	file    string
	start   int // line, 0 if unknown
	lang    string
	change  string // add|mod|rem|ctx
	exports []string
	isTest  bool // test func/file — shown in table, excluded from graph nodes
}

type edge struct {
	from, to, label, kind string // kind: calls|called-by|imports|overflow
	dashed                bool
}

// routeCtx — per changed SvelteKit route page, its parent layout + sibling
// routes at BASE_REF (prototype §5.2.2 frontend existing-context surfacing).
// Sourced from the TS slice's directory listing; faithful per §5.4 (real repo
// tree only — never speculative "related modules").
type routeCtx struct {
	File, ParentLayout string   // full repo paths
	Siblings           []string // full repo paths, excludes the changed file
}

// importCtx — per changed file, its resolved import targets that are NOT
// themselves changed (prototype §5.2.2: the reused existing roots, e.g.
// event-feed.ts → api/client.ts). Real repo modules only.
type importCtx struct {
	From    string
	Targets []importTarget
}
type importTarget struct {
	File    string
	Exports []string
}

type graph struct {
	nodes        map[string]*node
	order        []string // insertion order for stable output
	edges        []edge
	notes        []string
	routes       []routeCtx     // frontend route context (roots view)
	imports      []importCtx    // frontend import-target context (roots view)
	callOverflow map[string]int // per-source overflow count for the "… +N more" summary
}

func newGraph() *graph { return &graph{nodes: map[string]*node{}} }

func (g *graph) addNode(n *node) *node {
	if existing, ok := g.nodes[n.key]; ok {
		// upgraded ctx -> a real change wins
		if n.change != classCtx && existing.change == classCtx {
			existing.change = n.change
		}
		if n.label != "" {
			existing.label = n.label
		}
		return existing
	}
	g.nodes[n.key] = n
	g.order = append(g.order, n.key)
	return n
}

func (g *graph) hasNode(k string) bool { _, ok := g.nodes[k]; return ok }

// ---------------- backend (Go) static analysis (§2.1 Approach A) ----------------

type funcInfo struct {
	canon   string // (*TaskService).ActivateTask  or  ConfigureHarness
	plain   string // ActivateTask / ConfigureHarness
	recv    string // TaskService / ""
	file    string // file the func lives in (set for dependents)
	start   int
	end     int
	callees []string // deduped callee plain-names from *ast.CallExpr in body
}

type goAnalyzer struct {
	base, head string
	g          *graph
	// parsed: ref -> file -> canon -> info (cache)
	parsed map[string]map[string]map[string]*funcInfo
}

func (a *goAnalyzer) analyzeFile(file string) {
	// ponytail: test files are coverage, not blast radius. Their functions are
	// not graph nodes (they would drown the cap); test dependents stay tracked
	// (isTest) but are excluded from the diagram.
	if isTestFile(file) {
		return
	}
	baseFuncs := a.funcs(a.base, file)
	headFuncs := a.funcs(a.head, file)

	// changed hunks in HEAD coordinates — the modification signal (spec §2.1).
	hunks, _ := diffHunks(a.base, a.head, file)
	changes := classifyFuncs(baseFuncs, headFuncs, hunks)

	// ponytail: iterate in SOURCE order (by start line), not map order — Go map
	// iteration is randomized, which would make the diagram non-reproducible
	// across runs. Reproducible output is required for CI (Phase 3).
	for _, canon := range sortedByLine(headFuncs) {
		change, ok := changes[canon]
		if !ok {
			continue // unchanged context — not a primary node
		}
		a.emitChangedFunc(file, headFuncs[canon], change)
	}
	// removals: in base, gone from head (source-ordered for determinism)
	for _, canon := range sortedByLine(baseFuncs) {
		change, ok := changes[canon]
		if !ok {
			continue
		}
		bf := baseFuncs[canon]
		// ponytail: removal classification uses presence only; a rename shows as
		// rem+add pair (known limitation, no type info to link them).
		a.emitChangedFunc(file, &funcInfo{
			canon: bf.canon, plain: bf.plain, recv: bf.recv, start: bf.start, end: bf.end,
		}, change)
	}
}

// classifyFuncs returns canon -> change (add|mod|rem) given the base/head func
// maps and HEAD-side diff hunks. Pure (no git/AST side effects) so it is
// unit-testable on fake func maps (M3).
//   - in head, not base                                   -> add
//   - in both, span intersects a hunk                      -> mod
//   - in base, not head                                   -> rem
func classifyFuncs(baseFuncs, headFuncs map[string]*funcInfo, hunks []hunk) map[string]string {
	out := map[string]string{}
	for canon, hf := range headFuncs {
		switch {
		case baseFuncs[canon] == nil:
			out[canon] = classAdd
		case intersects(hf.start, hf.end, hunks):
			out[canon] = classMod
		}
	}
	for canon := range baseFuncs {
		if headFuncs[canon] == nil {
			out[canon] = classRem
		}
	}
	return out
}

func (a *goAnalyzer) emitChangedFunc(file string, f *funcInfo, change string) {
	n := a.g.addNode(&node{
		key: "go::" + f.canon + "::" + file, label: f.canon, file: file,
		start: f.start, lang: "go", change: change,
	})

	// 1-hop callees (§2.1): *ast.CallExpr in the changed function body.
	// ponytail: interface-dispatch blind spot — for `h.Spawn(ctx)` we record "Spawn"
	// and cannot resolve the concrete type behind the interface var without full
	// type information (would need x/tools, deliberately excluded per §2.3).
	for _, callee := range f.callees {
		target := a.calleeNodeKey(callee)
		if target == n.key {
			continue // ponytail: suppress self-loop from name-collision (e.g. (*execCmd).Start calling exec.Cmd.Start)
		}
		a.g.edges = append(a.g.edges, edge{from: n.key, to: target, label: "calls", kind: "calls"})
	}

	// 1-hop dependents (§2.1): git grep of callers, receiver-qualified for methods.
	for _, dep := range a.callersOf(f, file) {
		depNode := a.g.addNode(&node{
			key: "go::" + dep.canon + "::" + dep.file, label: dep.canon, file: dep.file,
			start: dep.start, lang: "go", change: classCtx, isTest: isTestFile(dep.file),
		})
		// edge points FROM the caller TO the changed func = "caller is impacted".
		a.g.edges = append(a.g.edges, edge{from: depNode.key, to: n.key, label: "calls", kind: "called-by"})
	}
}

// calleeNodeKey returns the key of an existing changed/context node matching the
// callee name, else a shared external context node key.
func (a *goAnalyzer) calleeNodeKey(plain string) string {
	for _, k := range a.g.order {
		n := a.g.nodes[k]
		if n.lang == "go" && (n.label == plain || methodTail(n.label) == plain) {
			return k
		}
	}
	// external / unresolved -> shared gray context node; edge will be dashed (unexpected).
	ext := &node{key: "ext::go::" + plain, label: plain + " (callee)", lang: "go", change: classCtx}
	a.g.addNode(ext)
	return ext.key
}

func (a *goAnalyzer) callersOf(f *funcInfo, ownFile string) []funcInfo {
	// ponytail: bare-name grep `\.ActivateTask(` or `ConfigureHarness(`.
	// High recall; a method name shared across two types can over-match — the
	// known false-positive cost of no type info. Upgrade path: x/tools callee
	// resolution (excluded per §2.3 zero-dep rule).
	var pat string
	if f.recv != "" {
		pat = `\.` + f.plain + `\(`
	} else {
		pat = `\b` + f.plain + `\(`
	}
	hits, _ := gitGrep(pat, a.head, "*.go")
	var out []funcInfo
	for _, h := range hits {
		if h.file == ownFile && f.start != 0 && within(h.line, f.start, f.end) {
			continue // the declaration line itself
		}
		if strings.HasPrefix(h.file, "scripts/code-impact-go/") {
			continue
		}
		if enc := a.enclosingFunc(h.file, h.line); enc != nil {
			enc.file = h.file
			out = append(out, *enc)
		} else {
			// top-level call (init / package-level) — represent as file-level context.
			out = append(out, funcInfo{canon: "<pkg> (" + h.file + ")", plain: "", file: h.file, start: h.line, end: h.line})
		}
	}
	return dedupFuncs(out)
}

func (a *goAnalyzer) enclosingFunc(file string, line int) *funcInfo {
	funcs := a.funcs(a.head, file) // working-tree callers are at HEAD
	for _, fi := range funcs {
		if within(line, fi.start, fi.end) {
			return fi
		}
	}
	return nil
}

func (a *goAnalyzer) funcs(ref, file string) map[string]*funcInfo {
	if a.parsed == nil {
		a.parsed = map[string]map[string]map[string]*funcInfo{}
	}
	byRef, ok := a.parsed[ref]
	if !ok {
		byRef = map[string]map[string]*funcInfo{}
		a.parsed[ref] = byRef
	}
	if m, ok := byRef[file]; ok {
		return m
	}
	m := map[string]*funcInfo{}
	byRef[file] = m
	src, err := gitShow(ref, file)
	// ponytail: WIP support — if HEAD has no committed blob for a brand-new file,
	// read the working tree so uncommitted PRs still diagram (spec §2 goal).
	if (err != nil || len(src) == 0) && ref == a.head {
		if b, e := os.ReadFile(file); e == nil {
			src = b
		}
	}
	if len(src) == 0 {
		return m // new file at HEAD / deleted at BASE — empty set is correct.
	}
	fset := token.NewFileSet()
	// ponytail: ignore parse errors on purpose. go/parser is error-tolerant and
	// returns a partial AST, which is exactly what lets us diagram a WIP PR that
	// does not compile (no go build needed).
	f, _ := parser.ParseFile(fset, file, src, parser.ParseComments)
	if f == nil {
		return m
	}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		plain := fn.Name.Name
		recv := ""
		if fn.Recv != nil && len(fn.Recv.List) > 0 {
			recv = recvTypeName(fn.Recv.List[0].Type)
		}
		canon := plain
		if recv != "" {
			canon = "(*" + recv + ")." + plain
		}
		start := fset.Position(fn.Pos()).Line
		end := fset.Position(fn.End()).Line
		fi := &funcInfo{canon: canon, plain: plain, recv: recv, start: start, end: end}
		fi.callees = callees(fn)
		m[canon] = fi
	}
	return m
}

// callees walks *ast.CallExpr in the body and collects callee plain-names.
func callees(fn *ast.FuncDecl) []string {
	if fn.Body == nil {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch fun := call.Fun.(type) {
		case *ast.Ident:
			if !seen[fun.Name] {
				seen[fun.Name] = true
				out = append(out, fun.Name)
			}
		case *ast.SelectorExpr:
			// ponytail: method or package-qualified call; we take Sel.Name only.
			// Concrete-type resolution needs type info (x/tools) — out of scope (§2.3).
			if !seen[fun.Sel.Name] {
				seen[fun.Sel.Name] = true
				out = append(out, fun.Sel.Name)
			}
		}
		return true
	})
	return out
}

func recvTypeName(t ast.Expr) string {
	switch x := t.(type) {
	case *ast.StarExpr:
		return recvTypeName(x.X)
	case *ast.ParenExpr:
		return recvTypeName(x.X)
	case *ast.Ident:
		return x.Name
	case *ast.IndexExpr: // C[T]
		return recvTypeName(x.X)
	case *ast.IndexListExpr: // C[T, U]
		return recvTypeName(x.X)
	}
	return ""
}

// ---------------- frontend (TS/Svelte) via node script (§2.2) ----------------

type tsNodeJSON struct {
	Key     string   `json:"key"`
	Label   string   `json:"label"`
	File    string   `json:"file"`
	Start   int      `json:"start"`
	Lang    string   `json:"lang"`
	Change  string   `json:"change"`
	Exports []string `json:"exports"`
	IsTest  bool     `json:"isTest"`
}
type tsEdgeJSON struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Label string `json:"label"`
	Kind  string `json:"kind"`
}
type tsRouteJSON struct {
	File         string   `json:"file"`
	ParentLayout string   `json:"parentLayout"`
	Siblings     []string `json:"siblings"`
}
type tsImportTargetJSON struct {
	File    string   `json:"file"`
	Exports []string `json:"exports"`
}
type tsImportJSON struct {
	From    string               `json:"from"`
	Targets []tsImportTargetJSON `json:"targets"`
}
type tsOut struct {
	Nodes   []tsNodeJSON   `json:"nodes"`
	Edges   []tsEdgeJSON   `json:"edges"`
	Routes  []tsRouteJSON  `json:"routes"`
	Imports []tsImportJSON `json:"imports"`
	Note    string         `json:"note"`
}

func runTSScript(override, base, head string, files []string) (nodes []*node, edges []edge, routes []routeCtx, imports []importCtx, note string, err error) {
	script := override
	if script == "" {
		exe, e := os.Executable()
		if e == nil {
			script = joinDir(exe, "..", "code-impact-ts", "index.js")
		}
		if script == "" {
			script = "scripts/code-impact-ts/index.js"
		}
	}
	if _, e := os.Stat(script); e != nil {
		return nil, nil, nil, nil, "", fmt.Errorf("ts script missing: %s", script)
	}
	args := []string{script, "-base", base, "-head", head}
	args = append(args, files...)
	cmd := exec.Command("node", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if e := cmd.Run(); e != nil {
		return nil, nil, nil, nil, "", fmt.Errorf("node: %v (%s)", e, strings.TrimSpace(stderr.String()))
	}
	var o tsOut
	// the script prints one JSON object; tolerate trailing whitespace.
	if e := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &o); e != nil {
		return nil, nil, nil, nil, "", fmt.Errorf("parse ts json: %v", e)
	}
	for _, n := range o.Nodes {
		nodes = append(nodes, &node{
			key: n.Key, label: n.Label, file: n.File, start: n.Start,
			lang: n.Lang, change: n.Change, exports: n.Exports, isTest: n.IsTest,
		})
	}
	for _, e := range o.Edges {
		edges = append(edges, edge{from: e.From, to: e.To, label: "imports", kind: "imports"})
	}
	for _, r := range o.Routes {
		sibs := make([]string, len(r.Siblings))
		copy(sibs, r.Siblings)
		routes = append(routes, routeCtx{File: r.File, ParentLayout: r.ParentLayout, Siblings: sibs})
	}
	for _, im := range o.Imports {
		var tgts []importTarget
		for _, t := range im.Targets {
			tgts = append(tgts, importTarget{File: t.File, Exports: t.Exports})
		}
		imports = append(imports, importCtx{From: im.From, Targets: tgts})
	}
	return nodes, edges, routes, imports, o.Note, nil
}

// ---------------- rendering (capped neighborhood, spec §2 + §1 colors) ----------------

func (g *graph) render(cap int) string {
	// Production neighborhood only: test nodes are excluded (spec §2 cap).
	changed, dependents, callees := g.partition()

	included := map[string]bool{}

	if len(changed) > cap {
		// truncate even changed nodes when a PR is very large; warn on stderr.
		for _, k := range changed[:cap] {
			included[k] = true
		}
		g.notes = append(g.notes, fmt.Sprintf("graph capped at %d changed symbols; %d more changed symbols not drawn.", cap, len(changed)-cap))
	} else {
		for _, k := range changed {
			included[k] = true
		}
		budget := cap - len(changed)
		for _, k := range dependents {
			if budget > 0 {
				included[k] = true
				budget--
			}
		}
		perNodeCallOverflow := map[string]int{}
		for _, k := range callees {
			if budget > 0 {
				included[k] = true
				budget--
			} else {
				// attribute callee overflow to each changed source that calls it
				for _, e := range g.edges {
					if e.kind == "calls" && e.to == k && included[e.from] {
						perNodeCallOverflow[e.from]++
					}
				}
			}
		}
		g.callOverflow = perNodeCallOverflow
	}

	var b strings.Builder
	b.WriteString("```mermaid\n")
	b.WriteString("flowchart LR\n")
	// spec §1 color convention via classDef — matches architecture-impact baseline.
	b.WriteString("  classDef add fill:#d7f5dd,stroke:#2e7d32,stroke-width:2px,color:#1b5e20;\n")
	b.WriteString("  classDef mod fill:#ffe8b3,stroke:#b8860b,stroke-width:2px,color:#7a5a00;\n")
	b.WriteString("  classDef rem fill:#ffd6d6,stroke:#c62828,stroke-width:2px,color:#b71c1c;\n")
	b.WriteString("  classDef ctx fill:#eeeeee,stroke:#9e9e9e,color:#555;\n")

	// nodes (stable order) — test nodes never drawn.
	idOf := map[string]string{}
	idx := 0
	for _, k := range g.order {
		if !included[k] || g.nodes[k].isTest {
			continue
		}
		idx++
		id := fmt.Sprintf("n%d", idx)
		idOf[k] = id
		n := g.nodes[k]
		cls := n.change
		if cls == "" {
			cls = classCtx
		}
		b.WriteString(fmt.Sprintf("  %s[%q]:::%s;\n", id, mermaidLabel(n), cls))
	}

	// edges between included, non-test nodes only.
	for _, e := range g.edges {
		fn, tn := g.nodes[e.from], g.nodes[e.to]
		if !included[e.from] || !included[e.to] || (fn != nil && fn.isTest) || (tn != nil && tn.isTest) {
			continue
		}
		// dashed = unexpected (spec §1): unresolved/external callee target.
		arrow := "-->"
		if tn != nil && strings.HasPrefix(tn.key, "ext::") {
			arrow = "-.->"
		}
		b.WriteString(fmt.Sprintf("  %s %s %s;\n", idOf[e.from], arrow, idOf[e.to]))
	}

	// "… +N more" callee-overflow summaries (spec §2).
	for src, n := range g.callOverflow {
		if n <= 0 || !included[src] {
			continue
		}
		idx++
		ov := fmt.Sprintf("n%d", idx)
		b.WriteString(fmt.Sprintf("  %s[%q]:::ctx;\n", ov, fmt.Sprintf("… +%d more callees", n)))
		b.WriteString(fmt.Sprintf("  %s -.-> %s;\n", idOf[src], ov))
	}
	b.WriteString("```\n\n")
	// one-line markdown legend — the sole non-diagram line in the block.
	b.WriteString("🟩 green = PR addition (new growth) · 🟨 amber = PR modification (graft point) · 🟥 red = removal · ⬜ gray = existing code\n")
	return b.String()
}

// partition returns (changed, dependent, callee) NON-TEST node keys in stable order.
func (g *graph) partition() (changed, deps, calls []string) {
	for _, k := range g.order {
		n := g.nodes[k]
		if n.isTest {
			continue
		}
		if n.change != classCtx && n.change != "" {
			changed = append(changed, k)
		}
	}
	depSet, callSet := map[string]bool{}, map[string]bool{}
	for _, e := range g.edges {
		switch e.kind {
		case "called-by", "imports":
			// e.from is the dependent (caller/importer)
			if d := g.nodes[e.from]; d != nil && d.change == classCtx && !d.isTest {
				depSet[e.from] = true
			}
		case "calls":
			if c := g.nodes[e.to]; c != nil && c.change == classCtx && !c.isTest {
				callSet[e.to] = true
			}
		}
	}
	for _, k := range g.order {
		if depSet[k] {
			deps = append(deps, k)
		}
	}
	for _, k := range g.order {
		if callSet[k] {
			calls = append(calls, k)
		}
	}
	return
}

// ---------------- root-system view (prototype §5) ----------------
//
// renderRoots is the additive-PR view: a rooted flowchart TD that surfaces the
// ESTABLISHED gray roots (parent layouts, sibling routes, reused import
// targets) the PR grafts onto, marks the graft point amber, and adds a legend.
// Layered on the SAME static analysis as blast — no new analyzer, no new deps
// (§5.3). Faithfulness (§5.4): every gray node is a real file at BASE_REF or a
// real import target of a changed file; nothing speculative.

func (g *graph) renderRoots(cap int) string {
	// 1. Surface existing context (§5.2.2): shells/layouts/siblings/import-targets.
	g.surfaceRootsContext()

	changed, ctx := g.partitionRoots()
	included, capNotes := g.capRoots(changed, ctx, cap)
	g.notes = append(g.notes, capNotes...)

	// declaration order: spine (shells+layouts) → changed growth → gray context.
	var spine, sibCtx, tgtCtx, otherCtx, declOrder []string
	for _, k := range ctx {
		n := g.nodes[k]
		switch {
		case strings.HasPrefix(k, "ctx::shell::") || isRouteLayout(n.file):
			spine = append(spine, k)
		case isRoutePage(n.file):
			sibCtx = append(sibCtx, k)
		case isImportTargetKey(k, g.imports):
			tgtCtx = append(tgtCtx, k)
		default:
			otherCtx = append(otherCtx, k)
		}
	}
	declOrder = append(declOrder, spine...)
	declOrder = append(declOrder, changed...)
	declOrder = append(declOrder, sibCtx...)
	declOrder = append(declOrder, tgtCtx...)
	declOrder = append(declOrder, otherCtx...)

	var b strings.Builder
	b.WriteString("```mermaid\n")
	b.WriteString("flowchart TD\n") // §5.2.1 rooted top-down
	b.WriteString("  classDef add fill:#d7f5dd,stroke:#2e7d32,stroke-width:2px,color:#1b5e20;\n")
	b.WriteString("  classDef mod fill:#ffe8b3,stroke:#b8860b,stroke-width:2px,color:#7a5a00;\n")
	b.WriteString("  classDef rem fill:#ffd6d6,stroke:#c62828,stroke-width:2px,color:#b71c1c;\n")
	b.WriteString("  classDef ctx fill:#eeeeee,stroke:#9e9e9e,color:#555;\n")

	idOf := map[string]string{}
	idx := 0
	for _, k := range declOrder {
		if !included[k] {
			continue
		}
		n := g.nodes[k]
		if n.isTest {
			continue
		}
		idx++
		id := fmt.Sprintf("n%d", idx)
		idOf[k] = id
		cls := n.change
		if cls == "" {
			cls = classCtx
		}
		b.WriteString(fmt.Sprintf("  %s[%q]:::%s;\n", id, rootsLabel(n), cls))
	}

	// edges (unlabeled — attachment topology over edge-type precision, §4): both
	// endpoints included & non-test. Dedup identical from→to (multiple changed
	// routes share a layout/shell edge).
	seen := map[string]bool{}
	for _, e := range g.edges {
		fn, tn := g.nodes[e.from], g.nodes[e.to]
		if !included[e.from] || !included[e.to] || (fn != nil && fn.isTest) || (tn != nil && tn.isTest) {
			continue
		}
		pair := e.from + ">" + e.to
		if seen[pair] {
			continue
		}
		seen[pair] = true
		b.WriteString(fmt.Sprintf("  %s --> %s;\n", idOf[e.from], idOf[e.to]))
	}

	b.WriteString("```\n\n")
	// §5.2.4 one-line markdown legend — the sole non-diagram line in the block.
	b.WriteString("🟩 green = PR addition (new growth) · 🟨 amber = PR modification (graft point) · 🟥 red = removal · ⬜ gray = existing code\n")
	return b.String()
}

// surfaceRootsContext adds the established-root gray nodes + graft edges the
// blast view lacks (prototype §3 failures 1–3). Non-clobbering: never
// downgrades a real change node to ctx, never overwrites its label, so a
// modified +layout.svelte stays the amber graft point.
func (g *graph) surfaceRootsContext() {
	for _, r := range g.routes {
		if r.ParentLayout == "" {
			continue
		}
		layKey := "ts::" + r.ParentLayout
		g.ensureCtx(layKey, shortRouteFile(r.ParentLayout), r.ParentLayout, "svelte", nil)

		// graft-point edge: parent layout → child route — the route-containment
		// edge today's import-only scanner misses (§3 failure 2 / §5.2.3).
		if childKey := "ts::" + r.File; g.hasNode(childKey) {
			g.edges = append(g.edges, edge{from: layKey, to: childKey, kind: "contains"})
		}
		// gray sibling routes sharing this layout (§5.2.2).
		for _, s := range r.Siblings {
			sibKey := "ts::" + s
			g.ensureCtx(sibKey, shortRouteFile(s), s, langOf(s), nil)
			g.edges = append(g.edges, edge{from: layKey, to: sibKey, kind: "contains"})
		}
		// conceptual shell root = the layout's route directory (§5.2.1 single root).
		shellDir := shellDirOf(r.ParentLayout) // routes/orchestration
		shellKey := "ctx::shell::" + shellDir
		g.ensureCtx(shellKey, lastSegment(shellDir)+" UI shell", shellDir, "svelte", nil)
		g.edges = append(g.edges, edge{from: shellKey, to: layKey, kind: "contains"})
	}
	// reused import targets (§5.2.2): changed file → existing gray module.
	for _, im := range g.imports {
		fromKey := "ts::" + im.From
		for _, t := range im.Targets {
			tgtKey := "ts::" + t.File
			g.ensureCtx(tgtKey, strings.TrimPrefix(t.File, "frontend/src/"), t.File, langOf(t.File), t.Exports)
			g.edges = append(g.edges, edge{from: fromKey, to: tgtKey, kind: "imports"})
		}
	}
}

// ensureCtx adds a ctx node only if absent — never clobbers an existing node.
func (g *graph) ensureCtx(key, label, file, lang string, exports []string) {
	if g.hasNode(key) {
		return
	}
	g.addNode(&node{key: key, label: label, file: file, lang: lang, change: classCtx, exports: exports})
}

// partitionRoots returns changed vs ctx non-test node keys in insertion order.
func (g *graph) partitionRoots() (changed, ctx []string) {
	for _, k := range g.order {
		n := g.nodes[k]
		if n.isTest {
			continue
		}
		if n.change != "" && n.change != classCtx {
			changed = append(changed, k)
		} else {
			ctx = append(ctx, k)
		}
	}
	return
}

// capRoots applies the 25-node budget with ≥6 slots reserved for established
// gray context (§5.2.2), prioritizing the structural spine (shells/layouts)
// when trimming.
func (g *graph) capRoots(changed, ctx []string, cap int) (map[string]bool, []string) {
	included := map[string]bool{}
	const minCtx = 6
	var spine, sib, tgt, other []string
	for _, k := range ctx {
		n := g.nodes[k]
		switch {
		case strings.HasPrefix(k, "ctx::shell::") || isRouteLayout(n.file):
			spine = append(spine, k)
		case isRoutePage(n.file):
			sib = append(sib, k)
		case isImportTargetKey(k, g.imports):
			tgt = append(tgt, k)
		default:
			other = append(other, k)
		}
	}
	gray := append(append(append(spine, sib...), tgt...), other...)
	if len(changed)+len(ctx) <= cap {
		for _, k := range changed {
			included[k] = true
		}
		for _, k := range gray {
			included[k] = true
		}
		return included, nil
	}
	changedShown := len(changed)
	if changedShown > cap-minCtx {
		changedShown = cap - minCtx
	}
	if changedShown < 0 {
		changedShown = 0
	}
	grayBudget := cap - changedShown
	for i, k := range changed {
		if i < changedShown {
			included[k] = true
		}
	}
	for i, k := range gray {
		if i < grayBudget {
			included[k] = true
		}
	}
	var notes []string
	if len(changed) > changedShown {
		notes = append(notes, fmt.Sprintf("graph capped at %d nodes; %d more changed symbols not drawn.", cap, len(changed)-changedShown))
	}
	return included, notes
}

// chooseView implements §5.1 auto-selection: roots for additive PRs (≥70%
// additions) that have honest established context to surface; blast otherwise.
// Explicit --view roots|blast always wins. No graph mutation — reads only.
func chooseView(req string, g *graph) string {
	switch req {
	case "roots", "blast":
		return req
	}
	changed, additions := 0, 0
	for _, k := range g.order {
		n := g.nodes[k]
		if n.isTest || n.change == "" || n.change == classCtx {
			continue
		}
		changed++
		if n.change == classAdd {
			additions++
		}
	}
	if changed == 0 || float64(additions)/float64(changed) < 0.70 {
		return "blast"
	}
	if !canSurfaceRootsCtx(g) {
		return "blast"
	}
	return "roots"
}

// canSurfaceRootsCtx reports whether surfaceRootsContext can honestly add gray
// established context. Gated to FRONTEND only (PR #14 review M1 / decision A):
// surfaceRootsContext has no backend context path, so auto-roots on an additive
// backend PR would drop blast's richer/honest callee + overflow signal for a
// misleadingly information-poorer view. Additive backend PRs auto-select blast;
// --view roots stays manually available for backend if ever wanted.
func canSurfaceRootsCtx(g *graph) bool {
	for _, r := range g.routes {
		if r.ParentLayout != "" {
			return true
		}
	}
	for _, im := range g.imports {
		if len(im.Targets) > 0 {
			return true
		}
	}
	return false
}

func isImportTargetKey(key string, imports []importCtx) bool {
	for _, im := range imports {
		for _, t := range im.Targets {
			if "ts::"+t.File == key {
				return true
			}
		}
	}
	return false
}

// ---------------- root-system path/label helpers ----------------

func isRoutePage(f string) bool {
	return strings.HasSuffix(f, "+page.svelte") && strings.Contains(f, "/routes/")
}
func isRouteLayout(f string) bool {
	return strings.HasSuffix(f, "+layout.svelte") && strings.Contains(f, "/routes/")
}

// routePath: frontend/src/routes/orchestration/live/+page.svelte -> /orchestration/live
func routePath(file string) string {
	const pfx = "frontend/src/routes/"
	if !strings.HasPrefix(file, pfx) {
		return file
	}
	rel := strings.TrimPrefix(file, pfx)
	for _, sfx := range []string{"/+page.svelte", "/+layout.svelte", "+page.svelte", "+layout.svelte"} {
		rel = strings.TrimSuffix(rel, sfx)
	}
	rel = strings.TrimSuffix(rel, "/")
	if rel == "" {
		return "/"
	}
	return "/" + rel
}

// shortRouteFile: .../orchestration/live/+page.svelte -> live/+page.svelte
func shortRouteFile(file string) string {
	rel := strings.TrimPrefix(file, "frontend/src/routes/")
	parts := strings.Split(rel, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-2] + "/" + parts[len(parts)-1]
	}
	return rel
}

func shellDirOf(layoutFile string) string {
	return strings.TrimPrefix(fileDir(layoutFile), "frontend/src/")
}

func fileDir(p string) string {
	if i := strings.LastIndexByte(p, '/'); i >= 0 {
		return p[:i]
	}
	return p
}

func lastSegment(p string) string {
	p = strings.TrimRight(p, "/")
	if i := strings.LastIndexByte(p, '/'); i >= 0 {
		return p[i+1:]
	}
	return p
}

func langOf(file string) string {
	if strings.HasSuffix(file, ".svelte") {
		return "svelte"
	}
	return "ts"
}

// rootsLabel: path-derived, informative labels for the root-system view.
func rootsLabel(n *node) string {
	if strings.HasPrefix(n.key, "ctx::shell::") {
		return fmt.Sprintf("%s UI shell<br/>%s/", lastSegment(n.file), n.file)
	}
	switch {
	case isRouteLayout(n.file), isRoutePage(n.file):
		if len(n.exports) > 0 {
			return fmt.Sprintf("%s<br/>%s · [%s]", shortRouteFile(n.file), routePath(n.file), strings.Join(n.exports, ", "))
		}
		return fmt.Sprintf("%s<br/>%s", shortRouteFile(n.file), routePath(n.file))
	default:
		rel := strings.TrimPrefix(n.file, "frontend/src/")
		if len(n.exports) > 0 {
			return fmt.Sprintf("%s<br/>[%s]", rel, strings.Join(n.exports, ", "))
		}
		if n.lang == "go" && n.label != "" {
			return fmt.Sprintf("%s<br/>%s", n.label, shortFile(n.file))
		}
		return rel
	}
}

// ---------------- helpers ----------------

func mermaidLabel(n *node) string {
	var s string
	if len(n.exports) > 0 {
		// M1: <br/> not \n — Mermaid v11 renders \n as literal text inside quoted
		// labels; <br/> is the reliable line break. %q in the renderer leaves it
		// unescaped (no special chars), so it reaches GitHub Markdown intact.
		s = fmt.Sprintf("%s<br/>[%s]", n.file, strings.Join(n.exports, ", "))
	} else {
		s = n.label
	}
	if n.file != "" && n.lang == "go" {
		s = fmt.Sprintf("%s<br/>%s", n.label, shortFile(n.file))
	}
	return s
}

func shortFile(f string) string {
	// keep last two path segments for readability
	parts := strings.Split(f, "/")
	if len(parts) >= 3 {
		return strings.Join(parts[len(parts)-3:], "/")
	}
	return f
}

func isTestFile(f string) bool {
	return strings.HasSuffix(f, "_test.go") ||
		strings.HasSuffix(f, ".test.ts") ||
		strings.HasSuffix(f, ".spec.ts")
}

func methodTail(canon string) string {
	if i := strings.LastIndexByte(canon, '.'); i >= 0 {
		return canon[i+1:]
	}
	return canon
}

func within(line, start, end int) bool { return line >= start && line <= end }

// sortedByLine returns func keys sorted by source start line then canon, for
// deterministic, reproducible output (map iteration is randomized in Go).
func sortedByLine(m map[string]*funcInfo) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if m[keys[i]].start != m[keys[j]].start {
			return m[keys[i]].start < m[keys[j]].start
		}
		return keys[i] < keys[j]
	})
	return keys
}

func intersects(start, end int, hunks []hunk) bool {
	for _, h := range hunks {
		if end >= h.start && start <= h.end {
			return true
		}
	}
	return false
}

type hunk struct{ start, end int }

// diffHunks returns the HEAD-side changed line ranges (the + lines) of a file.
func diffHunks(base, head, file string) ([]hunk, error) {
	out, err := gitOutput("diff", "--unified=0", base+"..."+head, "--", file)
	if err != nil {
		return nil, err
	}
	return parseHunks(out), nil
}

// parseHunks extracts the HEAD-side (+) line ranges from a unified diff. Pure
// (no git) so it is unit-testable on synthetic @@ -a,b +c,d @@ strings.
func parseHunks(diff string) []hunk {
	var hs []hunk
	for _, line := range strings.Split(diff, "\n") {
		if !strings.HasPrefix(line, "@@") {
			continue
		}
		// @@ -a,b +c,d @@  — we want the +c..(+c+d-1) range.
		i := strings.Index(line, "+")
		if i < 0 {
			continue
		}
		rest := line[i+1:]
		j := strings.IndexAny(rest, " @")
		if j < 0 {
			j = len(rest)
		}
		plus := rest[:j]
		parts := strings.SplitN(plus, ",", 2)
		start, _ := strconv.Atoi(parts[0])
		count := 1
		if len(parts) == 2 {
			count, _ = strconv.Atoi(parts[1])
		}
		if count == 0 {
			continue // pure deletion hunk has no + lines
		}
		hs = append(hs, hunk{start: start, end: start + count - 1})
	}
	return hs
}

func changedFiles(base, head string) ([]string, error) {
	out, err := gitOutput("diff", "--name-only", base+"..."+head)
	if err != nil {
		return nil, err
	}
	var fs []string
	for _, l := range strings.Split(out, "\n") {
		l = strings.TrimSpace(l)
		if l != "" {
			fs = append(fs, l)
		}
	}
	return fs, nil
}

func gitShow(ref, file string) ([]byte, error) {
	return gitOutputBytes("show", ref+":"+file)
}

type grepHit struct {
	file string
	line int
}

func gitGrep(pattern, head, pathspec string) ([]grepHit, error) {
	// M2: ref-qualify to HEAD_REF — `git grep <rev> -- <pathspec>` searches that
	// rev's tree, not the working tree. In Phase 3 CI the working tree is `main`
	// with pr-head as a ref only, so a working-tree grep would use wrong callers.
	out, err := gitOutput("grep", "-n", "-E", "--no-color", pattern, head, "--", pathspec)
	if err != nil {
		return nil, nil // no matches -> git grep exits 1; treat as empty.
	}
	// rev-mode output is `<head>:file:line:content` — strip the `<head>:` prefix
	// so file/line parse the same as working-tree mode.
	prefix := head + ":"
	var hits []grepHit
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		line = strings.TrimPrefix(line, prefix)
		// form: file:line:content (paths in this tree have no ':' so first colon is the sep)
		c1 := strings.IndexByte(line, ':')
		c2 := strings.IndexByte(line[c1+1:], ':')
		if c1 < 0 || c2 < 0 {
			continue
		}
		file := line[:c1]
		ln, _ := strconv.Atoi(line[c1+1 : c1+1+c2])
		hits = append(hits, grepHit{file: file, line: ln})
	}
	return hits, nil
}

func gitExists() bool { _, err := exec.LookPath("git"); return err == nil }

func gitOutput(args ...string) (string, error) {
	b, err := gitOutputBytes(args...)
	return string(b), err
}

func gitOutputBytes(args ...string) ([]byte, error) {
	var stdout bytes.Buffer
	cmd := exec.Command("git", args...)
	cmd.Stdout = &stdout
	err := cmd.Run()
	return bytes.TrimSpace(stdout.Bytes()), err
}

func dedupFuncs(in []funcInfo) []funcInfo {
	seen := map[string]bool{}
	var out []funcInfo
	for _, f := range in {
		k := f.canon + "|" + f.file
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, f)
	}
	return out
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// joinDir joins the first path segment of exe with the rest (rough dirname join).
func joinDir(exe string, parts ...string) string {
	dir := exe
	if i := strings.LastIndexByte(exe, '/'); i >= 0 {
		dir = exe[:i]
	}
	return strings.Join(append([]string{dir}, parts...), "/")
}
