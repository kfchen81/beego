package vanilla

import (
	"context"
)

type RepositoryBase struct {
	Ctx context.Context
}

type ServiceBase struct {
	Ctx context.Context
}

type EntityBase struct {
	Ctx   context.Context
	Model interface{}
}
