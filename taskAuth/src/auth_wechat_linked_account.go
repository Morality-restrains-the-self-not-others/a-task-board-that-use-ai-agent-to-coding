package main

import (
	"log/slog"
	"net/http"
	"strings"
	"unicode/utf8"

	"tracelog"
)

const maxWechatLinkedAccountQueryRunes = 128

func normalizeWechatLinkedAccountQuery(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", nil
	}
	if utf8.RuneCountInString(s) > maxWechatLinkedAccountQueryRunes {
		return "", errWechatLinkedAccountQueryTooLong
	}
	return s, nil
}

var errWechatLinkedAccountQueryTooLong = errString("wechat account query too long")

type errString string

func (e errString) Error() string { return string(e) }

func resolveWechatLinkedUserIDs(q string) ([]string, error) {
	seen := make(map[string]struct{})
	var ids []string
	addRows := func(query string, args ...interface{}) error {
		rows, err := db.Query(query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				return err
			}
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
			if len(ids) >= 50 {
				return nil
			}
		}
		return rows.Err()
	}

	if err := addRows(`
		SELECT DISTINCT CAST(user_id AS CHAR) COLLATE utf8mb4_unicode_ci
		FROM wechat_identity
		WHERE nickname COLLATE utf8mb4_unicode_ci = ?
		   OR openid COLLATE utf8mb4_unicode_ci = ?
		   OR unionid COLLATE utf8mb4_unicode_ci = ?
		LIMIT 50`, q, q, q); err != nil {
		return nil, err
	}
	if len(ids) >= 50 {
		return ids, nil
	}
	qLower := strings.ToLower(q)
	if err := addRows(`
		SELECT DISTINCT lm.object_id
		FROM auth_login_method lm
		INNER JOIN wechat_identity wi
		  ON CAST(wi.user_id AS CHAR) COLLATE utf8mb4_unicode_ci = lm.object_id
		WHERE lm.binding_voided_at IS NULL
		  AND lm.method_type IN ('phone', 'email', 'username')
		  AND lm.identifier COLLATE utf8mb4_unicode_ci IN (?, ?)
		LIMIT 50`, q, qLower); err != nil {
		return nil, err
	}
	if len(ids) >= 50 {
		return ids, nil
	}
	if err := addRows(`
		SELECT DISTINCT p.user_id
		FROM auth_user_profile p
		INNER JOIN wechat_identity wi
		  ON CAST(wi.user_id AS CHAR) COLLATE utf8mb4_unicode_ci = p.user_id
		WHERE p.username COLLATE utf8mb4_unicode_ci IN (?, ?)
		LIMIT 50`, q, qLower); err != nil {
		return nil, err
	}
	return ids, nil
}

// handleInternalWechatLinkedAccount GET /api/internal/users/wechat-linked-account/?q=
func handleInternalWechatLinkedAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireInternalSecret(r) {
		writeErrorDetail(w, r, http.StatusForbidden, "forbidden")
		return
	}
	q, err := normalizeWechatLinkedAccountQuery(r.URL.Query().Get("q"))
	if err != nil {
		writeErrorDetail(w, r, http.StatusBadRequest, "q too long")
		return
	}
	if q == "" {
		writeErrorDetail(w, r, http.StatusBadRequest, "q required")
		return
	}
	ids, err := resolveWechatLinkedUserIDs(q)
	if err != nil {
		slog.ErrorContext(r.Context(), "wechat_linked_account_lookup_failed",
			"query_len", utf8.RuneCountInString(q),
			"trace_id", tracelog.TraceIDFromContext(r.Context()),
		)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	if ids == nil {
		ids = []string{}
	}
	slog.InfoContext(r.Context(), "wechat_linked_account_lookup",
		"query_len", utf8.RuneCountInString(q),
		"user_count", len(ids),
		"trace_id", tracelog.TraceIDFromContext(r.Context()),
	)
	writeJSON(w, http.StatusOK, map[string]interface{}{"user_ids": ids})
}
