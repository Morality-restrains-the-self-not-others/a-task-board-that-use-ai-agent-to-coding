package main

import (
	"testing"
	"time"
)

func TestParseListeningPortsFromLsof(t *testing.T) {
	output := `runAll  123 user   48u  IPv4 111      0t0  TCP *:9999 (LISTEN)
python  456 user   10u  IPv4 222      0t0  TCP 127.0.0.1:3000 (LISTEN)
node    789 user   11u  IPv6 333      0t0  TCP [::1]:4000 (LISTEN)
`
	ports := parseListeningPortsFromLsof(output)
	for _, want := range []string{"9999", "3000", "4000"} {
		if _, ok := ports[want]; !ok {
			t.Fatalf("missing port %s in %#v", want, ports)
		}
	}
	if len(ports) != 3 {
		t.Fatalf("ports = %#v, want 3 entries", ports)
	}
}

func TestPrefillStartablePortProbeCache_UsesBatchSnapshot(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"svc-healthy", "svc-stopped"})
	store.Update("svc-healthy", StatusHealthy, "")
	store.Update("svc-stopped", StatusStopped, "")

	lsofCalls := 0
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{Name: "g1", Services: []Service{
			{Name: "svc-healthy", Command: "sleep 1", HealthCheck: HealthCheck{URL: "http://127.0.0.1:18001"}},
			{Name: "svc-stopped", Command: "sleep 1", HealthCheck: HealthCheck{URL: "http://127.0.0.1:18002"}},
		}}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	runner.listenerPIDsFn = func(string) ([]int, error) {
		lsofCalls++
		return nil, nil
	}

	services := store.All()
	runner.prefillStartablePortProbeCache(services)
	payload := buildStatusPayload(store, runner)
	if len(payload) != 2 {
		t.Fatalf("payload len = %d, want 2", len(payload))
	}
	if lsofCalls != 0 {
		t.Fatalf("per-port lsof calls = %d, want 0 (batch snapshot only)", lsofCalls)
	}
	for _, item := range payload {
		if item.Name == "svc-stopped" && item.ListenPortActive {
			t.Fatalf("stopped service with free port should report listen_port_active=false")
		}
	}
}

func TestListeningTCPPortsSnapshot_NoInPlaceMutationRace(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{Name: "g1", Services: []Service{
			{Name: "svc-a", Command: "sleep 1", HealthCheck: HealthCheck{URL: "http://127.0.0.1:19991"}},
		}}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	// Seed a published snapshot, then refresh while readers keep using the old map.
	old := map[string]struct{}{"19991": {}}
	runner.listeningPortsSnapMu.Lock()
	runner.listeningPortsSnap = old
	runner.listeningPortsSnapExpires = time.Now().Add(-time.Second) // force refresh
	runner.listeningPortsSnapMu.Unlock()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 10000; i++ {
			_ = serviceHasActivePortInSnapshot(&Service{
				Name:        "svc-a",
				HealthCheck: HealthCheck{URL: "http://127.0.0.1:19991"},
			}, old)
		}
	}()

	for i := 0; i < 100; i++ {
		runner.listeningPortsSnapMu.Lock()
		runner.listeningPortsSnapExpires = time.Now().Add(-time.Second)
		runner.listeningPortsSnapMu.Unlock()
		_ = runner.listeningTCPPortsSnapshot()
	}
	<-done
}
