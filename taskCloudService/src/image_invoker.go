package main

import (
	"net/http"
	"strings"
)

// resolveImageInvokerUserID returns the image invoker (评论人员 / 镜像调用人员).
// Prefer explicit body fields from TTS @-mention path; else authenticated start user.
func resolveImageInvokerUserID(r *http.Request, body map[string]interface{}) string {
	if body != nil {
		for _, key := range []string{"image_invoker_user_id", "comment_created_by_id"} {
			if v := strings.TrimSpace(strField(body, key)); v != "" {
				return v
			}
		}
	}
	if r == nil {
		return ""
	}
	if u := strings.TrimSpace(getAuthUser(r)); u != "" {
		return u
	}
	return strings.TrimSpace(r.Header.Get("X-User-Id"))
}

func setCloudServerImageInvokerUserID(companyID, workspaceID, taskID, invokerID string) error {
	companyID = strings.TrimSpace(companyID)
	taskID = strings.TrimSpace(taskID)
	invokerID = strings.TrimSpace(invokerID)
	if companyID == "" || taskID == "" || invokerID == "" {
		return nil
	}
	q := `UPDATE cloud_server_configs SET image_invoker_user_id=?, updated_at=CURRENT_TIMESTAMP
		WHERE company_id=? AND task_id=?`
	args := []interface{}{invokerID, companyID, taskID}
	if ws := strings.TrimSpace(workspaceID); ws != "" {
		q += ` AND workspace_id=?`
		args = append(args, ws)
	}
	_, err := db.Exec(q, args...)
	return err
}
