package main

import (
	"context"
	"errors"
	"testing"

	domainservices "valueStream/domain/services"
)

type stubImpactEvaluator struct {
	results []domainservices.StreamImpactResult
	err     error
}

func (s *stubImpactEvaluator) Evaluate(context.Context, []domainservices.ImpactStream) ([]domainservices.StreamImpactResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.results, nil
}

func TestStatusStore_FailedStepsFollowYAMLOrder(t *testing.T) {
	cfg := &Config{
		ValueStreams: []ValueStream{{
			Name:   "flow-a",
			Domain: "auth",
			Steps: []Step{
				{Name: "step-z", TestFile: "z.py"},
				{Name: "step-a", TestFile: "a.py"},
				{Name: "step-m", TestFile: "m.py"},
			},
		}},
	}
	store := NewStatusStore(cfg)
	store.BeginStreamRun("flow-a")
	store.FinishStep("flow-a", "step-z", TestRun{Passed: false, ExitCode: 1})
	store.FinishStep("flow-a", "step-a", TestRun{Passed: false, ExitCode: 1})
	store.FinishStep("flow-a", "step-m", TestRun{Passed: true, ExitCode: 0})
	store.EndStreamRun("flow-a")

	view := store.StreamView("flow-a")
	want := []string{"step-z", "step-a"}
	if len(view.FailedSteps) != len(want) {
		t.Fatalf("failed_steps = %v, want %v", view.FailedSteps, want)
	}
	for i := range want {
		if view.FailedSteps[i] != want[i] {
			t.Fatalf("failed_steps = %v, want YAML order %v", view.FailedSteps, want)
		}
	}
}

func TestStatusStore_StreamRunSetsFailedSteps(t *testing.T) {
	cfg := minimalConfig()
	store := NewStatusStore(cfg)
	store.BeginStreamRun("flow-a")
	store.SetStepRunning("flow-a", "s1")
	store.FinishStep("flow-a", "s1", TestRun{Passed: true, ExitCode: 0})
	store.SetStepRunning("flow-a", "s2")
	store.FinishStep("flow-a", "s2", TestRun{Passed: false, ExitCode: 1, LogTail: "fail"})
	store.SetStepRunning("flow-a", "s3")
	store.FinishStep("flow-a", "s3", TestRun{Passed: true, ExitCode: 0})
	store.EndStreamRun("flow-a")

	view := store.StreamView("flow-a")
	if view.StreamStatus != StreamStatusFailed {
		t.Fatalf("stream_status = %q, want failed", view.StreamStatus)
	}
	if len(view.FailedSteps) != 1 || view.FailedSteps[0] != "s2" {
		t.Fatalf("failed_steps = %v, want [s2]", view.FailedSteps)
	}
}

func TestStatusStore_SingleStepDoesNotChangeStreamStatus(t *testing.T) {
	cfg := minimalConfig()
	store := NewStatusStore(cfg)
	if store.StreamView("flow-a").StreamStatus != StreamStatusUnknown {
		t.Fatalf("initial = %q", store.StreamView("flow-a").StreamStatus)
	}
	store.SetStepRunning("flow-a", "s1")
	store.FinishStep("flow-a", "s1", TestRun{Passed: true, ExitCode: 0})
	view := store.StreamView("flow-a")
	if view.StreamStatus != StreamStatusUnknown {
		t.Fatalf("after single step stream_status = %q, want unknown", view.StreamStatus)
	}
	if view.Steps[0].Status != StepStatusPassed {
		t.Fatalf("step status = %q", view.Steps[0].Status)
	}
}

func TestStatusStore_StepFieldsInView(t *testing.T) {
	cfg := &Config{
		ValueStreams: []ValueStream{{
			Name:   "f",
			Domain: "platform",
			Steps: []Step{{
				Name:     "s1",
				TestFile: "a.py",
				Fields: []StepField{{
					Name: "saas-backend.auth_user.email",
				}},
			}},
		}},
	}
	store := NewStatusStore(cfg)
	resp := store.AllStreams()
	if resp.Streams[0].Domain != "platform" {
		t.Fatalf("domain = %q, want platform", resp.Streams[0].Domain)
	}
	if len(resp.Streams[0].Steps[0].Fields) != 1 {
		t.Fatalf("fields len = %d", len(resp.Streams[0].Steps[0].Fields))
	}
	if resp.Streams[0].Steps[0].Fields[0].ProviderService != "saas-backend" {
		t.Fatalf("provider = %q", resp.Streams[0].Steps[0].Fields[0].ProviderService)
	}
}

func TestStatusStore_AllStreamsTrimsDomainWhitespace(t *testing.T) {
	cfg := &Config{
		ValueStreams: []ValueStream{{
			Name:   "flow-a",
			Domain: "  auth  ",
			Steps: []Step{
				{Name: "s1", TestFile: "a.py"},
			},
		}},
	}
	store := NewStatusStore(cfg)

	resp := store.AllStreams()
	if got := resp.Streams[0].Domain; got != "auth" {
		t.Fatalf("domain = %q, want auth", got)
	}
}

func TestStatusStore_BeginStreamRunResetsStepStatuses(t *testing.T) {
	cfg := minimalConfig()
	store := NewStatusStore(cfg)
	store.FinishStep("flow-a", "s1", TestRun{Passed: true, ExitCode: 0})
	store.FinishStep("flow-a", "s2", TestRun{Passed: false, ExitCode: 1})

	store.BeginStreamRun("flow-a")

	view := store.StreamView("flow-a")
	if view.StreamStatus != StreamStatusRunning {
		t.Fatalf("stream_status = %q, want running", view.StreamStatus)
	}
	for _, step := range view.Steps {
		if step.Status != StepStatusPending {
			t.Fatalf("step %q status = %q, want pending", step.Name, step.Status)
		}
	}
}

func TestStatusStore_EndStreamRunListsNonPassedStepsInOrder(t *testing.T) {
	cfg := minimalConfig()
	store := NewStatusStore(cfg)
	store.BeginStreamRun("flow-a")
	store.FinishStep("flow-a", "s1", TestRun{Passed: false, ExitCode: 1})
	store.EndStreamRun("flow-a")

	view := store.StreamView("flow-a")
	want := []string{"s1", "s2", "s3"}
	if len(view.FailedSteps) != len(want) {
		t.Fatalf("failed_steps = %v, want %v", view.FailedSteps, want)
	}
	for i := range want {
		if view.FailedSteps[i] != want[i] {
			t.Fatalf("failed_steps = %v, want %v", view.FailedSteps, want)
		}
	}
}

func TestStatusStore_PlannedStepKeepsPlannedAndNotAffectStreamResult(t *testing.T) {
	cfg := &Config{
		ValueStreams: []ValueStream{{
			Name:   "flow-a",
			Domain: "auth",
			Steps: []Step{
				{Name: "future", Lifecycle: "planned", TestFile: "future.py"},
				{Name: "active", Lifecycle: "active", TestFile: "active.py"},
			},
		}},
	}
	store := NewStatusStore(cfg)

	view := store.StreamView("flow-a")
	if view.Steps[0].Status != StepStatusPlanned {
		t.Fatalf("planned step initial status = %q, want planned", view.Steps[0].Status)
	}

	store.BeginStreamRun("flow-a")
	store.FinishStep("flow-a", "active", TestRun{Passed: true, ExitCode: 0})
	store.EndStreamRun("flow-a")

	view = store.StreamView("flow-a")
	if view.StreamStatus != StreamStatusPassed {
		t.Fatalf("stream status = %q, want passed", view.StreamStatus)
	}
	if len(view.FailedSteps) != 0 {
		t.Fatalf("failed_steps = %v, want empty", view.FailedSteps)
	}
	if view.Steps[0].Status != StepStatusPlanned {
		t.Fatalf("planned step status = %q, want planned", view.Steps[0].Status)
	}
}

func TestStatusStore_AllStreamsAppliesKnownImpactAndImpactedSteps(t *testing.T) {
	cfg := minimalConfig()
	evaluator := &stubImpactEvaluator{
		results: []domainservices.StreamImpactResult{{
			Name:          "flow-a",
			Impacted:      true,
			ImpactStatus:  domainservices.ImpactStatusKnown,
			ImpactedSteps: []string{"s1", "s3"},
			Steps: []domainservices.StepImpactResult{
				{Name: "s1", Impacted: true, Reason: "step test_file changed in HEAD"},
				{Name: "s2", Impacted: false, Reason: "step test_file not changed in HEAD"},
				{Name: "s3", Impacted: true, Reason: "step test_file changed in HEAD"},
			},
		}},
	}
	store := NewStatusStore(cfg, evaluator)

	view := store.StreamView("flow-a")
	if view.ImpactStatus != string(domainservices.ImpactStatusKnown) {
		t.Fatalf("impact_status = %q, want known", view.ImpactStatus)
	}
	if !view.Impacted {
		t.Fatalf("impacted = false, want true")
	}
	wantImpactedSteps := []string{"s1", "s3"}
	if len(view.ImpactedSteps) != len(wantImpactedSteps) {
		t.Fatalf("impacted_steps = %v, want %v", view.ImpactedSteps, wantImpactedSteps)
	}
	for i := range wantImpactedSteps {
		if view.ImpactedSteps[i] != wantImpactedSteps[i] {
			t.Fatalf("impacted_steps = %v, want %v", view.ImpactedSteps, wantImpactedSteps)
		}
	}
	if !view.Steps[0].Impacted || view.Steps[0].ImpactReason == "" {
		t.Fatalf("step s1 impact = %+v, want impacted with reason", view.Steps[0])
	}
	if view.Steps[1].Impacted {
		t.Fatalf("step s2 impacted = true, want false")
	}
}

func TestStatusStore_AllStreamsFallsBackToUnknownImpact(t *testing.T) {
	cfg := minimalConfig()
	evaluator := &stubImpactEvaluator{err: errors.New("git show failed")}
	store := NewStatusStore(cfg, evaluator)

	view := store.StreamView("flow-a")
	if view.ImpactStatus != string(domainservices.ImpactStatusUnknown) {
		t.Fatalf("impact_status = %q, want unknown", view.ImpactStatus)
	}
	if view.Impacted {
		t.Fatalf("impacted = true, want false")
	}
	if len(view.ImpactedSteps) != 0 {
		t.Fatalf("impacted_steps = %v, want empty", view.ImpactedSteps)
	}
	for _, step := range view.Steps {
		if step.Impacted {
			t.Fatalf("step %q impacted = true, want false", step.Name)
		}
	}
}

func minimalConfig() *Config {
	return &Config{
		ValueStreams: []ValueStream{{
			Name:   "flow-a",
			Domain: "auth",
			Steps: []Step{
				{Name: "s1", TestFile: "a.py"},
				{Name: "s2", TestFile: "b.py"},
				{Name: "s3", TestFile: "c.py"},
			},
		}},
	}
}
