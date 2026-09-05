package main

import (
	"database/sql"
	"strconv"
	"strings"
	"time"

	"taskAuth/domain"
)

type inboxMessageRow struct {
	ID                     int64
	RecipientUserID        string
	Kind                   string
	Title                  string
	Body                   string
	Reason                 string
	ActorUserID            string
	ImpersonationSessionID sql.NullInt64
	CreatedAt              time.Time
	ReadAt                 sql.NullTime
}

func listInboxMessages(recipientUserID string, limit int) ([]inboxMessageRow, error) {
	recipientUserID = strings.TrimSpace(recipientUserID)
	if recipientUserID == "" {
		return nil, sql.ErrNoRows
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := db.Query(`
		SELECT id, recipient_user_id, kind, title, body, reason, actor_user_id,
		       impersonation_session_id, created_at, read_at
		FROM auth_user_inbox_message
		WHERE recipient_user_id = ?
		ORDER BY created_at DESC, id DESC
		LIMIT ?`, recipientUserID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]inboxMessageRow, 0)
	for rows.Next() {
		var row inboxMessageRow
		if err := rows.Scan(
			&row.ID, &row.RecipientUserID, &row.Kind, &row.Title, &row.Body, &row.Reason,
			&row.ActorUserID, &row.ImpersonationSessionID, &row.CreatedAt, &row.ReadAt,
		); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func markInboxMessageRead(recipientUserID, messageID string) error {
	recipientUserID = strings.TrimSpace(recipientUserID)
	messageID = strings.TrimSpace(messageID)
	if recipientUserID == "" || messageID == "" {
		return sql.ErrNoRows
	}
	res, err := db.Exec(`
		UPDATE auth_user_inbox_message
		SET read_at = UTC_TIMESTAMP()
		WHERE id = ? AND recipient_user_id = ? AND read_at IS NULL`,
		messageID, recipientUserID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		var exists int
		err = db.QueryRow(`
			SELECT 1 FROM auth_user_inbox_message WHERE id = ? AND recipient_user_id = ? LIMIT 1`,
			messageID, recipientUserID,
		).Scan(&exists)
		if err != nil {
			return sql.ErrNoRows
		}
	}
	return nil
}

func inboxMessageJSON(row inboxMessageRow) map[string]interface{} {
	item := map[string]interface{}{
		"id":                strconv.FormatInt(row.ID, 10),
		"recipient_user_id": row.RecipientUserID,
		"kind":              row.Kind,
		"title":             row.Title,
		"body":              domain.StripEmbeddedInboxReason(row.Body, row.Reason),
		"reason":            row.Reason,
		"actor_user_id":     row.ActorUserID,
		"created_at":        row.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		"read":              row.ReadAt.Valid,
	}
	if row.ImpersonationSessionID.Valid {
		item["impersonation_session_id"] = strconv.FormatInt(row.ImpersonationSessionID.Int64, 10)
	}
	if row.ReadAt.Valid {
		item["read_at"] = row.ReadAt.Time.UTC().Format("2006-01-02T15:04:05Z")
	}
	return item
}
