package domain

import "testing"

func TestNewServiceLifecyclePlan_EmptyNamesIsNoOp(t *testing.T) {
	plan, err := NewServiceLifecyclePlan(LifecycleOperationStart, nil)
	if err != nil {
		t.Fatalf("empty start plan: %v", err)
	}
	if plan.Operation != LifecycleOperationStart {
		t.Fatalf("operation = %q, want start", plan.Operation)
	}
	if len(plan.OrderedNames) != 0 {
		t.Fatalf("ordered names = %#v, want empty", plan.OrderedNames)
	}

	plan, err = NewServiceLifecyclePlan(LifecycleOperationStop, []string{"", "  "})
	if err != nil {
		t.Fatalf("blank-only stop plan: %v", err)
	}
	if plan.Operation != LifecycleOperationStop {
		t.Fatalf("operation = %q, want stop", plan.Operation)
	}
	if len(plan.OrderedNames) != 0 {
		t.Fatalf("ordered names = %#v, want empty", plan.OrderedNames)
	}
}

func TestNewServiceLifecyclePlan_UnknownOperation(t *testing.T) {
	_, err := NewServiceLifecyclePlan("restart", []string{"svc"})
	if err == nil {
		t.Fatal("expected error for unknown operation")
	}
}
