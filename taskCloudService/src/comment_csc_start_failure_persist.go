package main

import (
	"log"
	"strings"
	"time"
)

// persistCommentCSCStartFailure 启机失败时把人话原因写入评论级 CSC，
// 并把 last_runtime_status 收口为 Failed，避免 runtime-status 长期报「创建中」。
func persistCommentCSCStartFailure(b *CommentContainerBinding, rawMsg string) {
	if b == nil {
		return
	}
	reason := humanizeStartVmError(strings.TrimSpace(rawMsg))
	if reason == "" {
		reason = runtimeMsgEmptyInstanceAfterFail
	}
	cscID := strings.TrimSpace(b.CSCID)
	if cscID == "" {
		cfg, err := loadCloudServerConfigForComment(b.CompanyID, b.WorkspaceID, b.TaskID, b.CommentID)
		if err == nil && cfg != nil {
			cscID = strings.TrimSpace(cfg.ID)
			b.CSCID = cscID
		}
	}
	if cscID == "" {
		return
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	_, err := db.Exec(
		`UPDATE cloud_server_configs SET error_reason=?, last_runtime_status=?, updated_at=? WHERE id=?`,
		reason, machineRuntimeFailed, now, cscID,
	)
	if err != nil {
		log.Printf("[taskCloudService] event=comment_csc_start_failure_persist_err csc_id=%s err=%v", cscID, err)
	}
}
