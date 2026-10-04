// Package graph builds a type dependency graph from
// resolved wire providers and determines reachability.
package graph

import (
	"go/types"

	"github.com/tandetat/wire-unused/internal/parser"
	"golang.org/x/tools/go/types/typeutil"
)

// TypeGraph maps types to their providers and tracks
// the dependency relationships between them.
type TypeGraph struct {
	// type -> provider that produces it
	providers typeutil.Map
	// provider -> types it needs as input
	deps map[*parser.ResolvedProvider][]types.Type
	// interface -> concrete type (from wire.Bind)
	bindings typeutil.Map
	// target struct providers (wire.Struct)
	targets []*parser.StructProvider
	// all provider sets and their providers
	setProviders map[*parser.ProviderSetRef][]*parser.ResolvedProvider
}

// Build constructs a TypeGraph from resolved args.
func Build(resolved *parser.ResolvedArgs) *TypeGraph {
	g := &TypeGraph{
		deps: make(
			map[*parser.ResolvedProvider][]types.Type,
		),
		targets:      resolved.Targets,
		setProviders: resolved.SetProviders,
	}

	for _, p := range resolved.Providers {
		for _, out := range p.Outputs {
			g.providers.Set(out, p)
		}
		g.deps[p] = p.Inputs
	}

	for _, b := range resolved.Bindings {
		g.bindings.Set(b.Iface, b.Concrete)
	}

	return g
}

// resolveType follows bindings to find the concrete type
// that satisfies a given type.
func (g *TypeGraph) resolveType(
	typ types.Type,
) types.Type {
	if concrete := g.bindings.At(typ); concrete != nil {
		return concrete.(types.Type)
	}
	return typ
}

// providerFor finds the provider that outputs typ.
func (g *TypeGraph) providerFor(
	typ types.Type,
) *parser.ResolvedProvider {
	v := g.providers.At(typ)
	if v == nil {
		return nil
	}
	return v.(*parser.ResolvedProvider)
}
