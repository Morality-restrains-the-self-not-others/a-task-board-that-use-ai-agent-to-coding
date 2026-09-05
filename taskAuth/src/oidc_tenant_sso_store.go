package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"taskAuth/domain"
)

func generateOidcClientSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func insertTenantOidcClient(companyID, name, redirectURIsJSON, secretPlain string) error {
	clientID, err := domain.TenantGitLabOidcClientID(companyID)
	if err != nil {
		return err
	}
	id := fmt.Sprintf("%d", generateSnowflakeID())
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	_, err = db.Exec(`
		INSERT INTO auth_oidc_client
		  (id, client_id, client_secret_hash, name, redirect_uris, managed_by, owner_company_id, purpose, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, clientID, hashClientSecret(secretPlain), name, redirectURIsJSON,
		domain.TenantGitLabOidcManagedBy, companyID, domain.TenantGitLabOidcPurpose, now, now)
	return err
}

func updateTenantOidcClientSecretAndRedirect(clientID, secretPlain, redirectURIsJSON string) error {
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	_, err := db.Exec(`
		UPDATE auth_oidc_client
		SET client_secret_hash = ?, redirect_uris = ?, updated_at = ?
		WHERE client_id = ? AND managed_by = ?`,
		hashClientSecret(secretPlain), redirectURIsJSON, now, clientID, domain.TenantGitLabOidcManagedBy)
	return err
}

func deleteTenantOidcClient(clientID string) error {
	if _, err := db.Exec(`DELETE FROM auth_oidc_authorization WHERE client_id = ?`, clientID); err != nil {
		return err
	}
	_, err := db.Exec(`DELETE FROM auth_oidc_client WHERE client_id = ? AND managed_by = ?`,
		clientID, domain.TenantGitLabOidcManagedBy)
	return err
}

func loadTenantOidcClient(companyID string) (*oidcClientRow, error) {
	clientID, err := domain.TenantGitLabOidcClientID(companyID)
	if err != nil {
		return nil, err
	}
	return loadOidcClient(clientID)
}

// cleanupOidcSsoIdempotency 分批删除热窗口之外的历史幂等行（OPT-20260825-030）。
//
// 签发/轮换/关闭 SSO 会把 Idempotency-Key 写入 auth_oidc_sso_idempotency，
// 表随操作次数无限增长。由 taskEvents timer 周期调用内部端点清理：
// 禁止业务进程内 ticker（元规则 51），清理必须走 timer worker。
// maxAgeDays <= 0 时用 7d 热窗口；limit 单批上限（默认 500），循环直到整批 < limit。
func cleanupOidcSsoIdempotency(maxAgeDays int, limit int64) (int64, error) {
	if maxAgeDays <= 0 {
		maxAgeDays = 7
	}
	if limit <= 0 {
		limit = 500
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -maxAgeDays).Format("2006-01-02 15:04:05.000000")
	var total int64
	for {
		res, err := db.Exec(`DELETE FROM auth_oidc_sso_idempotency WHERE applied_at < ? LIMIT ?`, cutoff, limit)
		if err != nil {
			return total, err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return total, err
		}
		total += n
		if n < limit {
			break
		}
	}
	return total, nil
}

func claimOidcSsoIdempotency(companyID, operation, key string) (already bool, err error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return false, nil
	}
	_, err = db.Exec(`
		INSERT INTO auth_oidc_sso_idempotency (company_id, operation, idempotency_key)
		VALUES (?, ?, ?)`, companyID, operation, key)
	if err != nil {
		if isDuplicateKey(err) {
			return true, nil
		}
		return false, err
	}
	return false, nil
}

func tenantOidcRedirectURIsJSON(redirectURI string) (string, error) {
	b, err := json.Marshal([]string{redirectURI})
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func firstRedirectURI(row *oidcClientRow) string {
	if row == nil {
		return ""
	}
	var uris []string
	if err := json.Unmarshal([]byte(row.RedirectURIs), &uris); err != nil || len(uris) == 0 {
		return ""
	}
	return uris[0]
}

func ownerCompanyIDOfClient(row *oidcClientRow) string {
	if row == nil {
		return ""
	}
	if s := strings.TrimSpace(row.OwnerCompanyID); s != "" {
		return s
	}
	cid, err := domain.OwnerCompanyIDFromClientID(row.ClientID)
	if err != nil {
		return ""
	}
	return cid
}
