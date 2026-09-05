package domain

import "fmt"

// 五个开发工具的名称常量，用于 DevToolLogRecorder 的 tool 参数。
const (
	ToolConfSync           = "conf-sync"
	ToolConfSyncRemote     = "conf-sync-remote"
	ToolDbClear            = "db-clear"
	ToolDbInit             = "db-init"
	ToolObservabilityClear = "observability-clear"
)

// AllDevTools 是所有合法开发工具名的列表。
var AllDevTools = []string{
	ToolConfSync,
	ToolConfSyncRemote,
	ToolDbClear,
	ToolDbInit,
	ToolObservabilityClear,
}

// DevToolLogRecorder 将开发工具操作的日志写入文件，支持按工具名读写。
// 每个开发者工具对应一个独立的日志文件，可通过 Tail 读取尾行、Clear 清空。
type DevToolLogRecorder interface {
	// Append 追加一行日志到指定工具的日志文件。
	// tool 必须是预定义的五个工具名之一，否则返回 error。
	Append(tool string, message string) error
	// Tail 返回日志文件的最后 N 行（按时间顺序，旧→新）。
	// 文件不存在时返回空切片，不报错。
	Tail(tool string, lines int) ([]string, error)
	// Clear 清空指定工具的日志文件。
	// 文件不存在时不报错。
	Clear(tool string) error
	// ClearAll 清空全部开发工具日志文件。
	ClearAll() error
}

// IsValidDevTool 检查 tool 名是否在预定义的五种开发工具中。
func IsValidDevTool(tool string) bool {
	for _, t := range AllDevTools {
		if t == tool {
			return true
		}
	}
	return false
}

// ValidateDevTool 校验 tool 名，非法时返回 error。
func ValidateDevTool(tool string) error {
	if IsValidDevTool(tool) {
		return nil
	}
	return fmt.Errorf("unknown dev tool: %q (valid: %v)", tool, AllDevTools)
}
