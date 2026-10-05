package main

import (
	"path/filepath"
	"runtime"
	"sort"
	"testing"

	"github.com/tandetat/wire-unused/internal/graph"
	"github.com/tandetat/wire-unused/internal/loader"
	"github.com/tandetat/wire-unused/internal/parser"
)

func testdataDir(name string) string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(
		filepath.Dir(file), "testdata", name,
	)
}

type testResult struct {
	funcName    string
	numSets     int
	numProvs    int
	unusedSets  []string
	unusedProvs []string
}

func runAnalysis(
	t *testing.T, dir string,
) testResult {
	t.Helper()

	pkgs, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(pkgs) == 0 {
		t.Fatal("no packages loaded")
	}

	pkg := pkgs[0]
	injectors, err := parser.FindInjectors(pkg)
	if err != nil {
		t.Fatalf("find injectors: %v", err)
	}
	if len(injectors) == 0 {
		t.Fatal("no injectors found")
	}

	inj := injectors[0]
	args, err := parser.ClassifyArgs(pkg, inj)
	if err != nil {
		t.Fatalf("classify: %v", err)
	}

	resolved, err := parser.ResolveAll(pkg, args)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	tg := graph.Build(resolved)
	unused := tg.FindUnused()

	var unusedSetNames []string
	for _, s := range unused.Sets {
		unusedSetNames = append(
			unusedSetNames, s.QualifiedName)
	}
	sort.Strings(unusedSetNames)

	var unusedProvNames []string
	for _, p := range unused.Providers {
		unusedProvNames = append(
			unusedProvNames, p.QualifiedName)
	}
	sort.Strings(unusedProvNames)

	return testResult{
		funcName:    inj.FuncName,
		numSets:     len(args.ProviderSets),
		numProvs:    len(args.StandaloneProviders),
		unusedSets:  unusedSetNames,
		unusedProvs: unusedProvNames,
	}
}

func TestSimpleUnused(t *testing.T) {
	r := runAnalysis(t, testdataDir("simple_unused"))

	if r.funcName != "InitializeDeps" {
		t.Errorf(
			"func = %q, want InitializeDeps",
			r.funcName)
	}
	if r.numSets != 2 {
		t.Errorf("sets = %d, want 2", r.numSets)
	}

	want := []string{"pkg_b.ProviderSet"}
	assertSliceEqual(t, "unused sets",
		r.unusedSets, want)
	assertSliceEqual(t, "unused provs",
		r.unusedProvs, nil)
}

func TestAllUsed(t *testing.T) {
	r := runAnalysis(t, testdataDir("all_used"))

	if r.numSets != 2 {
		t.Errorf("sets = %d, want 2", r.numSets)
	}

	assertSliceEqual(t, "unused sets",
		r.unusedSets, nil)
	assertSliceEqual(t, "unused provs",
		r.unusedProvs, nil)
}

func TestMultipleUnused(t *testing.T) {
	r := runAnalysis(t, testdataDir("multiple_unused"))

	if r.numSets != 3 {
		t.Errorf("sets = %d, want 3", r.numSets)
	}

	want := []string{
		"pkg_b.ProviderSet",
		"pkg_c.ProviderSet",
	}
	assertSliceEqual(t, "unused sets",
		r.unusedSets, want)
}

func TestNestedSets(t *testing.T) {
	r := runAnalysis(t, testdataDir("nested_sets"))

	if r.numSets != 2 {
		t.Errorf("sets = %d, want 2", r.numSets)
	}

	want := []string{"pkg_c.ProviderSet"}
	assertSliceEqual(t, "unused sets",
		r.unusedSets, want)
}

func TestBindAndStruct(t *testing.T) {
	r := runAnalysis(t, testdataDir("bind_and_struct"))

	if r.numSets != 3 {
		t.Errorf("sets = %d, want 3", r.numSets)
	}

	want := []string{"pkg_c.ProviderSet"}
	assertSliceEqual(t, "unused sets",
		r.unusedSets, want)
}

func TestStandaloneUnused(t *testing.T) {
	r := runAnalysis(t, testdataDir("standalone_unused"))

	if r.numProvs != 2 {
		t.Errorf("provs = %d, want 2", r.numProvs)
	}

	assertSliceEqual(t, "unused sets",
		r.unusedSets, nil)
	assertSliceEqual(t, "unused provs",
		r.unusedProvs, []string{"pkg_b.NewServiceB"})
}

func assertSliceEqual(
	t *testing.T,
	label string,
	got, want []string,
) {
	t.Helper()
	if len(got) == 0 {
		got = nil
	}
	if len(want) == 0 {
		want = nil
	}
	if len(got) != len(want) {
		t.Errorf("%s: got %v, want %v",
			label, got, want)
		return
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("%s[%d]: got %q, want %q",
				label, i, got[i], want[i])
		}
	}
}
