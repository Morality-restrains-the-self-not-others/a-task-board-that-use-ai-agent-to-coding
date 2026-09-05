package main

import "testing"

func TestMachineRuntimeCountsAsStarted(t *testing.T) {
	cases := []struct {
		id, status string
		want       bool
	}{
		{"i-1", "Running", true},
		{"i-1", "running", true},
		{"mock-abc", "", true},
		{"mock-abc", "Starting", true},
		{"i-1", "Starting", false},
		{"i-1", "Pending", false},
		{"i-1", "Initializing", false},
		{"i-1", "Stopped", false},
		{"i-1", "Released", false},
		{"", "Running", false},
		{"i-1", "", true}, // legacy unknown until reconciled
	}
	for _, tc := range cases {
		if got := machineRuntimeCountsAsStarted(tc.id, tc.status); got != tc.want {
			t.Fatalf("started(%q,%q)=%v want %v", tc.id, tc.status, got, tc.want)
		}
	}
}

func TestMachineRuntimeCountsAsStarting(t *testing.T) {
	if !machineRuntimeCountsAsStarting("i-1", "Starting") {
		t.Fatal("Starting should count as starting")
	}
	if machineRuntimeCountsAsStarting("i-1", "Running") {
		t.Fatal("Running must not count as starting")
	}
	if machineRuntimeCountsAsStarting("mock-x", "Starting") {
		t.Fatal("mock is always started, not starting")
	}
}

func TestMachineRuntimeIsReleasedOrMissing(t *testing.T) {
	if !machineRuntimeIsReleasedOrMissing("", false) {
		t.Fatal("not found => released")
	}
	if !machineRuntimeIsReleasedOrMissing("Released", true) {
		t.Fatal("Released status")
	}
	if machineRuntimeIsReleasedOrMissing("Running", true) {
		t.Fatal("Running is not released")
	}
}
