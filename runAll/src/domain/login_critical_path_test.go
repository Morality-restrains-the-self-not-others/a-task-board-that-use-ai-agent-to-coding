package domain

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeStarter struct {
	running map[string]bool
	fail    map[string]error
	starts  []string
}

func (f *fakeStarter) IsServiceRunning(name string) bool {
	return f.running[name]
}

func (f *fakeStarter) StartService(_ context.Context, name string) error {
	f.starts = append(f.starts, name)
	if err, ok := f.fail[name]; ok && err != nil {
		return err
	}
	if f.running == nil {
		f.running = map[string]bool{}
	}
	f.running[name] = true
	return nil
}

func TestEnsureLoginCriticalPath_StartsMissingInOrder(t *testing.T) {
	f := &fakeStarter{running: map[string]bool{}}
	var progress []string
	started, failed, err := EnsureLoginCriticalPath(context.Background(), f, func(msg string) {
		progress = append(progress, msg)
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(failed) != 0 {
		t.Fatalf("failed=%v", failed)
	}
	want := []string{"task-auth", "task-gateway", "taskFE"}
	if strings.Join(started, ",") != strings.Join(want, ",") {
		t.Fatalf("started=%v want=%v", started, want)
	}
	if strings.Join(f.starts, ",") != strings.Join(want, ",") {
		t.Fatalf("start calls=%v want=%v", f.starts, want)
	}
	if len(progress) == 0 {
		t.Fatal("expected progress callbacks")
	}
}

func TestEnsureLoginCriticalPath_SkipsAlreadyRunning(t *testing.T) {
	f := &fakeStarter{running: map[string]bool{
		"task-auth":    true,
		"task-gateway": true,
	}}
	started, failed, err := EnsureLoginCriticalPath(context.Background(), f, nil)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(failed) != 0 {
		t.Fatalf("failed=%v", failed)
	}
	if len(f.starts) != 1 || f.starts[0] != "taskFE" {
		t.Fatalf("starts=%v want [taskFE]", f.starts)
	}
	if len(started) != 1 || started[0] != "taskFE" {
		t.Fatalf("started=%v", started)
	}
}

func TestEnsureLoginCriticalPath_PartialFailure(t *testing.T) {
	f := &fakeStarter{
		running: map[string]bool{},
		fail:    map[string]error{"task-gateway": errors.New("boom")},
	}
	started, failed, err := EnsureLoginCriticalPath(context.Background(), f, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if len(failed) != 1 || failed[0] != "task-gateway" {
		t.Fatalf("failed=%v", failed)
	}
	// Continues after failure: taskFE still attempted
	if len(f.starts) != 3 {
		t.Fatalf("starts=%v want 3 attempts", f.starts)
	}
	if len(started) != 2 { // auth + taskFE succeeded
		t.Fatalf("started=%v", started)
	}
}

func TestEnsureLoginCriticalPath_NilStarter(t *testing.T) {
	_, _, err := EnsureLoginCriticalPath(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoginCriticalPathServices_Order(t *testing.T) {
	want := []string{"task-auth", "task-gateway", "taskFE"}
	if strings.Join(LoginCriticalPathServices, ",") != strings.Join(want, ",") {
		t.Fatalf("got %v", LoginCriticalPathServices)
	}
}
