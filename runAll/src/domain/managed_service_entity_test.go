package domain

import (
	"strings"
	"testing"
)

func TestManagedService_CanStop(t *testing.T) {
	service, err := NewManagedService("api", "platform", ServiceStatusHealthy, nil)
	if err != nil {
		t.Fatalf("NewManagedService: %v", err)
	}

	if err := service.CanStop(nil); err != nil {
		t.Fatalf("CanStop should allow healthy service without dependents: %v", err)
	}

	err = service.CanStop([]string{"web"})
	if err == nil || !strings.Contains(err.Error(), "active downstream dependencies") {
		t.Fatalf("CanStop should reject active dependents, got: %v", err)
	}

	retrying, err := NewManagedService("api", "platform", ServiceStatusRetrying, nil)
	if err != nil {
		t.Fatalf("NewManagedService: %v", err)
	}
	if err := retrying.CanStop(nil); err != nil {
		t.Fatalf("CanStop should allow retrying service without dependents: %v", err)
	}
}

func TestManagedService_CanStart(t *testing.T) {
	stopped, err := NewManagedService("api", "platform", ServiceStatusStopped, nil)
	if err != nil {
		t.Fatalf("NewManagedService: %v", err)
	}
	if !stopped.CanStart() {
		t.Fatal("stopped service should be startable")
	}

	healthy, err := NewManagedService("api", "platform", ServiceStatusHealthy, nil)
	if err != nil {
		t.Fatalf("NewManagedService: %v", err)
	}
	if healthy.CanStart() {
		t.Fatal("healthy service should not be startable")
	}

	for _, status := range []string{
		ServiceStatusFailed,
		ServiceStatusSkipped,
		ServiceStatusPending,
	} {
		service, err := NewManagedService("api", "platform", status, nil)
		if err != nil {
			t.Fatalf("NewManagedService(%q): %v", status, err)
		}
		if !service.CanStart() {
			t.Fatalf("%q service should be startable", status)
		}
	}

	// Retrying means a launch is already in flight: the global predicate must keep
	// excluding it so cascade/start-all planning never double-binds a service that
	// another goroutine is launching. Only the explicit single-service Start path
	// may reclaim Retrying (see runner claimStartForOperator / TestStartService_FromRetrying).
	retrying, err := NewManagedService("api", "platform", ServiceStatusRetrying, nil)
	if err != nil {
		t.Fatalf("NewManagedService(%q): %v", ServiceStatusRetrying, err)
	}
	if retrying.CanStart() {
		t.Fatal("retrying service must not be startable via the global predicate (in-flight launch); only explicit single-service Start may reclaim it")
	}
}
