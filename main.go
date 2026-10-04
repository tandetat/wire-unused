package main

import (
	"fmt"
	"os"

	"github.com/tandetat/wire-unused/internal/graph"
	"github.com/tandetat/wire-unused/internal/loader"
	"github.com/tandetat/wire-unused/internal/parser"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr,
			"usage: wire-unused <package-pattern>\n")
		os.Exit(1)
	}
	pattern := os.Args[1]

	pkgs, err := loader.Load(pattern)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load: %v\n", err)
		os.Exit(1)
	}

	for _, pkg := range pkgs {
		injectors, err := parser.FindInjectors(pkg)
		if err != nil {
			fmt.Fprintf(os.Stderr,
				"parse %s: %v\n", pkg.PkgPath, err)
			continue
		}

		for _, inj := range injectors {
			args, err := parser.ClassifyArgs(pkg, inj)
			if err != nil {
				fmt.Fprintf(os.Stderr,
					"classify %s: %v\n",
					inj.FuncName, err)
				continue
			}

			resolved, err := parser.ResolveAll(
				pkg, args,
			)
			if err != nil {
				fmt.Fprintf(os.Stderr,
					"resolve %s: %v\n",
					inj.FuncName, err)
				continue
			}

			tg := graph.Build(resolved)

			unused := tg.FindUnused()

			printResult(inj, args, unused)
		}
	}
}

func printResult(
	inj *parser.Injector,
	args *parser.ClassifiedArgs,
	unused *graph.UnusedResult,
) {
	fmt.Printf("Analyzing: %s\n", inj.FuncName)
	fmt.Printf("  wire.Build() at %s:%d\n",
		inj.Filename, inj.Line)
	fmt.Println()
	fmt.Printf("  %d provider sets, %d standalone providers\n",
		len(args.ProviderSets),
		len(args.StandaloneProviders))
	fmt.Println()

	fmt.Println("  Unused provider sets:")
	if len(unused.Sets) == 0 {
		fmt.Println("    (none)")
	}
	for _, s := range unused.Sets {
		fmt.Printf("    - %s\n", s.QualifiedName)
		fmt.Printf("        (%s)\n", s.PkgPath)
		fmt.Printf("        provides: %s\n", s.Provides)
		fmt.Printf("        %s:%d\n", inj.Filename, s.Line)
	}

	fmt.Println()
	fmt.Println("  Unused providers:")
	if len(unused.Providers) == 0 {
		fmt.Println("    (none)")
	}
	for _, p := range unused.Providers {
		fmt.Printf("    - %s\n", p.QualifiedName)
		fmt.Printf("        provides: %s\n", p.Provides)
		fmt.Printf("        %s:%d\n", inj.Filename, p.Line)
	}
}
