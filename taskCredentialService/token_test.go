package token_test

import (
	"database/sql"
	"testing"

	"taskCredentialService/domain"
	"taskCredentialService/infrastructure"
)

// setupTestDB creates a dedicated MySQL test database and runs the migration.
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db := setupMySQLTestDB(t)

	// Create tables (MySQL DDL)
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
		t.Fatalf("migrate test db: %v", err)
	}
	return db
}

func TestTokenRepository_SaveAndFind(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := infrastructure.NewSQLiteTokenRepository(db)
	token := &domain.ContainerToken{
		ID:                    "tok-001",
		TaskID:                "task-1",
		CompanyID:             "tenant-1",
		WorkspaceID:           "ws-1",
		ContainerAccessToken:  "access-abc123",
		ContainerRefreshToken: "refresh-xyz789",
	}
	token.ContainerAccessTokenExpiresAt = "2099-12-31 23:59:59"

	saved, err := repo.Save(token)
	if err != nil {
		t.Fatalf("save token: %v", err)
	}
	if saved.ContainerAccessToken != "access-abc123" {
		t.Errorf("expected access-abc123, got %s", saved.ContainerAccessToken)
	}

	found, err := repo.FindByAccessToken("access-abc123")
	if err != nil {
		t.Fatalf("find by access token: %v", err)
	}
	if found == nil {
		t.Fatal("expected token, got nil")
	}
	if found.TaskID != "task-1" {
		t.Errorf("expected task-1, got %s", found.TaskID)
	}
}

func TestTokenRepository_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := infrastructure.NewSQLiteTokenRepository(db)
	found, err := repo.FindByAccessToken("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != nil {
		t.Errorf("expected nil for nonexistent token")
	}
}

// TestTokenRepository_FindByAccessToken_NullExpires 回归：container_access_token_expires_at
// 为 NULL（未设置过期时间的令牌，手工重置/初始化路径产生）时 FindByAccessToken 必须正常返回，
// 不得报 converting NULL to string（OPT-20260809-024：NULL 导致令牌被误判
// TOKEN_NOT_FOUND → exchange-refresh 401 TOKEN_ACCESS_INVALID → 容器 fail-closed）。
func TestTokenRepository_FindByAccessToken_NullExpires(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec(`
		INSERT INTO credential_container_tokens (id, task_id, company_id, workspace_id,
			container_access_token, container_access_token_expires_at, container_refresh_token)
		VALUES (?, ?, ?, ?, ?, NULL, ?)
	`, "tok-null-exp", "task-null-exp", "tenant-1", "ws-1", "access-null-exp", "refresh-1")
	if err != nil {
		t.Fatalf("insert token with NULL expires: %v", err)
	}

	repo := infrastructure.NewSQLiteTokenRepository(db)
	found, err := repo.FindByAccessToken("access-null-exp")
	if err != nil {
		t.Fatalf("find by access token with NULL expires: %v", err)
	}
	if found == nil {
		t.Fatal("expected token, got nil")
	}
	if found.TaskID != "task-null-exp" {
		t.Errorf("expected task-null-exp, got %s", found.TaskID)
	}
	if found.ContainerAccessTokenExpiresAt != "" {
		t.Errorf("expected empty expires, got %q", found.ContainerAccessTokenExpiresAt)
	}
}

func TestTokenService_IssueToken(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_ = infrastructure.NewSQLiteTokenRepository(db)
	auditRepo := infrastructure.NewSQLiteAuditRepository(db)

	// The application service is tested via integration tests.
	// Verify audit repo works.
	event := &domain.TokenAuditEvent{
		ID:                "audit-001",
		TaskID:            "task-1",
		EventType:         "token_issued",
		AccessTokenSHA256: "sha256test",
	}
	if err := auditRepo.Save(event); err != nil {
		t.Fatalf("save audit event: %v", err)
	}
	events, err := auditRepo.FindByTaskID("task-1", 10)
	if err != nil {
		t.Fatalf("find audit events: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}
}

func TestTokenRepository_FindByTaskID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := infrastructure.NewSQLiteTokenRepository(db)
	t1 := &domain.ContainerToken{
		ID: "tok-001", TaskID: "task-1", CompanyID: "tenant-1",
		ContainerAccessToken: "access-old", ContainerRefreshToken: "refresh-old",
		ContainerAccessTokenExpiresAt: "2099-01-01 00:00:00",
	}
	t2 := &domain.ContainerToken{
		ID: "tok-002", TaskID: "task-1", CompanyID: "tenant-1",
		ContainerAccessToken: "access-new", ContainerRefreshToken: "refresh-new",
		ContainerAccessTokenExpiresAt: "2099-01-02 00:00:00",
	}
	repo.Save(t1)
	repo.Save(t2)

	found, err := repo.FindByTaskID("tenant-1", "task-1")
	if err != nil {
		t.Fatalf("find by task: %v", err)
	}
	if found == nil {
		t.Fatal("expected token")
	}
	// A token should be returned.
	if found.ID != "tok-001" && found.ID != "tok-002" {
		t.Errorf("expected tok-001 or tok-002, got %s", found.ID)
	}
}

func TestTaskScope_Validate(t *testing.T) {
	tests := []struct {
		name  string
		scope domain.TaskScope
		valid bool
	}{
		{"all set", domain.TaskScope{TenantID: "t", WorkspaceID: "w", TaskID: "tk"}, true},
		{"missing tenant", domain.TaskScope{WorkspaceID: "w", TaskID: "tk"}, false},
		{"missing workspace", domain.TaskScope{TenantID: "t", TaskID: "tk"}, false},
		{"missing task", domain.TaskScope{TenantID: "t", WorkspaceID: "w"}, false},
		{"all empty", domain.TaskScope{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.scope.Validate()
			if tt.valid && err != nil {
				t.Errorf("expected valid, got: %v", err)
			}
			if !tt.valid && err == nil {
				t.Errorf("expected error")
			}
		})
	}
}

func TestTokenRepository_FindByRefreshToken(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := infrastructure.NewSQLiteTokenRepository(db)
	token := &domain.ContainerToken{
		ID: "tok-rf-001", TaskID: "task-1", CompanyID: "tenant-1",
		ContainerAccessToken: "access-1", ContainerRefreshToken: "refresh-abc",
		ContainerAccessTokenExpiresAt: "2099-01-01 00:00:00",
	}
	repo.Save(token)

	found, err := repo.FindByRefreshToken("refresh-abc")
	if err != nil {
		t.Fatalf("find by refresh: %v", err)
	}
	if found == nil {
		t.Fatal("expected token")
	}
	if found.ID != "tok-rf-001" {
		t.Errorf("expected tok-rf-001, got %s", found.ID)
	}

	// Non-existent refresh token
	notFound, err := repo.FindByRefreshToken("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notFound != nil {
		t.Errorf("expected nil for nonexistent refresh token")
	}
}

func TestTokenRepository_UpdateBusinessAPIEndpoint(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := infrastructure.NewSQLiteTokenRepository(db)
	token := &domain.ContainerToken{
		ID: "tok-be-001", TaskID: "task-1", CompanyID: "tenant-1",
		ContainerAccessToken: "access-1",
	}
	repo.Save(token)

	endpoint := "https://biz.example.com/api"
	if err := repo.UpdateBusinessAPIEndpoint("tok-be-001", endpoint); err != nil {
		t.Fatalf("update business api endpoint: %v", err)
	}

	found, _ := repo.FindByAccessToken("access-1")
	if found == nil {
		t.Fatal("expected token")
	}
	if found.BusinessAPIEndpoint != endpoint {
		t.Errorf("expected %s, got %s", endpoint, found.BusinessAPIEndpoint)
	}
}

func TestTokenService_ValidateToken_ScopeMismatch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := infrastructure.NewSQLiteTokenRepository(db)
	auditRepo := infrastructure.NewSQLiteAuditRepository(db)
	svc := infrastructure.NewTestTokenService(repo, auditRepo)

	token := &domain.ContainerToken{
		ID: "tok-val-001", TaskID: "task-1", CompanyID: "tenant-1", WorkspaceID: "ws-1",
		ContainerAccessToken:          "access-scope-test",
		ContainerAccessTokenExpiresAt: "2099-01-01 00:00:00",
	}
	repo.Save(token)

	// Correct scope
	validScope := domain.TaskScope{TenantID: "tenant-1", WorkspaceID: "ws-1", TaskID: "task-1"}
	_, err := svc.ValidateToken("access-scope-test", validScope)
	if err != nil {
		t.Errorf("expected valid, got: %v", err)
	}

	// Wrong scope
	wrongScope := domain.TaskScope{TenantID: "tenant-2", WorkspaceID: "ws-1", TaskID: "task-1"}
	_, err = svc.ValidateToken("access-scope-test", wrongScope)
	if err == nil {
		t.Errorf("expected scope mismatch error")
	}
}

func TestTokenService_RefreshAccess_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := infrastructure.NewSQLiteTokenRepository(db)
	auditRepo := infrastructure.NewSQLiteAuditRepository(db)
	svc := infrastructure.NewTestTokenService(repo, auditRepo)

	token := &domain.ContainerToken{
		ID: "tok-ra-001", TaskID: "task-1", CompanyID: "tenant-1", WorkspaceID: "ws-1",
		ContainerAccessToken: "", ContainerRefreshToken: "refresh-ra-test",
	}
	repo.Save(token)

	scope := domain.TaskScope{TenantID: "tenant-1", WorkspaceID: "ws-1", TaskID: "task-1"}
	newAccess, expiresAt, err := svc.RefreshAccess("refresh-ra-test", scope)
	if err != nil {
		t.Fatalf("refresh access: %v", err)
	}
	if newAccess == "" {
		t.Fatal("expected non-empty access token")
	}
	if expiresAt == "" {
		t.Fatal("expected non-empty expires_at")
	}
}

func TestTokenService_RefreshAccess_InvalidRefreshToken(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := infrastructure.NewSQLiteTokenRepository(db)
	auditRepo := infrastructure.NewSQLiteAuditRepository(db)
	svc := infrastructure.NewTestTokenService(repo, auditRepo)

	scope := domain.TaskScope{TenantID: "tenant-1", WorkspaceID: "ws-1", TaskID: "task-1"}
	_, _, err := svc.RefreshAccess("nonexistent-refresh", scope)
	if err == nil {
		t.Fatal("expected error for invalid refresh token")
	}
}

func TestTokenService_EnsureAccessByScope_RefreshesAfterExchangeClearedAccess(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := infrastructure.NewSQLiteTokenRepository(db)
	auditRepo := infrastructure.NewSQLiteAuditRepository(db)
	svc := infrastructure.NewTestTokenService(repo, auditRepo)

	// exchange-refresh 后：access 清空、expires 空、仅剩 refresh —— 旧 by-scope 会 200 空 token → 网关 409
	token := &domain.ContainerToken{
		ID: "tok-ensure-001", TaskID: "task-1", CompanyID: "tenant-1", WorkspaceID: "ws-1",
		ContainerAccessToken: "", ContainerAccessTokenExpiresAt: "",
		ContainerRefreshToken: "refresh-ensure-test",
	}
	if _, err := repo.Save(token); err != nil {
		t.Fatalf("save: %v", err)
	}

	scope := domain.TaskScope{TenantID: "tenant-1", WorkspaceID: "ws-1", TaskID: "task-1"}
	got, err := svc.EnsureAccessByScope(scope)
	if err != nil {
		t.Fatalf("EnsureAccessByScope: %v", err)
	}
	if got == nil || got.ContainerAccessToken == "" {
		t.Fatal("expected non-empty access after ensure")
	}
	if got.ContainerAccessTokenExpiresAt == "" {
		t.Fatal("expected expires_at after ensure")
	}
}

func TestTokenService_EnsureAccessByScope_ReusesUsableAccess(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := infrastructure.NewSQLiteTokenRepository(db)
	auditRepo := infrastructure.NewSQLiteAuditRepository(db)
	svc := infrastructure.NewTestTokenService(repo, auditRepo)

	token := &domain.ContainerToken{
		ID: "tok-ensure-002", TaskID: "task-1", CompanyID: "tenant-1", WorkspaceID: "ws-1",
		ContainerAccessToken: "access-live", ContainerAccessTokenExpiresAt: "2099-01-01 00:00:00",
		ContainerRefreshToken: "refresh-unused",
	}
	if _, err := repo.Save(token); err != nil {
		t.Fatalf("save: %v", err)
	}

	scope := domain.TaskScope{TenantID: "tenant-1", WorkspaceID: "ws-1", TaskID: "task-1"}
	got, err := svc.EnsureAccessByScope(scope)
	if err != nil {
		t.Fatalf("EnsureAccessByScope: %v", err)
	}
	if got.ContainerAccessToken != "access-live" {
		t.Fatalf("expected reuse access-live, got %q", got.ContainerAccessToken)
	}
}

func TestTokenService_EnsureAccessByScope_ExpiredReturnsStaleAccessWithoutRefresh(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := infrastructure.NewSQLiteTokenRepository(db)
	auditRepo := infrastructure.NewSQLiteAuditRepository(db)
	svc := infrastructure.NewTestTokenService(repo, auditRepo)

	token := &domain.ContainerToken{
		ID: "tok-ensure-003", TaskID: "task-1", CompanyID: "tenant-1", WorkspaceID: "ws-1",
		ContainerAccessToken: "access-stale", ContainerAccessTokenExpiresAt: "2000-01-01 00:00:00",
		ContainerRefreshToken: "refresh-present",
	}
	if _, err := repo.Save(token); err != nil {
		t.Fatalf("save: %v", err)
	}

	scope := domain.TaskScope{TenantID: "tenant-1", WorkspaceID: "ws-1", TaskID: "task-1"}
	got, err := svc.EnsureAccessByScope(scope)
	if err != nil {
		t.Fatalf("EnsureAccessByScope: %v", err)
	}
	// 容器 auth 按 env 精确比对、不看 expires_at；过期非空 access 必须原样返回，禁止单方换新。
	if got.ContainerAccessToken != "access-stale" {
		t.Fatalf("expected stale access-stale unchanged, got %q", got.ContainerAccessToken)
	}
	if got.ContainerAccessTokenExpiresAt != "2000-01-01 00:00:00" {
		t.Fatalf("expected expires_at unchanged, got %q", got.ContainerAccessTokenExpiresAt)
	}
}
