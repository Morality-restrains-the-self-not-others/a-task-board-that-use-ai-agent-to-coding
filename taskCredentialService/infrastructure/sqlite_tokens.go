package infrastructure

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	dbload "dbload"

	_ "github.com/go-sql-driver/mysql"

	"taskCredentialService/domain"
)

// SQLiteTokenRepository implements ports.ContainerTokenRepository using MySQL.
type SQLiteTokenRepository struct {
	db *sql.DB
}

func NewSQLiteTokenRepository(db *sql.DB) *SQLiteTokenRepository {
	return &SQLiteTokenRepository{db: db}
}

// OpenTokensDB opens the tokens MySQL database.
func OpenTokensDB(monorepoRoot string) (*sql.DB, error) {
	dsn, err := dbload.ResolveMySQLDSN("container", monorepoRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve container DSN: %w", err)
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open tokens db: %w", err)
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping tokens db: %w", err)
	}
	return db, nil
}

const tokenSelectCols = `id, task_id, company_id, workspace_id, COALESCE(comment_id,''),
			container_access_token, container_access_token_expires_at, container_refresh_token,
			server_url, business_api_endpoint, container_vscode_url,
			authorization_id, image_id, instance_type, region_id, zone_id,
			created_at, updated_at`

func (r *SQLiteTokenRepository) Save(token *domain.ContainerToken) (*domain.ContainerToken, error) {
	start := time.Now()
	_, err := r.db.Exec(`
		INSERT INTO credential_container_tokens (id, task_id, company_id, workspace_id, comment_id,
			container_access_token, container_access_token_expires_at, container_refresh_token,
			server_url, business_api_endpoint, container_vscode_url,
			authorization_id, image_id, instance_type, region_id, zone_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			container_access_token=VALUES(container_access_token),
			container_access_token_expires_at=VALUES(container_access_token_expires_at),
			container_refresh_token=VALUES(container_refresh_token),
			updated_at=CURRENT_TIMESTAMP
	`,
		token.ID, token.TaskID, token.CompanyID, token.WorkspaceID, token.CommentID,
		token.ContainerAccessToken, token.ContainerAccessTokenExpiresAt, token.ContainerRefreshToken,
		token.ServerURL, token.BusinessAPIEndpoint, token.ContainerVscodeURL,
		token.AuthorizationID, token.ImageID, token.InstanceType, token.RegionID, token.ZoneID,
	)
	if err != nil {
		return nil, err
	}
	log.Printf("[task-credential-service] sqlite token saved: task=%s comment=%s duration=%dms", token.TaskID, token.CommentID, time.Since(start).Milliseconds())
	return token, nil
}

func (r *SQLiteTokenRepository) FindByAccessToken(accessToken string) (*domain.ContainerToken, error) {
	row := r.db.QueryRow(`SELECT `+tokenSelectCols+` FROM credential_container_tokens WHERE container_access_token = ?`, accessToken)
	return scanToken(row)
}

func (r *SQLiteTokenRepository) FindByTaskID(companyID, taskID string) (*domain.ContainerToken, error) {
	row := r.db.QueryRow(`SELECT `+tokenSelectCols+` FROM credential_container_tokens WHERE company_id = ? AND task_id = ?
		ORDER BY updated_at DESC LIMIT 1`, companyID, taskID)
	return scanToken(row)
}

// FindByScope looks up the most recent token for tenant/workspace/task/comment.
func (r *SQLiteTokenRepository) FindByScope(companyID, workspaceID, taskID, commentID string) (*domain.ContainerToken, error) {
	row := r.db.QueryRow(`SELECT `+tokenSelectCols+` FROM credential_container_tokens
		WHERE company_id = ? AND workspace_id = ? AND task_id = ? AND COALESCE(comment_id,'') = ?
		ORDER BY updated_at DESC LIMIT 1`, companyID, workspaceID, taskID, strings.TrimSpace(commentID))
	return scanToken(row)
}

func (r *SQLiteTokenRepository) CountByTaskScope(companyID, workspaceID, taskID string) (int, error) {
	var n int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM credential_container_tokens
		WHERE company_id = ? AND workspace_id = ? AND task_id = ?`, companyID, workspaceID, taskID).Scan(&n)
	return n, err
}

func (r *SQLiteTokenRepository) UpdateAccessToken(id, newAccessToken string, expiresAt string) error {
	_, err := r.db.Exec(`
		UPDATE credential_container_tokens SET container_access_token = ?, container_access_token_expires_at = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, newAccessToken, expiresAt, id)
	return err
}

func (r *SQLiteTokenRepository) UpdateRefreshToken(id, newRefreshToken string) error {
	_, err := r.db.Exec(`
		UPDATE credential_container_tokens SET container_refresh_token = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, newRefreshToken, id)
	return err
}

func (r *SQLiteTokenRepository) FindByRefreshToken(refreshToken string) (*domain.ContainerToken, error) {
	row := r.db.QueryRow(`SELECT `+tokenSelectCols+` FROM credential_container_tokens WHERE container_refresh_token = ?`, refreshToken)
	return scanToken(row)
}

func (r *SQLiteTokenRepository) UpdateBusinessAPIEndpoint(id, endpoint string) error {
	_, err := r.db.Exec(`
		UPDATE credential_container_tokens SET business_api_endpoint = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, endpoint, id)
	return err
}

func scanToken(row *sql.Row) (*domain.ContainerToken, error) {
	var t domain.ContainerToken
	// container_access_token_expires_at 允许为 NULL（未设置过期时间的令牌）——
	// 直接扫入 string 会报 converting NULL to string 错误，导致 FindByAccessToken
	// 失败、令牌被误判 TOKEN_NOT_FOUND（OPT-20260809-024 调试期踩坑：手工重置
	// 令牌为 expires NULL 后 exchange-refresh 全部 401 TOKEN_ACCESS_INVALID）。
	var expiresAt sql.NullString
	err := row.Scan(&t.ID, &t.TaskID, &t.CompanyID, &t.WorkspaceID, &t.CommentID,
		&t.ContainerAccessToken, &expiresAt, &t.ContainerRefreshToken,
		&t.ServerURL, &t.BusinessAPIEndpoint, &t.ContainerVscodeURL,
		&t.AuthorizationID, &t.ImageID, &t.InstanceType, &t.RegionID, &t.ZoneID,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if expiresAt.Valid {
		t.ContainerAccessTokenExpiresAt = expiresAt.String
	}
	return &t, nil
}

// SQLiteAuditRepository implements ports.TokenAuditEventRepository.
type SQLiteAuditRepository struct {
	db *sql.DB
}

func NewSQLiteAuditRepository(db *sql.DB) *SQLiteAuditRepository {
	return &SQLiteAuditRepository{db: db}
}

func (r *SQLiteAuditRepository) Save(event *domain.TokenAuditEvent) error {
	_, err := r.db.Exec(`
		INSERT INTO credential_token_audit_events (id, task_id, comment_id, event_type, access_token_sha256,
			prev_access_token_sha256, new_access_token_sha256, refresh_token_sha256,
			source_component, trace_id, error_code, error_detail, seq)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, event.ID, event.TaskID, event.CommentID, event.EventType, event.AccessTokenSHA256,
		event.PrevAccessTokenSHA256, event.NewAccessTokenSHA256, event.RefreshTokenSHA256,
		event.SourceComponent, event.TraceID, event.ErrorCode, event.ErrorDetail, event.Seq,
	)
	return err
}

func (r *SQLiteAuditRepository) FindByTaskID(taskID string, limit int) ([]domain.TokenAuditEvent, error) {
	rows, err := r.db.Query(`
		SELECT id, task_id, COALESCE(comment_id,''), event_type, access_token_sha256,
			prev_access_token_sha256, new_access_token_sha256, refresh_token_sha256,
			source_component, trace_id, error_code, error_detail, seq, created_at
		FROM credential_token_audit_events WHERE task_id = ? ORDER BY created_at DESC LIMIT ?
	`, taskID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []domain.TokenAuditEvent
	for rows.Next() {
		var e domain.TokenAuditEvent
		if err := rows.Scan(&e.ID, &e.TaskID, &e.CommentID, &e.EventType, &e.AccessTokenSHA256,
			&e.PrevAccessTokenSHA256, &e.NewAccessTokenSHA256, &e.RefreshTokenSHA256,
			&e.SourceComponent, &e.TraceID, &e.ErrorCode, &e.ErrorDetail, &e.Seq, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}
