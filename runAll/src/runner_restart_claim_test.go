package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClaimServiceRestart_AllowsRetryingAndSkipped(t *testing.T) {
	t.Parallel()
	for _, from := range []Status{
		StatusHealthy, StatusFailed, StatusStopped, StatusPending,
		StatusRetrying, StatusSkipped,
	} {
		from := from
		t.Run(string(from), func(t *testing.T) {
			t.Parallel()
			store := NewStatusStore()
			store.Init([]string{"svc"})
			store.Update("svc", from, "prev")
			got, err := claimServiceRestart(store, "svc")
			if err != nil {
				t.Fatalf("claim from %s: %v", from, err)
			}
			if got != from {
				t.Fatalf("previous = %q, want %q", got, from)
			}
			st := store.Get("svc")
			if st == nil || st.Status != StatusRestarting {
				t.Fatalf("status = %v, want restarting", st)
			}
		})
	}
}

func TestClaimServiceRestart_RejectsInFlightLaunch(t *testing.T) {
	t.Parallel()
	for _, from := range []Status{StatusStarting, StatusRestarting, StatusBuilding} {
		from := from
		t.Run(string(from), func(t *testing.T) {
			t.Parallel()
			store := NewStatusStore()
			store.Init([]string{"svc"})
			store.Update("svc", from, "")
			_, err := claimServiceRestart(store, "svc")
			if err == nil {
				t.Fatalf("claim from %s: want error", from)
			}
			if !strings.Contains(err.Error(), string(from)) {
				t.Fatalf("error %q should mention %s", err, from)
			}
			if store.Get("svc").Status != from {
				t.Fatalf("status mutated on rejected claim")
			}
		})
	}
}

func TestStartupAlreadyClaimed_RetryingOnlyWithForceFreshStart(t *testing.T) {
	t.Parallel()
	if !startupAlreadyClaimed(&ServiceStatus{Status: StatusRestarting}, false) {
		t.Fatal("restarting is pre-claimed")
	}
	if !startupAlreadyClaimed(&ServiceStatus{Status: StatusRetrying}, true) {
		t.Fatal("forceFreshStart after canary fallback must own Retrying")
	}
	if startupAlreadyClaimed(&ServiceStatus{Status: StatusRetrying}, false) {
		t.Fatal("Retrying without forceFreshStart must not skip the start CAS")
	}
	if startupAlreadyClaimed(&ServiceStatus{Status: StatusFailed}, true) {
		t.Fatal("Failed must go through transitionServiceToStarting")
	}
}

func TestShouldCanaryOverlap_OnlyServingOrFailedLiveProcess(t *testing.T) {
	t.Parallel()
	if !shouldCanaryOverlap(StatusHealthy, true) {
		t.Fatal("healthy + live process must overlap")
	}
	if !shouldCanaryOverlap(StatusFailed, true) {
		t.Fatal("failed + live process must overlap (alive-but-unhealthy restart)")
	}
	if shouldCanaryOverlap(StatusRetrying, true) {
		t.Fatal("retrying is mid-health-check; overlap would double-bind the port")
	}
	if shouldCanaryOverlap(StatusHealthy, false) {
		t.Fatal("no live process: cannot overlap")
	}
}

func TestRestartService_FromRetryingDoesNotReject(t *testing.T) {
	health := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(health.Close)

	const name = "svc-retrying-restart"
	cfg := &Config{Groups: []Group{{Services: []Service{{
		Name:    name,
		Command: "sleep 30",
		HealthCheck: HealthCheck{
			URL: health.URL, Timeout: 2, Retries: 2, CheckInterval: 1,
			Backoff: Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5},
		},
	}}}}}
	runner, store := testPreciseRestartRunner(t, cfg)
	store.Init([]string{name})
	store.Update(name, StatusRetrying, "waiting for health")
	t.Cleanup(func() { runner.stopProcess(name) })

	if err := runner.restartService(context.Background(), name); err != nil {
		t.Fatalf("restart from retrying: %v", err)
	}
	st := store.Get(name)
	if st == nil || st.Status != StatusHealthy {
		t.Fatalf("status after restart = %v, want healthy", st)
	}
}
