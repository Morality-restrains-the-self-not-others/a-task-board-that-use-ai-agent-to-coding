package autorunstartvm

import "strings"

// Cloud VM start entry runtime_source codes (stored on history.runtime_source).
const (
	RuntimeSourceCloudVM                = "cloud_vm"
	RuntimeSourceCloudVMManual          = "cloud_vm_manual"
	RuntimeSourceCloudVMTemplate        = "cloud_vm_template"
	RuntimeSourceCloudVMAutoRun         = "cloud_vm_auto_run"
	RuntimeSourceCloudVMCommentMention  = "cloud_vm_comment_mention"
	RuntimeSourceCloudVMTerminalMigrate = "cloud_vm_terminal_migrate"
	RuntimeSourceRelayLocal             = "relay_local"
	RuntimeSourceMockRun                = "mock_run"
)

// ApplyRuntimeSource sets body.runtime_source when non-empty.
func ApplyRuntimeSource(body map[string]interface{}, runtimeSource string) {
	if body == nil {
		return
	}
	if s := strings.TrimSpace(runtimeSource); s != "" {
		body["runtime_source"] = s
	}
}
