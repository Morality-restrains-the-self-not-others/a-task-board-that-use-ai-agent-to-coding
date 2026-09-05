package main

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestDeletionPrecheckNoBlockers(t *testing.T) {
	setupAuthTestDB(t)
	fetchDeletionBlockersBillFn = func(ctx context.Context, userID string) ([]deletionBlocker, error) {
		return nil, nil
	}
	fetchDeletionBlockersTenantFn = func(ctx context.Context, userID string) ([]deletionBlocker, error) {
		return nil, nil
	}
	fetchDeletionBlockersCloudFn = func(ctx context.Context, userID string) ([]deletionBlocker, error) {
		return nil, nil
	}
	t.Cleanup(func() {
		fetchDeletionBlockersBillFn = fetchBillDeletionBlockers
		fetchDeletionBlockersTenantFn = fetchTenantDeletionBlockers
		fetchDeletionBlockersCloudFn = fetchCloudDeletionBlockers
	})

	userID := insertTestUser(t, "del-precheck@test.com", "secret123")
	payload, err := deletionPrecheckPayload(context.Background(), userID)
	if err != nil {
		t.Fatalf("precheck: %v", err)
	}
	if can, _ := payload["can_request"].(bool); !can {
		t.Fatalf("expected can_request true, got %v blockers=%v", payload["can_request"], payload["blockers"])
	}
}

func TestDeletionRequestCancelLifecycle(t *testing.T) {
	setupAuthTestDB(t)
	fetchDeletionBlockersBillFn = func(ctx context.Context, userID string) ([]deletionBlocker, error) {
		return nil, nil
	}
	fetchDeletionBlockersTenantFn = func(ctx context.Context, userID string) ([]deletionBlocker, error) {
		return nil, nil
	}
	fetchDeletionBlockersCloudFn = func(ctx context.Context, userID string) ([]deletionBlocker, error) {
		return nil, nil
	}
	t.Cleanup(func() {
		fetchDeletionBlockersBillFn = fetchBillDeletionBlockers
		fetchDeletionBlockersTenantFn = fetchTenantDeletionBlockers
		fetchDeletionBlockersCloudFn = fetchCloudDeletionBlockers
	})

	userID := insertTestUser(t, "del-life@test.com", "secret123")
	ctx := context.Background()
	precheck, err := deletionPrecheckPayload(ctx, userID)
	if err != nil {
		t.Fatalf("precheck: %v", err)
	}
	req, err := createDeletionRequest(ctx, userID, precheck)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if req.Status != deletionStatusPending {
		t.Fatalf("status=%s", req.Status)
	}
	if err := cancelDeletionRequest(ctx, userID); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	status, err := deletionStatusPayload(userID)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status["status"] != "none" {
		t.Fatalf("expected none after cancel, got %v", status["status"])
	}
}

func TestExecuteDueDeletionArchivesUser(t *testing.T) {
	setupAuthTestDB(t)
	fetchDeletionBlockersBillFn = func(ctx context.Context, userID string) ([]deletionBlocker, error) {
		return nil, nil
	}
	fetchDeletionBlockersTenantFn = func(ctx context.Context, userID string) ([]deletionBlocker, error) {
		return nil, nil
	}
	fetchDeletionBlockersCloudFn = func(ctx context.Context, userID string) ([]deletionBlocker, error) {
		return nil, nil
	}
	// OPT-20260820-019: after archiving the last member, invite cleanup is invoked.
	var cleanupCalled []string
	cleanupTenantInvitesFn = func(ctx context.Context, userID string) error {
		cleanupCalled = append(cleanupCalled, userID)
		return nil
	}
	t.Cleanup(func() {
		fetchDeletionBlockersBillFn = fetchBillDeletionBlockers
		fetchDeletionBlockersTenantFn = fetchTenantDeletionBlockers
		fetchDeletionBlockersCloudFn = fetchCloudDeletionBlockers
		cleanupTenantInvitesFn = cleanupTenantInvites
	})

	userID := insertTestUser(t, "del-exec@test.com", "secret123")
	ctx := context.Background()
	precheck, _ := deletionPrecheckPayload(ctx, userID)
	req, err := createDeletionRequest(ctx, userID, precheck)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	_, err = db.Exec(`UPDATE auth_account_deletion_request SET effective_at=? WHERE id=?`,
		time.Now().UTC().Add(-time.Minute), req.ID)
	if err != nil {
		t.Fatalf("backdate effective_at: %v", err)
	}
	n, err := executeDueDeletionRequests(ctx, 10)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if n != 1 {
		t.Fatalf("processed=%d want 1", n)
	}
	var archived int
	if err := db.QueryRow(`SELECT COALESCE(is_archived,0) FROM auth_user WHERE id=?`, userID).Scan(&archived); err != nil || archived != 1 {
		t.Fatalf("archived=%d err=%v", archived, err)
	}
	if len(cleanupCalled) != 1 || cleanupCalled[0] != userID {
		t.Fatalf("cleanupCalled=%v want [%s]", cleanupCalled, userID)
	}
}

func TestExecuteDueDeletionCleanupFailureDoesNotBlock(t *testing.T) {
	setupAuthTestDB(t)
	fetchDeletionBlockersBillFn = func(ctx context.Context, userID string) ([]deletionBlocker, error) {
		return nil, nil
	}
	fetchDeletionBlockersTenantFn = func(ctx context.Context, userID string) ([]deletionBlocker, error) {
		return nil, nil
	}
	fetchDeletionBlockersCloudFn = func(ctx context.Context, userID string) ([]deletionBlocker, error) {
		return nil, nil
	}
	cleanupTenantInvitesFn = func(ctx context.Context, userID string) error {
		return fmt.Errorf("tenant service unavailable")
	}
	t.Cleanup(func() {
		fetchDeletionBlockersBillFn = fetchBillDeletionBlockers
		fetchDeletionBlockersTenantFn = fetchTenantDeletionBlockers
		fetchDeletionBlockersCloudFn = fetchCloudDeletionBlockers
		cleanupTenantInvitesFn = cleanupTenantInvites
	})

	userID := insertTestUser(t, "del-cleanup-fail@test.com", "secret123")
	ctx := context.Background()
	precheck, _ := deletionPrecheckPayload(ctx, userID)
	req, err := createDeletionRequest(ctx, userID, precheck)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	_, err = db.Exec(`UPDATE auth_account_deletion_request SET effective_at=? WHERE id=?`,
		time.Now().UTC().Add(-time.Minute), req.ID)
	if err != nil {
		t.Fatalf("backdate effective_at: %v", err)
	}
	n, err := executeDueDeletionRequests(ctx, 10)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if n != 1 {
		t.Fatalf("processed=%d want 1 (cleanup failure must not block deletion)", n)
	}
	var archived int
	if err := db.QueryRow(`SELECT COALESCE(is_archived,0) FROM auth_user WHERE id=?`, userID).Scan(&archived); err != nil || archived != 1 {
		t.Fatalf("archived=%d err=%v", archived, err)
	}
}

func insertTestUser(t *testing.T, email, password string) string {
	t.Helper()
	userID := fmt.Sprintf("%d", generateSnowflakeID())
	now := time.Now().UTC()
	_, err := db.Exec(`INSERT INTO auth_user (id, password, is_active, is_staff, is_superuser, is_tenant, date_joined, is_archived)
		VALUES (?, '', 1, 0, 0, 0, ?, 0)`, userID, now)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	hash, err := hashPassword(password)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	contentTypeID := cfg.UserContentTypeID
	if contentTypeID == 0 {
		contentTypeID = 4
	}
	lmID := generateSnowflakeID()
	_, err = db.Exec(`INSERT INTO auth_login_method
		(id, content_type_id, object_id, method_type, identifier, password_hash, is_verified, created_at, updated_at)
		VALUES (?, ?, ?, 'email', ?, ?, 1, ?, ?)`, lmID, contentTypeID, userID, email, hash, now, now)
	if err != nil {
		t.Fatalf("insert login method: %v", err)
	}
	return userID
}
