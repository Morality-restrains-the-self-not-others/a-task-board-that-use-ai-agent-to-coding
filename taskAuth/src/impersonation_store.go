package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"authz"
	"taskAuth/domain"

	"tracelog"
)

const impersonationTokenPrefix = "imp_"
const impersonationTTL = time.Hour

const impersonationSessionSelect = `id, actor_user_id, target_user_id, token_key, idempotency_key, started_at, expires_at, ended_at, reason`

type impersonationSessionRow struct {
	ID             int64
	ActorUserID    string
	TargetUserID   string
	TokenKey       string
	IdempotencyKey string
	StartedAt      time.Time
	ExpiresAt      time.Time
	EndedAt        sql.NullTime
	Reason         string
}

func scanImpersonationSession(row *impersonationSessionRow) []any {
	return []any{
		&row.ID, &row.ActorUserID, &row.TargetUserID, &row.TokenKey, &row.IdempotencyKey,
		&row.StartedAt, &row.ExpiresAt, &row.EndedAt, &row.Reason,
	}
}

func generateImpersonationToken() (string, error) {
	buf := make([]byte, 20)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("impersonation token: %w", err)
	}
	return impersonationTokenPrefix + hex.EncodeToString(buf), nil
}

func isImpersonationToken(tokenKey string) bool {
	return strings.HasPrefix(strings.TrimSpace(tokenKey), impersonationTokenPrefix)
}

func lookupOpenImpersonationByToken(tokenKey string) (*impersonationSessionRow, error) {
	tokenKey = strings.TrimSpace(tokenKey)
	if tokenKey == "" || !isImpersonationToken(tokenKey) {
		return nil, sql.ErrNoRows
	}
	row := impersonationSessionRow{}
	err := db.QueryRow(`
		SELECT `+impersonationSessionSelect+`
		FROM auth_impersonation_session
		WHERE token_key = ? AND ended_at IS NULL AND expires_at > UTC_TIMESTAMP()`,
		tokenKey,
	).Scan(scanImpersonationSession(&row)...)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func lookupImpersonationByToken(tokenKey string) (*impersonationSessionRow, error) {
	tokenKey = strings.TrimSpace(tokenKey)
	if tokenKey == "" || !isImpersonationToken(tokenKey) {
		return nil, sql.ErrNoRows
	}
	row := impersonationSessionRow{}
	err := db.QueryRow(`
		SELECT `+impersonationSessionSelect+`
		FROM auth_impersonation_session
		WHERE token_key = ?`,
		tokenKey,
	).Scan(scanImpersonationSession(&row)...)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func findOpenImpersonationForActor(actorUserID string) (*impersonationSessionRow, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" {
		return nil, sql.ErrNoRows
	}
	row := impersonationSessionRow{}
	err := db.QueryRow(`
		SELECT `+impersonationSessionSelect+`
		FROM auth_impersonation_session
		WHERE actor_user_id = ? AND ended_at IS NULL AND expires_at > UTC_TIMESTAMP()
		ORDER BY id DESC LIMIT 1`,
		actorUserID,
	).Scan(scanImpersonationSession(&row)...)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func findOpenImpersonationByIdempotency(actorUserID, key string) (*impersonationSessionRow, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, sql.ErrNoRows
	}
	row := impersonationSessionRow{}
	err := db.QueryRow(`
		SELECT `+impersonationSessionSelect+`
		FROM auth_impersonation_session
		WHERE actor_user_id = ? AND idempotency_key = ?
		  AND ended_at IS NULL AND expires_at > UTC_TIMESTAMP()
		ORDER BY id DESC LIMIT 1`,
		actorUserID, key,
	).Scan(scanImpersonationSession(&row)...)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func insertImpersonationSession(actorID, targetID, token, idemKey, reason string, ttl time.Duration) (*impersonationSessionRow, int64, error) {
	id := generateSnowflakeID()
	now := time.Now().UTC()
	exp := now.Add(ttl)
	reason = domain.NormalizeImpersonationReason(reason)

	tx, err := db.Begin()
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.Exec(`
		INSERT INTO auth_impersonation_session
		  (id, actor_user_id, target_user_id, token_key, idempotency_key, reason, started_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, actorID, targetID, token, idemKey, reason,
		now, exp,
	)
	if err != nil {
		return nil, 0, err
	}

	msgID := generateSnowflakeID()
	title := "系统管理员以你的身份登录"
	body := domain.ImpersonationNoticeBody(actorID)
	_, err = tx.Exec(`
		INSERT INTO auth_user_inbox_message
		  (id, recipient_user_id, kind, title, body, reason, actor_user_id, impersonation_session_id, created_at)
		VALUES (?, ?, 'impersonation_notice', ?, ?, ?, ?, ?, ?)`,
		msgID, targetID, title, body, reason, actorID, id, now,
	)
	if err != nil {
		return nil, 0, err
	}
	if err := tx.Commit(); err != nil {
		return nil, 0, err
	}
	return &impersonationSessionRow{
		ID: id, ActorUserID: actorID, TargetUserID: targetID,
		TokenKey: token, IdempotencyKey: idemKey, Reason: reason,
		StartedAt: now, ExpiresAt: exp,
	}, msgID, nil
}

func endImpersonationSession(tokenKey string) error {
	_, err := db.Exec(`
		UPDATE auth_impersonation_session
		SET ended_at = UTC_TIMESTAMP()
		WHERE token_key = ? AND ended_at IS NULL`,
		tokenKey,
	)
	return err
}

func userIsPlatformAdmin(userID string) (bool, error) {
	_, isSuper, _, _, err := loadUserAuthFlags(userID)
	if err != nil {
		return false, err
	}
	if isSuper {
		return true, nil
	}
	roles, err := loadPlatformRoles(userID)
	if err != nil {
		return false, err
	}
	for _, role := range roles {
		if role == "super_admin" {
			return true, nil
		}
		if authz.PlatformRolePerms(role)[authz.PermPlatformManage] {
			return true, nil
		}
	}
	return false, nil
}

func impersonationLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !tracelog.ImpersonationFromContext(r.Context()).Active() {
			tok := requestAuthToken(r)
			if sess, err := lookupOpenImpersonationByToken(tok); err == nil && sess != nil {
				tracelog.SetImpersonation(r.Context(), tracelog.Impersonation{
					ImpersonatorUserID: sess.ActorUserID,
					ImpersonatedUserID: sess.TargetUserID,
					SessionID:          strconv.FormatInt(sess.ID, 10),
				})
			}
		}
		next.ServeHTTP(w, r)
	})
}
