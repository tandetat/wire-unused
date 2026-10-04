package pkg_a

import "github.com/google/wire"

var ProviderSet = wire.NewSet(NewServiceA)

type ServiceA struct{}

func NewServiceA() *ServiceA {
	return &ServiceA{}
}
