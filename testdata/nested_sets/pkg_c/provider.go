package pkg_c

import "github.com/google/wire"

var ProviderSet = wire.NewSet(NewServiceC)

type ServiceC struct{}

func NewServiceC() *ServiceC {
	return &ServiceC{}
}
