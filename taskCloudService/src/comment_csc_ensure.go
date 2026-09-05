package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
)

func commentCSCPlatformIsMockOrEmpty(platform string) bool {
	p := strings.ToLower(trim(platform))
	return p == "" || p == "mock" || p == "relay-local" || strings.HasPrefix(p, "relay")
}

func commentCSCCloudMetaIllegal(cfg *CloudServerConfig) bool {
	if cfg == nil {
		return true
	}
	return commentCSCPlatformIsMockOrEmpty(cfg.Platform) ||
		trim(cfg.Region) == "" ||
		trim(cfg.AuthorizationID) == ""
}

func fillIfEmpty(dst *string, src string) bool {
	if dst == nil || trim(*dst) != "" || trim(src) == "" {
		return false
	}
	*dst = src
	return true
}

// fillCommentCSCIllegalCloudMeta 只补评论行的非法/空云账号字段，不覆盖已有合法值，
// 不改 instance/server_url。任务模板仅作默认值来源。
func fillCommentCSCIllegalCloudMeta(cfg *CloudServerConfig, companyID, workspaceID, taskID string) bool {
	if cfg == nil {
		return false
	}
	base, bErr := loadCloudServerConfig(companyID, workspaceID, taskID)
	if bErr != nil || base == nil || commentCSCPlatformIsMockOrEmpty(base.Platform) {
		return false
	}
	changed := false
	if commentCSCPlatformIsMockOrEmpty(cfg.Platform) {
		cfg.Platform = base.Platform
		changed = true
	}
	if fillIfEmpty(&cfg.Region, base.Region) {
		changed = true
	}
	if fillIfEmpty(&cfg.ZoneID, base.ZoneID) {
		changed = true
	}
	if fillIfEmpty(&cfg.AuthorizationID, base.AuthorizationID) {
		changed = true
	}
	if fillIfEmpty(&cfg.SecurityGroupID, base.SecurityGroupID) {
		changed = true
	}
	if fillIfEmpty(&cfg.VswitchID, base.VswitchID) {
		changed = true
	}
	if fillIfEmpty(&cfg.ImageInvokerUserID, base.ImageInvokerUserID) {
		changed = true
	}
	return changed
}

// applyTaskBaseMetaToCommentCSC 新建评论行：只填空字段（新行为空，等价于克隆模板）。
func applyTaskBaseMetaToCommentCSC(cfg *CloudServerConfig, companyID, workspaceID, taskID string) {
	fillCommentCSCIllegalCloudMeta(cfg, companyID, workspaceID, taskID)
}

// upgradeCommentCSCMetaFromTaskBase 存量脏行按字段补齐。返回是否有变更。
func upgradeCommentCSCMetaFromTaskBase(cfg *CloudServerConfig, companyID, workspaceID, taskID string) bool {
	return fillCommentCSCIllegalCloudMeta(cfg, companyID, workspaceID, taskID)
}

// healCommentCSCMockMetaFromTaskBase 读路径自愈非法/空字段并落库，不覆盖评论级合法覆盖。
func healCommentCSCMockMetaFromTaskBase(cfg *CloudServerConfig) {
	if cfg == nil || trim(cfg.CommentID) == "" {
		return
	}
	if !fillCommentCSCIllegalCloudMeta(cfg, cfg.CompanyID, cfg.WorkspaceID, cfg.TaskID) {
		return
	}
	if err := upsertCloudServerConfig(*cfg); err != nil {
		log.Printf("[taskCloudService] event=comment_csc_heal_from_base_failed csc_id=%s task_id=%s comment_id=%s err=%v",
			cfg.ID, cfg.TaskID, cfg.CommentID, err)
		return
	}
	log.Printf("[taskCloudService] event=comment_csc_healed_from_base company_id=%s task_id=%s comment_id=%s csc_id=%s platform=%s region=%s",
		cfg.CompanyID, cfg.TaskID, cfg.CommentID, cfg.ID, cfg.Platform, cfg.Region)
}

// resolveWorkspaceIDForTask 从任务级/任意 CSC 行解析 workspace_id。
func resolveWorkspaceIDForTask(companyID, taskID string) string {
	companyID = trim(companyID)
	taskID = trim(taskID)
	if taskID == "" {
		return ""
	}
	q := `SELECT workspace_id FROM cloud_server_configs WHERE task_id=?`
	args := []interface{}{taskID}
	if companyID != "" {
		q += ` AND company_id=?`
		args = append(args, companyID)
	}
	q += ` ORDER BY CASE WHEN COALESCE(comment_id,'')='' THEN 0 ELSE 1 END, updated_at DESC LIMIT 1`
	var wid string
	if err := db.QueryRow(q, args...).Scan(&wid); err != nil {
		return ""
	}
	return trim(wid)
}

// ensureCommentCloudServerConfig 为评论确保独立 CSC 行（可并行挂不同机器）。
// comment_id 空：返回任务级默认 CSC；非空：按 UNIQUE(workspace_id,task_id,comment_id) 创建/复用。
func ensureCommentCloudServerConfig(companyID, workspaceID, taskID, commentID string) (*CloudServerConfig, error) {
	companyID = trim(companyID)
	workspaceID = trim(workspaceID)
	taskID = trim(taskID)
	commentID = trim(commentID)
	if companyID == "" || taskID == "" {
		return nil, fmt.Errorf("company_id and task_id required")
	}
	if workspaceID == "" {
		workspaceID = resolveWorkspaceIDForTask(companyID, taskID)
	}
	if workspaceID == "" {
		return nil, fmt.Errorf("workspace_id required for comment CSC")
	}

	existing, err := loadCloudServerConfigForComment(companyID, workspaceID, taskID, commentID)
	if err == nil && existing != nil {
		wasIllegal := commentCSCCloudMetaIllegal(existing)
		if commentID != "" && fillCommentCSCIllegalCloudMeta(existing, companyID, workspaceID, taskID) {
			if uErr := upsertCloudServerConfig(*existing); uErr != nil {
				log.Printf("[taskCloudService] event=comment_csc_upgrade_from_base_failed csc_id=%s err=%v", existing.ID, uErr)
			} else {
				log.Printf("[taskCloudService] event=comment_csc_upgraded_from_base company_id=%s task_id=%s comment_id=%s csc_id=%s platform=%s region=%s",
					companyID, taskID, commentID, existing.ID, existing.Platform, existing.Region)
				if wasIllegal && !commentCSCCloudMetaIllegal(existing) {
					recoverFailedCommentBindingForComment(companyID, taskID, commentID, existing.ID, "comment_csc_upgraded_from_base")
				}
			}
		}
		return existing, nil
	}
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	cfg := CloudServerConfig{
		ID:          genID("csc"),
		CompanyID:   companyID,
		WorkspaceID: workspaceID,
		TaskID:      taskID,
		CommentID:   commentID,
	}
	if commentID != "" {
		applyTaskBaseMetaToCommentCSC(&cfg, companyID, workspaceID, taskID)
	}
	if commentID != "" && commentCSCCloudMetaIllegal(&cfg) {
		return nil, fmt.Errorf("评论级云主机需要真实云平台、地域与授权（platform/region/authorization_id 不能为空或 mock）")
	}

	if err := upsertCloudServerConfig(cfg); err != nil {
		return nil, err
	}
	log.Printf("[taskCloudService] event=comment_csc_ensured company_id=%s workspace_id=%s task_id=%s comment_id=%s csc_id=%s",
		companyID, workspaceID, taskID, commentID, cfg.ID)
	return &cfg, nil
}
