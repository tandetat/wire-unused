//go:build wireinject

package all_used

import (
	"example.com/all_used/pkg_a"
	"example.com/all_used/pkg_b"
	"github.com/google/wire"
)

type Deps struct {
	B *pkg_b.ServiceB
}

func InitializeDeps() (*Deps, error) {
	wire.Build(
		pkg_a.ProviderSet,
		pkg_b.ProviderSet,
		wire.Struct(new(Deps), "*"),
	)
	return nil, nil
}
