package interfaces

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"taskCredentialService/application"
	"taskCredentialService/domain"
	"taskCredentialService/infrastructure"
)

type stubBizRepo struct {
	snap  *domain.TaskSnapshot
	repos []domain.TaskRepoSnapshot
}

func (s *stubBizRepo) FetchTaskSnapshot(taskID, commentID string) (*domain.TaskSnapshot, error) {
	return s.snap, nil
}
func (s *stubBizRepo) FetchTaskRepos(taskID, commentID string) ([]domain.TaskRepoSnapshot, error) {
	if s.repos == nil {
		return []domain.TaskRepoSnapshot{}, nil
	}
	return s.repos, nil
}
func (s *stubBizRepo) FetchRepoIdentities(taskID, commentID string) ([]domain.GitIdentitySnapshot, error) {
	return nil, nil
}
func (s *stubBizRepo) FetchCommentCreatedByUserID(commentID string) (int64, error) {
	return 0, nil
}
func (s *stubBizRepo) FetchUserGitIdentities(userID int64) ([]domain.GitIdentitySnapshot, error) {
	return nil, nil
}

func setupTokenDB(t *testing.T) *sql.DB {
	t.Helper()
	db := setupMySQLTestDB(t)

	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS credential_container_tokens (
			id VARCHAR(64) PRIMARY KEY, task_id VARCHAR(64) NOT NULL, company_id VARCHAR(64) NOT NULL DEFAULT '',
			workspace_id VARCHAR(64) NOT NULL DEFAULT '',
			comment_id VARCHAR(64) NOT NULL DEFAULT '',
			container_access_token VARCHAR(512) NOT NULL DEFAULT '',
			container_access_token_expires_at VARCHAR(64) NULL,
			container_refresh_token VARCHAR(512) NOT NULL DEFAULT '',
			server_url VARCHAR(1024) NOT NULL DEFAULT '',
			business_api_endpoint VARCHAR(1024) NOT NULL DEFAULT '',
			container_vscode_url VARCHAR(1024) NOT NULL DEFAULT '',
			authorization_id VARCHAR(64) NOT NULL DEFAULT '',
			image_id VARCHAR(64) NOT NULL DEFAULT '',
			instance_type VARCHAR(128) NOT NULL DEFAULT '',
			region_id VARCHAR(64) NOT NULL DEFAULT '',
			zone_id VARCHAR(64) NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
		CREATE TABLE IF NOT EXISTS credential_token_audit_events (
			id VARCHAR(64) PRIMARY KEY, task_id VARCHAR(64) NOT NULL DEFAULT '',
			comment_id VARCHAR(64) NOT NULL DEFAULT '',
			event_type VARCHAR(64) NOT NULL DEFAULT '',
			access_token_sha256 VARCHAR(64) NOT NULL DEFAULT '',
			prev_access_token_sha256 VARCHAR(64) NOT NULL DEFAULT '',
			new_access_token_sha256 VARCHAR(64) NOT NULL DEFAULT '',
			refresh_token_sha256 VARCHAR(64) NOT NULL DEFAULT '',
			source_component VARCHAR(128) NOT NULL DEFAULT '',
			trace_id VARCHAR(64) NOT NULL DEFAULT '',
			error_code VARCHAR(64) NOT NULL DEFAULT '',
			error_detail TEXT,
			seq INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`)
	if err != nil {
		if isMySQLDiskFull(err) {
			t.Skipf("MySQL test DB storage full, skipping: %v", err)
		}
		t.Fatalf("schema: %v", err)
	}
	return db
}

func TestHandleTaskDetailIncludesAtMentionFields(t *testing.T) {
	db := setupTokenDB(t)
	tokenRepo := infrastructure.NewSQLiteTokenRepository(db)
	auditRepo := infrastructure.NewSQLiteAuditRepository(db)
	tokenSvc := application.NewTokenService(tokenRepo, auditRepo)
	scope := domain.TaskScope{TenantID: "t1", WorkspaceID: "w1", TaskID: "task_1", CommentID: "cmt-pack"}
	issued, err := tokenSvc.IssueToken(scope, nil)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}

	biz := &stubBizRepo{
		snap: &domain.TaskSnapshot{
			ID: "task_1", Title: "demo", Description: "d",
			CompanyID: "t1", WorkspaceID: "w1", AutoRun: false,
		},
		repos: []domain.TaskRepoSnapshot{},
	}
	svc := &infrastructure.AppServices{
		Token:      tokenSvc,
		Credential: application.NewCredentialService(tokenRepo, biz, nil),
		TaskDetail: application.NewTaskDetailService(biz),
	}
	h := NewHandlers(svc)
	h.fetchActiveContextPack = func(taskID string) map[string]interface{} {
		if taskID != "task_1" {
			return nil
		}
		return map[string]interface{}{
			"at_mention_run": map[string]interface{}{
				"run_id":            "agent-9",
				"parent_comment_id": "cmt-1",
				"agent_comment_id":  "agent-9",
				"installed_image":   map[string]string{"id": "img-1", "name": "Img"},
			},
			"comment_thread": []map[string]interface{}{
				{"kind": "human", "id": "cmt-1", "content": "@Img run"},
			},
		}
	}

	body := `{"access_token":"` + issued.ContainerAccessToken + `"}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/workspace/w1/task/task_1/comment/cmt-1/cloud/server-container-token/task-detail/",
		strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.handleTaskDetail(rec, req, "t1", "w1", "task_1", "cmt-pack")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	at, _ := resp["at_mention_run"].(map[string]interface{})
	if at == nil || at["run_id"] != "agent-9" {
		t.Fatalf("at_mention_run=%#v", resp["at_mention_run"])
	}
	thread, ok := resp["comment_thread"]
	if !ok || thread == nil {
		t.Fatalf("missing comment_thread: %#v", resp)
	}
}

func TestHandleTaskDetailOmitsPackWhenFetcherNil(t *testing.T) {
	db := setupTokenDB(t)
	tokenRepo := infrastructure.NewSQLiteTokenRepository(db)
	auditRepo := infrastructure.NewSQLiteAuditRepository(db)
	tokenSvc := application.NewTokenService(tokenRepo, auditRepo)
	scope := domain.TaskScope{TenantID: "t1", WorkspaceID: "w1", TaskID: "task_2", CommentID: "cmt-pack2"}
	issued, err := tokenSvc.IssueToken(scope, nil)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	biz := &stubBizRepo{
		snap: &domain.TaskSnapshot{
			ID: "task_2", Title: "demo", Description: "d",
			CompanyID: "t1", WorkspaceID: "w1",
		},
	}
	h := NewHandlers(&infrastructure.AppServices{
		Token:      tokenSvc,
		Credential: application.NewCredentialService(tokenRepo, biz, nil),
		TaskDetail: application.NewTaskDetailService(biz),
	})
	body := `{"access_token":"` + issued.ContainerAccessToken + `"}`
	req := httptest.NewRequest(http.MethodPost, "/task-detail", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.handleTaskDetail(rec, req, "t1", "w1", "task_2", "cmt-pack2")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if _, ok := resp["at_mention_run"]; ok {
		t.Fatalf("expected no at_mention_run, got %#v", resp["at_mention_run"])
	}
	if resp["idle_recycle_minutes"] != float64(5) {
		t.Fatalf("idle_recycle_minutes=%v want 5", resp["idle_recycle_minutes"])
	}
	idle, _ := resp["instruction_idle"].(map[string]interface{})
	if idle == nil || idle["enabled"] != true || idle["minutes"] != float64(5) {
		t.Fatalf("instruction_idle=%#v", resp["instruction_idle"])
	}
	if _, ok := resp["machine_release_sts"]; ok {
		t.Fatalf("machine_release_sts must be omitted: %#v", resp)
	}
}

func TestHandleTaskDetailSynthesizesAtMentionWhenNoActiveAgent(t *testing.T) {
	db := setupTokenDB(t)
	tokenRepo := infrastructure.NewSQLiteTokenRepository(db)
	auditRepo := infrastructure.NewSQLiteAuditRepository(db)
	tokenSvc := application.NewTokenService(tokenRepo, auditRepo)
	scope := domain.TaskScope{TenantID: "t1", WorkspaceID: "w1", TaskID: "task_syn", CommentID: "cmt-parent"}
	issued, err := tokenSvc.IssueToken(scope, nil)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	biz := &stubBizRepo{
		snap: &domain.TaskSnapshot{
			ID: "task_syn", Title: "demo", Description: "d",
			CompanyID: "t1", WorkspaceID: "w1", InstalledImageID: "img-9", AutoRun: false,
		},
	}
	h := NewHandlers(&infrastructure.AppServices{
		Token:      tokenSvc,
		Credential: application.NewCredentialService(tokenRepo, biz, nil),
		TaskDetail: application.NewTaskDetailService(biz),
	})
	h.fetchActiveContextPack = func(taskID string) map[string]interface{} { return nil }
	body := `{"access_token":"` + issued.ContainerAccessToken + `"}`
	req := httptest.NewRequest(http.MethodPost, "/task-detail", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.handleTaskDetail(rec, req, "t1", "w1", "task_syn", "cmt-parent")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	at, _ := resp["at_mention_run"].(map[string]interface{})
	if at == nil || at["parent_comment_id"] != "cmt-parent" {
		t.Fatalf("at_mention_run=%#v", resp["at_mention_run"])
	}
	if at["agent_comment_id"] != nil && at["agent_comment_id"] != "" {
		t.Fatalf("agent_comment_id must be omitted before container POST: %#v", at)
	}
	img, _ := at["installed_image"].(map[string]interface{})
	if img == nil || img["id"] != "img-9" {
		t.Fatalf("installed_image=%#v", at["installed_image"])
	}
	task, _ := resp["task"].(map[string]interface{})
	if task["installed_image_id"] != "img-9" {
		t.Fatalf("task.installed_image_id=%#v", task["installed_image_id"])
	}
}
