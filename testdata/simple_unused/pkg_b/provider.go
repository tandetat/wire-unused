package pkg_b

import "github.com/google/wire"

var ProviderSet = wire.NewSet(NewServiceB)

type ServiceB struct{}

func NewServiceB() *ServiceB {
	return &ServiceB{}
}
