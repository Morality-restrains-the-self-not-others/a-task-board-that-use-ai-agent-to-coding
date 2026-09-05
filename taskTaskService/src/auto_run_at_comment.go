package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"taskTaskService/src/domain"
)

const (
	autoRunAtCommentSource = "auto_run"
	autoRunAtCommentPrefix = "【自动运行】"
)

// ensureAutoRunAtCommentFn is replaceable in tests.
var ensureAutoRunAtCommentFn = ensureAutoRunAtComment

// resolveAutoRunCommentID requires a non-empty comment ID before start-vm.
func resolveAutoRunCommentID(p autoRunTriggerParams) (string, error) {
	id, err := ensureAutoRunAtCommentFn(p)
	if err != nil {
		return "", err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return "", fmt.Errorf("auto_run comment_id required")
	}
	return id, nil
}

// attachStartVmCommentID writes comment_id and parent_comment_id (mention-path alignment),
// plus container_name derived with the same rule as cloudcommon.DeriveContainerName
// (taskID 已带 task_ 前缀时不重复加前缀，避免 task_task_ 双前缀)。
func attachStartVmCommentID(body map[string]interface{}, commentID, taskID string) map[string]interface{} {
	if body == nil {
		body = map[string]interface{}{}
	}
	cid := strings.TrimSpace(commentID)
	if cid == "" {
		return body
	}
	body["comment_id"] = cid
	body["parent_comment_id"] = cid
	if cn, _ := body["container_name"].(string); strings.TrimSpace(cn) == "" {
		tid := strings.TrimSpace(taskID)
		if tid != "" {
			if strings.HasPrefix(tid, "task_") {
				body["container_name"] = tid + "_" + cid
			} else {
				body["container_name"] = "task_" + tid + "_" + cid
			}
		}
	}
	return body
}

// composeAutoRunAtCommentContent builds human comment body for synthetic auto_run @ mention.
// When startSkipReason is non-empty (soft-skip start-vm), append the reason so operators see why
// the server was not started while the auto-run comment is still posted.
func composeAutoRunAtCommentContent(title, description, startSkipReason string) string {
	t := strings.TrimSpace(title)
	d := strings.TrimSpace(description)
	body := ""
	switch {
	case t != "" && d != "":
		body = t + "\n\n" + d
	case t != "":
		body = t
	default:
		body = d
	}
	if body == "" {
		body = "自动运行"
	}
	out := autoRunAtCommentPrefix + "\n" + body
	if reason := strings.TrimSpace(startSkipReason); reason != "" {
		out += "\n\n未启动服务器：" + reason
	}
	return out
}

type activeAutoRunAgent struct {
	ParentCommentID string
	AgentCommentID  string
}

func findActiveAutoRunAgentByTask(taskID string) (*activeAutoRunAgent, error) {
	base := strings.TrimSpace(cfg.AICommentServiceURL)
	if base == "" {
		if local := findLocalAutoRunParentComment(taskID); local != "" {
			return &activeAutoRunAgent{ParentCommentID: local}, nil
		}
		return nil, nil
	}
	u := strings.TrimRight(base, "/") + "/api/internal/task-ai-comment/container-agent-comments/active-by-task?task_id=" + url.QueryEscape(taskID)
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	setAICommentInternalSecret(req)
	resp, err := aiCommentHTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusNotFound {
		if local := findLocalAutoRunParentComment(taskID); local != "" {
			return &activeAutoRunAgent{ParentCommentID: local}, nil
		}
		return nil, nil
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("active-by-task status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	pack, _ := payload["context_pack"].(map[string]interface{})
	if pack == nil {
		return nil, nil
	}
	at, _ := pack["at_mention_run"].(map[string]interface{})
	if at == nil {
		return nil, nil
	}
	if strings.TrimSpace(fmt.Sprintf("%v", at["source"])) != autoRunAtCommentSource {
		return nil, nil
	}
	parentID := strings.TrimSpace(fmt.Sprintf("%v", at["parent_comment_id"]))
	agentID := strings.TrimSpace(fmt.Sprintf("%v", payload["id"]))
	if agentID == "" || agentID == "<nil>" {
		agentID = strings.TrimSpace(fmt.Sprintf("%v", at["agent_comment_id"]))
		if agentID == "<nil>" {
			agentID = ""
		}
	}
	if parentID == "" || parentID == "<nil>" {
		if local := findLocalAutoRunParentComment(taskID); local != "" {
			return &activeAutoRunAgent{ParentCommentID: local, AgentCommentID: agentID}, nil
		}
		return nil, nil
	}
	return &activeAutoRunAgent{ParentCommentID: parentID, AgentCommentID: agentID}, nil
}

func findLocalAutoRunParentComment(taskID string) string {
	if db == nil || strings.TrimSpace(taskID) == "" {
		return ""
	}
	var id string
	err := db.QueryRow(
		`SELECT id FROM task_comments WHERE task_id=? AND content LIKE ? ORDER BY created_at DESC, id DESC LIMIT 1`,
		taskID, autoRunAtCommentPrefix+"%",
	).Scan(&id)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(id)
}

// ensureAutoRunAtComment creates a synthetic @镜像 human comment for auto_run.
// Pending Agent comments are created by the container after bootstrap (OPT-20260823-008).
func ensureAutoRunAtComment(p autoRunTriggerParams) (parentCommentID string, err error) {
	if db == nil {
		return "", fmt.Errorf("db not open")
	}
	if strings.TrimSpace(p.TaskID) == "" || strings.TrimSpace(p.ImageID) == "" {
		return "", fmt.Errorf("task_id and image_id required")
	}
	if existing, findErr := findActiveAutoRunAgentByTask(p.TaskID); findErr != nil {
		log.Printf("[taskTaskService] event=auto_run_at_comment_lookup_failed task_id=%s err=%v", p.TaskID, findErr)
	} else if existing != nil {
		idents := prepareAutoRunRepoIdentities(existing.ParentCommentID, strings.TrimSpace(p.UserID), p.TaskID, p.GrantTicket, p.RepoIdentities)
		if len(idents) > 0 {
			if _, updErr := db.Exec(
				`UPDATE task_comments SET repo_identities_json=? WHERE id=? AND task_id=?`,
				RepoIdentitiesJSON(idents), existing.ParentCommentID, p.TaskID,
			); updErr != nil {
				log.Printf(
					"[taskTaskService] event=auto_run_at_comment_identities_update_failed task_id=%s comment_id=%s err=%v",
					p.TaskID, existing.ParentCommentID, updErr,
				)
			} else {
				log.Printf(
					"[taskTaskService] event=auto_run_at_comment_identities_updated task_id=%s comment_id=%s repo_count=%d",
					p.TaskID, existing.ParentCommentID, len(p.RepoIdentities),
				)
			}
		}
		log.Printf(
			"[taskTaskService] event=auto_run_at_comment_reuse task_id=%s comment_id=%s agent_id=%s",
			p.TaskID, existing.ParentCommentID, existing.AgentCommentID,
		)
		return existing.ParentCommentID, nil
	}

	t, err := loadTask(p.TaskID)
	if err != nil {
		return "", err
	}
	img, err := lookupInstalledImageFn(p.TenantID, p.ImageID)
	if err != nil {
		return "", err
	}
	imageName := strings.TrimSpace(img.Name)
	if imageName == "" {
		imageName = p.ImageID
	}
	content := composeAutoRunAtCommentContent(t.Title, t.Description, p.StartSkipReason)
	mentions := []domain.ImageMention{{
		Type: domain.MentionTypeInstalledImage,
		ID:   p.ImageID,
		Name: imageName,
	}}
	_ = domain.ApplyMentionSkill(&mentions[0], img.ImageSkills)
	mentionsJSON := domain.MentionsJSON(mentions)
	id := genID("cmt")
	now := time.Now().UTC()
	createdBy := strings.TrimSpace(p.UserID)
	if createdBy == "" {
		createdBy = strings.TrimSpace(t.OwnerID)
	}
	identitiesJSON := RepoIdentitiesJSON(prepareAutoRunRepoIdentities(id, createdBy, p.TaskID, p.GrantTicket, p.RepoIdentities))
	if _, err := db.Exec(
		`INSERT INTO task_comments(id,task_id,created_by_id,content,mentions_json,repo_identities_json,created_at) VALUES(?,?,?,?,?,?,?)`,
		id, p.TaskID, createdBy, content, mentionsJSON, identitiesJSON, now,
	); err != nil {
		return "", err
	}
	log.Printf(
		"[taskTaskService] event=auto_run_at_comment_create task_id=%s comment_id=%s image_id=%s repo_count=%d",
		p.TaskID, id, p.ImageID, len(p.RepoIdentities),
	)
	if err := notifyContainerAgentPendingWithModels(
		p.TenantID, p.WorkspaceID, p.TaskID, id, p.ImageID, imageName, content, createdBy, p.AgentModels, autoRunAtCommentSource,
	); err != nil {
		log.Printf("[taskTaskService] event=auto_run_at_comment_agent_failed task_id=%s comment_id=%s err=%v", p.TaskID, id, err)
		// Human comment already visible; Agent pending failure must not block start-vm.
		return id, nil
	}
	log.Printf("[taskTaskService] event=auto_run_at_comment_ok task_id=%s comment_id=%s", p.TaskID, id)
	return id, nil
}
