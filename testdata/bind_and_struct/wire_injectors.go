//go:build wireinject

package bind_and_struct

import (
	"example.com/bind_and_struct/pkg_a"
	"example.com/bind_and_struct/pkg_b"
	"example.com/bind_and_struct/pkg_c"
	"github.com/google/wire"
)

type Deps struct {
	B *pkg_b.ServiceB
}

// pkg_a provides the Repository binding used by pkg_b.
// pkg_c is unused.
func InitializeDeps() (*Deps, error) {
	wire.Build(
		pkg_a.ProviderSet,
		pkg_b.ProviderSet,
		pkg_c.ProviderSet,
		wire.Struct(new(Deps), "*"),
	)
	return nil, nil
}
