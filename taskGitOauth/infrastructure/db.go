package infrastructure

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"mysqlmeta"
)

type DB struct {
	*sql.DB
}

func OpenDB(dsn, repoRoot string) (*DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	out := &DB{DB: db}
	// Schema/seed: dataMigrate/taskGitOauth via 9999 init / migrate CLI only.
	// OpenDB must not EnsureSchema — see 40_app_process_independent_of_db_migrate.md.
	_ = repoRoot
	return out, nil
}

// EnsureSchema runs SQL migration files from dataMigrate/taskGitOauth/ against the database.
// Migration tracking uses the data_migrate_log table to ensure each SQL file runs exactly once.
func (d *DB) EnsureSchema(repoRoot string) error {
	migDir := filepath.Join(repoRoot, "dataMigrate", "taskGitOauth")
	entries, err := os.ReadDir(migDir)
	if err != nil {
		return fmt.Errorf("taskGitOauth EnsureSchema read %s: %w", migDir, err)
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		files = append(files, e.Name())
	}
	sort.Strings(files)
	if len(files) == 0 {
		return fmt.Errorf("taskGitOauth EnsureSchema: no .sql files in %s", migDir)
	}

	// Create tracking table first (this is the only inline DDL — it IS the tracking mechanism).
	if _, err := d.Exec(`CREATE TABLE IF NOT EXISTS data_migrate_log (
		step_key VARCHAR(255) PRIMARY KEY,
		applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		checksum VARCHAR(64) NOT NULL DEFAULT ''
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`); err != nil {
		return fmt.Errorf("taskGitOauth data_migrate_log: %w", err)
	}

	for _, name := range files {
		var exists int
		if err := d.QueryRow(`SELECT 1 FROM data_migrate_log WHERE step_key = ?`, name).Scan(&exists); err == nil {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(migDir, name))
		if err != nil {
			return fmt.Errorf("taskGitOauth read %s: %w", name, err)
		}
		if _, err := d.Exec(mysqlmeta.StripMySQLClientMeta(string(raw))); err != nil {
			return fmt.Errorf("taskGitOauth migration %s: %w", name, err)
		}
		if _, err := d.Exec(`INSERT INTO data_migrate_log (step_key, applied_at) VALUES (?, NOW())`, name); err != nil {
			return fmt.Errorf("taskGitOauth record migration %s: %w", name, err)
		}
	}
	return nil
}

func (d *DB) PingOK() (latencyMs int, err error) {
	start := time.Now()
	if err := d.Ping(); err != nil {
		return 0, err
	}
	var one int
	if err := d.QueryRow("SELECT 1").Scan(&one); err != nil {
		return 0, err
	}
	return int(time.Since(start).Milliseconds()), nil
}

type CredentialRow struct {
	ID                 int64
	Provider           string
	Task2appUserID     string
	RefreshTokenCipher string
	RemoteUserID       string
	RemoteLogin        string
	Scope              string
	BindStatus         string
	BindError          string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// ExpandProviderKeys returns lookup aliases so bare "github" and
// compound "github:github-official*" resolve the same credential rows.
// Also aliases legacy github-official ↔ github-official-daydaymoney after
// daydaymoney App rename (Django catalog vs taskGitOauth storage).
func ExpandProviderKeys(providerKey string) []string {
	pk := strings.ToLower(strings.TrimSpace(providerKey))
	if pk == "" {
		return nil
	}
	seen := map[string]struct{}{pk: {}}
	keys := []string{pk}
	add := func(k string) {
		k = strings.ToLower(strings.TrimSpace(k))
		if k == "" {
			return
		}
		if _, ok := seen[k]; ok {
			return
		}
		seen[k] = struct{}{}
		keys = append(keys, k)
	}
	if i := strings.Index(pk, ":"); i >= 0 {
		provider, sp := pk[:i], pk[i+1:]
		add(provider)
		if provider == "github" {
			switch sp {
			case "github-official":
				add("github:github-official-daydaymoney")
			case "github-official-daydaymoney":
				add("github:github-official")
			}
		}
	} else if pk == "github" {
		add("github:github-official")
		add("github:github-official-daydaymoney")
		add("github:default")
	}
	return keys
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	parts := make([]string, n)
	for i := range parts {
		parts[i] = "?"
	}
	return strings.Join(parts, ",")
}

func (d *DB) FindActiveCredential(providerKey string, userID string, remoteUserID string) (*CredentialRow, error) {
	keys := ExpandProviderKeys(providerKey)
	if len(keys) == 0 {
		return nil, nil
	}
	q := `
SELECT id, provider, task2app_user_id, refresh_token_cipher, remote_user_id, remote_login,
       scope, bind_status, bind_error, created_at, updated_at
FROM git_oauth_appusercredential
WHERE provider IN (` + placeholders(len(keys)) + `) AND task2app_user_id = ? AND bind_status = 'active'`
	args := make([]any, 0, len(keys)+2)
	for _, k := range keys {
		args = append(args, k)
	}
	args = append(args, userID)
	if remoteUserID != "" {
		q += ` AND remote_user_id = ?`
		args = append(args, remoteUserID)
	}
	q += ` ORDER BY updated_at DESC, id DESC LIMIT 1`
	return d.scanCredential(d.QueryRow(q, args...))
}

func (d *DB) FindActiveCredentialPrefix(providerPrefix string, userID string, remoteUserID string) (*CredentialRow, error) {
	q := `
SELECT id, provider, task2app_user_id, refresh_token_cipher, remote_user_id, remote_login,
       scope, bind_status, bind_error, created_at, updated_at
FROM git_oauth_appusercredential
WHERE provider LIKE ? AND task2app_user_id = ? AND bind_status = 'active'`
	args := []any{providerPrefix + "%", userID}
	if remoteUserID != "" {
		q += ` AND remote_user_id = ?`
		args = append(args, remoteUserID)
	}
	q += ` ORDER BY updated_at DESC, id DESC LIMIT 1`
	return d.scanCredential(d.QueryRow(q, args...))
}

func (d *DB) GetCredentialByIDForUpdate(tx *sql.Tx, id int64) (*CredentialRow, error) {
	q := `
SELECT id, provider, task2app_user_id, refresh_token_cipher, remote_user_id, remote_login,
       scope, bind_status, bind_error, created_at, updated_at
FROM git_oauth_appusercredential WHERE id = ? FOR UPDATE`
	row := tx.QueryRow(q, id)
	return d.scanCredential(row)
}

func (d *DB) UpdateRefreshCipher(tx *sql.Tx, id int64, cipher string) error {
	_, err := tx.Exec(`
UPDATE git_oauth_appusercredential
SET refresh_token_cipher = ?, updated_at = ?
WHERE id = ?`, cipher, time.Now().UTC().Format("2006-01-02 15:04:05"), id)
	return err
}

func (d *DB) UpsertCredential(row *CredentialRow) (*CredentialRow, error) {
	now := time.Now().UTC()
	nowStr := now.Format("2006-01-02 15:04:05")
	res, err := d.Exec(`
INSERT INTO git_oauth_appusercredential
  (provider, task2app_user_id, refresh_token_cipher, remote_user_id, remote_login,
   scope, bind_status, bind_error, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
  refresh_token_cipher = VALUES(refresh_token_cipher),
  remote_login = VALUES(remote_login),
  scope = VALUES(scope),
  bind_status = VALUES(bind_status),
  bind_error = VALUES(bind_error),
  updated_at = VALUES(updated_at)`,
		row.Provider, row.Task2appUserID, row.RefreshTokenCipher, row.RemoteUserID, row.RemoteLogin,
		row.Scope, row.BindStatus, row.BindError, nowStr, nowStr,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	if id == 0 {
		var existingID int64
		_ = d.QueryRow(`
SELECT id FROM git_oauth_appusercredential
WHERE provider = ? AND task2app_user_id = ? AND remote_user_id = ?`,
			row.Provider, row.Task2appUserID, row.RemoteUserID).Scan(&existingID)
		id = existingID
	}
	return d.GetCredentialByID(id)
}

func (d *DB) GetCredentialByID(id int64) (*CredentialRow, error) {
	q := `
SELECT id, provider, task2app_user_id, refresh_token_cipher, remote_user_id, remote_login,
       scope, bind_status, bind_error, created_at, updated_at
FROM git_oauth_appusercredential WHERE id = ?`
	return d.scanCredential(d.QueryRow(q, id))
}

func (d *DB) UpdateBindStatus(id int64, status, bindError string) error {
	_, err := d.Exec(`
UPDATE git_oauth_appusercredential
SET bind_status = ?, bind_error = ?, updated_at = ?
WHERE id = ?`, status, bindError, time.Now().UTC().Format("2006-01-02 15:04:05"), id)
	return err
}

func (d *DB) DeleteCredentials(providerKey string, userID string, remoteUserID string) (int64, error) {
	keys := ExpandProviderKeys(providerKey)
	if len(keys) == 0 {
		return 0, nil
	}
	q := `DELETE FROM git_oauth_appusercredential WHERE provider IN (` + placeholders(len(keys)) + `) AND task2app_user_id = ?`
	args := make([]any, 0, len(keys)+2)
	for _, k := range keys {
		args = append(args, k)
	}
	args = append(args, userID)
	if remoteUserID != "" {
		q += ` AND remote_user_id = ?`
		args = append(args, remoteUserID)
	}
	res, err := d.Exec(q, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// DeleteCredentialsByProviderKey removes all user credentials for a provider key (all users).
func (d *DB) DeleteCredentialsByProviderKey(providerKey string) (int64, error) {
	keys := ExpandProviderKeys(providerKey)
	if len(keys) == 0 {
		return 0, nil
	}
	q := `DELETE FROM git_oauth_appusercredential WHERE provider IN (` + placeholders(len(keys)) + `)`
	args := make([]any, len(keys))
	for i, k := range keys {
		args[i] = k
	}
	res, err := d.Exec(q, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (d *DB) ListUserIDs(providerKey string) ([]int64, error) {
	keys := ExpandProviderKeys(providerKey)
	if len(keys) == 0 {
		return []int64{}, nil
	}
	q := `
SELECT DISTINCT task2app_user_id FROM git_oauth_appusercredential
WHERE provider IN (` + placeholders(len(keys)) + `) ORDER BY task2app_user_id`
	args := make([]any, len(keys))
	for i, k := range keys {
		args[i] = k
	}
	rows, err := d.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	if out == nil {
		out = []int64{}
	}
	return out, rows.Err()
}

func (d *DB) ListCredentialsForUser(providerKey string, userID string) ([]CredentialRow, error) {
	return d.ListCredentialsForUserKeys([]string{providerKey}, userID)
}

// ExpandProviderKeysAll returns the deduplicated union of ExpandProviderKeys
// across multiple provider keys (lookup aliases for credential rows).
func ExpandProviderKeysAll(providerKeys []string) []string {
	seen := map[string]struct{}{}
	var keys []string
	for _, pk := range providerKeys {
		for _, k := range ExpandProviderKeys(pk) {
			if _, ok := seen[k]; ok {
				continue
			}
			seen[k] = struct{}{}
			keys = append(keys, k)
		}
	}
	return keys
}

// ListCredentialsForUserKeys lists credential rows matching ANY of the given
// provider keys (each expanded via ExpandProviderKeysAll, deduplicated). Used
// by the connection-status check so credentials stored under the configured
// service_provider key (e.g. github:github-official-daydaymoney) are found even
// when the caller only resolved the bare provider (e.g. repo_url-only checks).
func (d *DB) ListCredentialsForUserKeys(providerKeys []string, userID string) ([]CredentialRow, error) {
	keys := ExpandProviderKeysAll(providerKeys)
	if len(keys) == 0 {
		return nil, nil
	}
	q := `
SELECT id, provider, task2app_user_id, refresh_token_cipher, remote_user_id, remote_login,
       scope, bind_status, bind_error, created_at, updated_at
FROM git_oauth_appusercredential
WHERE provider IN (` + placeholders(len(keys)) + `) AND task2app_user_id = ?
ORDER BY updated_at DESC, id DESC`
	args := make([]any, 0, len(keys)+1)
	for _, k := range keys {
		args = append(args, k)
	}
	args = append(args, userID)
	rows, err := d.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CredentialRow
	for rows.Next() {
		r, err := scanCredentialRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

type scannable interface {
	Scan(dest ...any) error
}

func (d *DB) scanCredential(row scannable) (*CredentialRow, error) {
	var r CredentialRow
	var created, updated string
	err := row.Scan(
		&r.ID, &r.Provider, &r.Task2appUserID, &r.RefreshTokenCipher, &r.RemoteUserID, &r.RemoteLogin,
		&r.Scope, &r.BindStatus, &r.BindError, &created, &updated,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.CreatedAt = parseTime(created)
	r.UpdatedAt = parseTime(updated)
	return &r, nil
}

func scanCredentialRows(rows *sql.Rows) (*CredentialRow, error) {
	var r CredentialRow
	var created, updated string
	err := rows.Scan(
		&r.ID, &r.Provider, &r.Task2appUserID, &r.RefreshTokenCipher, &r.RemoteUserID, &r.RemoteLogin,
		&r.Scope, &r.BindStatus, &r.BindError, &created, &updated,
	)
	if err != nil {
		return nil, err
	}
	r.CreatedAt = parseTime(created)
	r.UpdatedAt = parseTime(updated)
	return &r, nil
}

func parseTime(s string) time.Time {
	s = strings.TrimSpace(s)
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999",
		"2006-01-02 15:04:05",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func BeginImmediate(db *sql.DB) (*sql.Tx, error) {
	// MySQL InnoDB：与 GetCredentialByIDForUpdate 的 FOR UPDATE 组成行锁，
	// 避免并行 GitLab refresh 把旧 refresh_token 用两次（后到者 400 invalid_grant）。
	return db.Begin()
}
