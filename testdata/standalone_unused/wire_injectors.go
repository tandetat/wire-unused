//go:build wireinject

package standalone_unused

import (
	"example.com/standalone_unused/pkg_a"
	"example.com/standalone_unused/pkg_b"
	"github.com/google/wire"
)

type Deps struct {
	A *pkg_a.ServiceA
	C *pkg_b.ServiceC
}

func InitializeDeps() (*Deps, error) {
	wire.Build(
		pkg_a.ProviderSet,
		pkg_b.NewServiceB,
		pkg_b.NewServiceC,
		wire.Struct(new(Deps), "*"),
	)
	return nil, nil
}
