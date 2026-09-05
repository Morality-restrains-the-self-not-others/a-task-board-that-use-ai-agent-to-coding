package main

import (
	"strings"

	"stopreason"
)

// stopReasonTriggerLabel 将 stop_reason 码转为用户可读的触发说明。
// SSOT：委托 shareLib/stopreason，与 taskEvents 共用，避免码表漂移。
func stopReasonTriggerLabel(reason string) string {
	return stopreason.TriggerLabel(reason)
}

func annotateStopServerMessage(msg, reason string) string {
	return stopreason.AnnotateMessage(msg, reason)
}

func stopReasonFromComputeBody(body map[string]interface{}) string {
	r := strings.TrimSpace(strField(body, "stop_reason"))
	if r == "" {
		r = strings.TrimSpace(strField(body, "reason"))
	}
	if r == "" || r == "<nil>" {
		return "user_stop"
	}
	return r
}
