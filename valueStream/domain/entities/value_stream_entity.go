package entities

import (
	"fmt"
	"strings"

	"valueStream/domain/value_objects"
)

type ValueStreamID string

// ValueStream 是配置上下文中的核心实体。
type ValueStream struct {
	id     ValueStreamID
	name   string
	domain value_objects.DomainName
}

func NewValueStream(id, name string, domain value_objects.DomainName) (ValueStream, error) {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return ValueStream{}, fmt.Errorf("value stream name cannot be empty")
	}
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return ValueStream{}, fmt.Errorf("value stream id cannot be empty")
	}
	return ValueStream{
		id:     ValueStreamID(trimmedID),
		name:   trimmedName,
		domain: domain,
	}, nil
}

func (v ValueStream) ID() ValueStreamID {
	return v.id
}

func (v ValueStream) Name() string {
	return v.name
}

func (v ValueStream) Domain() value_objects.DomainName {
	return v.domain
}

