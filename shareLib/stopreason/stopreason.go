// Package stopreason 提供 stop_reason 码 → 中文标签的 SSOT 映射。
//
// taskCloudService（stop_reason_display.go）与 taskEvents
// （cloudcommon/stop_reason.go）共用本包，避免两处 Go 码表漂移；
// 前端 serverStartHistoryDisplay.js 的 STOP_REASON_LABELS 由
// db/scripts/ci/check_stop_reason_labels.py 断言 key 集合覆盖 Codes()。
package stopreason

import "strings"

// TriggerLabel 将 stop_reason 码转为用户可读的触发说明。
// 与历史两处实现保持完全一致；新增码必须同步前端 STOP_REASON_LABELS。
func TriggerLabel(reason string) string {
	switch strings.TrimSpace(reason) {
	case "", "<nil>":
		return "未标明来源"
	case "user_stop", "stop_vm":
		return "用户点击停止服务器"
	case "instruction_idle":
		return "容器指令空闲超时回收"
	case "idle_recycle":
		return "工作区机器空闲回收"
	case "schedule_window_end":
		return "排期窗口结束自动关闭"
	case "superseded_by_new_start":
		return "新启动替换旧实例"
	case "task_status_cancelled":
		return "任务进度变为已取消"
	case "task_status_completed":
		return "任务完成后释放"
	case "inbound_task_terminal":
		return "任务已终态（容器入站补偿释放）"
	case "reconcile_task_terminal":
		return "任务终态对账释放"
	default:
		if strings.HasPrefix(reason, "task_status_") {
			return "任务进度变更释放（" + strings.TrimPrefix(reason, "task_status_") + "）"
		}
		return reason
	}
}

// AnnotateMessage 给停服消息追加触发来源说明（幂等：已含「触发：」则不再追加）。
func AnnotateMessage(msg, reason string) string {
	return AnnotateMessageWithLabel(msg, reason, "")
}

// AnnotateMessageWithLabel 使用显式 label 追加触发说明；label 为空时回退 TriggerLabel。
func AnnotateMessageWithLabel(msg, reason, label string) string {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return msg
	}
	if strings.Contains(msg, "触发：") {
		return msg
	}
	l := strings.TrimSpace(label)
	if l == "" {
		l = TriggerLabel(reason)
	}
	return msg + "（触发：" + l + "）"
}

// Codes 返回 SSOT 定义的全部 stop_reason 码（不含空值与别名）。
// CI 用于断言前端 STOP_REASON_LABELS key 集合覆盖本表。
func Codes() []string {
	return []string{
		"user_stop",
		"stop_vm",
		"instruction_idle",
		"idle_recycle",
		"schedule_window_end",
		"superseded_by_new_start",
		"task_status_cancelled",
		"task_status_completed",
		"inbound_task_terminal",
		"reconcile_task_terminal",
	}
}
