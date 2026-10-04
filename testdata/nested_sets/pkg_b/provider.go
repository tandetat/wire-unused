package pkg_b

import (
	"example.com/nested_sets/pkg_a"
	"github.com/google/wire"
)

// ProviderSet includes pkg_a.ProviderSet nested inside.
var ProviderSet = wire.NewSet(
	pkg_a.ProviderSet,
	NewServiceB,
)

type ServiceB struct {
	A *pkg_a.ServiceA
}

func NewServiceB(a *pkg_a.ServiceA) *ServiceB {
	return &ServiceB{A: a}
}
