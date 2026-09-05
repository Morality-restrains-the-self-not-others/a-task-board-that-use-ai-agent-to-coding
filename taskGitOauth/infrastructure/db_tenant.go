package infrastructure

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// TenantGitLabOAuthConnectionRow is the persistence model for git_oauth_tenant_gitlab_oauth_connections.
type TenantGitLabOAuthConnectionRow struct {
	ID              string
	CompanyID       string
	BaseURL         string
	ClientID        string
	ClientSecretEnc string
	Remark          string
	RedirectURI     string
	Scope           string
	Active          bool
	Intranet        bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func tenantGitLabHostMatches(baseURL, host string) bool {
	want := strings.ToLower(strings.TrimSpace(hostnameOf(host)))
	if want == "" {
		want = strings.ToLower(strings.TrimSpace(host))
	}
	if want == "" {
		return false
	}
	got := strings.ToLower(strings.TrimSpace(hostnameOf(baseURL)))
	return got != "" && got == want
}

// GetTenantGitLabConnectionByHost returns the active Path A connection whose
// base_url hostname matches host (e.g. "115.29.110.74"). Used when gitsite
// resolve has no YAML provider for a tenant self-hosted GitLab.
func (d *DB) GetTenantGitLabConnectionByHost(host string) (*TenantGitLabOAuthConnectionRow, error) {
	if strings.TrimSpace(host) == "" {
		return nil, nil
	}
	q := `
SELECT id, company_id, base_url, client_id, client_secret_enc, remark, redirect_uri, scope, active, intranet, created_at, updated_at
FROM git_oauth_tenant_gitlab_oauth_connections WHERE active = 1`
	rows, err := d.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		r, err := scanTenantGitLabConnectionRow(rows)
		if err != nil {
			return nil, err
		}
		if tenantGitLabHostMatches(r.BaseURL, host) {
			return r, nil
		}
	}
	return nil, rows.Err()
}

func (d *DB) GetTenantGitLabConnection(companyID string) (*TenantGitLabOAuthConnectionRow, error) {
	cid := strings.TrimSpace(companyID)
	if cid == "" {
		return nil, nil
	}
	q := `
SELECT id, company_id, base_url, client_id, client_secret_enc, remark, redirect_uri, scope, active, intranet, created_at, updated_at
FROM git_oauth_tenant_gitlab_oauth_connections WHERE company_id = ? LIMIT 1`
	r, err := scanTenantGitLabConnectionRow(d.QueryRow(q, cid))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r, nil
}

type tenantRowScanner interface {
	Scan(dest ...any) error
}

func scanTenantGitLabConnectionRow(sc tenantRowScanner) (*TenantGitLabOAuthConnectionRow, error) {
	var r TenantGitLabOAuthConnectionRow
	var active, intranet int
	var created, updated string
	err := sc.Scan(
		&r.ID, &r.CompanyID, &r.BaseURL, &r.ClientID, &r.ClientSecretEnc, &r.Remark,
		&r.RedirectURI, &r.Scope, &active, &intranet, &created, &updated,
	)
	if err != nil {
		return nil, err
	}
	r.Active = active != 0
	r.Intranet = intranet != 0
	r.CreatedAt = parseTime(created)
	r.UpdatedAt = parseTime(updated)
	return &r, nil
}

func (d *DB) UpsertTenantGitLabConnection(row *TenantGitLabOAuthConnectionRow) (*TenantGitLabOAuthConnectionRow, error) {
	if row == nil {
		return nil, nil
	}
	now := time.Now().UTC()
	nowStr := now.Format("2006-01-02 15:04:05")
	active := 0
	if row.Active {
		active = 1
	}
	intranet := 0
	if row.Intranet {
		intranet = 1
	}
	existing, err := d.GetTenantGitLabConnection(row.CompanyID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		if strings.TrimSpace(row.ID) == "" {
			row.ID = newTenantConnectionID()
		}
		_, err = d.Exec(`
INSERT INTO git_oauth_tenant_gitlab_oauth_connections
  (id, company_id, base_url, client_id, client_secret_enc, remark, redirect_uri, scope, active, intranet, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			row.ID, row.CompanyID, row.BaseURL, row.ClientID, row.ClientSecretEnc, row.Remark,
			row.RedirectURI, row.Scope, active, intranet, nowStr, nowStr,
		)
		if err != nil {
			return nil, err
		}
		return d.GetTenantGitLabConnection(row.CompanyID)
	}
	_, err = d.Exec(`
UPDATE git_oauth_tenant_gitlab_oauth_connections
SET base_url = ?, client_id = ?, client_secret_enc = ?, remark = ?, redirect_uri = ?, scope = ?, active = ?, intranet = ?, updated_at = ?
WHERE company_id = ?`,
		row.BaseURL, row.ClientID, row.ClientSecretEnc, row.Remark, row.RedirectURI, row.Scope, active, intranet, nowStr, row.CompanyID,
	)
	if err != nil {
		return nil, err
	}
	return d.GetTenantGitLabConnection(row.CompanyID)
}

func (d *DB) DeleteTenantGitLabConnection(companyID string) (bool, error) {
	cid := strings.TrimSpace(companyID)
	if cid == "" {
		return false, nil
	}
	res, err := d.Exec(`DELETE FROM git_oauth_tenant_gitlab_oauth_connections WHERE company_id = ?`, cid)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func newTenantConnectionID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("%x", b[:])
}
