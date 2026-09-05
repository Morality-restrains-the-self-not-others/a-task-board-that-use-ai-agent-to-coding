package infrastructure

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const defaultGrantTicketTTL = 30 * time.Minute

// IssueGrantTicket stores a one-shot L2 ticket bound to user + gitsite.
func (d *DB) IssueGrantTicket(userID, gitsite, remoteUserID string, ttl time.Duration) (string, error) {
	if d == nil {
		return "", fmt.Errorf("nil db")
	}
	uid := strings.TrimSpace(userID)
	site := strings.ToLower(strings.TrimSpace(gitsite))
	if uid == "" || site == "" {
		return "", fmt.Errorf("grant ticket missing user or gitsite")
	}
	if ttl <= 0 {
		ttl = defaultGrantTicketTTL
	}
	id := RandomTokenURLSafe(24)
	_, err := d.Exec(`
INSERT INTO git_oauth_grant_ticket (id, task2app_user_id, gitsite, remote_user_id, expires_at, created_at)
VALUES (?, ?, ?, ?, ?, NOW())`,
		id, uid, site, strings.TrimSpace(remoteUserID), time.Now().UTC().Add(ttl).Format("2006-01-02 15:04:05"))
	if err != nil {
		return "", err
	}
	return id, nil
}

// ConsumeGrantTicket marks a ticket used. Returns remote_user_id when consumed.
func (d *DB) ConsumeGrantTicket(id, userID, gitsite string) (remoteUserID string, ok bool, err error) {
	if d == nil {
		return "", false, fmt.Errorf("nil db")
	}
	ticketID := strings.TrimSpace(id)
	uid := strings.TrimSpace(userID)
	site := strings.ToLower(strings.TrimSpace(gitsite))
	if ticketID == "" || uid == "" || site == "" {
		return "", false, nil
	}
	res, err := d.Exec(`
UPDATE git_oauth_grant_ticket
SET consumed_at = NOW()
WHERE id = ? AND task2app_user_id = ? AND gitsite = ?
  AND consumed_at IS NULL AND expires_at > NOW()`,
		ticketID, uid, site)
	if err != nil {
		return "", false, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return "", false, nil
	}
	err = d.QueryRow(`SELECT remote_user_id FROM git_oauth_grant_ticket WHERE id = ?`, ticketID).Scan(&remoteUserID)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return strings.TrimSpace(remoteUserID), true, nil
}
