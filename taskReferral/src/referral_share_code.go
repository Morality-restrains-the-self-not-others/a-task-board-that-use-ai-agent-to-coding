package main

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"unicode/utf8"

	"github.com/go-sql-driver/mysql"
)

const shareCodeLen = 10
const defaultChannelName = "默认"
const maxReferralChannels = 20

// Unambiguous alphabet (no 0/O, 1/l/I) so copied links stay readable.
const shareCodeAlphabet = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"

type referralChannelRow struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	IsDefault bool   `json:"is_default"`
	Status    string `json:"status"`
}

func generateShareCode() (string, error) {
	buf := make([]byte, shareCodeLen)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, shareCodeLen)
	n := len(shareCodeAlphabet)
	for i := range buf {
		out[i] = shareCodeAlphabet[int(buf[i])%n]
	}
	return string(out), nil
}

func isMySQLDuplicate(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}

func validateChannelName(name string) (string, error) {
	name = strings.TrimSpace(name)
	n := utf8.RuneCountInString(name)
	if n < 1 || n > 32 {
		return "", fmt.Errorf("invalid_name")
	}
	return name, nil
}

func lookupShareCodeByUser(userID string) (string, error) {
	var code string
	err := db.QueryRow(
		`SELECT code FROM referral_share_code
		 WHERE user_id = ? AND is_default = 1 AND status = 'active' LIMIT 1`, userID,
	).Scan(&code)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return code, nil
}

// lookupShareCodesByUsersBatch returns user_id → 默认分享码（is_default=1 active）。
// 请求的每个 id 都在结果集中；无默认码的用户为空串。用于申请列表批量水合，避免 N+1。
func lookupShareCodesByUsersBatch(userIDs []string) (map[string]string, error) {
	out := make(map[string]string, len(userIDs))
	ids := make([]string, 0, len(userIDs))
	seen := map[string]bool{}
	for _, raw := range userIDs {
		id := strings.TrimSpace(raw)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
		out[id] = ""
	}
	if len(ids) == 0 {
		return out, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	rows, err := db.Query(`
		SELECT user_id, code FROM referral_share_code
		WHERE user_id IN (`+strings.Join(placeholders, ",")+`) AND is_default = 1 AND status = 'active'`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var userID, code string
		if err := rows.Scan(&userID, &code); err != nil {
			continue
		}
		if code != "" {
			out[userID] = code
		}
	}
	return out, rows.Err()
}

// lookupShareCodeOwner resolves an opaque share code to user_id.
// Unknown / empty / disabled codes return "" (not an error). Must not parse u{userId}.
func lookupShareCodeOwner(code string) (string, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return "", nil
	}
	var userID string
	err := db.QueryRow(
		`SELECT user_id FROM referral_share_code WHERE code = ? AND status = 'active'`, code,
	).Scan(&userID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(userID), nil
}

func insertShareCode(code, userID, channelName string, isDefault bool) error {
	def := 0
	if isDefault {
		def = 1
	}
	_, err := db.Exec(
		`INSERT INTO referral_share_code (code, user_id, channel_name, is_default, status)
		 VALUES (?, ?, ?, ?, 'active')`,
		code, userID, channelName, def,
	)
	return err
}

func allocateShareCode(userID, channelName string, isDefault bool) (string, error) {
	for attempt := 0; attempt < 8; attempt++ {
		code, err := generateShareCode()
		if err != nil {
			return "", err
		}
		err = insertShareCode(code, userID, channelName, isDefault)
		if err == nil {
			return code, nil
		}
		if !isMySQLDuplicate(err) {
			return "", err
		}
		if isDefault {
			existing, lookupErr := lookupShareCodeByUser(userID)
			if lookupErr != nil {
				return "", lookupErr
			}
			if existing != "" {
				return existing, nil
			}
		} else {
			existing, lookupErr := lookupChannelByName(userID, channelName)
			if lookupErr != nil {
				return "", lookupErr
			}
			if existing != nil {
				return existing.Code, nil
			}
		}
	}
	return "", fmt.Errorf("share code allocate retries exhausted")
}

// ensureUserShareCode returns the default opaque share code for the user.
func ensureUserShareCode(userID string) (string, error) {
	id := strings.TrimSpace(userID)
	if id == "" {
		return "", fmt.Errorf("user_id required")
	}
	existing, err := lookupShareCodeByUser(id)
	if err != nil {
		return "", err
	}
	if existing != "" {
		return existing, nil
	}
	code, err := allocateShareCode(id, defaultChannelName, true)
	if err != nil {
		return "", err
	}
	log.Printf("[taskReferral] share code allocated user_id=%s", id)
	return code, nil
}

func lookupChannelByName(userID, name string) (*referralChannelRow, error) {
	var row referralChannelRow
	var isDefault int
	err := db.QueryRow(
		`SELECT code, channel_name, is_default, status FROM referral_share_code
		 WHERE user_id = ? AND channel_name = ? LIMIT 1`, userID, name,
	).Scan(&row.Code, &row.Name, &isDefault, &row.Status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	row.IsDefault = isDefault == 1
	return &row, nil
}

func listReferralChannels(userID string) ([]referralChannelRow, error) {
	if _, err := ensureUserShareCode(userID); err != nil {
		return nil, err
	}
	rows, err := db.Query(
		`SELECT code, channel_name, is_default, status FROM referral_share_code
		 WHERE user_id = ? ORDER BY is_default DESC, created_at ASC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]referralChannelRow, 0)
	for rows.Next() {
		var row referralChannelRow
		var isDefault int
		if err := rows.Scan(&row.Code, &row.Name, &isDefault, &row.Status); err != nil {
			return nil, err
		}
		row.IsDefault = isDefault == 1
		out = append(out, row)
	}
	return out, rows.Err()
}

func countActiveChannels(userID string) (int, error) {
	var n int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM referral_share_code WHERE user_id = ? AND status = 'active'`, userID,
	).Scan(&n)
	return n, err
}

func createReferralChannel(userID, name string) (*referralChannelRow, error) {
	id := strings.TrimSpace(userID)
	if id == "" {
		return nil, fmt.Errorf("user_id required")
	}
	name, err := validateChannelName(name)
	if err != nil {
		return nil, err
	}
	if _, err := ensureUserShareCode(id); err != nil {
		return nil, err
	}
	existing, err := lookupChannelByName(id, name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}
	n, err := countActiveChannels(id)
	if err != nil {
		return nil, err
	}
	if n >= maxReferralChannels {
		return nil, fmt.Errorf("channel_limit")
	}
	code, err := allocateShareCode(id, name, false)
	if err != nil {
		return nil, err
	}
	log.Printf("[taskReferral] referral event REFERRAL_CHANNEL_CREATED key=%s", id)
	return &referralChannelRow{Code: code, Name: name, IsDefault: false, Status: "active"}, nil
}

func disableReferralChannel(userID, code string) error {
	id := strings.TrimSpace(userID)
	code = strings.TrimSpace(code)
	if id == "" || code == "" {
		return fmt.Errorf("not_found")
	}
	var isDefault int
	var status string
	err := db.QueryRow(
		`SELECT is_default, status FROM referral_share_code WHERE user_id = ? AND code = ?`,
		id, code,
	).Scan(&isDefault, &status)
	if err == sql.ErrNoRows {
		return fmt.Errorf("not_found")
	}
	if err != nil {
		return err
	}
	if isDefault == 1 {
		return fmt.Errorf("cannot_disable_default")
	}
	if status == "disabled" {
		return nil
	}
	_, err = db.Exec(
		`UPDATE referral_share_code SET status = 'disabled' WHERE user_id = ? AND code = ? AND is_default = 0`,
		id, code,
	)
	if err == nil {
		log.Printf("[taskReferral] referral event REFERRAL_CHANNEL_DISABLED key=%s", id)
	}
	return err
}

// deleteReferralChannel hard-deletes a non-default channel. Historical
// referral edges in the bill service keep the channel_code string, so past
// settlements are unaffected; the code simply stops resolving for new binds.
func deleteReferralChannel(userID, code string) error {
	id := strings.TrimSpace(userID)
	code = strings.TrimSpace(code)
	if id == "" || code == "" {
		return fmt.Errorf("not_found")
	}
	var isDefault int
	err := db.QueryRow(
		`SELECT is_default FROM referral_share_code WHERE user_id = ? AND code = ?`,
		id, code,
	).Scan(&isDefault)
	if err == sql.ErrNoRows {
		return fmt.Errorf("not_found")
	}
	if err != nil {
		return err
	}
	if isDefault == 1 {
		return fmt.Errorf("cannot_delete_default")
	}
	_, err = db.Exec(
		`DELETE FROM referral_share_code WHERE user_id = ? AND code = ? AND is_default = 0`,
		id, code,
	)
	if err == nil {
		log.Printf("[taskReferral] referral event REFERRAL_CHANNEL_DELETED key=%s", id)
	}
	return err
}

func channelNameByCode(ownerUserID, code string) string {
	if strings.TrimSpace(code) == "" {
		return "历史（未分渠道）"
	}
	var name string
	err := db.QueryRow(
		`SELECT channel_name FROM referral_share_code WHERE user_id = ? AND code = ?`,
		ownerUserID, code,
	).Scan(&name)
	if err != nil {
		return code
	}
	return name
}
