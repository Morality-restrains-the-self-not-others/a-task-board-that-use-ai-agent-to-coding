package domain

import (
	"testing"
)

func TestNewBuildGroupResult_ok(t *testing.T) {
	result, err := NewBuildGroupResult(
		BuildGroupStatusOK,
		3, 3,
		nil, nil, nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != BuildGroupStatusOK {
		t.Errorf("expected status ok, got %s", result.Status)
	}
	if result.Total != 3 {
		t.Errorf("expected total 3, got %d", result.Total)
	}
	if result.Built != 3 {
		t.Errorf("expected built 3, got %d", result.Built)
	}
}

func TestNewBuildGroupResult_partial(t *testing.T) {
	result, err := NewBuildGroupResult(
		BuildGroupStatusPartial,
		5, 3,
		[]string{"svc-a", "svc-b"},
		[]string{"svc-c"},
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != BuildGroupStatusPartial {
		t.Errorf("expected status partial, got %s", result.Status)
	}
	if len(result.Failed) != 2 {
		t.Errorf("expected 2 failed, got %d", len(result.Failed))
	}
	if len(result.Skipped) != 1 {
		t.Errorf("expected 1 skipped, got %d", len(result.Skipped))
	}
}

func TestNewBuildGroupResult_none(t *testing.T) {
	result, err := NewBuildGroupResult(
		BuildGroupStatusNone,
		0, 0,
		nil, nil,
		[]string{"svc-x", "svc-y"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != BuildGroupStatusNone {
		t.Errorf("expected status none, got %s", result.Status)
	}
	if len(result.NoBuild) != 2 {
		t.Errorf("expected 2 no_build, got %d", len(result.NoBuild))
	}
}

func TestNewBuildGroupResult_invalidStatus(t *testing.T) {
	_, err := NewBuildGroupResult("unknown", 0, 0, nil, nil, nil)
	if err == nil {
		t.Error("expected error for invalid status")
	}
}

func TestNewBuildGroupResult_negativeTotal(t *testing.T) {
	_, err := NewBuildGroupResult(BuildGroupStatusNone, -1, 0, nil, nil, nil)
	if err == nil {
		t.Error("expected error for negative total")
	}
}

func TestNewBuildGroupResult_builtExceedsTotal(t *testing.T) {
	_, err := NewBuildGroupResult(BuildGroupStatusOK, 1, 5, nil, nil, nil)
	if err == nil {
		t.Error("expected error for built > total")
	}
}

func TestComputeBuildGroupStatus_ok(t *testing.T) {
	status := ComputeBuildGroupStatus(3, 3, nil)
	if status != BuildGroupStatusOK {
		t.Errorf("expected ok, got %s", status)
	}
}

func TestComputeBuildGroupStatus_partial(t *testing.T) {
	status := ComputeBuildGroupStatus(5, 3, []string{"a"})
	if status != BuildGroupStatusPartial {
		t.Errorf("expected partial, got %s", status)
	}
}

func TestComputeBuildGroupStatus_none(t *testing.T) {
	status := ComputeBuildGroupStatus(0, 0, nil)
	if status != BuildGroupStatusNone {
		t.Errorf("expected none, got %s", status)
	}
}

func TestIsTerminalBuildStatus(t *testing.T) {
	tests := []struct {
		status   string
		expected bool
	}{
		{ServiceStatusHealthy, true},
		{ServiceStatusFailed, true},
		{ServiceStatusStopped, true},
		{ServiceStatusBuilding, false},    // only building is excluded
		{ServiceStatusStarting, true},     // buildable — compilation is disk-only
		{ServiceStatusRetrying, true},     // buildable — compilation is disk-only
		{ServiceStatusRestarting, true},   // buildable — compilation is disk-only
		{ServiceStatusPending, true},      // buildable — compilation is disk-only
		{ServiceStatusSkipped, true},      // buildable — compilation is disk-only
	}
	for _, tt := range tests {
		got := IsTerminalBuildStatus(tt.status)
		if got != tt.expected {
			t.Errorf("IsTerminalBuildStatus(%q) = %v, want %v", tt.status, got, tt.expected)
		}
	}
}
