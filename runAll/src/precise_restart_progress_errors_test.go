package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// 回归：精准编译重启在 failed>0 的中途进度事件必须携带 Errors，
// 否则 status UI 的 prog-errors-list（含「复制日志」按钮）一直 display:none。
func TestPreciseRestart_MidProgressEventsIncludeErrorsWhenFailed(t *testing.T) {
	healthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(healthServer.Close)

	cfg := &Config{
		Groups: []Group{{Services: []Service{
			{
				Name:        "svc-precise-ok",
				Command:     "sleep 30",
				HealthCheck: HealthCheck{URL: healthServer.URL, Timeout: 2, Retries: 2, CheckInterval: 1, Backoff: Backoff{Initial: 0.1, Max: 0.2, Multiplier: 1.5}},
			},
		}}},
	}
	runner, store := testPreciseRestartRunner(t, cfg)
	runner.progressBroadcaster = NewProgressBroadcaster()
	oldReleaseWait := servicePortReleaseWait
	servicePortReleaseWait = 300 * time.Millisecond
	t.Cleanup(func() { servicePortReleaseWait = oldReleaseWait })
	store.Init([]string{"svc-precise-ok"})
	store.Update("svc-precise-ok", StatusStopped, "")

	path := filepath.Join(t.TempDir(), "reg.txt")
	t.Setenv("RUNALL_PRECISE_RESTART_FILE", path)
	// 未知服务先计入 failed/allErrors，随后真实服务进度事件也必须带 Errors。
	if err := writeRegisteredServices(path, []string{"ghost-svc", "svc-precise-ok"}); err != nil {
		t.Fatal(err)
	}
	if !runner.TryBeginPreciseRestart("precise-restart-errors-test") {
		t.Fatal("TryBeginPreciseRestart failed")
	}
	defer runner.endPreciseRestart()

	ch := runner.progressBroadcaster.Subscribe("precise-restart-errors-test")
	defer runner.progressBroadcaster.Unsubscribe("precise-restart-errors-test", ch)

	done := make(chan struct{})
	var midFailedWithoutErrors []StartAllProgressEvent
	go func() {
		defer close(done)
		deadline := time.After(20 * time.Second)
		for {
			select {
			case <-deadline:
				return
			case ev, ok := <-ch:
				if !ok {
					return
				}
				if ev.Failed > 0 && len(ev.Errors) == 0 {
					midFailedWithoutErrors = append(midFailedWithoutErrors, ev)
				}
				if ev.Done {
					return
				}
			}
		}
	}()

	keep, err := runner.PreciseRestart(context.Background(), "test-session")
	if err != nil {
		t.Fatalf("PreciseRestart: %v", err)
	}
	<-done

	foundGhost := false
	for _, k := range keep {
		if k == "ghost-svc" {
			foundGhost = true
		}
	}
	if !foundGhost {
		t.Fatalf("keep = %v, want ghost-svc retained", keep)
	}
	if len(midFailedWithoutErrors) > 0 {
		var samples []string
		for i, ev := range midFailedWithoutErrors {
			if i >= 3 {
				break
			}
			samples = append(samples, ev.ToSSE())
		}
		t.Fatalf("found %d progress events with Failed>0 but empty Errors; samples:\n%s",
			len(midFailedWithoutErrors), strings.Join(samples, ""))
	}
}
