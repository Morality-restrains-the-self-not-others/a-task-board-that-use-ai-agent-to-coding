package token_test

import (
	"testing"

	"taskCredentialService/domain"
	"taskCredentialService/infrastructure"
)

func TestTokenService_IssueTokenRequiresCommentID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	svc := infrastructure.NewTestTokenService(
		infrastructure.NewSQLiteTokenRepository(db),
		infrastructure.NewSQLiteAuditRepository(db),
	)
	_, err := svc.IssueToken(domain.TaskScope{TenantID: "t", WorkspaceID: "w", TaskID: "task-1"}, nil)
	if err == nil {
		t.Fatal("expected COMMENT_ID_REQUIRED")
	}
	de, ok := err.(*domain.DomainError)
	if !ok || de.Code != "COMMENT_ID_REQUIRED" {
		t.Fatalf("got %v", err)
	}
}

func TestTokenService_TwoCommentsIndependentExchange(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := infrastructure.NewSQLiteTokenRepository(db)
	svc := infrastructure.NewTestTokenService(repo, infrastructure.NewSQLiteAuditRepository(db))

	scopeA := domain.TaskScope{TenantID: "t1", WorkspaceID: "w1", TaskID: "task-multi", CommentID: "cmt-a"}
	scopeB := domain.TaskScope{TenantID: "t1", WorkspaceID: "w1", TaskID: "task-multi", CommentID: "cmt-b"}
	tokA, err := svc.IssueToken(scopeA, nil)
	if err != nil {
		t.Fatalf("issue A: %v", err)
	}
	tokB, err := svc.IssueToken(scopeB, nil)
	if err != nil {
		t.Fatalf("issue B: %v", err)
	}
	if tokA.ContainerAccessToken == tokB.ContainerAccessToken {
		t.Fatal("two comments must not share access token")
	}

	rtA, err := svc.ExchangeRefresh(tokA.ContainerAccessToken, "https://biz.example.com/a", scopeA)
	if err != nil {
		t.Fatalf("exchange A: %v", err)
	}
	rtB, err := svc.ExchangeRefresh(tokB.ContainerAccessToken, "https://biz.example.com/b", scopeB)
	if err != nil {
		t.Fatalf("exchange B after A must not 403, got %v", err)
	}
	if rtA == "" || rtB == "" || rtA == rtB {
		t.Fatalf("expected distinct refresh tokens, A=%q B=%q", rtA, rtB)
	}

	_, err = svc.ExchangeRefresh(tokB.ContainerAccessToken, "https://biz.example.com/a", scopeA)
	if err == nil {
		t.Fatal("B access must not exchange under A scope")
	}
}

func TestTokenService_IssueTokenReusesRowWhenRefreshExists(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := infrastructure.NewSQLiteTokenRepository(db)
	svc := infrastructure.NewTestTokenService(repo, infrastructure.NewSQLiteAuditRepository(db))
	scope := domain.TaskScope{TenantID: "t1", WorkspaceID: "w1", TaskID: "task-reuse", CommentID: "cmt-r"}
	first, err := svc.IssueToken(scope, nil)
	if err != nil {
		t.Fatalf("first issue: %v", err)
	}
	rt, err := svc.ExchangeRefresh(first.ContainerAccessToken, "https://biz.example.com/", scope)
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	second, err := svc.IssueToken(scope, nil)
	if err != nil {
		t.Fatalf("second issue: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("expected same row id, first=%s second=%s", first.ID, second.ID)
	}
	found, err := repo.FindByRefreshToken(rt)
	if err != nil || found == nil {
		t.Fatalf("refresh must remain on same row: %v", err)
	}
	if found.ContainerRefreshToken != rt {
		t.Fatalf("refresh cleared unexpectedly: %q", found.ContainerRefreshToken)
	}
}
