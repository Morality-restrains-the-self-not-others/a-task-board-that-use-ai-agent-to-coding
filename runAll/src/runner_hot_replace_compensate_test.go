package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"runAll/src/domain"
	"runAll/src/infrastructure"
)

func newCompensateTestRunner(t *testing.T, names []string, withOwnership bool) (*Runner, *StatusStore, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	store := NewStatusStore()
	services := make([]Service, 0, len(names))
	for _, n := range names {
		services = append(services, Service{
			Name:    n,
			Command: "sleep 30",
			HealthCheck: HealthCheck{
				URL:           srv.URL,
				Timeout:       5,
				Retries:       2,
				CheckInterval: 1,
				Backoff:       Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5},
			},
		})
	}
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups:  []Group{{Name: "g1", Services: services}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	stubNoPortListenersForTest(runner)

	repo := infrastructure.NewFileServiceOwnershipRepository(filepath.Join(t.TempDir(), "ownership.json"))
	runner.ownershipRepo = repo
	runner.ownershipGuard = domain.NewServiceOwnershipGuardService(repo)

	if withOwnership {
		for _, n := range names {
			own, err := domain.NewServiceOwnership(n, "prev-session", os.Getpid(), defaultOwnershipConfigRef, srv.URL, time.Now())
			if err != nil {
				t.Fatalf("NewServiceOwnership(%s): %v", n, err)
			}
			if err := repo.Save(own); err != nil {
				t.Fatalf("Save ownership(%s): %v", n, err)
			}
		}
	}
	return runner, store, srv
}

func TestCompensateHotReplaceDownServices_AutoStartsDownServiceWithOwnership(t *testing.T) {
	runner, store, _ := newCompensateTestRunner(t, []string{"comp-a"}, true)
	// 热替换后 store 回填前该服务保持 pending（adopt 未收养）。
	if st := store.Get("comp-a"); st == nil || st.Status != StatusPending {
		t.Fatalf("pre-status=%v, want pending", st)
	}

	n := runner.compensateHotReplaceDownServices(context.Background())
	if n != 1 {
		t.Fatalf("compensated=%d, want 1 (status=%v)", n, store.Get("comp-a"))
	}
	got := store.Get("comp-a")
	if got == nil || got.Status != StatusHealthy {
		t.Fatalf("post-status=%v, want healthy", got)
	}
	// 补偿后应重建 monitoring。
	runner.monitorMu.Lock()
	_, monitorExists := runner.monitors["comp-a"]
	runner.monitorMu.Unlock()
	if !monitorExists {
		t.Fatal("expected monitor to be resumed after compensation")
	}

	if err := runner.StopService(context.Background(), "comp-a"); err != nil {
		t.Fatalf("cleanup StopService: %v", err)
	}
}

func TestCompensateHotReplaceDownServices_SkipsServiceWithoutOwnership(t *testing.T) {
	runner, store, _ := newCompensateTestRunner(t, []string{"comp-no-own"}, false)

	n := runner.compensateHotReplaceDownServices(context.Background())
	if n != 0 {
		t.Fatalf("compensated=%d, want 0 (no ownership = never started / manually stopped)", n)
	}
	if st := store.Get("comp-no-own"); st == nil || st.Status != StatusPending {
		t.Fatalf("status=%v, want still pending", st)
	}
}

func TestCompensateHotReplaceDownServices_SkipsAlreadyHealthy(t *testing.T) {
	runner, store, srv := newCompensateTestRunner(t, []string{"comp-up"}, true)
	store.Update("comp-up", StatusHealthy, "")

	n := runner.compensateHotReplaceDownServices(context.Background())
	if n != 0 {
		t.Fatalf("compensated=%d, want 0 (already healthy)", n)
	}
	if st := store.Get("comp-up"); st == nil || st.Status != StatusHealthy {
		t.Fatalf("status=%v, want healthy", st)
	}
	_ = srv
}

func TestCompensateHotReplaceDownServices_SkipsServiceWithoutProbe(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{
			Name: "g1",
			Services: []Service{{
				Name:    "comp-noprobe",
				Command: "sleep 30",
				// 无健康探针：不参与补偿，维持人工 start-all。
			}},
		}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	stubNoPortListenersForTest(runner)
	repo := infrastructure.NewFileServiceOwnershipRepository(filepath.Join(t.TempDir(), "ownership.json"))
	runner.ownershipRepo = repo
	runner.ownershipGuard = domain.NewServiceOwnershipGuardService(repo)
	own, err := domain.NewServiceOwnership("comp-noprobe", "prev-session", os.Getpid(), defaultOwnershipConfigRef, srv.URL, time.Now())
	if err != nil {
		t.Fatalf("NewServiceOwnership: %v", err)
	}
	if err := repo.Save(own); err != nil {
		t.Fatalf("Save: %v", err)
	}

	n := runner.compensateHotReplaceDownServices(context.Background())
	if n != 0 {
		t.Fatalf("compensated=%d, want 0 (no probe configured)", n)
	}
}
