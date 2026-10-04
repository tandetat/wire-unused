//go:build wireinject

package multiple_unused

import (
	"example.com/multiple_unused/pkg_a"
	"example.com/multiple_unused/pkg_b"
	"example.com/multiple_unused/pkg_c"
	"github.com/google/wire"
)

type Deps struct {
	A *pkg_a.ServiceA
}

func InitializeDeps() (*Deps, error) {
	wire.Build(
		pkg_a.ProviderSet,
		pkg_b.ProviderSet,
		pkg_c.ProviderSet,
		wire.Struct(new(Deps), "*"),
	)
	return nil, nil
}
