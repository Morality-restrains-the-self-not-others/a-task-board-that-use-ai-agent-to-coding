package domain

import (
	"strings"
	"testing"
)

func TestRenderStartupLogObjectKeyDefault(t *testing.T) {
	key, err := RenderStartupLogObjectKey(DefaultStartupLogPathRule, StartupLogIDs{
		WorkspaceID: "ws_-1", TaskID: "task_2", CommentID: "cmt_3",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "workspace_ws_-1/task_task_2/comment_cmt_3/startup_logs.json"
	if key != want {
		t.Fatalf("key=%q want %q", key, want)
	}
}

func TestValidateStartupLogPathRuleRejectsDotDot(t *testing.T) {
	if err := ValidateStartupLogPathRule("workspace_{workspaceId}/../x"); err == nil {
		t.Fatal("expected reject")
	}
}

func TestValidateStartupLogPathRuleRequiresPlaceholders(t *testing.T) {
	if err := ValidateStartupLogPathRule("only/{taskId}/startup_logs.json"); err == nil {
		t.Fatal("expected missing workspace/comment")
	}
}

func TestStartupLogIDsValidate(t *testing.T) {
	if err := (StartupLogIDs{WorkspaceID: "w", TaskID: "t", CommentID: "c"}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (StartupLogIDs{WorkspaceID: "w", TaskID: "t"}).Validate(); err == nil {
		t.Fatal("missing comment should fail")
	}
}

func TestRenderStartupLogObjectKeyEmptyRuleUsesDefault(t *testing.T) {
	key, err := RenderStartupLogObjectKey("", StartupLogIDs{WorkspaceID: "w", TaskID: "t", CommentID: "c"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(key, "/startup_logs.json") {
		t.Fatalf("key=%q", key)
	}
}
