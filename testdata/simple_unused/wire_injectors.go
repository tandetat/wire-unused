//go:build wireinject

package simple_unused

import (
	"example.com/simple_unused/pkg_a"
	"example.com/simple_unused/pkg_b"
	"github.com/google/wire"
)

type Deps struct {
	A *pkg_a.ServiceA
}

func InitializeDeps() (*Deps, error) {
	wire.Build(
		pkg_a.ProviderSet,
		pkg_b.ProviderSet,
		wire.Struct(new(Deps), "*"),
	)
	return nil, nil
}
