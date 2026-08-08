package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"testing"
)

// M3: the smallest check that fails if the non-trivial classification/parser
// logic breaks. Pure functions only — no git, no fixtures, no suites (YAGNI).

func TestParseHunks(t *testing.T) {
	cases := []struct {
		name string
		diff string
		want []hunk
	}{
		{"simple", "@@ -1,3 +1,5 @@ ctx\n+x", []hunk{{1, 5}}},
		{"pure deletion has no +lines", "@@ -1,1 +1,0 @@", nil},
		{"single +line defaults count=1", "@@ -10,2 +12 @@", []hunk{{12, 12}}},
		{"multi hunk", "@@ -1,1 +5,2 @@ ctx\na\n@@ -20,1 +30,3 @@", []hunk{{5, 6}, {30, 32}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := parseHunks(c.diff)
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("parseHunks(%q) = %+v, want %+v", c.diff, got, c.want)
			}
		})
	}
}

func TestIntersects(t *testing.T) {
	cases := []struct {
		name        string
		start, end  int
		hunks       []hunk
		want        bool
	}{
		{"no overlap", 10, 20, []hunk{{1, 5}}, false},
		{"overlap", 10, 20, []hunk{{15, 25}}, true},
		{"boundary touch", 10, 20, []hunk{{20, 20}}, true},
		{"empty hunks", 10, 20, nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := intersects(c.start, c.end, c.hunks); got != c.want {
				t.Fatalf("intersects(%d,%d,%+v) = %v, want %v", c.start, c.end, c.hunks, got, c.want)
			}
		})
	}
}

func firstFunc(t *testing.T, src string) *ast.FuncDecl {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), "x.go", src, parser.ParseComments)
	if err != nil || f == nil {
		t.Fatalf("parse: %v", err)
	}
	for _, d := range f.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok {
			return fn
		}
	}
	t.Fatal("no func decl")
	return nil
}

func TestCallees(t *testing.T) {
	fn := firstFunc(t, `package p
func f(x int) {
	g(x)
	h.Foo()
	bar(1, 2)
}`)
	want := []string{"g", "Foo", "bar"}
	if got := callees(fn); !reflect.DeepEqual(got, want) {
		t.Fatalf("callees = %v, want %v", got, want)
	}
}

func TestClassifyFuncs(t *testing.T) {
	base := map[string]*funcInfo{
		"A": {canon: "A", start: 10, end: 20},
		"B": {canon: "B", start: 30, end: 40},
	}
	head := map[string]*funcInfo{
		"A": {canon: "A", start: 10, end: 20},
		"B": {canon: "B", start: 30, end: 40},
		"C": {canon: "C", start: 50, end: 60},
	}
	hunks := []hunk{{15, 18}} // touches A only; B untouched, C is new
	// A in both + span hits hunk -> mod; B in both, untouched -> omitted; C new -> add.
	want := map[string]string{"A": classMod, "C": classAdd}
	got := classifyFuncs(base, head, hunks)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("classifyFuncs = %v, want %v", got, want)
	}

	// removal: D only in base.
	got = classifyFuncs(map[string]*funcInfo{"D": {canon: "D"}}, nil, nil)
	if got["D"] != classRem {
		t.Fatalf("expected D=rem, got %v", got)
	}
}

// root-system view pure logic (prototype §5): view auto-selection (§5.1), route
// path derivation, and the ≥6-slot ctx reservation in the cap (§5.2.2).
func TestRoutePath(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"page nested", "frontend/src/routes/orchestration/live/+page.svelte", "/orchestration/live"},
		{"layout", "frontend/src/routes/orchestration/+layout.svelte", "/orchestration"},
		{"index page", "frontend/src/routes/orchestration/+page.svelte", "/orchestration"},
		{"root page", "frontend/src/routes/+page.svelte", "/"},
		{"non-route", "frontend/src/lib/api/client.ts", "frontend/src/lib/api/client.ts"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := routePath(c.in); got != c.want {
				t.Fatalf("routePath(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestChooseViewAuto(t *testing.T) {
	g := newGraph()
	// additive: 4 adds + 1 mod = 80% additions, with a route parent layout → roots.
	addNodes(g, []string{"a1", "a2", "a3", "a4"}, classAdd)
	addNodes(g, []string{"m1"}, classMod)
	g.routes = []routeCtx{{File: "x", ParentLayout: "frontend/src/routes/orchestration/+layout.svelte"}}
	if got := chooseView("auto", g); got != "roots" {
		t.Fatalf("additive+ctx: chooseView= %q, want roots", got)
	}
	// explicit always wins.
	if got := chooseView("blast", g); got != "blast" {
		t.Fatalf("explicit blast overridden: %q", got)
	}
	if got := chooseView("roots", g); got != "roots" {
		t.Fatalf("explicit roots overridden: %q", got)
	}
	// heavily-modifying: 1 add + 4 mod = 20% → blast even with ctx.
	g2 := newGraph()
	addNodes(g2, []string{"a1"}, classAdd)
	addNodes(g2, []string{"m1", "m2", "m3", "m4"}, classMod)
	g2.routes = []routeCtx{{File: "x", ParentLayout: "frontend/src/routes/orchestration/+layout.svelte"}}
	if got := chooseView("auto", g2); got != "blast" {
		t.Fatalf("modifying: chooseView= %q, want blast", got)
	}
	// additive but no honest context to surface → blast (graceful degrade).
	g3 := newGraph()
	addNodes(g3, []string{"a1", "a2", "a3"}, classAdd)
	if got := chooseView("auto", g3); got != "blast" {
		t.Fatalf("no-ctx: chooseView= %q, want blast", got)
	}
}

func TestCapRootsReservesCtx(t *testing.T) {
	g := newGraph()
	// 20 changed + 8 ctx (all shells so they are spine) under a 25 cap: changed is
	// trimmed to 19 but ctx keeps ≥6 slots (here all 8 fit since 19+8>25 → ctx=6).
	var changed []string
	for i := 0; i < 20; i++ {
		k := fmt.Sprintf("ts::c%d", i)
		g.addNode(&node{key: k, change: classAdd, lang: "ts"})
		changed = append(changed, k)
	}
	var ctx []string
	for i := 0; i < 8; i++ {
		k := fmt.Sprintf("ctx::shell::routes/d%d", i)
		g.addNode(&node{key: k, change: classCtx, file: "routes/d" + fmt.Sprint(i)})
		ctx = append(ctx, k)
	}
	included, _ := g.capRoots(changed, ctx, defaultCap)
	ctxShown := 0
	for _, k := range ctx {
		if included[k] {
			ctxShown++
		}
	}
	if ctxShown < 6 {
		t.Fatalf("ctx reserved = %d, want >=6 (§5.2.2)", ctxShown)
	}
	total := 0
	for _, v := range included {
		if v {
			total++
		}
	}
	if total > defaultCap {
		t.Fatalf("total included = %d, want <= cap %d", total, defaultCap)
	}
}

func addNodes(g *graph, keys []string, change string) {
	for _, k := range keys {
		g.addNode(&node{key: k, change: change, lang: "ts"})
	}
}
