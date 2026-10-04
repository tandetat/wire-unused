//go:build wireinject

package nested_sets

import (
	"example.com/nested_sets/pkg_b"
	"example.com/nested_sets/pkg_c"
	"github.com/google/wire"
)

type Deps struct {
	B *pkg_b.ServiceB
}

// pkg_b.ProviderSet nests pkg_a.ProviderSet, so both
// should be marked as used. pkg_c is unused.
func InitializeDeps() (*Deps, error) {
	wire.Build(
		pkg_b.ProviderSet,
		pkg_c.ProviderSet,
		wire.Struct(new(Deps), "*"),
	)
	return nil, nil
}
