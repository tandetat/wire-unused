package pkg_b

import (
	"example.com/bind_and_struct/pkg_a"
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(NewServiceB)

type ServiceB struct {
	Repo pkg_a.Repository
}

func NewServiceB(repo pkg_a.Repository) *ServiceB {
	return &ServiceB{Repo: repo}
}
