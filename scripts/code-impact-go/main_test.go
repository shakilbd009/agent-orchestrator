package main

import (
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
