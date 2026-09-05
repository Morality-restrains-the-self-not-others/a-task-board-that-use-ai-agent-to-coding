package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

type feedbackGroupRow struct {
	ID         string
	Name       string
	SortOrder  int
	Enabled    bool
	CreatedAt  string
	UpdatedAt  string
	Thresholds []feedbackThresholdRow
	Links      []feedbackLinkRow
}

type feedbackThresholdRow struct {
	ID           string
	ResourceKind string
	MinQuantity  float64
}

type feedbackLinkRow struct {
	ID        string
	Title     string
	URL       string
	SortOrder int
	Enabled   bool
}

func listFeedbackResourceKinds() ([]map[string]interface{}, error) {
	rows, err := db.Query(`SELECT kind, display_name, unit, enabled FROM billing_feedback_resource_kind WHERE enabled=1 ORDER BY kind`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var kind, name, unit string
		var enabled int
		if err := rows.Scan(&kind, &name, &unit, &enabled); err != nil {
			return nil, err
		}
		out = append(out, map[string]interface{}{
			"kind": kind, "display_name": name, "unit": unit, "enabled": enabled == 1,
		})
	}
	return out, rows.Err()
}

func kindEnabled(kind string) (bool, error) {
	var n int
	err := db.QueryRow(`SELECT COUNT(*) FROM billing_feedback_resource_kind WHERE kind=? AND enabled=1`, kind).Scan(&n)
	return n > 0, err
}

func loadFeedbackGroup(id string) (*feedbackGroupRow, error) {
	g := &feedbackGroupRow{ID: id}
	var enabled int
	err := db.QueryRow(`SELECT name, sort_order, enabled, created_at, updated_at FROM billing_feedback_link_group WHERE id=?`, id).
		Scan(&g.Name, &g.SortOrder, &enabled, &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		return nil, err
	}
	g.Enabled = enabled == 1
	if err := attachFeedbackChildren(g); err != nil {
		return nil, err
	}
	return g, nil
}

func listFeedbackGroups() ([]feedbackGroupRow, error) {
	rows, err := db.Query(`SELECT id, name, sort_order, enabled, created_at, updated_at FROM billing_feedback_link_group ORDER BY sort_order ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var groups []feedbackGroupRow
	for rows.Next() {
		var g feedbackGroupRow
		var enabled int
		if err := rows.Scan(&g.ID, &g.Name, &g.SortOrder, &enabled, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, err
		}
		g.Enabled = enabled == 1
		groups = append(groups, g)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range groups {
		if err := attachFeedbackChildren(&groups[i]); err != nil {
			return nil, err
		}
	}
	if groups == nil {
		groups = []feedbackGroupRow{}
	}
	return groups, nil
}

func attachFeedbackChildren(g *feedbackGroupRow) error {
	trows, err := db.Query(`SELECT id, resource_kind, min_quantity FROM billing_feedback_link_threshold WHERE group_id=? ORDER BY resource_kind`, g.ID)
	if err != nil {
		return err
	}
	defer trows.Close()
	g.Thresholds = []feedbackThresholdRow{}
	for trows.Next() {
		var t feedbackThresholdRow
		if err := trows.Scan(&t.ID, &t.ResourceKind, &t.MinQuantity); err != nil {
			return err
		}
		g.Thresholds = append(g.Thresholds, t)
	}
	lrows, err := db.Query(`SELECT id, title, url, sort_order, enabled FROM billing_feedback_link WHERE group_id=? ORDER BY sort_order ASC, id ASC`, g.ID)
	if err != nil {
		return err
	}
	defer lrows.Close()
	g.Links = []feedbackLinkRow{}
	for lrows.Next() {
		var l feedbackLinkRow
		var en int
		if err := lrows.Scan(&l.ID, &l.Title, &l.URL, &l.SortOrder, &en); err != nil {
			return err
		}
		l.Enabled = en == 1
		g.Links = append(g.Links, l)
	}
	return nil
}

func replaceFeedbackChildren(tx *sql.Tx, groupID string, thresholds []feedbackThresholdRow, links []feedbackLinkRow) error {
	if _, err := tx.Exec(`DELETE FROM billing_feedback_link_threshold WHERE group_id=?`, groupID); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM billing_feedback_link WHERE group_id=?`, groupID); err != nil {
		return err
	}
	for _, t := range thresholds {
		if _, err := tx.Exec(
			`INSERT INTO billing_feedback_link_threshold (id, group_id, resource_kind, min_quantity) VALUES (?,?,?,?)`,
			t.ID, groupID, t.ResourceKind, t.MinQuantity,
		); err != nil {
			return err
		}
	}
	for _, l := range links {
		en := 0
		if l.Enabled {
			en = 1
		}
		if _, err := tx.Exec(
			`INSERT INTO billing_feedback_link (id, group_id, title, url, sort_order, enabled) VALUES (?,?,?,?,?,?)`,
			l.ID, groupID, l.Title, l.URL, l.SortOrder, en,
		); err != nil {
			return err
		}
	}
	return nil
}

func saveFeedbackGroup(existingID string, name string, sortOrder int, enabled bool, thresholds []feedbackThresholdRow, links []feedbackLinkRow) (*feedbackGroupRow, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	id := existingID
	now := utcNow()
	en := 0
	if enabled {
		en = 1
	}
	if id == "" {
		id = formatID(generateSnowflakeID())
		if _, err := tx.Exec(
			`INSERT INTO billing_feedback_link_group (id, name, sort_order, enabled, created_at, updated_at) VALUES (?,?,?,?,?,?)`,
			id, name, sortOrder, en, now, now,
		); err != nil {
			return nil, err
		}
	} else {
		res, err := tx.Exec(
			`UPDATE billing_feedback_link_group SET name=?, sort_order=?, enabled=?, updated_at=? WHERE id=?`,
			name, sortOrder, en, now, id,
		)
		if err != nil {
			return nil, err
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return nil, sql.ErrNoRows
		}
	}
	if err := replaceFeedbackChildren(tx, id, thresholds, links); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return loadFeedbackGroup(id)
}

func deleteFeedbackGroup(id string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM billing_feedback_link_threshold WHERE group_id=?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM billing_feedback_link WHERE group_id=?`, id); err != nil {
		return err
	}
	res, err := tx.Exec(`DELETE FROM billing_feedback_link_group WHERE id=?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return tx.Commit()
}

func feedbackGroupAdminMap(g *feedbackGroupRow) map[string]interface{} {
	ths := make([]map[string]interface{}, 0, len(g.Thresholds))
	for _, t := range g.Thresholds {
		ths = append(ths, map[string]interface{}{
			"id": t.ID, "resource_kind": t.ResourceKind, "min_quantity": t.MinQuantity,
		})
	}
	links := make([]map[string]interface{}, 0, len(g.Links))
	for _, l := range g.Links {
		links = append(links, map[string]interface{}{
			"id": l.ID, "title": l.Title, "url": l.URL, "sort_order": l.SortOrder, "enabled": l.Enabled,
		})
	}
	return map[string]interface{}{
		"id": g.ID, "name": g.Name, "sort_order": g.SortOrder, "enabled": g.Enabled,
		"created_at": g.CreatedAt, "updated_at": g.UpdatedAt,
		"thresholds": ths, "links": links,
	}
}

func feedbackGroupTenantMap(g *feedbackGroupRow) map[string]interface{} {
	links := make([]map[string]interface{}, 0)
	for _, l := range g.Links {
		if !l.Enabled {
			continue
		}
		links = append(links, map[string]interface{}{
			"id": l.ID, "title": l.Title, "url": l.URL, "sort_order": l.SortOrder,
		})
	}
	return map[string]interface{}{
		"id": g.ID, "name": g.Name, "sort_order": g.SortOrder, "links": links,
	}
}

func loadFeedbackIdempotency(key string) (status int, body []byte, ok bool, err error) {
	var code int
	var raw string
	err = db.QueryRow(`SELECT status_code, response_body FROM billing_feedback_idempotency WHERE idempotency_key=?`, key).Scan(&code, &raw)
	if err == sql.ErrNoRows {
		return 0, nil, false, nil
	}
	if err != nil {
		return 0, nil, false, err
	}
	return code, []byte(raw), true, nil
}

func saveFeedbackIdempotency(key, method, groupID string, status int, payload interface{}) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = db.Exec(
		`INSERT INTO billing_feedback_idempotency (idempotency_key, method, group_id, status_code, response_body, created_at)
		 VALUES (?,?,?,?,?,?)`,
		key, method, groupID, status, string(raw), utcNow(),
	)
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "duplicate") {
		return nil
	}
	return err
}

func parseFeedbackSaveBody(body map[string]interface{}) (name string, sortOrder int, enabled bool, ths []feedbackThresholdRow, links []feedbackLinkRow, err error) {
	name = strings.TrimSpace(stringField(body, "name"))
	if name == "" {
		return "", 0, false, nil, nil, fmt.Errorf("name required")
	}
	if v, ok := body["sort_order"]; ok && v != nil {
		n, e := parseInt64Field(v)
		if e != nil {
			return "", 0, false, nil, nil, fmt.Errorf("invalid sort_order")
		}
		sortOrder = int(n)
	}
	enabled = true
	if _, ok := body["enabled"]; ok {
		enabled = boolField(body, "enabled")
	}
	rawTh, _ := body["thresholds"].([]interface{})
	seenKind := map[string]struct{}{}
	for _, item := range rawTh {
		m, ok := item.(map[string]interface{})
		if !ok {
			return "", 0, false, nil, nil, fmt.Errorf("invalid threshold")
		}
		kind := strings.TrimSpace(stringField(m, "resource_kind"))
		if kind == "" {
			return "", 0, false, nil, nil, fmt.Errorf("resource_kind required")
		}
		okKind, e := kindEnabled(kind)
		if e != nil {
			return "", 0, false, nil, nil, e
		}
		if !okKind {
			return "", 0, false, nil, nil, fmt.Errorf("unknown resource_kind")
		}
		if _, dup := seenKind[kind]; dup {
			return "", 0, false, nil, nil, fmt.Errorf("duplicate resource_kind")
		}
		seenKind[kind] = struct{}{}
		qty, e := parseFloat64Field(m["min_quantity"])
		if e != nil {
			return "", 0, false, nil, nil, fmt.Errorf("invalid min_quantity")
		}
		ths = append(ths, feedbackThresholdRow{
			ID: formatID(generateSnowflakeID()), ResourceKind: kind, MinQuantity: qty,
		})
	}
	rawLinks, _ := body["links"].([]interface{})
	for _, item := range rawLinks {
		m, ok := item.(map[string]interface{})
		if !ok {
			return "", 0, false, nil, nil, fmt.Errorf("invalid link")
		}
		title := strings.TrimSpace(stringField(m, "title"))
		url := strings.TrimSpace(stringField(m, "url"))
		if title == "" {
			return "", 0, false, nil, nil, fmt.Errorf("link title required")
		}
		if e := validateHttpsURL(url); e != nil {
			return "", 0, false, nil, nil, e
		}
		so := 0
		if v, ok := m["sort_order"]; ok && v != nil {
			n, e := parseInt64Field(v)
			if e != nil {
				return "", 0, false, nil, nil, fmt.Errorf("invalid link sort_order")
			}
			so = int(n)
		}
		en := true
		if _, ok := m["enabled"]; ok {
			en = boolField(m, "enabled")
		}
		links = append(links, feedbackLinkRow{
			ID: formatID(generateSnowflakeID()), Title: title, URL: url, SortOrder: so, Enabled: en,
		})
	}
	return name, sortOrder, enabled, ths, links, nil
}
