package main

import (
	"errors"
	"net/http"
	"strings"
)

var errCommentIDRequired = errors.New("缺少评论ID")

// resolveScopedCloudServerConfig 只读指定评论 CSC。comment_id 必填，禁止回退任务级或其它评论。
func resolveScopedCloudServerConfig(companyID, workspaceID, taskID, commentID string) (*CloudServerConfig, error) {
	commentID = strings.TrimSpace(commentID)
	if commentID == "" {
		return nil, errCommentIDRequired
	}
	cfg, err := loadCloudServerConfigForComment(companyID, workspaceID, taskID, commentID)
	if err != nil || cfg == nil {
		return cfg, err
	}
	healCommentCSCMockMetaFromTaskBase(cfg)
	return cfg, nil
}

func containerForwardHasAddress(cfg *CloudServerConfig) bool {
	if cfg == nil {
		return false
	}
	origin, _ := normalizeURLOrigin(cfg.ServerURL)
	if origin != "" {
		return true
	}
	origin, _ = normalizeURLOrigin(cfg.BusinessAPIEndpoint)
	return origin != ""
}

func healAndKeepIfAddress(cfg *CloudServerConfig, err error) *CloudServerConfig {
	if err != nil || cfg == nil {
		return nil
	}
	healCommentCSCMockMetaFromTaskBase(cfg)
	if containerForwardHasAddress(cfg) {
		return cfg
	}
	return nil
}

// loadCloudServerConfigForGatewayForward 解析 SaaS→容器转发用的 CSC。
// csc_id / comment_id 优先（comment_id 非空时禁止回退其它评论）；
// 否则取最新已登记地址的评论级 CSC。任务级 comment_id=” 多为空模板，不能挡住评论级可达地址。
func loadCloudServerConfigForGatewayForward(companyID, workspaceID, taskID, commentID, cscID string) (*CloudServerConfig, error) {
	cscID = strings.TrimSpace(cscID)
	commentID = strings.TrimSpace(commentID)
	if cscID != "" {
		return loadCloudServerConfigByID(cscID)
	}
	if commentID != "" {
		return resolveScopedCloudServerConfig(companyID, workspaceID, taskID, commentID)
	}
	if cfg := healAndKeepIfAddress(loadNewestCloudServerConfigWithInstance(companyID, workspaceID, taskID)); cfg != nil {
		return cfg, nil
	}
	if cfg := healAndKeepIfAddress(loadNewestCloudServerConfigForTask(companyID, workspaceID, taskID)); cfg != nil {
		return cfg, nil
	}
	return loadCloudServerConfig(companyID, workspaceID, taskID)
}

func commentIDFromPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i := 0; i+1 < len(parts); i++ {
		if parts[i] == "comment_id" {
			return strings.TrimSpace(parts[i+1])
		}
	}
	return ""
}

func commentIDFromComputeRequest(r *http.Request, body map[string]interface{}) string {
	if r != nil {
		if v := commentIDFromPath(r.URL.Path); v != "" {
			return v
		}
	}
	if body != nil {
		if v := strField(body, "comment_id"); v != "" {
			return v
		}
	}
	if r == nil {
		return ""
	}
	return strings.TrimSpace(r.URL.Query().Get("comment_id"))
}

// workspaceIDFromComputeRequest 从路径/头/query 解析 workspace_id，供评论绑定等内部请求落库。
func workspaceIDFromComputeRequest(r *http.Request) string {
	if r == nil {
		return ""
	}
	if v := pathKV(r.URL.Path, "workspace_id"); v != "" {
		return v
	}
	if v := strings.TrimSpace(r.Header.Get("X-Workspace-Id")); v != "" {
		return v
	}
	return strings.TrimSpace(r.URL.Query().Get("workspace_id"))
}

func rejectMissingComputeCommentID(w http.ResponseWriter, r *http.Request, commentID string) bool {
	if strings.TrimSpace(commentID) != "" {
		return false
	}
	writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{
		"status": "error", "message": "缺少评论ID",
	})
	return true
}
