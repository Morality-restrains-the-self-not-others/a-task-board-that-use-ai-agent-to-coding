package main

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"taskTaskService/src/domain"
)

const (
	humanCommentDefaultLimit = 50
	humanCommentMaxLimit     = 200
	detailCommentEnrichLimit = 50
)

type humanCommentPageParams struct {
	limit           int
	cursorCreatedAt time.Time
	cursorID        string
	hasCursor       bool
	paginate        bool
}

func parseHumanCommentPageQuery(r *http.Request) humanCommentPageParams {
	q := r.URL.Query()
	p := humanCommentPageParams{limit: humanCommentDefaultLimit}
	if lim := strings.TrimSpace(q.Get("limit")); lim != "" {
		p.paginate = true
		n, err := strconv.Atoi(lim)
		if err != nil || n < 1 {
			n = humanCommentDefaultLimit
		}
		if n > humanCommentMaxLimit {
			n = humanCommentMaxLimit
		}
		p.limit = n
	}
	if cur := strings.TrimSpace(q.Get("cursor")); cur != "" {
		if ts, id, ok := decodeHumanCommentCursor(cur); ok {
			p.hasCursor = true
			p.cursorCreatedAt = ts
			p.cursorID = id
		}
	}
	return p
}

func encodeHumanCommentCursor(createdAt time.Time, id string) string {
	raw := createdAt.UTC().Format(time.RFC3339Nano) + "|" + id
	return url.QueryEscape(raw)
}

func decodeHumanCommentCursor(raw string) (time.Time, string, bool) {
	decoded, err := url.QueryUnescape(strings.TrimSpace(raw))
	if err != nil {
		decoded = strings.TrimSpace(raw)
	}
	parts := strings.SplitN(decoded, "|", 2)
	if len(parts) != 2 || parts[1] == "" {
		return time.Time{}, "", false
	}
	ts, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		ts, err = time.Parse(time.RFC3339, parts[0])
		if err != nil {
			return time.Time{}, "", false
		}
	}
	return ts, parts[1], true
}

func scanHumanCommentRow(s humanCommentScan, ca time.Time) map[string]interface{} {
	item := map[string]interface{}{
		"id": s.id,
		"created_by": map[string]interface{}{
			"id": s.author, "username": s.author, "email": "",
		},
		"content":                s.content,
		"execution_mode":         s.executionMode,
		"depends_on_comment_ids": parseDependsOnCommentIDsJSON(s.dependsOnIDsJSON),
		"repo_identities":        repoIdentitiesFromJSONColumn(s.identitiesJSON),
		"created_at":             ca.UTC().Format(time.RFC3339Nano), // RFC3339 with Z
	}
	if parsed, err := domain.ParseMentionsJSON(s.mentionsJSON); err == nil && len(parsed) > 0 {
		item["mentions"] = parsed
	}
	applyGitPRToCommentItem(item, s.parentCommentID, s.gitPRHTMLURL, s.gitPRJSON)
	return item
}

func scanHumanCommentRowFromRows(rows interface {
	Scan(dest ...any) error
}) (map[string]interface{}, time.Time, error) {
	var s humanCommentScan
	var createdRaw string
	if err := rows.Scan(
		&s.id, &s.author, &s.content, &s.mentionsJSON, &s.executionMode, &s.dependsOnIDsJSON, &s.identitiesJSON,
		&s.parentCommentID, &s.gitPRHTMLURL, &s.gitPRJSON, &createdRaw,
	); err != nil {
		return nil, time.Time{}, err
	}
	ca := parseMySQLUTCDateTime(createdRaw)
	return scanHumanCommentRow(s, ca), ca, nil
}

func loadHumanCommentsForTask(taskID string) []map[string]interface{} {
	rows, err := db.Query(
		`SELECT `+humanCommentSelectCols+` FROM task_comments WHERE task_id=? ORDER BY created_at ASC, id ASC`,
		taskID,
	)
	if err != nil {
		return []map[string]interface{}{}
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		item, _, err := scanHumanCommentRowFromRows(rows)
		if err != nil {
			continue
		}
		out = append(out, item)
	}
	return out
}

func listHumanCommentsPage(taskID string, p humanCommentPageParams) ([]map[string]interface{}, bool, time.Time, string, error) {
	limit := p.limit
	if limit < 1 {
		limit = humanCommentDefaultLimit
	}
	query := `SELECT ` + humanCommentSelectCols + ` FROM task_comments WHERE task_id=?`
	args := []interface{}{taskID}
	if p.hasCursor {
		query += ` AND (created_at < ? OR (created_at = ? AND id < ?))`
		cursor := formatMySQLUTCDateTime(p.cursorCreatedAt)
		args = append(args, cursor, cursor, p.cursorID)
	}
	query += ` ORDER BY created_at DESC, id DESC LIMIT ?`
	args = append(args, limit+1)
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, false, time.Time{}, "", err
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	var lastCreatedAt time.Time
	var lastID string
	for rows.Next() {
		item, ca, err := scanHumanCommentRowFromRows(rows)
		if err != nil {
			return nil, false, time.Time{}, "", err
		}
		out = append(out, item)
		id, _ := item["id"].(string)
		lastID = id
		lastCreatedAt = ca
	}
	hasMore := len(out) > limit
	if hasMore {
		out = out[:limit]
	}
	return out, hasMore, lastCreatedAt, lastID, rows.Err()
}

func humanCommentsPageResponse(results []map[string]interface{}, hasMore bool, lastCreatedAt time.Time, lastID string) map[string]interface{} {
	out := map[string]interface{}{
		"results":  results,
		"has_more": hasMore,
	}
	if hasMore && !lastCreatedAt.IsZero() && lastID != "" {
		out["next_cursor"] = encodeHumanCommentCursor(lastCreatedAt, lastID)
	} else {
		out["next_cursor"] = nil
	}
	return out
}

func enrichTaskDetailComments(out map[string]interface{}, t *taskRecord) {
	humanPage := humanCommentPageParams{
		limit:    detailCommentEnrichLimit,
		paginate: true,
	}
	comments, hasMore, lastCreatedAt, lastID, err := listHumanCommentsPage(t.ID, humanPage)
	if err != nil {
		comments = []map[string]interface{}{}
		hasMore = false
	}
	if comments == nil {
		comments = []map[string]interface{}{}
	}
	out["comments"] = comments
	out["comments_has_more"] = hasMore
	if grants := loadCommentOAuthGrantsForTask(t.ID); len(grants) > 0 {
		out["comment_oauth_grants"] = grants
	} else {
		out["comment_oauth_grants"] = []map[string]string{}
	}
	if hasMore && !lastCreatedAt.IsZero() && lastID != "" {
		out["comments_next_cursor"] = encodeHumanCommentCursor(lastCreatedAt, lastID)
	} else {
		out["comments_next_cursor"] = nil
	}

	feedErrors := []string{}
	aiPage, aiErr := fetchAICommentsForTask(t.ID, detailCommentEnrichLimit)
	if aiErr != nil {
		feedErrors = append(feedErrors, "ai_comments: "+aiErr.Error())
		aiPage = commentPageResult{results: []map[string]interface{}{}}
	}
	out["ai_comments"] = aiPage.results
	out["ai_comments_has_more"] = aiPage.hasMore
	out["ai_comments_next_cursor"] = aiPage.nextCursor

	agentPage, agentErr := fetchContainerAgentCommentsForTask(t.ID, detailCommentEnrichLimit)
	if agentErr != nil {
		feedErrors = append(feedErrors, "container_agent_comments: "+agentErr.Error())
		agentPage = commentPageResult{results: []map[string]interface{}{}}
	}
	out["container_agent_comments"] = agentPage.results
	out["container_agent_comments_has_more"] = agentPage.hasMore
	out["container_agent_comments_next_cursor"] = agentPage.nextCursor

	if len(feedErrors) > 0 {
		out["comments_feed_errors"] = feedErrors
	}
}

// dropLegacyAITaskCommentsIfEmpty removed (originally lines 220-241).
// It checked SQLite's sqlite_master for ai_comment_task_comments table existence,
// and conditionally dropped it if empty. On MySQL, this is always a no-op
// because sqlite_master doesn't exist. The ai_comment_task_comments table was part
// of the legacy SQLite-era task_task.db; modern deployments use taskAIComment
// service exclusively.
// See git history for the original implementation.
