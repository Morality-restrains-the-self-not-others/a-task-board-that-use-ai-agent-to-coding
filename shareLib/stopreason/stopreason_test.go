package stopreason

import (
	"strings"
	"testing"
)

func TestTriggerLabel(t *testing.T) {
	cases := []struct {
		reason string
		want   string
	}{
		{"", "未标明来源"},
		{"<nil>", "未标明来源"},
		{"user_stop", "用户点击停止服务器"},
		{"stop_vm", "用户点击停止服务器"},
		{"instruction_idle", "容器指令空闲超时回收"},
		{"idle_recycle", "工作区机器空闲回收"},
		{"schedule_window_end", "排期窗口结束自动关闭"},
		{"superseded_by_new_start", "新启动替换旧实例"},
		{"task_status_cancelled", "任务进度变为已取消"},
		{"task_status_completed", "任务完成后释放"},
		{"inbound_task_terminal", "任务已终态（容器入站补偿释放）"},
		{"reconcile_task_terminal", "任务终态对账释放"},
		{"task_status_whatever", "任务进度变更释放（whatever）"},
		{"unknown_code", "unknown_code"},
	}
	for _, tc := range cases {
		if got := TriggerLabel(tc.reason); got != tc.want {
			t.Errorf("TriggerLabel(%q)=%q want %q", tc.reason, got, tc.want)
		}
	}
}

func TestAnnotateMessageIdempotent(t *testing.T) {
	got := AnnotateMessage("服务器已停止", "user_stop")
	want := "服务器已停止（触发：用户点击停止服务器）"
	if got != want {
		t.Fatalf("AnnotateMessage=%q want %q", got, want)
	}
	// 已含「触发：」不重复追加
	again := AnnotateMessage(want, "user_stop")
	if again != want {
		t.Fatalf("second annotate=%q want unchanged %q", again, want)
	}
	// 空消息不追加
	if empty := AnnotateMessage("  ", "user_stop"); empty != "" {
		t.Fatalf("empty msg=%q", empty)
	}
}

func TestAnnotateMessageWithLabelOverride(t *testing.T) {
	got := AnnotateMessageWithLabel("服务器已停止", "user_stop", "自定义")
	if !strings.Contains(got, "自定义") {
		t.Fatalf("label override not applied: %q", got)
	}
}

func TestCodesNonEmptyAndNoDuplicates(t *testing.T) {
	codes := Codes()
	if len(codes) == 0 {
		t.Fatal("Codes() empty")
	}
	seen := map[string]bool{}
	for _, c := range codes {
		if c == "" {
			t.Fatalf("Codes() contains empty string")
		}
		if seen[c] {
			t.Fatalf("duplicate code %q", c)
		}
		seen[c] = true
	}
}
