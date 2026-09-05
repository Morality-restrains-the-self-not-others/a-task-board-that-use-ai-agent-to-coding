package cloudcommon

import "testing"

func TestFormatStartupLogLabel(t *testing.T) {
	got := FormatStartupLogLabel(StartupLogScope{
		InstanceID:    "i-j6c57m3gs9rd14eoiue3",
		ContainerName: "task_t1_cmt-1",
	})
	want := "[i-j6c57m3gs9rd14eoiue3、task_t1_cmt-1]"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestFormatStartupLogLabelDerivesContainerName(t *testing.T) {
	got := FormatStartupLogLabel(StartupLogScope{
		InstanceID: "i-1",
		TaskID:     "task_abc",
		CommentID:  "cmt_9",
	})
	want := "[i-1、task_abc_cmt_9]"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestDeriveContainerNameNoDoubleTaskPrefix(t *testing.T) {
	got := DeriveContainerName("", "task_15666874162351520866", "cmt_9")
	want := "task_15666874162351520866_cmt_9"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if DeriveContainerName("", "task1", "cNew") != "task_task1_cNew" {
		t.Fatalf("bare task1 should still get one task_ prefix")
	}
}

func TestFormatStartupLogLabelMissingSlotsUseDash(t *testing.T) {
	got := FormatStartupLogLabel(StartupLogScope{ContainerName: "task2app-container"})
	want := "[-、task2app-container]"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestFormatStartupLogLabelEmpty(t *testing.T) {
	if FormatStartupLogLabel(StartupLogScope{}) != "" {
		t.Fatal("expected empty label")
	}
}

func TestApplyStartupLogScopePrefixesMessage(t *testing.T) {
	out := ApplyStartupLogScope(map[string]interface{}{
		"status":  "processing",
		"message": "正在根据 @镜像 启动运行环境...",
	}, StartupLogScope{InstanceID: "i-1", ContainerName: "task_t1_c1"})
	msg, _ := out["message"].(string)
	if msg != "[i-1、task_t1_c1] 正在根据 @镜像 启动运行环境..." {
		t.Fatalf("message=%q", msg)
	}
	if out["log_label"] != "[i-1、task_t1_c1]" {
		t.Fatalf("log_label=%v", out["log_label"])
	}
}

func TestScopeFromEventDataDerivesContainerName(t *testing.T) {
	s := ScopeFromEventData(map[string]interface{}{
		"task_id":           "t1",
		"parent_comment_id": "cmt-2",
		"instance_id":       "i-evt",
	})
	if s.ContainerName != "task_t1_cmt-2" {
		t.Fatalf("container=%q", s.ContainerName)
	}
	if FormatStartupLogLabel(s) != "[i-evt、task_t1_cmt-2]" {
		t.Fatalf("label=%q", FormatStartupLogLabel(s))
	}
}
