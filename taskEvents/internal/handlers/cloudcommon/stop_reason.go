package cloudcommon

import "stopreason"

// StopReasonTriggerLabel 将 stop_reason 码转为用户可读的触发说明。
// SSOT：委托 shareLib/stopreason，与 taskCloudService 共用，避免码表漂移。
func StopReasonTriggerLabel(reason string) string {
	return stopreason.TriggerLabel(reason)
}

func AnnotateStopServerMessage(msg, reason string) string {
	return stopreason.AnnotateMessage(msg, reason)
}

func AnnotateStopServerMessageWithLabel(msg, reason, label string) string {
	return stopreason.AnnotateMessageWithLabel(msg, reason, label)
}
