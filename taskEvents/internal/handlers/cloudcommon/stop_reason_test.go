package cloudcommon

import "testing"

func TestStopReasonTriggerLabel(t *testing.T) {
	if got := StopReasonTriggerLabel("instruction_idle"); got != "容器指令空闲超时回收" {
		t.Fatalf("got %q", got)
	}
	if got := StopReasonTriggerLabel("user_stop"); got != "用户点击停止服务器" {
		t.Fatalf("got %q", got)
	}
}

func TestAnnotateStopServerMessageWithLabel(t *testing.T) {
	got := AnnotateStopServerMessageWithLabel("正在调用aliyunAPI停止服务器...", "instruction_idle", "")
	want := "正在调用aliyunAPI停止服务器...（触发：容器指令空闲超时回收）"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if again := AnnotateStopServerMessageWithLabel(got, "user_stop", "用户点击停止服务器"); again != got {
		t.Fatalf("must be idempotent, got %q", again)
	}
	fromEvent := AnnotateStopServerMessageWithLabel("正在停止 Mock 实例...", "instruction_idle", "容器指令空闲超时回收")
	if fromEvent != "正在停止 Mock 实例...（触发：容器指令空闲超时回收）" {
		t.Fatalf("got %q", fromEvent)
	}
}
