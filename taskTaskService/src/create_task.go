package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"tracelog"
)

// 服务端创建任务幂等（OPT-20260819-036）：fork_from+owner+标题 短时间窗去重，或客户端
// Idempotency-Key 去重。API 客户端/多标签/重试打出两次相同 POST 时只创建一次，
// 第二次返回首次已创建任务，不再重复扣任务帖配额与起机。
var (
	forkCreateGuardMu  sync.Mutex
	forkCreateGuardMap = map[string]*forkCreateClaim{}

	taskCreateDedupWindow = 10 * time.Second
	taskCreateClaimWait   = 15 * time.Second
)

type forkCreateClaim struct {
	key       string
	taskID    string
	claimedAt time.Time
	done      chan struct{}
}

func (c *forkCreateClaim) resolve(taskID string) {
	forkCreateGuardMu.Lock()
	if cur, ok := forkCreateGuardMap[c.key]; ok && cur == c {
		cur.taskID = taskID
	}
	forkCreateGuardMu.Unlock()
	close(c.done)
}

func (c *forkCreateClaim) abort() {
	forkCreateGuardMu.Lock()
	if cur, ok := forkCreateGuardMap[c.key]; ok && cur == c {
		delete(forkCreateGuardMap, c.key)
	}
	forkCreateGuardMu.Unlock()
	close(c.done)
}

// taskCreateDedupKey 构建去重键：优先客户端 Idempotency-Key，其次 fork 短时间窗键。
// 普通创建且无 Idempotency-Key 返回空（不去重）。
func taskCreateDedupKey(r *http.Request, tenantID, workspaceID, ownerID, title string, body map[string]interface{}) string {
	if idem := strings.TrimSpace(r.Header.Get("Idempotency-Key")); idem != "" {
		return "idem\x00" + tenantID + "\x00" + idem
	}
	forkFrom := strings.TrimSpace(strField(body, "fork_from"))
	if forkFrom == "" {
		return ""
	}
	return strings.Join([]string{
		"fork\x00", tenantID, workspaceID, ownerID, forkFrom, strings.TrimSpace(title),
	}, "\x00")
}

// claimTaskCreateDedup 尝试认领创建去重键。
//   - existingID != "" → 窗口内已有同键任务，应直接返回该任务（不重复创建/扣配额）。
//   - claim != nil → 本调用取得认领权，创建完成后必须 resolve/abort。
//   - 两者皆空 → 无去重（普通创建）。
func claimTaskCreateDedup(r *http.Request, tenantID, workspaceID, ownerID, title string, body map[string]interface{}) (existingID string, claim *forkCreateClaim) {
	key := taskCreateDedupKey(r, tenantID, workspaceID, ownerID, title, body)
	if key == "" {
		return "", nil
	}
	forkCreateGuardMu.Lock()
	defer forkCreateGuardMu.Unlock()
	for {
		cur, ok := forkCreateGuardMap[key]
		if !ok {
			nc := &forkCreateClaim{key: key, claimedAt: time.Now(), done: make(chan struct{})}
			forkCreateGuardMap[key] = nc
			return "", nc
		}
		age := time.Since(cur.claimedAt)
		if cur.taskID != "" && age < taskCreateDedupWindow {
			return cur.taskID, nil
		}
		if cur.taskID == "" && age < taskCreateClaimWait {
			// 并发创建进行中：等待其完成后再判（覆盖 92ms 级双 POST 竞态）。
			done := cur.done
			forkCreateGuardMu.Unlock()
			select {
			case <-done:
			case <-time.After(taskCreateClaimWait - age):
			}
			forkCreateGuardMu.Lock()
			continue
		}
		// 过期/超时认领：删除后重新认领（创建方已失败/丢弃）。
		delete(forkCreateGuardMap, key)
	}
}

func handleCreateTask(w http.ResponseWriter, r *http.Request, tenantID, userID string) {
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, errMsgInvalidJSON)
		return
	}
	workspaceID := strField(body, "workspace_id")
	if workspaceID == "" {
		writeError(w, r, http.StatusBadRequest, errMsgWorkspaceIDRequired)
		return
	}
	if _, err := verifyWorkspace(r.Context(), tenantID, workspaceID); err != nil {
		writeError(w, r, http.StatusNotFound, err.Error())
		return
	}
	if !hasWorkspaceAccess(r.Context(), tenantID, workspaceID, userID) {
		writeError(w, r, http.StatusForbidden, errMsgForbidden)
		return
	}
	ownerID := ownerFromBody(body)
	if ownerID == "" {
		ownerID = userID
	}
	operatorID := operatorFromBody(body)
	title := strField(body, "title")
	if title == "" {
		writeError(w, r, http.StatusBadRequest, errMsgTitleRequired)
		return
	}
	// OPT-20260818-001：Fork（fork_from 非空）由服务端强制落到工作区进度体系第一列，
	// 忽略其它 API 客户端复制的源任务列。须在进度列校验之前执行。
	applyForkProgressColumnOverride(body, tenantID, workspaceID)
	if _, err := validateTaskFieldsFromRequest(r, tenantID, workspaceID, body); err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	completed := boolField(body, "completed")
	autoRun := boolField(body, "auto_run")
	imageID := strField(body, "container_image_id")
	autoRunIdents, identErr := resolveAutoRunRepoIdentities(r.Context(), autoRun, userID, tenantID, "", body)
	if identErr != nil {
		writeErrorMap(w, r, http.StatusBadRequest, map[string]interface{}{
			"error":  identErr.Error(),
			"detail": identErr.Error(),
			"code":   "repo_identities_required",
		})
		return
	}
	var autoRunTpl map[string]interface{}
	if autoRun {
		tpl, gateErr := validateAutoRunPrerequisites(tenantID, imageID, linkedProjectIDsFromBody(body))
		if gateErr != nil {
			if writeAutoRunGateError(w, gateErr) {
				return
			}
			writeError(w, r, http.StatusBadRequest, gateErr.Error())
			return
		}
		autoRunTpl = tpl
	}
	// OPT-20260823-006：服务端一次请求批量派生（fork_count 2..99）。逐副本走既有创建
	// 与幂等逻辑，客户端单次 RTT 即可创建 N 个副本，配额不足时返回已创建数。
	forkCount := forkCountFromBody(body)
	autoRunSkipReason := ""
	if autoRun {
		autoRunSkipReason = probeGitAccessForAutoRun(r.Context(), userID, tenantID, linkedProjectIDsFromBody(body))
	}
	if forkCount > 1 {
		handleForkBatchCreate(w, r, tenantID, workspaceID, userID, body, ownerID, operatorID, title,
			completed, autoRun, imageID, autoRunIdents, autoRunTpl, autoRunSkipReason, forkCount)
		return
	}
	handleSingleTaskCreate(w, r, tenantID, workspaceID, userID, body, ownerID, operatorID, title,
		completed, autoRun, imageID, autoRunIdents, autoRunTpl, autoRunSkipReason)
}

// handleSingleTaskCreate 单任务创建（fork_count<=1）：createTaskOnce 成功后以 201 写回任务 JSON。
func handleSingleTaskCreate(w http.ResponseWriter, r *http.Request, tenantID, workspaceID, userID string,
	body map[string]interface{}, ownerID, operatorID, title string,
	completed, autoRun bool, imageID string, autoRunIdents []RepoIdentitySelection,
	autoRunTpl map[string]interface{}, autoRunSkipReason string) {

	_, payload, cerr := createTaskOnce(r, tenantID, workspaceID, userID, body, ownerID, operatorID, title,
		completed, autoRun, imageID, autoRunIdents, autoRunTpl, autoRunSkipReason)
	if cerr != nil {
		cerr.writeTo(w, r)
		return
	}
	writeJSON(w, http.StatusCreated, payload)
}

// forkCountFromBody 读取批量派生数：缺失/非法 → 1（单任务）；上限 99，与前端 clampForkCopyCount 对称。
func forkCountFromBody(body map[string]interface{}) int {
	n := intField(body, "fork_count")
	if n < 1 {
		return 1
	}
	if n > 99 {
		return 99
	}
	return n
}

// forkCopyIdempotencyKeyServer 服务端批量派生中每个副本使用的幂等键：<batch>:<i>，
// 与前端 forkCopyIdempotencyKey 同构，批请求重试时同一副本经既有幂等命中。
func forkCopyIdempotencyKeyServer(batchKey string, index int) string {
	return fmt.Sprintf("%s:%d", batchKey, index)
}

// createTaskErr 承载单次任务创建失败的状态与错误体；status 非零即失败。
type createTaskErr struct {
	status int
	msg    string
	body   map[string]interface{}
}

// writeTo 把失败写回 HTTP 响应（与原有 writeError/writeErrorMap 语义一致）。
func (e *createTaskErr) writeTo(w http.ResponseWriter, r *http.Request) {
	if e == nil || e.status == 0 {
		return
	}
	if e.body != nil {
		writeErrorMap(w, r, e.status, e.body)
		return
	}
	writeError(w, r, e.status, e.msg)
}

// responseBody 产出可随部分成功响应一并返回的错误体（供批量派生 partial 场景）。
func (e *createTaskErr) responseBody() map[string]interface{} {
	out := map[string]interface{}{"status": "error", "error": e.msg, "message": e.msg}
	if e != nil && e.body != nil {
		for k, v := range e.body {
			out[k] = v
		}
	}
	return out
}

// createTaskOnce 在单请求内创建一个任务（含扣配额/落库/自动运行接线）。
// 供单任务创建与批量 Fork 循环复用；所有写响应逻辑交由调用方。
// 返回 taskID 与最终 payload；cerr != nil 表示未创建成功。
func createTaskOnce(r *http.Request, tenantID, workspaceID, userID string,
	body map[string]interface{}, ownerID, operatorID, title string,
	completed, autoRun bool, imageID string, autoRunIdents []RepoIdentitySelection,
	autoRunTpl map[string]interface{}, autoRunSkipReason string) (taskID string, payload map[string]interface{}, cerr *createTaskErr) {

	// 服务端幂等（OPT-20260819-036）：fork 短时间窗 / Idempotency-Key 去重。
	// 必须在扣任务帖配额之前命中，避免重复 POST 重复扣费与重复起机。
	existingID, claim := claimTaskCreateDedup(r, tenantID, workspaceID, ownerID, title, body)
	if existingID != "" {
		if t, err := loadTask(existingID); err == nil && t != nil {
			log.Printf("[taskTaskService] event=task_create_dedup_hit task_id=%s tenant_id=%s workspace_id=%s owner_id=%s",
				existingID, tenantID, workspaceID, ownerID)
			return existingID, taskJSONWithCreatedBy(t, tenantID), nil
		}
		// 窗口内 claim 但任务已被删除/加载失败：继续创建新任务（不阻塞）。
		log.Printf("[taskTaskService] event=task_create_dedup_stale task_id=%s tenant_id=%s", existingID, tenantID)
	}
	createdID := ""
	if claim != nil {
		defer func() {
			if createdID != "" {
				claim.resolve(createdID)
			} else {
				claim.abort()
			}
		}()
	}
	// D4 契约：镜像技能绑定校验 + 名↔ID 映射快照（resolveTaskImageSkill）。
	// 在配额消费之前失败返回，避免无效绑定白扣任务帖配额。
	skillSnapshot, serr := resolveTaskImageSkill(tenantID, imageID,
		strField(body, "container_image_skill_id"), strField(body, "description"))
	if serr != nil {
		return "", nil, &createTaskErr{status: serr.status, body: map[string]interface{}{
			"error": serr.msg, "detail": serr.msg, "code": serr.code,
		}}
	}
	imageSkillID, snapshotJSON := "", ""
	if skillSnapshot != nil {
		imageSkillID = skillSnapshot.SkillID
		snapshotJSON, _ = skillSnapshot.encode()
	}
	agentModels, amErr := parseCreateTaskAgentModels(body, autoRun)
	if amErr != nil {
		return "", nil, &createTaskErr{status: http.StatusBadRequest, msg: amErr.Error()}
	}
	// OPT-20260825-014：把该份 auto_run 的 agent_models[0].model 持久化到任务行，
	// 详情页层级图「选择模型」默认展示此副本实际 overlay 的模型（不依赖任务级 feature-params）。
	agentModel := ""
	if len(agentModels) > 0 {
		agentModel = optionalJSONString(agentModels[0], "model")
	}
	id := genID("task")
	postExpiresAt := ""
	if !testModeSkipDjango(r) {
		expiresAt, err := consumeTaskPostQuota(tenantID, id, workspaceID, userID, firstProjectID(body))
		if err != nil {
			if ibe, ok := err.(*insufficientBalanceError); ok {
				return "", nil, &createTaskErr{status: http.StatusPaymentRequired, body: map[string]interface{}{
					"detail":          ibe.Message,
					"code":            "INSUFFICIENT_TASK_POST_QUOTA",
					"balance_points":  ibe.BalancePoints,
					"required_points": ibe.RequiredPoints,
				}}
			}
			return "", nil, &createTaskErr{status: http.StatusBadGateway, msg: err.Error()}
		}
		postExpiresAt = expiresAt
	}
	now := time.Now().UTC()
	if postExpiresAt == "" {
		// 计费服务未返回到期时间（兜底）：now + 12 个月
		postExpiresAt = now.AddDate(0, 12, 0).Format("2006-01-02 15:04:05.000000")
	}
	postExpiresAtVal := expiresAtDBValue(postExpiresAt) // driver 按 loc=Local 转北京墙钟存库
	orderNum := nextOrderNum(tenantID, workspaceID)
	fps := strField(body, "feature_params_source")
	if fps == "" {
		fps = "none"
	}
	pfpc := strField(body, "personal_feature_params_config_id")
	if hasFeatureParamsKeys(body) {
		if err := validateFeatureParamsSourceRequired(fps, pfpc); err != nil {
			return "", nil, &createTaskErr{status: http.StatusBadRequest, body: map[string]interface{}{
				"error":                 err.Error(),
				"message":               err.Error(),
				"code":                  "FEATURE_PARAMS_SOURCE_REQUIRED",
				"feature_params_source": []string{err.Error()},
			}}
		}
	}
	seq, revID, err := insertTaskWithWorkspaceSeq(
		id, tenantID, title, strField(body, "description"), boolToInt(completed), strFieldDefault(body, "priority", "medium"),
		orderNum, workspaceID, ownerID, operatorID, strField(body, "deliverable_obj_id"), strField(body, "progress_column_id"),
		strField(body, "parent_task"), strField(body, "fork_from"), imageID,
		imageSkillID, snapshotJSON,
		boolToInt(autoRun), boolToInt(boolField(body, "auto_commit_after_agent_complete")), fps, pfpc,
		strField(body, "task_kind"), strField(body, "code_lang"), agentModel, postExpiresAtVal, userID, now)
	if err != nil {
		log.Printf("[taskTaskService] event=workspace_seq_allocate_failed task_id=%s tenant_id=%s workspace_id=%s err=%v",
			id, tenantID, workspaceID, err)
		return "", nil, &createTaskErr{status: http.StatusInternalServerError, msg: err.Error()}
	}
	log.Printf("[taskTaskService] event=workspace_seq_allocated task_id=%s tenant_id=%s workspace_id=%s workspace_seq=%d",
		id, tenantID, workspaceID, seq)
	if assignees := memberIDsFromBody(body, "assignees"); assignees != nil {
		setAssignees(id, assignees)
	}
	if projs, ok := body["projects"].([]interface{}); ok {
		if err := saveTaskProjects(r.Context(), id, tenantID, projs); err != nil {
			db.Exec(`DELETE FROM task_tasks WHERE id=?`, id)
			return "", nil, &createTaskErr{status: http.StatusBadRequest, msg: err.Error()}
		}
	}
	saveBranchStrategy(id, body)
	t, _ := loadTask(id)
	if t != nil {
		if err := applyScheduleRhythmFromBody(t, body); err != nil {
			return "", nil, &createTaskErr{status: http.StatusBadRequest, msg: err.Error()}
		}
		if err := applyQueuedAutoRunFromBody(t, body, userID); err != nil {
			db.Exec(`DELETE FROM task_tasks WHERE id=?`, id)
			return "", nil, &createTaskErr{status: httpStatusOf(err, http.StatusBadRequest), msg: err.Error()}
		}
		t, _ = loadTask(id)
	}
	payload = taskJSONWithCreatedBy(t, tenantID)
	if autoRun {
		if autoRunSkipReason != "" {
			applyAutoRunStartSkip(payload, autoRunSkipReason)
			persistAutoRunStartSkipReason(id, autoRunSkipReason)
			log.Printf("[taskTaskService] auto_run start skipped task_id=%s reason=%s", id, autoRunSkipReason)
		} else {
			clearAutoRunStartSkipReason(id)
		}
	}
	createdID = id
	taskID = id
	if pubErr := publishTaskCreatedFn(r.Context(), tenantID, workspaceID, id, title, userID, postExpiresAt, seq); pubErr != nil {
		log.Printf("[taskTaskService] publish TASK_CREATED failed task_id=%s: %v", id, pubErr)
	}
	if revID != "" {
		if pubErr := publishTaskRevisionRecordedFn(r.Context(), tenantID, workspaceID, id, revID, 1, userID, "title,description"); pubErr != nil {
			log.Printf("[taskTaskService] publish TASK_REVISION_RECORDED failed task_id=%s revision_id=%s: %v", id, revID, pubErr)
		}
	}
	// Soft-skip still schedules comment-only auto_run (StartSkipReason set → no start-vm).
	// Joining the workspace queue skips immediate start-vm so dispatcher owns start.
	if autoRun {
		if deferImmediateAutoRunStart(body, false) {
			log.Printf("[taskTaskService] event=task_auto_run_deferred_to_queue task_id=%s tenant_id=%s workspace_id=%s",
				id, tenantID, workspaceID)
			tracelog.LogForwardStage(r.Context(), "task_auto_run_deferred_to_queue", map[string]any{
				"task_id": id, "tenant_id": tenantID, "workspace_id": workspaceID,
			})
		} else {
			scheduleTaskAutoRunFn(autoRunTriggerParams{
				TenantID:        tenantID,
				WorkspaceID:     workspaceID,
				TaskID:          id,
				UserID:          userID,
				ImageID:         imageID,
				RunTemplate:     autoRunTpl,
				ClientPublicIP:  resolveAutoRunClientPublicIP(r, body),
				StartSkipReason: autoRunSkipReason,
				RepoIdentities:  autoRunIdents,
				AgentModels:     agentModels,
				GrantTicket:     grantTicketFromBody(body),
			})
		}
	}
	return taskID, payload, nil
}

// handleForkBatchCreate 服务端一次请求批量派生 N 个副本（OPT-20260823-006）。
// 逐副本复用 createTaskOnce；每个副本以 <batchKey>:<i> 作为 Idempotency-Key，
// 使批请求重试时已建副本经既有服务端幂等命中，不再重复扣配额/起机。
// 响应：全部成功 → 201 {ids, created_count, first_id, tasks}；部分成功 → 200
// {ids, created_count, partial:true, error:{...}}；首副本即失败 → 该失败状态。
func handleForkBatchCreate(w http.ResponseWriter, r *http.Request, tenantID, workspaceID, userID string,
	body map[string]interface{}, ownerID, operatorID, title string,
	completed, autoRun bool, imageID string, autoRunIdents []RepoIdentitySelection,
	autoRunTpl map[string]interface{}, autoRunSkipReason string, forkCount int) {

	batchKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if batchKey == "" {
		// 无客户端幂等键时用服务端随机键兜底，保证批内各副本去重键互不相同。
		batchKey = "batch\x00" + genID("fkb")
	}
	var ids []string
	var payloads []map[string]interface{}
	for i := 1; i <= forkCount; i++ {
		req := r.Clone(r.Context())
		req.Header.Set("Idempotency-Key", forkCopyIdempotencyKeyServer(batchKey, i))
		taskID, payload, cerr := createTaskOnce(req, tenantID, workspaceID, userID, body, ownerID, operatorID, title,
			completed, autoRun, imageID, autoRunIdents, autoRunTpl, autoRunSkipReason)
		if cerr != nil {
			if len(ids) == 0 {
				cerr.writeTo(w, r)
				return
			}
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"ids":           ids,
				"created_count": len(ids),
				"partial":       true,
				"error":         cerr.responseBody(),
			})
			return
		}
		ids = append(ids, taskID)
		payloads = append(payloads, payload)
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"ids":           ids,
		"created_count": len(ids),
		"first_id":      ids[0],
		"tasks":         payloads,
	})
}

func nextOrderNum(tenantID, workspaceID string) int {
	var max sql.NullInt64
	db.QueryRow(`SELECT MAX(order_num) FROM task_tasks WHERE tenant_id=? AND workspace_id=?`, tenantID, workspaceID).Scan(&max)
	if max.Valid {
		return int(max.Int64) + 1
	}
	return 0
}

// applyForkProgressColumnOverride enforces the server-side invariant that a Fork
// (fork_from set) always lands on the workspace progress-system's first column,
// ignoring a client-supplied progress_column_id copied from the source task
// (OPT-20260818-001). Resolution failure is non-fatal: the client value is kept.
func applyForkProgressColumnOverride(body map[string]interface{}, tenantID, workspaceID string) {
	if strField(body, "fork_from") == "" {
		return
	}
	firstCol, err := resolveFirstProgressColumnViaProjectService(tenantID, workspaceID)
	if err != nil {
		log.Printf("[taskTaskService] fork first progress column resolve failed tenant_id=%s workspace_id=%s err=%v", tenantID, workspaceID, err)
		return
	}
	if firstCol != "" {
		body["progress_column_id"] = firstCol
	}
}

// resolveFirstProgressColumnViaProjectService returns the workspace progress-system's
// first column id via taskProjectService internal endpoint.
func resolveFirstProgressColumnViaProjectService(tenantID, workspaceID string) (string, error) {
	if strings.TrimSpace(cfg.ProjectServiceURL) == "" {
		return "", nil
	}
	out, status, err := projectServicePost("/api/internal/progress-columns/first", map[string]interface{}{
		"tenant_id":    tenantID,
		"workspace_id": workspaceID,
	})
	if err != nil {
		return "", err
	}
	if status != 200 {
		return "", fmt.Errorf("resolve first progress column status %d", status)
	}
	return strField(out, "progress_column_id"), nil
}
