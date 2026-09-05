package infrastructure

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func (d *DB) InsertAccessAudit(site string, userID string, companyID, workspaceID *int64, action, fingerprint string, detail map[string]any) error {
	if detail == nil {
		detail = map[string]any{}
	}
	blob, _ := json.Marshal(detail)
	_, err := d.Exec(`
INSERT INTO git_oauth_appaccesstokenuseaudit
  (site, task2app_user_id, company_id, workspace_id, action, access_token_fingerprint, detail, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		site, userID, nullInt64(companyID), nullInt64(workspaceID), action, fingerprint, string(blob),
		time.Now().UTC().Format("2006-01-02 15:04:05"),
	)
	return err
}

func (d *DB) InsertTaskAudit(provider, taskID, workspaceID string, companyID, userID *int64, action string, detail map[string]any) error {
	if detail == nil {
		detail = map[string]any{}
	}
	blob, _ := json.Marshal(detail)
	_, err := d.Exec(`
INSERT INTO git_oauth_taskcredentialaudit
  (provider, task2app_task_id, task2app_workspace_id, task2app_company_id, task2app_user_id, action, detail, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		provider, taskID, workspaceID, nullInt64(companyID), nullInt64(userID), action, string(blob),
		time.Now().UTC().Format("2006-01-02 15:04:05"),
	)
	return err
}

func nullInt64(p *int64) any {
	if p == nil {
		return nil
	}
	return *p
}

// MergeAuditRow 合并审计行：谁在何时点击了一键合并（审计回显用）。
type MergeAuditRow struct {
	UserID    string
	CreatedAt time.Time
}

// escapeLike 转义 MySQL LIKE 通配符（配合 ESCAPE '\\' 使用），使 URL 中的
// % _ \ 按字面匹配 detail JSON 文本。
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

// parseAuditCreatedAt 解析审计 created_at 列：该列写入时即为 UTC 墙钟（无时区
// 语义）。DSN 带 parseTime=true&loc=Local 时 driver 会按本地时区附加偏移
// （如 "2026-08-24T01:14:29+08:00"），此处剥离时区后缀并按 UTC 墙钟解析，
// 恢复真实的点击时刻；无偏移的原始字符串（"2006-01-02 15:04:05"）同样按 UTC。
func parseAuditCreatedAt(s string) time.Time {
	s = strings.TrimSpace(s)
	if len(s) >= 20 {
		switch s[19] {
		case 'Z', '+', '-':
			s = s[:19]
		}
	}
	s = strings.Replace(s, "T", " ", 1)
	layouts := []string{
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

// SelectLatestMergeAuditByURL 返回该 html_url 最近一次「一键合并」成功
// （result=merged / noop_already_merged）的审计行；无匹配返回 (nil, nil)。
// detail 是 JSON 文本，按 JSON 转义后的 html_url 精确子串匹配（Go json.Marshal
// 会把 & 写成 &，与写入时同一编码保证可匹配）。返回的 CreatedAt 优先取
// detail.merged_at（RFC3339，精确点击时间），历史行回退 created_at 列（UTC）。
func (d *DB) SelectLatestMergeAuditByURL(htmlURL string) (*MergeAuditRow, error) {
	escapedURL, err := json.Marshal(htmlURL)
	if err != nil {
		return nil, err
	}
	pattern := `%"html_url":"` + escapeLike(strings.Trim(string(escapedURL), `"`)) + `"%`
	var userID sql.NullString
	var createdAt, detail string
	err = d.QueryRow(`
SELECT task2app_user_id, created_at, detail
FROM git_oauth_taskcredentialaudit
WHERE action = 'merge_request_merge'
  AND detail LIKE ? ESCAPE '\\'
  AND (detail LIKE '%"result":"merged"%' OR detail LIKE '%"result":"noop_already_merged"%')
ORDER BY created_at DESC, id DESC
LIMIT 1`, pattern).Scan(&userID, &createdAt, &detail)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	row := &MergeAuditRow{CreatedAt: parseAuditCreatedAt(createdAt)}
	if userID.Valid {
		row.UserID = userID.String
	}
	// detail.merged_at（RFC3339 显式时区）优先于 created_at 列：精确且不受
	// DSN loc 影响。历史行无该字段时回退 parseAuditCreatedAt 的 UTC 墙钟解析。
	if strings.TrimSpace(detail) != "" {
		var detailMap map[string]any
		if json.Unmarshal([]byte(detail), &detailMap) == nil {
			if at := strings.TrimSpace(fmt.Sprint(detailMap["merged_at"])); at != "" && at != "<nil>" {
				row.CreatedAt = parseTime(at)
			}
		}
	}
	return row, nil
}
