package pkg_b

import (
	"example.com/all_used/pkg_a"
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(NewServiceB)

type ServiceB struct {
	A *pkg_a.ServiceA
}

func NewServiceB(a *pkg_a.ServiceA) *ServiceB {
	return &ServiceB{A: a}
}
