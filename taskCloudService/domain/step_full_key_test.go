package domain

import (
	"strings"
	"testing"
)

func TestRenderStepFullObjectKeyDefault(t *testing.T) {
	key, err := RenderStepFullObjectKey(DefaultStepFullPathRule, StepFullIDs{
		WorkspaceID: "ws_-1", TaskID: "task_2", CommentID: "cmt_3", JobID: "job_4", LayerID: "layer_5",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "workspace_ws_-1/task_task_2/comment_cmt_3/layer_layer_5/step_full.json"
	if key != want {
		t.Fatalf("key=%q want %q", key, want)
	}
}

func TestRenderStepFullObjectKeyRejectsDotDot(t *testing.T) {
	if err := ValidateStepFullPathRule("workspace_{workspaceId}/../x"); err == nil {
		t.Fatal("expected reject")
	}
}

func TestRenderStepFullObjectKeyRequiresPlaceholders(t *testing.T) {
	if err := ValidateStepFullPathRule("only/{jobId}/step_full.json"); err == nil {
		t.Fatal("expected missing workspace/task/comment")
	}
}

func TestSanitizePathTokenStripsSlash(t *testing.T) {
	got := SanitizePathToken("../a/b")
	if strings.Contains(got, "/") || strings.Contains(got, "..") {
		t.Fatalf("got=%q", got)
	}
}

func TestStepFullIDsValidate(t *testing.T) {
	if err := (StepFullIDs{WorkspaceID: "w", TaskID: "t", CommentID: "c", JobID: "j"}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (StepFullIDs{WorkspaceID: "w", TaskID: "t", CommentID: "c"}).Validate(); err == nil {
		t.Fatal("missing job should fail")
	}
}
