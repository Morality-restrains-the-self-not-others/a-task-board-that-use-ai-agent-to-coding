package token_test

import (
	"testing"

	"taskCredentialService/domain"
	"taskCredentialService/infrastructure"
)

// 回归：云主机 UserData 会 docker rm + docker run 同一预埋 ACCESS_TOKEN。
// 首次 exchange-refresh 已写入 refresh 并清空 access 后，新容器再调 exchange-refresh
// 必须幂等返回已有 refresh，否则 onlineServiceJS 无落盘 refresh → TOKEN_BOOTSTRAP_FAILED → 容器通信 idle。
func TestTokenService_ExchangeRefresh_IdempotentAfterFirstExchange(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := infrastructure.NewSQLiteTokenRepository(db)
	auditRepo := infrastructure.NewSQLiteAuditRepository(db)
	svc := infrastructure.NewTestTokenService(repo, auditRepo)

	scope := domain.TaskScope{TenantID: "tenant-1", WorkspaceID: "ws-1", TaskID: "task-idem-1", CommentID: "cmt-idem"}
	issued, err := svc.IssueToken(scope, nil)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	bootstrapAccess := issued.ContainerAccessToken
	if bootstrapAccess == "" {
		t.Fatal("expected non-empty bootstrap access")
	}

	first, err := svc.ExchangeRefresh(bootstrapAccess, "https://biz.example.com/api", scope)
	if err != nil {
		t.Fatalf("first exchange-refresh: %v", err)
	}
	if first == "" {
		t.Fatal("expected refresh token from first exchange")
	}

	second, err := svc.ExchangeRefresh(bootstrapAccess, "https://biz.example.com/api", scope)
	if err != nil {
		t.Fatalf("second exchange-refresh (container recreate) should be idempotent, got %v", err)
	}
	if second != first {
		t.Fatalf("idempotent exchange should return the same refresh, got %q want %q", second, first)
	}
}

func TestTokenService_ExchangeRefresh_IdempotentWhenCurrentAccessStillPresent(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := infrastructure.NewSQLiteTokenRepository(db)
	auditRepo := infrastructure.NewSQLiteAuditRepository(db)
	svc := infrastructure.NewTestTokenService(repo, auditRepo)

	token := &domain.ContainerToken{
		ID: "tok-cur-001", TaskID: "task-cur-1", CompanyID: "tenant-1", WorkspaceID: "ws-1", CommentID: "cmt-cur",
		ContainerAccessToken: "access-still-valid", ContainerRefreshToken: "existing-refresh",
		ContainerAccessTokenExpiresAt: "2099-01-01 00:00:00",
	}
	if _, err := repo.Save(token); err != nil {
		t.Fatalf("save: %v", err)
	}

	scope := domain.TaskScope{TenantID: "tenant-1", WorkspaceID: "ws-1", TaskID: "task-cur-1", CommentID: "cmt-cur"}
	got, err := svc.ExchangeRefresh("access-still-valid", "https://biz.example.com/api", scope)
	if err != nil {
		t.Fatalf("expected idempotent refresh, got %v", err)
	}
	if got != "existing-refresh" {
		t.Fatalf("got refresh %q, want existing-refresh", got)
	}
}

func TestTokenService_ExchangeRefresh_SuccessClearsAccess(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := infrastructure.NewSQLiteTokenRepository(db)
	auditRepo := infrastructure.NewSQLiteAuditRepository(db)
	svc := infrastructure.NewTestTokenService(repo, auditRepo)

	token := &domain.ContainerToken{
		ID: "tok-ex-001", TaskID: "task-1", CompanyID: "tenant-1", WorkspaceID: "ws-1", CommentID: "cmt-ex",
		ContainerAccessToken: "access-ex", ContainerRefreshToken: "",
		ContainerAccessTokenExpiresAt: "2099-01-01 00:00:00",
	}
	if _, err := repo.Save(token); err != nil {
		t.Fatalf("save: %v", err)
	}

	scope := domain.TaskScope{TenantID: "tenant-1", WorkspaceID: "ws-1", TaskID: "task-1", CommentID: "cmt-ex"}
	refreshToken, err := svc.ExchangeRefresh("access-ex", "https://biz.example.com/api", scope)
	if err != nil {
		t.Fatalf("exchange refresh: %v", err)
	}
	if refreshToken == "" {
		t.Fatal("expected non-empty refresh token")
	}
	found, _ := repo.FindByAccessToken("access-ex")
	if found != nil {
		t.Errorf("expected access token to be cleared")
	}
	foundByRefresh, _ := repo.FindByRefreshToken(refreshToken)
	if foundByRefresh == nil {
		t.Fatal("expected token with refresh token")
	}
}

func TestTokenService_ExchangeRefresh_UnknownAccessStillAlreadyDone(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := infrastructure.NewSQLiteTokenRepository(db)
	auditRepo := infrastructure.NewSQLiteAuditRepository(db)
	svc := infrastructure.NewTestTokenService(repo, auditRepo)

	token := &domain.ContainerToken{
		ID: "tok-unk-001", TaskID: "task-unk-1", CompanyID: "tenant-1", WorkspaceID: "ws-1", CommentID: "cmt-unk",
		ContainerAccessToken: "", ContainerRefreshToken: "secret-refresh",
	}
	if _, err := repo.Save(token); err != nil {
		t.Fatalf("save: %v", err)
	}

	scope := domain.TaskScope{TenantID: "tenant-1", WorkspaceID: "ws-1", TaskID: "task-unk-1", CommentID: "cmt-unk"}
	_, err := svc.ExchangeRefresh("attacker-guess", "https://biz.example.com/api", scope)
	if err == nil {
		t.Fatal("unknown access must not receive existing refresh")
	}
	de, ok := err.(*domain.DomainError)
	if !ok || de.Code != "TOKEN_EXCHANGE_ALREADY_DONE" {
		t.Fatalf("expected TOKEN_EXCHANGE_ALREADY_DONE, got %v", err)
	}
}
