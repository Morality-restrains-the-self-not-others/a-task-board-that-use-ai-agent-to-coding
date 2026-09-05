package repositories

import (
	"context"

	"valueStream/domain/entities"
	"valueStream/domain/value_objects"
)

// ValueStreamRepository 定义配置上下文的持久化契约。
type ValueStreamRepository interface {
	List(ctx context.Context) ([]entities.ValueStream, error)
	SaveDomainOrder(ctx context.Context, order value_objects.DomainOrder, streams []entities.ValueStream) error
}

