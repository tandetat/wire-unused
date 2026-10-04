package graph

import (
	"go/types"
	"sort"
	"strings"

	"github.com/tandetat/wire-unused/internal/parser"
)

// UnusedSet describes an unused provider set.
type UnusedSet struct {
	QualifiedName string
	PkgPath       string
	Provides      string
	Line          int
}

// UnusedProvider describes an unused standalone provider.
type UnusedProvider struct {
	QualifiedName string
	Provides      string
	Line          int
}

// UnusedResult holds all unused providers/sets.
type UnusedResult struct {
	Sets      []UnusedSet
	Providers []UnusedProvider
}

// FindUnused performs DFS from target types to find all
// needed providers, then reports those not reached.
func (g *TypeGraph) FindUnused() *UnusedResult {
	needed := g.findNeeded()
	result := &UnusedResult{}

	for setRef, providers := range g.setProviders {
		allUnused := true
		var providedTypes []string
		for _, p := range providers {
			if needed[p] {
				allUnused = false
				break
			}
			for _, out := range p.Outputs {
				providedTypes = append(
					providedTypes,
					shortTypeName(out))
			}
		}
		if allUnused && len(providers) > 0 {
			result.Sets = append(result.Sets,
				UnusedSet{
					QualifiedName: setRef.QualifiedName(),
					PkgPath:       setRef.PkgPath,
					Provides: strings.Join(
						providedTypes, ", "),
					Line: setRef.Line,
				},
			)
		}
	}

	sort.Slice(result.Sets, func(i, j int) bool {
		return result.Sets[i].QualifiedName <
			result.Sets[j].QualifiedName
	})

	return result
}

// findNeeded walks backwards from target types through
// the graph, marking all transitively needed providers.
func (g *TypeGraph) findNeeded() map[*parser.ResolvedProvider]bool {
	needed := make(map[*parser.ResolvedProvider]bool)
	visited := make(map[types.Type]bool)

	for _, target := range g.targets {
		underlying := target.TargetType.Underlying()
		st, ok := underlying.(*types.Struct)
		if !ok {
			continue
		}

		allFields := len(target.Fields) == 1 &&
			target.Fields[0] == "*"

		for i := 0; i < st.NumFields(); i++ {
			field := st.Field(i)
			if !field.Exported() {
				continue
			}
			if allFields || containsField(
				target.Fields, field.Name()) {
				g.markNeeded(
					field.Type(), needed, visited)
			}
		}
	}

	return needed
}

func containsField(fields []string, name string) bool {
	for _, f := range fields {
		if f == name {
			return true
		}
	}
	return false
}

// markNeeded recursively marks the provider for typ
// (and all its transitive deps) as needed.
func (g *TypeGraph) markNeeded(
	typ types.Type,
	needed map[*parser.ResolvedProvider]bool,
	visited map[types.Type]bool,
) {
	if visited[typ] {
		return
	}
	visited[typ] = true

	resolved := g.resolveType(typ)
	if resolved != typ {
		visited[resolved] = true
	}

	p := g.providerFor(typ)
	if p == nil {
		p = g.providerFor(resolved)
	}
	if p == nil {
		return
	}

	needed[p] = true

	for _, dep := range g.deps[p] {
		g.markNeeded(dep, needed, visited)
	}
}

func shortTypeName(typ types.Type) string {
	s := types.TypeString(typ, func(pkg *types.Package) string {
		name := pkg.Name()
		path := pkg.Path()
		parts := strings.Split(path, "/")
		if len(parts) > 0 {
			last := parts[len(parts)-1]
			if last != name {
				return name
			}
		}
		return name
	})
	return s
}
