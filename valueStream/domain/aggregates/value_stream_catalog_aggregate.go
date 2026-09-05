package aggregates

import (
	"fmt"
	"time"

	"valueStream/domain/entities"
	"valueStream/domain/events"
	"valueStream/domain/value_objects"
)

// ValueStreamCatalog 是业务域排序的一致性边界（聚合根）。
type ValueStreamCatalog struct {
	streams []entities.ValueStream
}

func NewValueStreamCatalog(streams []entities.ValueStream) (*ValueStreamCatalog, error) {
	if len(streams) == 0 {
		return nil, fmt.Errorf("value stream catalog cannot be empty")
	}
	copied := make([]entities.ValueStream, len(streams))
	copy(copied, streams)
	return &ValueStreamCatalog{streams: copied}, nil
}

func (c *ValueStreamCatalog) ReorderDomains(order value_objects.DomainOrder) (events.DomainOrderReordered, error) {
	byDomain := make(map[string][]entities.ValueStream)
	firstSeen := make([]string, 0)
	known := make(map[string]bool)
	for _, stream := range c.streams {
		domain := stream.Domain().Value()
		if !known[domain] {
			known[domain] = true
			firstSeen = append(firstSeen, domain)
		}
		byDomain[domain] = append(byDomain[domain], stream)
	}

	requested := order.Domains()
	requestedSet := make(map[string]bool, len(requested))
	reordered := make([]entities.ValueStream, 0, len(c.streams))
	for _, domain := range requested {
		name := domain.Value()
		if !known[name] {
			return events.DomainOrderReordered{}, fmt.Errorf("unknown domain %q", name)
		}
		requestedSet[name] = true
		reordered = append(reordered, byDomain[name]...)
	}
	for _, domain := range firstSeen {
		if requestedSet[domain] {
			continue
		}
		reordered = append(reordered, byDomain[domain]...)
	}
	c.streams = reordered
	orderedDomains := make([]string, 0, len(firstSeen))
	for _, stream := range c.streams {
		domain := stream.Domain().Value()
		if len(orderedDomains) == 0 || orderedDomains[len(orderedDomains)-1] != domain {
			orderedDomains = append(orderedDomains, domain)
		}
	}
	return events.DomainOrderReordered{
		OccurredAt:  time.Now(),
		DomainNames: orderedDomains,
	}, nil
}

func (c *ValueStreamCatalog) Streams() []entities.ValueStream {
	out := make([]entities.ValueStream, len(c.streams))
	copy(out, c.streams)
	return out
}

