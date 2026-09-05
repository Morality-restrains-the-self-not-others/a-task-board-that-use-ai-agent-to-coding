package main

import (
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"tracelog"
)

const commentContainerBindingSelectColumns = `id, company_id, COALESCE(workspace_id,''), task_id, comment_id, execution_mode, depends_on_comment_id,
			status, mock_container_name, csc_id, COALESCE(start_trace_id,''), created_at, updated_at`

// start-vm 常早于 ensure binding：UPDATE 0 行时暂存，INSERT 后 drain 回写，避免第二条评论缺启动 TraceId。
// 暂存优先落 MySQL 旁路表 cloud_binding_pending_start_trace（OPT-20260816-035，跨重启/多副本仍可回填），
// db 不可用时回退进程内 map。
var pendingBindingStartTrace sync.Map

func pendingBindingStartTraceKey(taskID, commentID string) string {
	return taskID + "\x1e" + commentID
}

func isTraceEqualToTaskID(traceID, taskID string) bool {
	tid := strings.TrimSpace(taskID)
	tr := strings.TrimSpace(traceID)
	if tid == "" || tr == "" {
		return false
	}
	return tr == tid
}

// stashPendingBindingStartTraceID 将 pending 启机 TraceId 写入旁路表（主路径），
// 旁路表写失败或 db 未初始化时回退进程内 map。
func stashPendingBindingStartTraceID(taskID, commentID, traceID string) {
	taskID = trim(taskID)
	commentID = trim(commentID)
	traceID = strings.TrimSpace(traceID)
	if taskID == "" || commentID == "" || traceID == "" {
		return
	}
	if db != nil {
		now := time.Now().UTC().Format("2006-01-02 15:04:05")
		if _, err := db.Exec(
			`INSERT INTO cloud_binding_pending_start_trace(task_id, comment_id, start_trace_id, updated_at)
			 VALUES (?,?,?,?)
			 ON DUPLICATE KEY UPDATE start_trace_id=VALUES(start_trace_id), updated_at=VALUES(updated_at)`,
			taskID, commentID, traceID, now,
		); err == nil {
			return
		} else {
			log.Printf("[taskCloudService] event=binding_start_trace_pending_persist_failed task_id=%s comment_id=%s err=%v",
				taskID, commentID, err)
		}
	}
	pendingBindingStartTrace.Store(pendingBindingStartTraceKey(taskID, commentID), traceID)
}

// loadPendingBindingStartTraceID 优先读进程内 map（本进程刚写入），再读旁路表。
func loadPendingBindingStartTraceID(taskID, commentID string) string {
	taskID = trim(taskID)
	commentID = trim(commentID)
	if taskID == "" || commentID == "" {
		return ""
	}
	if raw, ok := pendingBindingStartTrace.Load(pendingBindingStartTraceKey(taskID, commentID)); ok {
		if tid, ok := raw.(string); ok {
			if tid = strings.TrimSpace(tid); tid != "" {
				return tid
			}
		}
	}
	if db == nil {
		return ""
	}
	var id string
	if err := db.QueryRow(
		`SELECT start_trace_id FROM cloud_binding_pending_start_trace WHERE task_id=? AND comment_id=? LIMIT 1`,
		taskID, commentID,
	).Scan(&id); err != nil {
		return ""
	}
	return strings.TrimSpace(id)
}

// deletePendingBindingStartTraceID 清除 map 与旁路表两处暂存（drain 成功后调用）。
func deletePendingBindingStartTraceID(taskID, commentID string) {
	taskID = trim(taskID)
	commentID = trim(commentID)
	if taskID == "" || commentID == "" {
		return
	}
	pendingBindingStartTrace.Delete(pendingBindingStartTraceKey(taskID, commentID))
	if db != nil {
		_, _ = db.Exec(
			`DELETE FROM cloud_binding_pending_start_trace WHERE task_id=? AND comment_id=?`,
			taskID, commentID,
		)
	}
}

func drainPendingBindingStartTraceID(taskID, commentID string) {
	taskID = trim(taskID)
	commentID = trim(commentID)
	if taskID == "" || commentID == "" {
		return
	}
	traceID := loadPendingBindingStartTraceID(taskID, commentID)
	if traceID == "" {
		return
	}
	if isTraceEqualToTaskID(traceID, taskID) {
		// 陈旧旁路行写的是 task_id：不写库，直接清掉。
		deletePendingBindingStartTraceID(taskID, commentID)
		return
	}
	persistCommentBindingStartTraceID(taskID, commentID, traceID)
}

// persistCommentBindingStartTraceID 将本次启机 TraceId 写入该评论 binding。
// 无 comment_id 不扇出；等于 task_id 的值拒绝写入。
// binding 行尚不存在时暂存，待 INSERT 后 drainPendingBindingStartTraceID 回写。
func persistCommentBindingStartTraceID(taskID, commentID, traceID string) {
	taskID = trim(taskID)
	commentID = trim(commentID)
	traceID = strings.TrimSpace(traceID)
	if taskID == "" || commentID == "" || traceID == "" {
		return
	}
	if isTraceEqualToTaskID(traceID, taskID) {
		log.Printf("[taskCloudService] event=binding_start_trace_id_skip_task_id task_id=%s comment_id=%s",
			taskID, commentID)
		return
	}
	if db == nil {
		stashPendingBindingStartTraceID(taskID, commentID, traceID)
		return
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	res, err := db.Exec(
		`UPDATE cloud_comment_container_bindings SET start_trace_id=?, updated_at=? WHERE task_id=? AND comment_id=?`,
		traceID, now, taskID, commentID,
	)
	if err != nil {
		log.Printf("[taskCloudService] event=binding_start_trace_id_persist_failed task_id=%s comment_id=%s err=%v",
			taskID, commentID, err)
		stashPendingBindingStartTraceID(taskID, commentID, traceID)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		stashPendingBindingStartTraceID(taskID, commentID, traceID)
		logInfo("event=binding_start_trace_id_pending_no_row task_id="+taskID+" comment_id="+commentID, traceID)
		return
	}
	deletePendingBindingStartTraceID(taskID, commentID)
	logInfo("event=binding_start_trace_id_persisted task_id="+taskID+" comment_id="+commentID+
		" rows="+strconv.FormatInt(n, 10), traceID)
}

func loadBindingStartTraceID(taskID, commentID string) string {
	taskID = trim(taskID)
	commentID = trim(commentID)
	if db == nil || taskID == "" || commentID == "" {
		return ""
	}
	var id string
	err := db.QueryRow(
		`SELECT COALESCE(start_trace_id,'') FROM cloud_comment_container_bindings WHERE task_id=? AND comment_id=? LIMIT 1`,
		taskID, commentID,
	).Scan(&id)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(id)
}

// ensureCommentBindingStartTraceID 在列为空（或误写 task_id）时写入独立 TraceId。
// 已有独立 ID 不覆盖。preferred 为空或等于 task_id 时生成新 ID。
func ensureCommentBindingStartTraceID(taskID, commentID, preferred string) string {
	taskID = trim(taskID)
	commentID = trim(commentID)
	if taskID == "" || commentID == "" {
		return ""
	}
	// OPT-20260816-035：先回填旁路暂存（可能来自上一进程/另一副本），再读现有列值。
	drainPendingBindingStartTraceID(taskID, commentID)
	existing := loadBindingStartTraceID(taskID, commentID)
	if existing != "" && !isTraceEqualToTaskID(existing, taskID) {
		return existing
	}
	tid := strings.TrimSpace(preferred)
	if tid == "" || isTraceEqualToTaskID(tid, taskID) {
		tid = tracelog.NewTraceID()
	}
	persistCommentBindingStartTraceID(taskID, commentID, tid)
	if got := loadBindingStartTraceID(taskID, commentID); got != "" {
		if got != existing {
			logInfo("event=binding_start_trace_id_ensured task_id="+taskID+" comment_id="+commentID, got)
		}
		return got
	}
	return tid
}
