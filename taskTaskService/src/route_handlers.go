package main

import (
	"fmt"
	"net/http"
	"strings"
)

func handleTaskRoutes(w http.ResponseWriter, r *http.Request) {
	tenantID := getAuthTenant(r)
	if tenantID == "" {
		parts := cleanPath(r, "/api/tenant/")
		if len(parts) > 0 {
			tenantID = parts[0]
		}
	}
	userID := getAuthUser(r)
	workspaceID := r.Header.Get("X-Workspace-Id")
	taskID := r.Header.Get("X-Resource-Id")

	parts := cleanPath(r, "/api/tenant/")
	if len(parts) >= 4 && parts[1] == "workspace" && parts[3] == "todos" {
		workspaceID = parts[2]
		if len(parts) >= 5 {
			taskID = parts[4]
		}
	}

	// Sub-resource: repo-clone-git-identities
	if taskID != "" && strings.Contains(r.URL.Path, "/repo-clone-git-identities") {
		if r.Method == http.MethodPatch {
			handlePatchRepoCloneGitIdentities(w, r, tenantID, userID, workspaceID, taskID)
			return
		}
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if taskID != "" && strings.Contains(r.URL.Path, "/revisions") {
		handleTaskRevisionRoutes(w, r, tenantID, userID, workspaceID, taskID)
		return
	}

	// Sub-resource actions on task
	if taskID != "" {
		suffix := strings.TrimSuffix(r.URL.Path, "/")
		switch {
		case strings.HasSuffix(suffix, "/subtree"):
			if r.Method == http.MethodGet {
				handleGetTaskSubtree(w, r, tenantID, userID, taskID)
				return
			}
		case strings.HasSuffix(suffix, "/queued-auto-run"):
			if r.Method == http.MethodGet {
				handleGetQueuedAutoRun(w, r, tenantID, userID, taskID)
				return
			}
		case strings.HasSuffix(suffix, "/switch"):
			if r.Method == http.MethodPatch || r.Method == http.MethodPost {
				handleSwitchTask(w, r, tenantID, userID, taskID)
				return
			}
		case strings.HasSuffix(suffix, "/renew"):
			if r.Method == http.MethodPost {
				handleRenewTask(w, r, tenantID, userID, taskID)
				return
			}
		case strings.HasSuffix(suffix, "/associate"):
			if r.Method == http.MethodPost {
				handleAssociateTask(w, r, tenantID, userID, taskID)
				return
			}
		}
	}

	// batch-delete at collection level
	if workspaceID != "" && strings.HasSuffix(r.URL.Path, "/batch-delete") {
		if r.Method == http.MethodPost {
			handleBatchDeleteTasks(w, r, tenantID, userID)
			return
		}
	}

	if taskID == "" {
		switch r.Method {
		case http.MethodGet:
			if workspaceID != "" {
				handleListWorkspaceTasks(w, r, tenantID, userID, workspaceID)
			} else {
				handleListTasks(w, r, tenantID, userID)
			}
		case http.MethodPost:
			handleCreateTask(w, r, tenantID, userID)
		default:
			writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}

	switch r.Method {
	case http.MethodGet:
		handleGetTask(w, r, tenantID, userID, taskID)
	case http.MethodPut, http.MethodPatch:
		handleUpdateTask(w, r, tenantID, userID, taskID)
	case http.MethodDelete:
		handleDeleteTask(w, r, tenantID, userID, taskID)
	default:
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func handleCommentRoutes(w http.ResponseWriter, r *http.Request) {
	tenantID := getAuthTenant(r)
	userID := getAuthUser(r)
	taskID := r.Header.Get("X-Task-Id")
	commentID := ""
	parts := cleanPath(r, "/api/tenant/")
	if len(parts) >= 3 && parts[1] == "tasks" {
		taskID = parts[2]
		if len(parts) >= 5 && parts[3] == "comments" {
			commentID = parts[4]
		}
	}
	if taskID == "" {
		writeError(w, r, http.StatusBadRequest, "task id required")
		return
	}
	if commentID != "" {
		switch r.Method {
		case http.MethodPatch:
			handlePatchComment(w, r, tenantID, userID, taskID, commentID)
		default:
			writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}
	switch r.Method {
	case http.MethodGet:
		handleListComments(w, r, tenantID, userID, taskID)
	case http.MethodPost:
		handleCreateComment(w, r, tenantID, userID, taskID)
	default:
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func handlePatchRepoCloneGitIdentities(w http.ResponseWriter, r *http.Request, tenantID, userID, workspaceID, taskID string) {
	t, err := loadTask(taskID)
	if err != nil || t.TenantID != tenantID || t.WorkspaceID != workspaceID {
		writeError(w, r, http.StatusNotFound, "not found")
		return
	}
	if userID != "internal" && !hasWorkspaceAccess(r.Context(), tenantID, workspaceID, userID) {
		writeError(w, r, http.StatusForbidden, "forbidden")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	incoming, ok := body["repo_clone_git_identities"].(map[string]interface{})
	if !ok {
		writeError(w, r, http.StatusBadRequest, "repo_clone_git_identities must be object")
		return
	}
	result := map[string]string{}
	for repoURL, rawID := range incoming {
		repoURL = strings.TrimSpace(repoURL)
		if repoURL == "" {
			continue
		}
		idStr := strings.TrimSpace(fmt.Sprintf("%v", rawID))
		if rawID == nil || idStr == "" || idStr == "<nil>" {
			db.Exec(`DELETE FROM task_repo_identities WHERE task_id=? AND repo_url=?`, taskID, repoURL)
			continue
		}
		rid := genID("tri")
		db.Exec(`INSERT INTO task_repo_identities(id,task_id,repo_url,git_identity_id) VALUES(?,?,?,?)
			ON DUPLICATE KEY UPDATE git_identity_id=VALUES(git_identity_id)`,
			rid, taskID, repoURL, idStr)
		result[repoURL] = idStr
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":                        true,
		"repo_clone_git_identities": result,
	})
}
