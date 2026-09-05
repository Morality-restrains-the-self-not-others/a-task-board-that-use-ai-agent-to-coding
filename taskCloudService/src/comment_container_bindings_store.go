package main

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const (
	ccbExecutionWaitPrevious = "wait_previous"
	ccbExecutionIndependent  = "independent"

	ccbStageCSCAllocated     = "csc_allocated"
	ccbStageServerScheduling = "server_scheduling"
	ccbStageServerStarted    = "server_started"
	ccbStageServerFailed     = "server_failed"
	ccbStatusPending         = "pending"
	ccbStatusWaitingPrevious = "waiting_previous"
	ccbStatusStarting        = "starting"
	ccbStatusRunning         = "running"
	ccbStatusCompleted       = "completed"
	ccbStatusFailed          = "failed"
	ccbStatusReleased        = "released"
	ccbStatusCancelled       = "cancelled"
)

type CommentContainerBinding struct {
	ID                 string
	CompanyID          string
	WorkspaceID        string
	TaskID             string
	CommentID          string
	ExecutionMode      string
	DependsOnCommentID string
	Status             string
	MockContainerName  string
	CSCID              string
	StartTraceID       string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// CommentContainerBindingLog 评论容器绑定启动阶段事件（服务端权威时间线）。
// OPT-20260809-011：冷打开/换设备时还原「排队→启动→分配→就绪」完整历史，
// 前端 refreshBindings 将 logs 字段去重合并到本地 SSE 派生行。
type CommentContainerBindingLog struct {
	ID          string
	WorkspaceID string
	CompanyID   string
	TaskID      string
	CommentID   string
	BindingID   string
	Stage       string
	Message     string
	CreatedAt   time.Time
}

// ccbStageMessage 阶段事件 → 中文消息，与 taskFE BINDING_STATUS_LOG_MESSAGES /
// BINDING_STAGE_LOG_MESSAGES 保持同文案，前端按消息文本去重。
func ccbStageMessage(stage string) string {
	switch stage {
	case ccbStatusPending:
		return "容器调度排队中"
	case ccbStatusWaitingPrevious:
		return "等待前序任务完成"
	case ccbStatusStarting:
		return "正在启动容器实例"
	case ccbStageCSCAllocated:
		return "容器实例已分配，等待服务就绪"
	case ccbStatusRunning:
		return "容器已就绪，服务可用"
	case ccbStatusCompleted:
		return "容器执行完成"
	case ccbStatusFailed:
		return "容器启动失败"
	case ccbStatusReleased:
		return "容器已释放"
	case ccbStatusCancelled:
		return "用户已终止等待"
	default:
		return ""
	}
}

func normalizeCommentContainerExecutionMode(mode string) string {
	switch strings.TrimSpace(mode) {
	case ccbExecutionIndependent:
		return ccbExecutionIndependent
	default:
		return ccbExecutionWaitPrevious
	}
}

func insertCommentContainerBinding(companyID, taskID, commentID, executionMode, dependsOnCommentID string, workspaceIDs ...string) (*CommentContainerBinding, error) {
	companyID = trim(companyID)
	taskID = trim(taskID)
	commentID = trim(commentID)
	if companyID == "" || taskID == "" || commentID == "" {
		return nil, fmt.Errorf("company_id, task_id, comment_id required")
	}
	executionMode = normalizeCommentContainerExecutionMode(executionMode)
	dependsOnCommentID = trim(dependsOnCommentID)
	explicitWS := ""
	if len(workspaceIDs) > 0 {
		explicitWS = trim(workspaceIDs[0])
	}
	ws := explicitWS
	if ws == "" {
		ws = lookupWorkspaceIDByTask(taskID)
	}
	now := time.Now().UTC()
	row := CommentContainerBinding{
		ID:                 genID("ccb"),
		CompanyID:          companyID,
		WorkspaceID:        ws,
		TaskID:             taskID,
		CommentID:          commentID,
		ExecutionMode:      executionMode,
		DependsOnCommentID: dependsOnCommentID,
		Status:             ccbStatusPending,
		MockContainerName:  buildCommentMockContainerName(taskID, commentID),
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	_, err := db.Exec(
		`INSERT INTO cloud_comment_container_bindings(
			id, company_id, workspace_id, task_id, comment_id, execution_mode, depends_on_comment_id,
			status, mock_container_name, csc_id, created_at, updated_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		row.ID, row.CompanyID, row.WorkspaceID, row.TaskID, row.CommentID, row.ExecutionMode, row.DependsOnCommentID,
		row.Status, row.MockContainerName, row.CSCID,
		formatMySQLUTCDateTime(row.CreatedAt), formatMySQLUTCDateTime(row.UpdatedAt),
	)
	if err != nil {
		return nil, err
	}
	// OPT-20260809-011: 记录初始排队阶段（服务端权威时间线起点）
	logCommentContainerBindingStageBestEffort(&row, ccbStatusPending)
	drainPendingBindingStartTraceID(taskID, commentID)
	drainPendingBindingStartError(companyID, taskID, commentID)
	if updated, err := loadCommentContainerBinding(companyID, taskID, commentID); err == nil && updated != nil {
		return updated, nil
	}
	return &row, nil
}

// ensureCommentContainerBinding 创建或同步 execution_mode / depends（避免前端 ensure 遇 UNIQUE 后模式漂移）。
// 若已存在且从 wait_previous 改为 independent，且仍处于 waiting_previous，则重置为 pending 以便立即调度。
func ensureCommentContainerBinding(companyID, taskID, commentID, executionMode, dependsOnCommentID string, workspaceIDs ...string) (*CommentContainerBinding, bool, error) {
	existing, err := loadCommentContainerBinding(companyID, taskID, commentID)
	if err == sql.ErrNoRows {
		row, err := insertCommentContainerBinding(companyID, taskID, commentID, executionMode, dependsOnCommentID, workspaceIDs...)
		return row, true, err
	}
	if err != nil {
		return nil, false, err
	}
	executionMode = normalizeCommentContainerExecutionMode(executionMode)
	dependsOnCommentID = trim(dependsOnCommentID)
	wantName := buildCommentMockContainerName(taskID, commentID)
	prevMode := existing.ExecutionMode
	changed := prevMode != executionMode || existing.DependsOnCommentID != dependsOnCommentID
	if trim(existing.MockContainerName) != wantName {
		changed = true
	}
	newStatus := existing.Status
	if executionMode == ccbExecutionIndependent && existing.Status == ccbStatusWaitingPrevious {
		newStatus = ccbStatusPending
		changed = true
	}
	if !changed {
		drainPendingBindingStartTraceID(taskID, commentID)
		drainPendingBindingStartError(companyID, taskID, commentID)
		if updated, loadErr := loadCommentContainerBinding(companyID, taskID, commentID); loadErr == nil && updated != nil {
			return updated, false, nil
		}
		return existing, false, nil
	}
	now := time.Now().UTC()
	_, err = db.Exec(
		`UPDATE cloud_comment_container_bindings
		SET execution_mode=?, depends_on_comment_id=?, status=?, mock_container_name=?, updated_at=?
		WHERE id=?`,
		executionMode, dependsOnCommentID, newStatus, wantName, formatMySQLUTCDateTime(now), existing.ID,
	)
	if err != nil {
		return nil, false, err
	}
	existing.ExecutionMode = executionMode
	existing.DependsOnCommentID = dependsOnCommentID
	existing.Status = newStatus
	existing.MockContainerName = wantName
	existing.UpdatedAt = now
	drainPendingBindingStartTraceID(taskID, commentID)
	drainPendingBindingStartError(companyID, taskID, commentID)
	if updated, loadErr := loadCommentContainerBinding(companyID, taskID, commentID); loadErr == nil && updated != nil {
		return updated, false, nil
	}
	return existing, false, nil
}

func listCommentContainerBindings(companyID, taskID string) ([]CommentContainerBinding, error) {
	companyID = trim(companyID)
	taskID = trim(taskID)
	if companyID == "" || taskID == "" {
		return nil, fmt.Errorf("company_id, task_id required")
	}
	rows, err := db.Query(
		`SELECT `+commentContainerBindingSelectColumns+`
		FROM cloud_comment_container_bindings
		WHERE company_id=? AND task_id=?
		ORDER BY created_at ASC, id ASC`,
		companyID, taskID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out, err := scanCommentContainerBindingRows(rows)
	if err != nil {
		return nil, err
	}
	for i := range out {
		if trim(out[i].StartTraceID) != "" || trim(out[i].CSCID) == "" {
			continue
		}
		out[i].StartTraceID = ensureCommentBindingStartTraceID(out[i].TaskID, out[i].CommentID, "")
	}
	return out, nil
}

func loadCommentContainerBinding(companyID, taskID, commentID string) (*CommentContainerBinding, error) {
	companyID = trim(companyID)
	taskID = trim(taskID)
	commentID = trim(commentID)
	row := db.QueryRow(
		`SELECT `+commentContainerBindingSelectColumns+`
		FROM cloud_comment_container_bindings
		WHERE company_id=? AND task_id=? AND comment_id=?`,
		companyID, taskID, commentID,
	)
	return scanCommentContainerBinding(row)
}

func scanCommentContainerBinding(row *sql.Row) (*CommentContainerBinding, error) {
	var b CommentContainerBinding
	var created, updated string
	err := row.Scan(
		&b.ID, &b.CompanyID, &b.WorkspaceID, &b.TaskID, &b.CommentID, &b.ExecutionMode, &b.DependsOnCommentID,
		&b.Status, &b.MockContainerName, &b.CSCID, &b.StartTraceID, &created, &updated,
	)
	if err != nil {
		return nil, err
	}
	b.CreatedAt = parseCloudUTCDateTime(created)
	b.UpdatedAt = parseCloudUTCDateTime(updated)
	return &b, nil
}

func scanCommentContainerBindingRows(rows *sql.Rows) ([]CommentContainerBinding, error) {
	out := make([]CommentContainerBinding, 0)
	for rows.Next() {
		var b CommentContainerBinding
		var created, updated string
		if err := rows.Scan(
			&b.ID, &b.CompanyID, &b.WorkspaceID, &b.TaskID, &b.CommentID, &b.ExecutionMode, &b.DependsOnCommentID,
			&b.Status, &b.MockContainerName, &b.CSCID, &b.StartTraceID, &created, &updated,
		); err != nil {
			return nil, err
		}
		b.CreatedAt = parseCloudUTCDateTime(created)
		b.UpdatedAt = parseCloudUTCDateTime(updated)
		out = append(out, b)
	}
	return out, rows.Err()
}

func updateCommentContainerBindingStatus(id, status string) error {
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	_, err := db.Exec(
		`UPDATE cloud_comment_container_bindings SET status=?, updated_at=? WHERE id=?`,
		status, now, id,
	)
	return err
}

// commentBindingIsProvisioning 任务下是否存在尚未完成的评论容器绑定（排队/等待/启动中）。
// 供 server-runtime-status 在 CSC 尚无 instance_id 时判断是否应返回 Pending。
func commentBindingIsProvisioning(companyID, taskID string) bool {
	companyID = trim(companyID)
	taskID = trim(taskID)
	if taskID == "" {
		return false
	}
	q := `SELECT COUNT(1) FROM cloud_comment_container_bindings
		WHERE task_id=? AND status IN (?,?,?)`
	args := []interface{}{taskID, ccbStatusPending, ccbStatusWaitingPrevious, ccbStatusStarting}
	if companyID != "" {
		q += ` AND company_id=?`
		args = append(args, companyID)
	}
	var n int
	if err := db.QueryRow(q, args...).Scan(&n); err != nil {
		return false
	}
	return n > 0
}

func markCommentContainerBindingRunning(id, mockName, cscID string) error {
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	_, err := db.Exec(
		`UPDATE cloud_comment_container_bindings
		SET status=?, mock_container_name=?, csc_id=?, updated_at=?
		WHERE id=?`,
		ccbStatusRunning, mockName, cscID, now, id,
	)
	return err
}

// markCommentContainerBindingStarting 记录独立 mock 容器名，但不挂接任务级 CSC
// （CSC 已被其它 live binding 占用时，供 independent 并行调度使用）。
func markCommentContainerBindingStarting(id, mockName string) error {
	return markCommentContainerBindingStartingWithCSC(id, mockName, "")
}

// markCommentContainerBindingStartingWithCSC 保持 starting，可挂接评论级 CSC；
// 仅在 CSC server_url/reachability 就绪后再升为 running。
func markCommentContainerBindingStartingWithCSC(id, mockName, cscID string) error {
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	_, err := db.Exec(
		`UPDATE cloud_comment_container_bindings
		SET status=?, mock_container_name=?, csc_id=?, updated_at=?
		WHERE id=?`,
		ccbStatusStarting, mockName, strings.TrimSpace(cscID), now, id,
	)
	return err
}

func completeCommentContainerBinding(companyID, taskID, commentID string) (*CommentContainerBinding, error) {
	b, err := loadCommentContainerBinding(companyID, taskID, commentID)
	if err != nil {
		return nil, err
	}
	if b.Status != ccbStatusRunning {
		return nil, fmt.Errorf("binding status %s cannot complete (want running)", b.Status)
	}
	if err := updateCommentContainerBindingStatus(b.ID, ccbStatusCompleted); err != nil {
		return nil, err
	}
	b.Status = ccbStatusCompleted
	b.UpdatedAt = time.Now().UTC()
	// OPT-20260809-011: 记录执行完成阶段
	logCommentContainerBindingStageBestEffort(b, ccbStatusCompleted)
	return b, nil
}

func commentContainerBindingToJSON(b *CommentContainerBinding) map[string]interface{} {
	out := commentContainerBindingJSONBase(b)
	if out == nil {
		return nil
	}
	attachCSCRuntimeOntoBindingJSON(out, b.CSCID)
	return out
}

func commentContainerBindingJSONBase(b *CommentContainerBinding) map[string]interface{} {
	if b == nil {
		return nil
	}
	// 始终回写规范名：避免存量 task_ + task_* 双前缀继续污染前端展示。
	containerName := resolveCommentMockContainerName(b.TaskID, b.CommentID, b.MockContainerName)
	return map[string]interface{}{
		"id":                    b.ID,
		"company_id":            b.CompanyID,
		"workspace_id":          b.WorkspaceID,
		"task_id":               b.TaskID,
		"comment_id":            b.CommentID,
		"execution_mode":        b.ExecutionMode,
		"depends_on_comment_id": b.DependsOnCommentID,
		"status":                b.Status,
		"mock_container_name":   containerName,
		"container_name":        containerName,
		"csc_id":                b.CSCID,
		"start_trace_id":        b.StartTraceID,
		"created_at":            formatCloudUTCJSON(b.CreatedAt),
		"updated_at":            formatCloudUTCJSON(b.UpdatedAt),
		// OPT-20260809-011: 启动阶段事件时间线（list API 填充；其余响应为空数组）
		"logs": []map[string]interface{}{},
	}
}

func attachCSCRuntimeOntoBindingJSON(out map[string]interface{}, cscID string) {
	if out == nil {
		return
	}
	cscID = trim(cscID)
	if cscID == "" {
		return
	}
	cfg, err := loadCloudServerConfigByID(cscID)
	if err != nil || cfg == nil {
		return
	}
	attachLoadedCSCRuntimeOntoBindingJSON(out, cfg)
}

func attachLoadedCSCRuntimeOntoBindingJSON(out map[string]interface{}, cfg *CloudServerConfig) {
	if out == nil || cfg == nil {
		return
	}
	out["last_runtime_status"] = cfg.LastRuntimeStatus
	out["public_ip"] = cfg.PublicIP
	out["has_server_url"] = trim(cfg.ServerURL) != ""
	out["error_reason"] = cfg.ErrorReason
}

func commentContainerBindingsToJSON(rows []CommentContainerBinding) []map[string]interface{} {
	ids := make([]string, 0, len(rows))
	for i := range rows {
		ids = append(ids, rows[i].CSCID)
	}
	byID, err := loadCloudServerConfigsByIDs(ids)
	if err != nil {
		logWarn("event=comment_binding_csc_batch_load_failed err="+err.Error(), "")
		byID = map[string]*CloudServerConfig{}
	}
	out := make([]map[string]interface{}, 0, len(rows))
	for i := range rows {
		item := commentContainerBindingJSONBase(&rows[i])
		attachLoadedCSCRuntimeOntoBindingJSON(item, byID[trim(rows[i].CSCID)])
		out = append(out, item)
	}
	return out
}

func resolveTaskCloudServerConfigID(companyID, taskID string) string {
	cfg, err := loadCloudServerConfig(companyID, "", taskID)
	if err != nil || cfg == nil {
		return ""
	}
	return cfg.ID
}

// buildCommentMockContainerName 评论级容器名：{taskId}_{commentId}，且保证恰好一个 task_ 前缀。
// 任务 ID 已是 task_* 时不再二次拼接（避免 task_task_1566…_cmt_…）。
func buildCommentMockContainerName(taskID, commentID string) string {
	tid := trim(taskID)
	cid := trim(commentID)
	if tid == "" || cid == "" {
		return ""
	}
	if strings.HasPrefix(tid, "task_") {
		return tid + "_" + cid
	}
	return "task_" + tid + "_" + cid
}

// resolveCommentMockContainerName 优先规范名；纠正存量 task_+task_* 双前缀。
func resolveCommentMockContainerName(taskID, commentID, stored string) string {
	want := buildCommentMockContainerName(taskID, commentID)
	stored = trim(stored)
	if want == "" {
		return stored
	}
	if stored == "" {
		return want
	}
	legacy := "task_" + trim(taskID) + "_" + trim(commentID)
	if stored == legacy && stored != want {
		return want
	}
	return stored
}
