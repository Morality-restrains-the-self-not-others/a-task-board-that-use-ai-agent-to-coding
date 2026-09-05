package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type taskRecord struct {
	ID                            string
	TenantID                      string
	Title                         string
	Description                   string
	Completed                     bool
	Priority                      string
	OrderNum                      int
	WorkspaceID                   string
	OwnerID                       string
	OperatorID                    string
	DeliverableObjID              string
	ProgressColumnID              string
	ParentTaskID                  string
	ForkFromID                    string
	InstalledImageID              string
	ImageSkillID                  string // 镜像技能 ID（D1=B sk_*）；老前端按名反解，未绑定为空
	ContainerImageSnapshot        string // 保存时名↔ID 映射快照 JSON；未绑定为空
	AgentModel                    string // 派生副本所选智能体模型（auto_run agent_models[0].model），详情层图默认展示
	AutoRun                       bool
	AutoCommitAfterAgentComplete  bool
	FeatureParamsSource           string
	PersonalFeatureParamsConfigID string
	TaskKind                      string
	CodeLang                      string
	AutoRunStartSkipReason        string       // soft-skip start-vm reason; empty when not skipped / cleared after start
	PostExpiresAt                 sql.NullTime // 创建帖到期时间（v15 存续期模式）
	WorkspaceSeq                  int          // 工作空间人读序号，创建后不变
	CreatedAt                     time.Time
	UpdatedAt                     time.Time
}

func scanTaskValues(scan func(dest ...interface{}) error, t *taskRecord) error {
	var completed, autoRun, autoCommit int
	var ca, ua time.Time
	var skillID, snapshot sql.NullString
	if err := scan(
		&t.ID, &t.TenantID, &t.Title, &t.Description, &completed, &t.Priority, &t.OrderNum,
		&t.WorkspaceID, &t.OwnerID, &t.OperatorID, &t.DeliverableObjID, &t.ProgressColumnID,
		&t.ParentTaskID, &t.ForkFromID, &t.InstalledImageID, &skillID, &snapshot, &t.AgentModel, &autoRun, &autoCommit,
		&t.FeatureParamsSource, &t.PersonalFeatureParamsConfigID, &t.TaskKind, &t.CodeLang,
		&t.AutoRunStartSkipReason, &t.PostExpiresAt, &t.WorkspaceSeq, &ca, &ua,
	); err != nil {
		return err
	}
	t.ImageSkillID = skillID.String
	t.ContainerImageSnapshot = snapshot.String
	t.Completed = completed != 0
	t.AutoRun = autoRun != 0
	t.AutoCommitAfterAgentComplete = autoCommit != 0
	t.CreatedAt = ca
	t.UpdatedAt = ua
	return nil
}

func scanTask(row *sql.Row) (*taskRecord, error) {
	var t taskRecord
	if err := scanTaskValues(row.Scan, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

const taskSelectCols = `id,tenant_id,title,COALESCE(description,''),completed,priority,order_num,workspace_id,owner_id,operator_id,deliverable_obj_id,progress_column_id,parent_task_id,fork_from_id,installed_image_id,COALESCE(image_skill_id,''),COALESCE(container_image_snapshot,''),COALESCE(agent_model,''),auto_run,auto_commit_after_agent_complete,feature_params_source,personal_feature_params_config_id,task_kind,code_lang,COALESCE(auto_run_start_skip_reason,''),post_expires_at,COALESCE(workspace_seq,0),created_at,updated_at`

func loadTask(id string) (*taskRecord, error) {
	row := db.QueryRow(`SELECT `+taskSelectCols+` FROM task_tasks WHERE id=?`, id)
	return scanTask(row)
}

func loadAssignees(taskID string) []string {
	rows, err := db.Query(`SELECT company_member_id FROM task_assignees WHERE task_id=? ORDER BY company_member_id`, taskID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		rows.Scan(&id)
		out = append(out, id)
	}
	return out
}

func setAssignees(taskID string, ids []string) {
	db.Exec(`DELETE FROM task_assignees WHERE task_id=?`, taskID)
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		db.Exec(`INSERT IGNORE INTO task_assignees(task_id,company_member_id) VALUES(?,?)`, taskID, id)
	}
}

func loadProjects(taskID, tenantID string) []map[string]interface{} {
	_ = tenantID
	rows, _ := db.Query(`SELECT id,project_id,base_branch,target_branch,repo_address FROM task_projects WHERE task_id=? ORDER BY created_at`, taskID)
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id, pid, base, target, addr string
		rows.Scan(&id, &pid, &base, &target, &addr)
		addr = strings.TrimSpace(addr)
		// B-086: do not GET project (that path probes GitLab). Live URL / mismatch
		// are computed on the client from the workspace project catalog.
		out = append(out, map[string]interface{}{
			"project_id":            pid,
			"repo_index":            0,
			"base_branch":           base,
			"target_branch":         target,
			"stored_repo_address":   addr,
			"project_repo_url":      "",
			"repo_address_mismatch": false,
		})
	}
	return out
}

func loadBranchStrategy(taskID string) map[string]interface{} {
	var work, merge, target string
	err := db.QueryRow(`SELECT work_branch_name,merge_target_branch_name,target_branch_name FROM task_branch_strategies WHERE task_id=?`, taskID).
		Scan(&work, &merge, &target)
	if err != nil {
		return nil
	}
	return map[string]interface{}{
		"work_branch_name":         work,
		"merge_target_branch_name": merge,
		"target_branch_name":       target,
	}
}

func loadRepoIdentities(taskID string) map[string]string {
	rows, _ := db.Query(`SELECT repo_url,git_identity_id FROM task_repo_identities WHERE task_id=?`, taskID)
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var url, gid string
		rows.Scan(&url, &gid)
		if url != "" && gid != "" {
			out[url] = gid
		}
	}
	return out
}

func taskToJSON(t *taskRecord, tenantID string) map[string]interface{} {
	assignees := loadAssignees(t.ID)
	if assignees == nil {
		assignees = []string{}
	}
	out := map[string]interface{}{
		"id":                                t.ID,
		"tenant_id":                         t.TenantID,
		"workspace_seq":                     t.WorkspaceSeq,
		"title":                             t.Title,
		"description":                       t.Description,
		"completed":                         t.Completed,
		"progress_column_id":                nilIfEmpty(t.ProgressColumnID),
		"priority":                          t.Priority,
		"order":                             fmt.Sprintf("%d", t.OrderNum),
		"workspace_id":                      t.WorkspaceID,
		"owner":                             nilIfEmpty(t.OwnerID),
		"operator":                          nilIfEmpty(t.OperatorID),
		"assignees":                         assignees,
		"deliverable_obj_id":                nilIfEmpty(t.DeliverableObjID),
		"deliverable_obj":                   resolveDeliverableObj(tenantID, t.DeliverableObjID),
		"container_image_id":                nilIfEmpty(t.InstalledImageID),
		"image_skill_id":                    nilIfEmpty(t.ImageSkillID),
		"container_image_snapshot":          snapshotToJSON(t.ContainerImageSnapshot),
		"agent_model":                       nilIfEmpty(t.AgentModel),
		"container_image":                   nil,
		"fork_from":                         nilIfEmpty(t.ForkFromID),
		"parent_task":                       nilIfEmpty(t.ParentTaskID),
		"auto_run":                          t.AutoRun,
		"auto_commit_after_agent_complete":  t.AutoCommitAfterAgentComplete,
		"auto_run_start_skip_reason":        t.AutoRunStartSkipReason,
		"auto_run_start_skipped":            strings.TrimSpace(t.AutoRunStartSkipReason) != "",
		"feature_params_source":             t.FeatureParamsSource,
		"personal_feature_params_config_id": t.PersonalFeatureParamsConfigID,
		"task_kind":                         nilIfEmpty(t.TaskKind),
		"code_lang":                         nilIfEmpty(t.CodeLang),
		"post_expires_at":                   postExpiresAtJSON(t.PostExpiresAt),
		"post_expired":                      t.postExpired(),
		"projects":                          loadProjects(t.ID, tenantID),
		"comments":                          []interface{}{},
		"ai_comments":                       []interface{}{},
		"created_at":                        t.CreatedAt.UTC().Format(time.RFC3339Nano),
		"updated_at":                        t.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	if bs := loadBranchStrategy(t.ID); bs != nil {
		out["branch_strategy"] = bs
	}
	if ids := loadRepoIdentities(t.ID); len(ids) > 0 {
		out["repo_clone_git_identities"] = ids
	}
	enrichTaskJSONWithQueue(out, t.ID)
	return out
}

func nilIfEmpty(s string) interface{} {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

// parsePostExpiresAtUTC 将 taskBill 返回的到期时间字符串（UTC 墙钟
// "2006-01-02 15:04:05.000000"）解析为 UTC time.Time，用于落库与回传。
// 注意：post_expires_at 列的读回由 DSN loc=Local 处理为正确 UTC，勿直接解析读回字符串。
func parsePostExpiresAtUTC(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if t, err := time.Parse("2006-01-02 15:04:05.000000", s); err == nil {
		return t, true
	}
	if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
		return t, true
	}
	return time.Time{}, false
}

// postExpired 判断创建帖是否已过 12 个月有效期（post_expires_at 非空且早于当前 UTC 时间）。
// post_expires_at 为空（历史数据兜底）视为未过期，由每日到期扫描统一补齐。
func (t *taskRecord) postExpired() bool {
	return t.PostExpiresAt.Valid && t.PostExpiresAt.Time.Before(time.Now().UTC())
}

// postExpiresAtJSON 序列化到期时间：有效 → RFC3339Nano 字符串，无效 → nil。
func postExpiresAtJSON(pe sql.NullTime) interface{} {
	if !pe.Valid {
		return nil
	}
	return pe.Time.UTC().Format(time.RFC3339Nano)
}

// expiresAtDBValue 将 taskBill 到期字符串转为入库 time.Time（driver 按 loc=Local 转北京墙钟）。
// 解析失败时返回 nil，交由调用方兜底。
func expiresAtDBValue(s string) interface{} {
	if parsed, ok := parsePostExpiresAtUTC(s); ok {
		return parsed
	}
	return nil
}

func saveTaskProjects(ctx context.Context, taskID, tenantID string, projects []interface{}) error {
	if projects == nil {
		return nil
	}
	soleProjectID := ""
	seenRepoIndex := map[int]struct{}{}
	for i, raw := range projects {
		item, ok := raw.(map[string]interface{})
		if !ok {
			return errProjectsMustBeObject(i)
		}
		pid := strField(item, "project_id")
		if pid == "" {
			continue
		}
		if soleProjectID == "" {
			soleProjectID = pid
		} else if soleProjectID != pid {
			return fmt.Errorf("%s", errMsgOneProjectOnly)
		}
		repoIndex := intFromField(item, "repo_index")
		if _, dup := seenRepoIndex[repoIndex]; dup {
			return errProjectsDuplicateRepoIndex(i, repoIndex)
		}
		seenRepoIndex[repoIndex] = struct{}{}
		base := strField(item, "base_branch")
		if base == "" {
			return errProjectsBaseBranchRequired(i)
		}
		target := retargetEmbeddedTaskIDs(strField(item, "target_branch"), taskID)
		urls, _ := getProjectRepoURLs(ctx, tenantID, pid)
		addr := ""
		if len(urls) > 0 {
			if repoIndex >= 0 && repoIndex < len(urls) {
				addr = urls[repoIndex]
			} else {
				addr = urls[0]
			}
		}
		tpID := genID("tp")
		db.Exec(`INSERT INTO task_projects(id,task_id,project_id,base_branch,target_branch,repo_address) VALUES(?,?,?,?,?,?)`,
			tpID, taskID, pid, base, target, addr)
	}
	return nil
}

// expandBranchNameTemplate 将分支名中的任务 ID 模板占位符（${taskId} / {taskId}）
// 替换为真实 taskID。FE 创建任务时（新任务尚无 ID）会把占位符写进分支名预览并随
// 创建请求持久化，若服务端不做替换，容器按分支名建分支/合入目标时拿到字面量
// `${taskId}`（OPT-20260809-025：任务 task_15370673744378765865 的
// task_branch_strategies / task_projects.target_branch 均含未替换字面量）。
func expandBranchNameTemplate(name, taskID string) string {
	if strings.Contains(name, "${taskId}") {
		name = strings.ReplaceAll(name, "${taskId}", taskID)
	}
	if strings.Contains(name, "{taskId}") {
		name = strings.ReplaceAll(name, "{taskId}", taskID)
	}
	return name
}

func saveBranchStrategy(taskID string, body map[string]interface{}) {
	bs, ok := body["branch_strategy"].(map[string]interface{})
	if !ok {
		return
	}
	db.Exec(`INSERT INTO task_branch_strategies(task_id,work_branch_name,merge_target_branch_name,target_branch_name) VALUES(?,?,?,?)
		ON DUPLICATE KEY UPDATE work_branch_name=VALUES(work_branch_name),
		merge_target_branch_name=VALUES(merge_target_branch_name), target_branch_name=VALUES(target_branch_name)`,
		taskID,
		retargetEmbeddedTaskIDs(strField(bs, "work_branch_name"), taskID),
		expandBranchNameTemplate(strField(bs, "merge_target_branch_name"), taskID),
		retargetEmbeddedTaskIDs(strField(bs, "target_branch_name"), taskID))
}

func intFromField(m map[string]interface{}, key string) int {
	switch v := m[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case string:
		var n int
		fmt.Sscanf(v, "%d", &n)
		return n
	default:
		return 0
	}
}

func memberIDsFromBody(body map[string]interface{}, key string) []string {
	raw, ok := body[key]
	if !ok {
		return nil
	}
	switch v := raw.(type) {
	case []interface{}:
		out := []string{}
		for _, item := range v {
			out = append(out, fmt.Sprintf("%v", item))
		}
		return out
	default:
		return nil
	}
}

func ownerFromBody(body map[string]interface{}) string {
	if v := strField(body, "owner"); v != "" {
		return v
	}
	return strField(body, "owner_id")
}

func operatorFromBody(body map[string]interface{}) string {
	if v := strField(body, "operator"); v != "" {
		return v
	}
	return strField(body, "operator_id")
}
