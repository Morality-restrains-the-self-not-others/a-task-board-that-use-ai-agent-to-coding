package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"clientip"
	"taskAuth/domain"
	"tracelog"
)

type loginHistoryRow struct {
	ID         int64
	UserID     string
	LoggedInAt string
	ClientIP   string
	UserAgent  string
	Entry      string
	MethodType string
	Outcome    string
}

func insertLoginHistory(userID, clientIP, userAgent, entry, methodType string) error {
	userID = strings.TrimSpace(userID)
	entry = domain.NormalizeLoginEntry(entry)
	if userID == "" || entry == "" || db == nil {
		return nil
	}
	id := generateSnowflakeID()
	now := timeNowUTC()
	_, err := db.Exec(`
		INSERT INTO auth_login_history
		  (id, user_id, logged_in_at, client_ip, user_agent, entry, method_type, outcome)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, userID, now,
		strings.TrimSpace(clientIP),
		domain.TruncateLoginUserAgent(userAgent),
		entry,
		strings.TrimSpace(methodType),
		domain.LoginOutcomeSuccess,
	)
	return err
}

// insertLoginAttempt 记录一次已识别账号的失败登录尝试（fail-open：DB 错误仅告警不阻断）。
// outcome 必须是显式失败结果；success 由 insertLoginHistory 独占写入。
func insertLoginAttempt(userID, clientIP, userAgent, outcome, entry, methodType string) error {
	userID = strings.TrimSpace(userID)
	entry = domain.NormalizeLoginEntry(entry)
	outcome = strings.TrimSpace(outcome)
	if userID == "" || entry == "" || outcome == "" || outcome == domain.LoginOutcomeSuccess || db == nil {
		return nil
	}
	id := generateSnowflakeID()
	now := timeNowUTC()
	_, err := db.Exec(`
		INSERT INTO auth_login_history
		  (id, user_id, logged_in_at, client_ip, user_agent, entry, method_type, outcome)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, userID, now,
		strings.TrimSpace(clientIP),
		domain.TruncateLoginUserAgent(userAgent),
		entry,
		strings.TrimSpace(methodType),
		outcome,
	)
	return err
}

// recordFailedLoginFromRequest 在失败分支（user_id 已知）落一条失败尝试。
// 不存 identifier（邮箱/手机/用户名）；client_ip / user_agent 与成功行同口径截断。
func recordFailedLoginFromRequest(r *http.Request, userID, outcome, methodType, entry string) {
	if r == nil {
		return
	}
	ctx := r.Context()
	ip := resolveClientIP(r)
	if ip != "" && !clientip.IsPublicIPAddr(ip) {
		slog.WarnContext(ctx, "login_history_client_ip_not_public",
			"trace_id", tracelog.TraceIDFromContext(ctx),
			"client_ip", ip,
			"outcome", outcome,
		)
	}
	ua := r.UserAgent()
	if err := insertLoginAttempt(userID, ip, ua, outcome, entry, methodType); err != nil {
		slog.WarnContext(ctx, "login_attempt_insert_failed",
			"trace_id", tracelog.TraceIDFromContext(ctx),
			"user_id", userID,
			"outcome", outcome,
			"error", err.Error(),
		)
	} else {
		slog.InfoContext(ctx, "login_attempt_recorded",
			"trace_id", tracelog.TraceIDFromContext(ctx),
			"user_id", userID,
			"outcome", outcome,
			"method_type", methodType,
			"client_ip", ip,
		)
	}
}

func listLoginHistory(userID string, limit, offset int, includeFailures bool) ([]loginHistoryRow, int64, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" || db == nil {
		return nil, 0, nil
	}
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	// 默认只展示成功历史（outcome='success'）；includeFailures 时展示含失败尝试的全量。
	outcomeClause := ""
	if !includeFailures {
		outcomeClause = " AND outcome = '" + domain.LoginOutcomeSuccess + "'"
	}
	countQuery := `SELECT COUNT(*) FROM auth_login_history WHERE user_id = ?` + outcomeClause
	var total int64
	if err := db.QueryRow(countQuery, userID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := db.Query(`
		SELECT id, user_id, logged_in_at, client_ip, user_agent, entry, method_type, outcome
		FROM auth_login_history
		WHERE user_id = ?`+outcomeClause+`
		ORDER BY logged_in_at DESC, id DESC
		LIMIT ? OFFSET ?`, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := make([]loginHistoryRow, 0)
	for rows.Next() {
		var row loginHistoryRow
		var logged sql.NullTime
		if err := rows.Scan(&row.ID, &row.UserID, &logged, &row.ClientIP, &row.UserAgent, &row.Entry, &row.MethodType, &row.Outcome); err != nil {
			return nil, 0, err
		}
		if logged.Valid {
			row.LoggedInAt = logged.Time.UTC().Format("2006-01-02T15:04:05.000000Z")
		}
		out = append(out, row)
	}
	return out, total, rows.Err()
}

func loginHistoryJSON(row loginHistoryRow) map[string]interface{} {
	outcome := row.Outcome
	if outcome == "" {
		outcome = domain.LoginOutcomeSuccess
	}
	return map[string]interface{}{
		"id":            strconv.FormatInt(row.ID, 10),
		"logged_in_at":  row.LoggedInAt,
		"client_ip":     row.ClientIP,
		"user_agent":    row.UserAgent,
		"entry":         row.Entry,
		"method_type":   row.MethodType,
		"outcome":       outcome,
		"outcome_label": domain.LoginOutcomeLabel(outcome),
		"entry_label":   domain.LoginEntryLabel(row.Entry),
		"method_label":  domain.LoginMethodLabel(row.MethodType),
	}
}

func recordSuccessfulLoginFromRequest(r *http.Request, userID, identifier, methodType, entry, providerApp string) {
	if r == nil {
		return
	}
	ctx := r.Context()
	ip := resolveClientIP(r)
	if ip != "" && !clientip.IsPublicIPAddr(ip) {
		slog.WarnContext(ctx, "login_history_client_ip_not_public",
			"trace_id", tracelog.TraceIDFromContext(ctx),
			"client_ip", ip,
			"entry", entry,
		)
	}
	ua := r.UserAgent()
	if err := insertLoginHistory(userID, ip, ua, entry, methodType); err != nil {
		slog.WarnContext(ctx, "login_history_insert_failed",
			"trace_id", tracelog.TraceIDFromContext(ctx),
			"user_id", userID,
			"entry", entry,
			"error", err.Error(),
		)
	} else {
		slog.InfoContext(ctx, "login_history_recorded",
			"trace_id", tracelog.TraceIDFromContext(ctx),
			"user_id", userID,
			"entry", entry,
			"method_type", methodType,
			"client_ip", ip,
		)
	}
	publishUserLoggedInExtra(ctx, userID, identifier, methodType, providerApp, ip, ua, entry)
}

func publishUserLoggedInExtra(ctx context.Context, userID, identifier, methodType, providerApp, clientIP, userAgent, entry string) {
	data := map[string]interface{}{
		"user_id":     userID,
		"identifier":  identifier,
		"method_type": methodType,
	}
	if providerApp != "" {
		data["provider_app"] = providerApp
	}
	if clientIP != "" {
		data["client_ip"] = clientIP
	}
	if entry != "" {
		data["entry"] = entry
	}
	if ua := domain.TruncateLoginUserAgent(userAgent); ua != "" {
		data["user_agent"] = ua
	}
	if err := publishDomainEventKafka(ctx, "USER_LOGGED_IN", data, userID); err != nil {
		if err == errKafkaNotConfigured {
			return
		}
		slog.WarnContext(ctx, "USER_LOGGED_IN publish failed",
			"trace_id", tracelog.TraceIDFromContext(ctx),
			"error", err.Error(),
		)
	}
}
