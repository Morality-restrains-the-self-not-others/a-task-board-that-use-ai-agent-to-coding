package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"taskTaskService/src/domain"
	"time"
)

func handleListComments(w http.ResponseWriter, r *http.Request, tenantID, userID, taskID string) {
	t, err := loadTask(taskID)
	if err == sql.ErrNoRows || t.TenantID != tenantID {
		writeError(w, r, http.StatusNotFound, errMsgNotFound)
		return
	}
	if !hasWorkspaceAccess(r.Context(), tenantID, t.WorkspaceID, userID) {
		writeError(w, r, http.StatusForbidden, errMsgForbiddenRead)
		return
	}
	page := parseHumanCommentPageQuery(r)
	if page.paginate {
		results, hasMore, lastCA, lastID, err := listHumanCommentsPage(taskID, page)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, humanCommentsPageResponse(results, hasMore, lastCA, lastID))
		return
	}
	rows, _ := db.Query(`SELECT `+humanCommentSelectCols+` FROM task_comments WHERE task_id=? ORDER BY created_at ASC, id ASC`, taskID)
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		item, _, err := scanHumanCommentRowFromRows(rows)
		if err != nil {
			continue
		}
		out = append(out, item)
	}
	writeJSON(w, http.StatusOK, out)
}

func handleCreateComment(w http.ResponseWriter, r *http.Request, tenantID, userID, taskID string) {
	t, err := loadTask(taskID)
	if err == sql.ErrNoRows || t.TenantID != tenantID {
		writeError(w, r, http.StatusNotFound, errMsgNotFound)
		return
	}
	hints, hasReplyHints := commentReplyPathHintsFrom(r)
	if hasReplyHints && hints.workspaceID != "" && t.WorkspaceID != hints.workspaceID {
		logInfo("event=comment_reply_workspace_mismatch task_id="+taskID+" path_workspace="+hints.workspaceID, taskID)
		writeError(w, r, http.StatusNotFound, errMsgNotFound)
		return
	}
	if !hasWorkspaceAccess(r.Context(), tenantID, t.WorkspaceID, userID) {
		writeError(w, r, http.StatusForbidden, errMsgForbidden)
		return
	}
	if !requirePostActive(w, t) {
		return
	}
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, errMsgInvalidJSON)
		return
	}
	content := strField(body, "content")
	if content == "" {
		writeError(w, r, http.StatusBadRequest, errMsgContentRequired)
		return
	}
	if content == taskID {
		writeError(w, r, http.StatusBadRequest, errMsgCommentContentIsTaskID)
		return
	}
	mentions, err := domain.ParseMentions(body["mentions"])
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	if len(mentions) > 0 {
		atMode, err := readWorkspaceAtModeEnabled(tenantID, t.WorkspaceID)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, errMsgWorkspaceUnavailable)
			return
		}
		if err := domain.ValidateMentions(atMode, mentions); err != nil {
			writeError(w, r, http.StatusBadRequest, err.Error())
			return
		}
		img, err := lookupInstalledImageFn(tenantID, mentions[0].ID, mentions[0].Name)
		if err != nil {
			if errors.Is(err, ErrInstalledImageNotFound) {
				writeError(w, r, http.StatusBadRequest, errMsgInstalledImageNotFound)
				return
			}
			logInfo("event=at_mention_image_lookup_failed task_id="+taskID+" err="+err.Error(), taskID)
			writeError(w, r, http.StatusBadGateway, errMsgInstalledImageLookupFailed)
			return
		}
		if mentions[0].Name == "" && img.Name != "" {
			mentions[0].Name = img.Name
		}
		if img.ID != "" {
			mentions[0].ID = img.ID
		}
		if err := domain.ApplyMentionSkill(&mentions[0], img.ImageSkills); err != nil {
			writeError(w, r, http.StatusBadRequest, err.Error())
			return
		}
	}
	mentionsJSON := domain.MentionsJSON(mentions)
	executionMode, err := normalizeCommentExecutionMode(strField(body, "execution_mode"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	dependsOnIDs := normalizeDependsOnCommentIDs(body["depends_on_comment_ids"])
	if executionMode == executionModeIndependent {
		dependsOnIDs = nil
	}
	dependsOnJSON := dependsOnCommentIDsJSON(dependsOnIDs)
	identitiesJSON, err := resolveCommentRepoIdentitiesJSON(body, loadTaskLinkedRepoURLs(taskID), len(mentions) > 0)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	// OPT-20260901-028: 提前生成 comment id，让 grant_ticket 路径的
	// COMMENT_GIT_OAUTH_GRANTED 事件能用 seed 同形幂等键（含 comment_id）。
	id := genID("cmt")
	if ticket := grantTicketFromBody(body); ticket != "" || strings.TrimSpace(identitiesJSON) != "" {
		sels, parseErr := ParseRepoIdentities(repoIdentitiesFromJSONColumn(identitiesJSON))
		if parseErr == nil && len(sels) > 0 {
			identitiesJSON = RepoIdentitiesJSON(prepareCommentOAuthIdentities(id, userID, taskID, ticket, sels))
		}
	}
	// OPT-20260821-036: 服务端 Git OAuth 门禁 — @镜像 启停容器需要任务关联仓的 Git OAuth 绑定。
	// FE 已拦截，但非 SPA 客户端可直接 POST /comments 绕过；资金/云资源路径默认 ≥ L3。
	// GitOAuthURL 未配置（dev/test 单机）时跳过，避免无依赖环境阻断；生产必配。
	if len(mentions) > 0 && strings.TrimSpace(cfg.GitOAuthURL) != "" {
		sels, parseErr := ParseRepoIdentities(repoIdentitiesFromJSONColumn(identitiesJSON))
		if parseErr == nil {
			if site := missingOAuthCapableGrantSite(sels, loadTaskLinkedRepoURLs(taskID)...); site != "" {
				logInfo("event=at_mention_git_oauth_grant_missing task_id="+taskID+" gitsite="+site+" user_id="+userID, taskID)
				writeErrorMap(w, r, http.StatusBadRequest, map[string]interface{}{
					"error":   "该任务评论尚未完成 Git OAuth 使用授权（gitsite=" + site + "），请重新完成授权后再提交并运行",
					"code":    errCodeCommentOAuthGrantMissing,
					"gitsite": site,
				})
				return
			}
		}
		for _, ident := range repoIdentitiesFromJSONColumn(identitiesJSON) {
			repoURL := strings.TrimSpace(strField(ident, "repo_url"))
			if !isHTTPGitRepoURL(repoURL) {
				continue
			}
			connected, err := gitOAuthUserAppConnection(tenantID, userID, repoURL)
			if err != nil {
				logInfo("event=at_mention_git_oauth_check_failed task_id="+taskID+" repo_url="+repoURL+" err="+err.Error(), taskID)
				writeErrorMap(w, r, http.StatusServiceUnavailable, map[string]interface{}{
					"error": "无法校验 Git OAuth 绑定，请稍后重试",
					"code":  "git_oauth_check_unavailable",
				})
				return
			}
			if !connected {
				logInfo("event=at_mention_git_oauth_not_bound task_id="+taskID+" repo_url="+repoURL+" user_id="+userID, taskID)
				writeErrorMap(w, r, http.StatusBadRequest, map[string]interface{}{
					"error":    "未绑定 Git OAuth，无法使用 @镜像 启停容器。请先完成该仓库的 Git 授权。",
					"code":     "git_oauth_not_connected",
					"repo_url": repoURL,
				})
				return
			}
		}
	}
	parentCommentID := strings.TrimSpace(strField(body, "parent_comment_id"))
	if hasReplyHints && hints.parentCommentID != "" {
		parentCommentID = hints.parentCommentID
	}
	if parentCommentID != "" && !commentBelongsToTask(taskID, parentCommentID) {
		logInfo("event=comment_reply_parent_not_found task_id="+taskID+" parent_comment_id="+parentCommentID, taskID)
		writeError(w, r, http.StatusBadRequest, errMsgParentCommentNotFound)
		return
	}
	gitPR := parseGitPRFromBody(body)
	if gitPR.htmlURL != "" {
		if existingID := findCommentIDByGitPR(taskID, gitPR.htmlURL); existingID != "" {
			existing := loadHumanCommentByID(existingID)
			if existing != nil {
				logInfo("event=git_pr_comment_idempotent_skip task_id="+taskID+" comment_id="+existingID, taskID)
				writeJSON(w, http.StatusOK, existing)
				return
			}
		}
	}
	now := time.Now().UTC()
	if _, err := db.Exec(
		`INSERT INTO task_comments(id,task_id,created_by_id,content,mentions_json,execution_mode,depends_on_comment_ids,repo_identities_json,parent_comment_id,git_pr_html_url,git_pr_json,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
		id, taskID, userID, content, mentionsJSON, executionMode, dependsOnJSON, identitiesJSON, parentCommentID, gitPR.htmlURL, gitPR.json, formatMySQLUTCDateTime(now),
	); err != nil {
		logInfo("event=comment_create_failed task_id="+taskID+" err="+err.Error(), taskID)
		writeError(w, r, http.StatusInternalServerError, errMsgFailedToCreateComment)
		return
	}
	if len(mentions) > 0 {
		m := mentions[0]
		logInfo(
			"event=at_mention_comment_create task_id="+taskID+
				" workspace_id="+t.WorkspaceID+
				" comment_id="+id+
				" mention_type="+m.Type+
				" mention_id="+m.ID,
			taskID,
		)
		eventData := buildTaskCommentImageMentionedData(
			taskID, tenantID, t.WorkspaceID, id, m.ID, m.Name, content, userID, executionMode, dependsOnIDs,
		)
		if m.Skill != "" {
			eventData["image_skill"] = m.Skill
		}
		if parsedIdents := repoIdentitiesFromJSONColumn(identitiesJSON); len(parsedIdents) > 0 {
			eventData["repo_identities"] = parsedIdents
		}
		if tpl := optionalServerRunTemplate(body); tpl != nil {
			eventData["server_run_template"] = tpl
		}
		if err := notifyContainerAgentPendingFn(tenantID, t.WorkspaceID, taskID, id, m.ID, m.Name, content, userID); err != nil {
			logInfo("event=at_mention_notify_agent_failed comment_id="+id+" task_id="+taskID+" err="+err.Error(), taskID)
		}
		_ = publishDomainEventFn(r.Context(), "TASK_COMMENT_IMAGE_MENTIONED", eventData, taskID)
		logInfo("event=at_mention_event_published comment_id="+id+" task_id="+taskID, taskID)
		logInfo("event=comment_repo_identities_saved comment_id="+id+" task_id="+taskID+" repo_count="+fmt.Sprint(len(repoIdentitiesFromJSONColumn(identitiesJSON))), taskID)
	}
	if gitPR.htmlURL != "" {
		logInfo("event=git_pr_comment_created task_id="+taskID+" comment_id="+id+" parent_comment_id="+parentCommentID, taskID)
		publishGitPrReplyCreatedSideEffects(
			r.Context(), taskID, tenantID, t.WorkspaceID, id, parentCommentID, gitPR.htmlURL, gitPR.provider, userID,
		)
	}
	resp := map[string]interface{}{
		"id": id,
		"created_by": map[string]interface{}{
			"id": userID, "username": userID, "email": "",
		},
		"content":                content,
		"execution_mode":         executionMode,
		"depends_on_comment_ids": dependsOnIDs,
		"repo_identities":        repoIdentitiesFromJSONColumn(identitiesJSON),
		"created_at":             now.Format(time.RFC3339Nano),
	}
	if len(mentions) > 0 {
		resp["mentions"] = mentions
	}
	applyGitPRToCommentItem(resp, parentCommentID, gitPR.htmlURL, gitPR.json)
	writeJSON(w, http.StatusCreated, resp)
}

func handlePatchComment(w http.ResponseWriter, r *http.Request, tenantID, userID, taskID, commentID string) {
	t, err := loadTask(taskID)
	if err == sql.ErrNoRows || t.TenantID != tenantID {
		writeError(w, r, http.StatusNotFound, errMsgNotFound)
		return
	}
	if !hasWorkspaceAccess(r.Context(), tenantID, t.WorkspaceID, userID) {
		writeError(w, r, http.StatusForbidden, errMsgForbidden)
		return
	}
	if !requirePostActive(w, t) {
		return
	}
	var taskIDFromRow string
	var prevMode string
	err = db.QueryRow(
		`SELECT task_id, execution_mode FROM task_comments WHERE id=?`, commentID,
	).Scan(&taskIDFromRow, &prevMode)
	if err == sql.ErrNoRows || taskIDFromRow != taskID {
		writeError(w, r, http.StatusNotFound, errMsgNotFound)
		return
	}
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, errMsgInvalidJSON)
		return
	}
	mode, err := normalizeCommentExecutionMode(strField(body, "execution_mode"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	if mode == prevMode {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"id": commentID, "execution_mode": mode,
		})
		return
	}
	if err := updateHumanCommentExecutionMode(commentID, mode); err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	logInfo(
		"event=comment_execution_mode_changed task_id="+taskID+
			" comment_id="+commentID+
			" previous="+prevMode+
			" execution_mode="+mode,
		r.Header.Get("X-Trace-Id"),
	)
	_ = publishDomainEvent(r.Context(), "CommentExecutionModeChanged", map[string]interface{}{
		"task_id":                 taskID,
		"tenant_id":               tenantID,
		"company_id":              tenantID,
		"workspace_id":            t.WorkspaceID,
		"comment_id":              commentID,
		"previous_execution_mode": prevMode,
		"execution_mode":          mode,
		"comment_kind":            "human",
	}, taskID)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id": commentID, "execution_mode": mode,
	})
}
