package main

import "testing"

func TestStopReasonTriggerLabel(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"user_stop", "用户点击停止服务器"},
		{"stop_vm", "用户点击停止服务器"},
		{"instruction_idle", "容器指令空闲超时回收"},
		{"idle_recycle", "工作区机器空闲回收"},
		{"schedule_window_end", "排期窗口结束自动关闭"},
		{"task_status_cancelled", "任务进度变为已取消"},
		{"", "未标明来源"},
		{"task_status_foo", "任务进度变更释放（foo）"},
	}
	for _, c := range cases {
		if got := stopReasonTriggerLabel(c.in); got != c.want {
			t.Fatalf("reason=%q got %q want %q", c.in, got, c.want)
		}
	}
}

func TestAnnotateStopServerMessage(t *testing.T) {
	msg := annotateStopServerMessage("正在调用aliyunAPI停止服务器...", "instruction_idle")
	if msg != "正在调用aliyunAPI停止服务器...（触发：容器指令空闲超时回收）" {
		t.Fatalf("got %q", msg)
	}
	again := annotateStopServerMessage(msg, "user_stop")
	if again != msg {
		t.Fatalf("must be idempotent, got %q", again)
	}
}

func TestStopReasonFromComputeBody(t *testing.T) {
	if got := stopReasonFromComputeBody(nil); got != "user_stop" {
		t.Fatalf("nil body got %q", got)
	}
	if got := stopReasonFromComputeBody(map[string]interface{}{"reason": "schedule_window_end"}); got != "schedule_window_end" {
		t.Fatalf("reason field got %q", got)
	}
	if got := stopReasonFromComputeBody(map[string]interface{}{"stop_reason": "instruction_idle", "reason": "other"}); got != "instruction_idle" {
		t.Fatalf("stop_reason must win, got %q", got)
	}
}
