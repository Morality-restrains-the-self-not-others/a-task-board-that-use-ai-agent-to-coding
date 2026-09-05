package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// 个人信息导出核心逻辑（PIPL「导出权」；与 account_deletion.go 同族）。
// 数据边界：仅个人可归因数据，绝不包含 password_hash / token 原文 / 激活令牌 / 验证码。

const (
	personalDataExportStatusReady   = "ready"
	personalDataExportStatusPartial = "partial"

	personalDataExportFormatVersion = 1
)

type personalDataExportRow struct {
	ID          int64
	UserID      string
	Status      string
	Content     string
	GeneratedAt time.Time
	ExpiresAt   time.Time
}

// downloadable 判断快照是否仍在有效期内（未过期）。
func (r *personalDataExportRow) downloadable() bool {
	return r != nil && !r.ExpiresAt.Before(time.Now().UTC())
}

type personalDataSectionFailure struct {
	Section string `json:"section"`
	Reason  string `json:"reason"`
}

type personalDataExportPayload struct {
	FormatVersion int                          `json:"format_version"`
	ExportedAt    string                       `json:"exported_at"`
	UserID        string                       `json:"user_id"`
	Sections      map[string]interface{}       `json:"sections"`
	Unavailable   []personalDataSectionFailure `json:"unavailable,omitempty"`
}

// 远端服务 fetch 函数（测试可覆盖，与 deletion blockers 同模式）
var (
	fetchPersonalDataBillFn   = fetchBillPersonalData
	fetchPersonalDataTenantFn = fetchTenantPersonalData
	fetchPersonalDataCloudFn  = fetchCloudPersonalData
)

// personalDataExportUpstreamTimeout 单次上游 personal-data 调用的超时阈值。
// 同步聚合跨 4 服务，任一上游挂起都会拖死整次导出请求；超时后按 section
// 降级为 partial（sections_unavailable），不让单一服务拖垮 PIPL 导出权。
var personalDataExportUpstreamTimeout = 20 * time.Second

func personalDataExportRetentionDays() int {
	if cfg.PersonalDataExportRetentionDays > 0 {
		return cfg.PersonalDataExportRetentionDays
	}
	return 7
}

// nullableTimeRFC3339 将可空时间格式化为 RFC3339，无效时返回空串。
func nullableTimeRFC3339(t sql.NullTime) string {
	if !t.Valid || t.Time.IsZero() {
		return ""
	}
	return t.Time.UTC().Format(time.RFC3339)
}

// aggregateLocalPersonalDataSections 收集 taskAuth 本地数据。
func aggregateLocalPersonalDataSections(ctx context.Context, userID string) (map[string]interface{}, error) {
	sections := make(map[string]interface{})

	// account: auth_user 基础信息（不含 password）
	var dateJoined, lastLogin sql.NullTime
	var isActive, isArchived bool
	var deletionCompletedAt sql.NullTime
	err := db.QueryRow(`
		SELECT date_joined, last_login, is_active, COALESCE(is_archived,0),
		       deletion_completed_at
		FROM auth_user WHERE id = ?`, userID).
		Scan(&dateJoined, &lastLogin, &isActive, &isArchived, &deletionCompletedAt)
	if err != nil {
		return nil, err
	}
	account := map[string]interface{}{
		"user_id":     userID,
		"is_active":   isActive,
		"is_archived": isArchived,
	}
	if dateJoined.Valid && !dateJoined.Time.IsZero() {
		account["date_joined"] = dateJoined.Time.UTC().Format(time.RFC3339)
	}
	if lastLogin.Valid && !lastLogin.Time.IsZero() {
		account["last_login"] = lastLogin.Time.UTC().Format(time.RFC3339)
	}
	if deletionCompletedAt.Valid {
		account["deletion_completed_at"] = deletionCompletedAt.Time.UTC().Format(time.RFC3339)
	}
	sections["account"] = account

	// profile: auth_user_profile（昵称/头像 URL）
	profile := map[string]interface{}{}
	var username, avatar string
	if err := db.QueryRow(`SELECT COALESCE(username,''), COALESCE(avatar,'') FROM auth_user_profile WHERE user_id = ?`, userID).
		Scan(&username, &avatar); err == nil {
		profile["username"] = username
		profile["avatar"] = avatar
	}
	sections["profile"] = profile

	// login_methods: 绑定方式（邮箱/手机号等标识）
	rows, err := db.Query(`
		SELECT method_type, identifier, is_verified, created_at,
		       COALESCE(binding_voided_at, '')
		FROM auth_login_method WHERE object_id = ? ORDER BY created_at ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	loginMethods := make([]map[string]interface{}, 0)
	for rows.Next() {
		var methodType, identifier, voidedAt string
		var isVerified bool
		var createdAt sql.NullTime
		if err := rows.Scan(&methodType, &identifier, &isVerified, &createdAt, &voidedAt); err != nil {
			continue
		}
		item := map[string]interface{}{
			"method_type": methodType,
			"identifier":  identifier,
			"is_verified": isVerified,
			"created_at":  nullableTimeRFC3339(createdAt),
		}
		if voidedAt != "" {
			item["binding_voided_at"] = voidedAt
		}
		loginMethods = append(loginMethods, item)
	}
	sections["login_methods"] = loginMethods

	// access_tokens: 仅元数据 + last_4，绝不含 token 原文
	tokenRows, err := db.Query(`
		SELECT name, last_4, created_at, COALESCE(last_used_at, ''), COALESCE(expires_at, ''), is_revoked
		FROM auth_user_access_token WHERE user_id = ? ORDER BY created_at ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer tokenRows.Close()
	accessTokens := make([]map[string]interface{}, 0)
	for tokenRows.Next() {
		var name, last4, lastUsed, expires string
		var createdAt sql.NullTime
		var revoked bool
		if err := tokenRows.Scan(&name, &last4, &createdAt, &lastUsed, &expires, &revoked); err != nil {
			continue
		}
		item := map[string]interface{}{
			"name":       name,
			"last_4":     last4,
			"created_at": nullableTimeRFC3339(createdAt),
			"is_revoked": revoked,
		}
		if lastUsed != "" {
			item["last_used_at"] = lastUsed
		}
		if expires != "" {
			item["expires_at"] = expires
		}
		accessTokens = append(accessTokens, item)
	}
	sections["access_tokens"] = accessTokens

	// wechat_identity: 微信绑定信息
	wechatRows, err := db.Query(`
		SELECT app_key, COALESCE(app_id,''), COALESCE(openid,''), COALESCE(unionid,''),
		       COALESCE(nickname,''), COALESCE(avatar_url,''), created_at
		FROM wechat_identity WHERE user_id = ? ORDER BY created_at ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer wechatRows.Close()
	wechatIdentities := make([]map[string]interface{}, 0)
	for wechatRows.Next() {
		var appKey, appID, openid, unionid, nickname, avatarURL string
		var createdAt sql.NullTime
		if err := wechatRows.Scan(&appKey, &appID, &openid, &unionid, &nickname, &avatarURL, &createdAt); err != nil {
			continue
		}
		wechatIdentities = append(wechatIdentities, map[string]interface{}{
			"app_key":    appKey,
			"app_id":     appID,
			"openid":     openid,
			"unionid":    unionid,
			"nickname":   nickname,
			"avatar_url": avatarURL,
			"created_at": nullableTimeRFC3339(createdAt),
		})
	}
	sections["wechat_identity"] = wechatIdentities

	// OPT-20260825-035：PIPL 导出是否包含失败登录尝试待产品确认，保守先只导出成功历史。
	histRows, histErr := db.Query(`
		SELECT id, logged_in_at, client_ip, user_agent, entry, method_type
		FROM auth_login_history WHERE user_id = ? AND outcome = 'success'
		ORDER BY logged_in_at DESC, id DESC LIMIT 200`, userID)
	if histErr != nil {
		return nil, histErr
	}
	defer histRows.Close()
	loginHistory := make([]map[string]interface{}, 0)
	for histRows.Next() {
		var id int64
		var logged sql.NullTime
		var ip, ua, entry, method string
		if err := histRows.Scan(&id, &logged, &ip, &ua, &entry, &method); err != nil {
			continue
		}
		loginHistory = append(loginHistory, map[string]interface{}{
			"id":           fmt.Sprintf("%d", id),
			"logged_in_at": nullableTimeRFC3339(logged),
			"client_ip":    ip,
			"user_agent":   ua,
			"entry":        entry,
			"method_type":  method,
		})
	}
	sections["login_history"] = loginHistory

	// kyc: 实名等级与状态（无记录时为只读的固定结构，保证导出格式稳定）
	kyc := map[string]interface{}{
		"tier":         "",
		"status":       "",
		"effective_at": "",
		"expires_at":   "",
		"risk_flags":   "",
	}
	var tier, kycStatus, effectiveAt, expiresAt, riskFlags string
	if err := db.QueryRow(`
		SELECT COALESCE(tier,''), COALESCE(status,''), COALESCE(effective_at,''),
		       COALESCE(expires_at,''), COALESCE(risk_flags,'')
		FROM auth_kyc_profile WHERE user_id = ?`, userID).
		Scan(&tier, &kycStatus, &effectiveAt, &expiresAt, &riskFlags); err == nil {
		kyc["tier"] = tier
		kyc["status"] = kycStatus
		kyc["effective_at"] = effectiveAt
		kyc["expires_at"] = expiresAt
		kyc["risk_flags"] = riskFlags
	} else if err != sql.ErrNoRows {
		slog.WarnContext(ctx, "personal_data_export_kyc_read_failed", "user_id", userID, "error", err.Error())
	}
	sections["kyc"] = kyc

	// deletion_history: 删除权相关记录（最近一次 + 历史条数）
	delHistory := map[string]interface{}{}
	var delCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM auth_account_deletion_request WHERE user_id = ?`, userID).Scan(&delCount); err == nil {
		delHistory["request_count"] = delCount
	}
	var delStatus, requestedAt, effectiveAt2, cancelledAt, executedAt string
	err = db.QueryRow(`
		SELECT status, requested_at, effective_at,
		       COALESCE(cancelled_at, ''), COALESCE(executed_at, '')
		FROM auth_account_deletion_request WHERE user_id = ? ORDER BY requested_at DESC LIMIT 1`, userID).
		Scan(&delStatus, &requestedAt, &effectiveAt2, &cancelledAt, &executedAt)
	if err == nil {
		latest := map[string]interface{}{
			"status":       delStatus,
			"requested_at": requestedAt,
			"effective_at": effectiveAt2,
		}
		if cancelledAt != "" {
			latest["cancelled_at"] = cancelledAt
		}
		if executedAt != "" {
			latest["executed_at"] = executedAt
		}
		delHistory["latest"] = latest
	}
	sections["deletion_history"] = delHistory

	return sections, nil
}

// fetchBillPersonalData 拉取 taskBill 中该用户的个人数据（内部端点契约见设计文档）。
func fetchBillPersonalData(ctx context.Context, userID string) (interface{}, error) {
	url := fmt.Sprintf("%s/api/internal/taskbill/users/%s/personal-data/", billServiceBaseURL(), userID)
	return internalPersonalDataGet(ctx, url, "X-TaskBill-Internal-Secret", cfg.BillInternalSecret)
}

// fetchTenantPersonalData 拉取 taskTenantService 中该用户的成员关系。
func fetchTenantPersonalData(ctx context.Context, userID string) (interface{}, error) {
	base := tenantServiceBaseURL()
	if base == "" {
		return nil, fmt.Errorf("tenant service url not configured")
	}
	url := fmt.Sprintf("%s/api/internal/tenant/users/%s/personal-data/", base, userID)
	return internalPersonalDataGet(ctx, url, "X-Internal-Secret", cfg.InternalSecret)
}

// fetchCloudPersonalData 拉取 taskCloudService 中该用户调起的云资源。
func fetchCloudPersonalData(ctx context.Context, userID string) (interface{}, error) {
	url := fmt.Sprintf("%s/api/internal/cloud/users/%s/personal-data/", cloudServiceBaseURL(), userID)
	return internalPersonalDataGet(ctx, url, "X-Internal-Secret", cfg.InternalSecret)
}

// internalPersonalDataGet 调用内部服务 personal-data 端点；非 2xx 一律视为不可达。
// 每个上游调用带独立超时阈值：超时按 section 降级（generatePersonalDataExport 已聚合
// failures 为 partial），避免单一服务慢查询让整次同步导出挂起直至请求超时。
func internalPersonalDataGet(ctx context.Context, url, secretHeader, secretValue string) (interface{}, error) {
	ctx, cancel := context.WithTimeout(ctx, personalDataExportUpstreamTimeout)
	defer cancel()
	body, status, err := internalServiceGet(ctx, url, secretHeader, secretValue)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("personal-data status=%d", status)
	}
	var wrapper struct {
		Data interface{} `json:"data"`
	}
	if err := json.Unmarshal(body, &wrapper); err != nil {
		return nil, err
	}
	return wrapper.Data, nil
}

// generatePersonalDataExport 聚合全部 section 并生成导出快照（同步）。
func generatePersonalDataExport(ctx context.Context, userID string) (*personalDataExportRow, []personalDataSectionFailure, error) {
	sections, err := aggregateLocalPersonalDataSections(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	var failures []personalDataSectionFailure
	remote := []struct {
		name string
		fn   func(context.Context, string) (interface{}, error)
	}{
		{"billing", fetchPersonalDataBillFn},
		{"tenant_memberships", fetchPersonalDataTenantFn},
		{"cloud_servers", fetchPersonalDataCloudFn},
	}
	for _, r := range remote {
		data, err := r.fn(ctx, userID)
		if err != nil {
			slog.WarnContext(ctx, "personal_data_export_section_unavailable",
				"user_id", userID, "section", r.name, "error", err.Error())
			failures = append(failures, personalDataSectionFailure{Section: r.name, Reason: "unavailable"})
			continue
		}
		sections[r.name] = data
	}

	status := personalDataExportStatusReady
	if len(failures) > 0 {
		status = personalDataExportStatusPartial
	}
	now := time.Now().UTC()
	payload := personalDataExportPayload{
		FormatVersion: personalDataExportFormatVersion,
		ExportedAt:    now.Format(time.RFC3339),
		UserID:        userID,
		Sections:      sections,
		Unavailable:   failures,
	}
	content, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, nil, err
	}

	// 惰性清理：新生成覆盖同用户旧行（含过期行），天然每用户最多 1 行
	if _, err := db.Exec(`DELETE FROM auth_personal_data_export WHERE user_id = ?`, userID); err != nil {
		return nil, nil, err
	}
	id := generateSnowflakeID()
	expires := now.Add(time.Duration(personalDataExportRetentionDays()) * 24 * time.Hour)
	_, err = db.Exec(`
		INSERT INTO auth_personal_data_export (id, user_id, status, content, generated_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		id, userID, status, string(content), now, expires)
	if err != nil {
		return nil, nil, err
	}
	slog.InfoContext(ctx, "personal_data_export_generated",
		"user_id", userID, "export_id", id, "status", status, "sections", len(sections))
	return &personalDataExportRow{
		ID: id, UserID: userID, Status: status, Content: string(content),
		GeneratedAt: now, ExpiresAt: expires,
	}, failures, nil
}

// getLatestPersonalDataExport 取该用户最近一次导出（可能已过期，由调用方判断）。
func getLatestPersonalDataExport(userID string) (*personalDataExportRow, error) {
	row := db.QueryRow(`
		SELECT id, user_id, status, content, generated_at, expires_at
		FROM auth_personal_data_export
		WHERE user_id = ? ORDER BY generated_at DESC LIMIT 1`, userID)
	var r personalDataExportRow
	err := row.Scan(&r.ID, &r.UserID, &r.Status, &r.Content, &r.GeneratedAt, &r.ExpiresAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// exportStatusPayload 组装 status 接口响应。
func exportStatusPayload(userID string) (map[string]interface{}, error) {
	row, err := getLatestPersonalDataExport(userID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return map[string]interface{}{
			"status":         "none",
			"retention_days": personalDataExportRetentionDays(),
		}, nil
	}
	if row.ExpiresAt.Before(time.Now().UTC()) {
		return map[string]interface{}{
			"status":         "expired",
			"export_id":      fmt.Sprintf("%d", row.ID),
			"generated_at":   row.GeneratedAt.UTC().Format(time.RFC3339),
			"expires_at":     row.ExpiresAt.UTC().Format(time.RFC3339),
			"retention_days": personalDataExportRetentionDays(),
		}, nil
	}
	return map[string]interface{}{
		"status":         row.Status,
		"export_id":      fmt.Sprintf("%d", row.ID),
		"generated_at":   row.GeneratedAt.UTC().Format(time.RFC3339),
		"expires_at":     row.ExpiresAt.UTC().Format(time.RFC3339),
		"retention_days": personalDataExportRetentionDays(),
	}, nil
}
