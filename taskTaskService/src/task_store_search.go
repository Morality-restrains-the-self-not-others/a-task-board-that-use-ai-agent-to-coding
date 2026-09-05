package main

import (
	"log"
	"strings"
)

// taskSearchHit is a lightweight row for tenant-wide task search (billing filters, navbar, etc.).
// CommentID 仅在查询命中评论标识（cmt_* / 评论容器名）时非空，供前端定位到该评论。
type taskSearchHit struct {
	ID           string
	Title        string
	WorkspaceID  string
	OwnerID      string
	OperatorID   string
	WorkspaceSeq int
	Assignees    []string
	CommentID    string
}

func listDistinctTaskWorkspaceIDs(tenantID string) ([]string, error) {
	rows, err := db.Query(`SELECT DISTINCT workspace_id FROM task_tasks WHERE tenant_id=? AND workspace_id!=''`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var wid string
		if err := rows.Scan(&wid); err != nil {
			continue
		}
		wid = strings.TrimSpace(wid)
		if wid != "" {
			out = append(out, wid)
		}
	}
	return out, nil
}

func likeContainsPattern(q string) string {
	q = strings.ReplaceAll(q, `\`, `\\`)
	q = strings.ReplaceAll(q, `%`, `\%`)
	q = strings.ReplaceAll(q, `_`, `\_`)
	return "%" + q + "%"
}

// normalizeTaskSearchQuery rewrites common pasted task-id forms to the canonical
// genID("task") shape task_<digits>. Users often paste:
//   - service-prefixed ids like task-task_<digits>
//   - comment container / ECS instance names like task_<digits>_cmt_<commentId>
//
// which would otherwise miss id LIKE matches (query longer than the stored id).
func normalizeTaskSearchQuery(q string) string {
	q = strings.TrimSpace(q)
	if q == "" {
		return q
	}
	q = strings.TrimPrefix(q, "#")
	if strings.HasPrefix(q, "task-task_") {
		q = "task_" + strings.TrimPrefix(q, "task-task_")
	}
	if taskID := taskIDFromCommentContainerQuery(q); taskID != "" {
		return taskID
	}
	return q
}

func taskIDFromCommentContainerQuery(q string) string {
	const marker = "_cmt_"
	i := strings.Index(q, marker)
	if i <= 0 {
		return ""
	}
	head := q[:i]
	if strings.HasPrefix(head, "task_task_") {
		digits := strings.TrimPrefix(head, "task_task_")
		if isSearchIDDigits(digits) {
			return "task_" + digits
		}
		return ""
	}
	if strings.HasPrefix(head, "task_") {
		digits := strings.TrimPrefix(head, "task_")
		if isSearchIDDigits(digits) {
			return "task_" + digits
		}
	}
	return ""
}

func isSearchIDDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// commentIDFromSearchQuery extracts genID("cmt") from a pasted comment id or
// comment container name. Empty when the query is not a comment identifier.
func commentIDFromSearchQuery(q string) string {
	q = strings.TrimSpace(q)
	if q == "" {
		return ""
	}
	q = strings.TrimPrefix(q, "#")
	if strings.HasPrefix(q, "task-task_") {
		q = "task_" + strings.TrimPrefix(q, "task-task_")
	}
	if strings.HasPrefix(q, "cmt_") {
		return takeCommentIDToken(q)
	}
	const marker = "_cmt_"
	i := strings.Index(q, marker)
	if i < 0 {
		return ""
	}
	return takeCommentIDToken("cmt_" + q[i+len(marker):])
}

func takeCommentIDToken(s string) string {
	if !strings.HasPrefix(s, "cmt_") {
		return ""
	}
	rest := s[len("cmt_"):]
	n := 0
	for n < len(rest) {
		c := rest[n]
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' {
			n++
			continue
		}
		break
	}
	if n == 0 {
		return ""
	}
	return "cmt_" + rest[:n]
}

// parseAssigneeIDsCSV splits a comma-separated assignee_ids query into trimmed unique IDs.
func parseAssigneeIDsCSV(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, 4)
	for _, part := range strings.Split(raw, ",") {
		id := strings.TrimSpace(part)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// searchTasksInWorkspaces returns up to limit tasks in the given workspaces.
// Empty q (and no assigneeIDs) returns the most recently updated tasks.
// Non-empty q matches title, id, owner_id, operator_id, or assignee company_member_id (LIKE).
// Pasted comment container names are rewritten to task id; pasted cmt_* ids match task_comments.id.
// assigneeIDs (exact) OR into the same filter for assignee / owner / operator when provided.
func searchTasksInWorkspaces(tenantID string, workspaceIDs []string, q string, assigneeIDs []string, limit int) ([]taskSearchHit, error) {
	if len(workspaceIDs) == 0 || limit <= 0 {
		return []taskSearchHit{}, nil
	}
	placeholders := make([]string, len(workspaceIDs))
	args := make([]interface{}, 0, len(workspaceIDs)+12)
	args = append(args, tenantID)
	for i, wid := range workspaceIDs {
		placeholders[i] = "?"
		args = append(args, wid)
	}
	sqlStr := `SELECT id, title, workspace_id, COALESCE(owner_id,''), COALESCE(operator_id,''), COALESCE(workspace_seq,0) FROM task_tasks WHERE tenant_id=? AND workspace_id IN (` +
		strings.Join(placeholders, ",") + `)`
	origQ := q
	q = normalizeTaskSearchQuery(q)
	if origQ != q {
		log.Printf("[taskTaskService] tasks/search query_normalized orig_len=%d new_len=%d", len(origQ), len(q))
	}
	commentID := commentIDFromSearchQuery(origQ)
	orParts := make([]string, 0, 8)
	if q != "" {
		pat := likeContainsPattern(q)
		orParts = append(orParts,
			`title LIKE ? ESCAPE '\\'`,
			`id LIKE ? ESCAPE '\\'`,
			`owner_id LIKE ? ESCAPE '\\'`,
			`operator_id LIKE ? ESCAPE '\\'`,
			`EXISTS (SELECT 1 FROM task_assignees ta WHERE ta.task_id=task_tasks.id AND ta.company_member_id LIKE ? ESCAPE '\\')`,
		)
		args = append(args, pat, pat, pat, pat, pat)
		if seq, ok := parseNumericSearchSeq(q); ok {
			orParts = append(orParts, `workspace_seq=?`)
			args = append(args, seq)
		}
	}
	if commentID != "" {
		orParts = append(orParts, `EXISTS (SELECT 1 FROM task_comments tc WHERE tc.task_id=task_tasks.id AND tc.id=?)`)
		args = append(args, commentID)
	}
	if len(assigneeIDs) > 0 {
		aPh := make([]string, len(assigneeIDs))
		for i, aid := range assigneeIDs {
			aPh[i] = "?"
			args = append(args, aid)
		}
		inList := strings.Join(aPh, ",")
		orParts = append(orParts,
			`EXISTS (SELECT 1 FROM task_assignees ta2 WHERE ta2.task_id=task_tasks.id AND ta2.company_member_id IN (`+inList+`))`,
			`owner_id IN (`+inList+`)`,
			`operator_id IN (`+inList+`)`,
		)
		for range []int{0, 1} {
			for _, aid := range assigneeIDs {
				args = append(args, aid)
			}
		}
	}
	if len(orParts) > 0 {
		sqlStr += ` AND (` + strings.Join(orParts, " OR ") + `)`
	}
	sqlStr += ` ORDER BY updated_at DESC, id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := db.Query(sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]taskSearchHit, 0, limit)
	for rows.Next() {
		var hit taskSearchHit
		if err := rows.Scan(&hit.ID, &hit.Title, &hit.WorkspaceID, &hit.OwnerID, &hit.OperatorID, &hit.WorkspaceSeq); err != nil {
			continue
		}
		// 查询命中评论标识时，返回的 hit 均带该评论（SQL 已按 comment id 过滤），供前端滚动定位。
		hit.CommentID = commentID
		hit.Assignees = loadAssignees(hit.ID)
		if hit.Assignees == nil {
			hit.Assignees = []string{}
		}
		out = append(out, hit)
	}
	return out, nil
}
