package services

import (
	"context"
	"fmt"

	"valueStream/domain/aggregates"
	"valueStream/domain/events"
	"valueStream/domain/repositories"
	"valueStream/domain/value_objects"
)

// DomainOrderReorderService 编排“业务域改序”用例，不直接依赖基础设施实现。
type DomainOrderReorderService struct {
	repository repositories.ValueStreamRepository
}

func NewDomainOrderReorderService(repository repositories.ValueStreamRepository) *DomainOrderReorderService {
	return &DomainOrderReorderService{repository: repository}
}

func (s *DomainOrderReorderService) Reorder(ctx context.Context, rawDomainOrder []string) (events.DomainOrderReordered, error) {
	if s.repository == nil {
		return events.DomainOrderReordered{}, fmt.Errorf("value stream repository is required")
	}
	order, err := value_objects.NewDomainOrder(rawDomainOrder)
	if err != nil {
		return events.DomainOrderReordered{}, err
	}
	streams, err := s.repository.List(ctx)
	if err != nil {
		return events.DomainOrderReordered{}, err
	}
	catalog, err := aggregates.NewValueStreamCatalog(streams)
	if err != nil {
		return events.DomainOrderReordered{}, err
	}
	evt, err := catalog.ReorderDomains(order)
	if err != nil {
		return events.DomainOrderReordered{}, err
	}
	if err := s.repository.SaveDomainOrder(ctx, order, catalog.Streams()); err != nil {
		return events.DomainOrderReordered{}, err
	}
	return evt, nil
}

