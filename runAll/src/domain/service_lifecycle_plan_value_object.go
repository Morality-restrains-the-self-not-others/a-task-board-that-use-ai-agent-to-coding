package domain

import (
	"fmt"
	"strings"
)

const (
	LifecycleOperationStart         = "start"
	LifecycleOperationStop          = "stop"
	LifecycleOperationCanaryRestart = "canary-restart"
)

// ServiceLifecyclePlan is an ordered list of service names for cascade start or stop.
// An empty OrderedNames list is a valid no-op (already in the desired state).
type ServiceLifecyclePlan struct {
	Operation    string
	OrderedNames []string
}

func NewServiceLifecyclePlan(operation string, orderedNames []string) (ServiceLifecyclePlan, error) {
	operation = strings.TrimSpace(operation)
	switch operation {
	case LifecycleOperationStart, LifecycleOperationStop, LifecycleOperationCanaryRestart:
	default:
		return ServiceLifecyclePlan{}, fmt.Errorf("unknown lifecycle operation %q", operation)
	}
	names := make([]string, 0, len(orderedNames))
	seen := make(map[string]struct{}, len(orderedNames))
	for _, name := range orderedNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	if len(names) == 0 {
		// Empty plan is a valid no-op: everything already running / already stopped.
		return ServiceLifecyclePlan{Operation: operation, OrderedNames: names}, nil
	}
	return ServiceLifecyclePlan{
		Operation:    operation,
		OrderedNames: names,
	}, nil
}

func (p ServiceLifecyclePlan) String() string {
	return strings.Join(p.OrderedNames, " -> ")
}
