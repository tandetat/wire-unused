package pkg_a

import "github.com/google/wire"

type Repository interface {
	Find(id string) (string, error)
}

type DBRepository struct{}

func NewDBRepository() *DBRepository {
	return &DBRepository{}
}

func (r *DBRepository) Find(
	id string,
) (string, error) {
	return id, nil
}

var ProviderSet = wire.NewSet(
	NewDBRepository,
	wire.Bind(new(Repository), new(*DBRepository)),
)
