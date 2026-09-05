package domain

import "testing"

func TestMergeStepFullJobIdempotentByJob(t *testing.T) {
	id := StepFullIDs{WorkspaceID: "w", TaskID: "t", CommentID: "c", JobID: "j1"}
	b := EmptyStepFullBundle(id)
	b = MergeStepFullJob(b, id, "completed", []any{map[string]any{"step_number": 1}})
	b = MergeStepFullJob(b, id, "completed", []any{map[string]any{"step_number": 2}})
	if len(b.Jobs) != 1 {
		t.Fatalf("jobs=%d", len(b.Jobs))
	}
	steps := StepsForJob(b, "j1")
	if len(steps) != 1 {
		t.Fatalf("steps=%d", len(steps))
	}
	m, _ := steps[0].(map[string]any)
	if m["step_number"] != 2 {
		t.Fatalf("step=%v", steps[0])
	}
}

func TestMergeStepFullJobKeepsOtherJobs(t *testing.T) {
	id1 := StepFullIDs{WorkspaceID: "w", TaskID: "t", CommentID: "c", JobID: "j1"}
	id2 := id1
	id2.JobID = "j2"
	b := MergeStepFullJob(EmptyStepFullBundle(id1), id1, "completed", []any{map[string]any{"n": 1}})
	b = MergeStepFullJob(b, id2, "failed", []any{map[string]any{"n": 2}})
	if len(b.Jobs) != 2 {
		t.Fatalf("jobs=%d", len(b.Jobs))
	}
	if StepsForJob(b, "j1") == nil || StepsForJob(b, "j2") == nil {
		t.Fatal("missing job")
	}
}

func TestParseStepFullBundleRejectsArray(t *testing.T) {
	if _, err := ParseStepFullBundle([]byte(`[]`)); err == nil {
		t.Fatal("expected error")
	}
}
