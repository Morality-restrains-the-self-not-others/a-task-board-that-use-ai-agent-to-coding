package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"gatewayauth"
	"snowflake"
)

// --- Recommended LLM Providers ---
//
// Admin CRUD at /api/system-admin/recommended-llm-providers/
// Tenant read-only at /api/recommended-llm-providers/

func handleSystemAdminRecommendedLLMProviders(w http.ResponseWriter, r *http.Request) {
	gatewayauth.ApplyGatewayUser(r, cfg.GatewayInternalSecret)

	switch r.Method {
	case http.MethodGet:
		listRecommendedProviders(w, r)
	case http.MethodPost:
		createRecommendedProvider(w, r)
	case http.MethodPut:
		updateRecommendedProvider(w, r)
	case http.MethodDelete:
		deleteRecommendedProvider(w, r)
	default:
		writeErrorJSON(w, r, 405, "method not allowed")
	}
}

func handleTenantRecommendedLLMProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, 405, "method not allowed")
		return
	}
	listRecommendedProviders(w, r)
}

type recommendedProvider struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DocsURL     string `json:"docs_url"`
	Description string `json:"description"`
	SortOrder   int    `json:"sort_order"`
}

func listRecommendedProviders(w http.ResponseWriter, r *http.Request) {
	conn, err := featureParamsConn()
	if err != nil {
		writeErrorJSON(w, r, 502, "数据库连接失败")
		return
	}
	rows, err := conn.Query(
		`SELECT id, name, docs_url, description, sort_order
		 FROM cloud_recommended_llm_providers ORDER BY sort_order ASC, name ASC`,
	)
	if err != nil {
		writeErrorJSON(w, r, 500, "查询推荐供应商失败")
		return
	}
	defer rows.Close()

	items := make([]recommendedProvider, 0)
	for rows.Next() {
		var item recommendedProvider
		if err := rows.Scan(&item.ID, &item.Name, &item.DocsURL, &item.Description, &item.SortOrder); err != nil {
			writeErrorJSON(w, r, 500, "读取推荐供应商数据失败")
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		writeErrorJSON(w, r, 500, "遍历推荐供应商数据失败")
		return
	}

	writeJSON(w, 200, map[string]interface{}{
		"items": items,
	})
}

func createRecommendedProvider(w http.ResponseWriter, r *http.Request) {
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, r, 400, "请求体解析失败")
		return
	}

	name := strings.TrimSpace(strField(body, "name"))
	if name == "" {
		writeErrorJSON(w, r, 400, "供应商名称不能为空")
		return
	}

	conn, err := featureParamsConn()
	if err != nil {
		writeErrorJSON(w, r, 502, "数据库连接失败")
		return
	}

	docsURL := strings.TrimSpace(strField(body, "docs_url"))
	description := strings.TrimSpace(strField(body, "description"))
	sortOrder := intFromAny(body["sort_order"], 0)
	now := fpNowUTC()
	id := snowflake.GenerateIDString()

	_, err = conn.Exec(
		`INSERT INTO cloud_recommended_llm_providers (id, name, docs_url, description, sort_order, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, name, docsURL, description, sortOrder, now, now,
	)
	if err != nil {
		writeErrorJSON(w, r, 500, "创建推荐供应商失败")
		return
	}

	var item recommendedProvider
	err = conn.QueryRow(
		`SELECT id, name, docs_url, description, sort_order
		 FROM cloud_recommended_llm_providers WHERE id = ?`, id,
	).Scan(&item.ID, &item.Name, &item.DocsURL, &item.Description, &item.SortOrder)
	if err != nil {
		writeErrorJSON(w, r, 500, "读取新建供应商失败")
		return
	}

	writeJSON(w, 201, map[string]interface{}{
		"item": item,
	})
}

func updateRecommendedProvider(w http.ResponseWriter, r *http.Request) {
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, r, 400, "请求体解析失败")
		return
	}

	id := strings.TrimSpace(strField(body, "id"))
	if id == "" {
		writeErrorJSON(w, r, 400, "id 不能为空")
		return
	}

	conn, err := featureParamsConn()
	if err != nil {
		writeErrorJSON(w, r, 502, "数据库连接失败")
		return
	}

	// Check existence
	var existingID string
	err = conn.QueryRow(
		`SELECT id FROM cloud_recommended_llm_providers WHERE id = ?`, id,
	).Scan(&existingID)
	if err == sql.ErrNoRows {
		writeErrorJSON(w, r, 404, "供应商不存在")
		return
	}
	if err != nil {
		writeErrorJSON(w, r, 500, "查询供应商失败")
		return
	}

	now := fpNowUTC()

	// Build dynamic update — only set fields present in body
	setClauses := []string{"updated_at = ?"}
	args := []interface{}{now}

	if v, ok := body["name"]; ok {
		setClauses = append(setClauses, "name = ?")
		args = append(args, strings.TrimSpace(fmt.Sprintf("%v", v)))
	}
	if v, ok := body["docs_url"]; ok {
		setClauses = append(setClauses, "docs_url = ?")
		args = append(args, strings.TrimSpace(fmt.Sprintf("%v", v)))
	}
	if v, ok := body["description"]; ok {
		setClauses = append(setClauses, "description = ?")
		args = append(args, strings.TrimSpace(fmt.Sprintf("%v", v)))
	}
	if v, ok := body["sort_order"]; ok {
		setClauses = append(setClauses, "sort_order = ?")
		args = append(args, intFromAny(v, 0))
	}

	if len(setClauses) == 1 {
		writeErrorJSON(w, r, 400, "没有需要更新的字段")
		return
	}

	args = append(args, id)
	query := fmt.Sprintf(
		"UPDATE cloud_recommended_llm_providers SET %s WHERE id = ?",
		strings.Join(setClauses, ", "),
	)
	_, err = conn.Exec(query, args...)
	if err != nil {
		writeErrorJSON(w, r, 500, "更新推荐供应商失败")
		return
	}

	var item recommendedProvider
	err = conn.QueryRow(
		`SELECT id, name, docs_url, description, sort_order
		 FROM cloud_recommended_llm_providers WHERE id = ?`, id,
	).Scan(&item.ID, &item.Name, &item.DocsURL, &item.Description, &item.SortOrder)
	if err != nil {
		writeErrorJSON(w, r, 500, "读取更新后供应商失败")
		return
	}

	writeJSON(w, 200, map[string]interface{}{
		"item": item,
	})
}

func deleteRecommendedProvider(w http.ResponseWriter, r *http.Request) {
	body, err := readJSONBody(r)
	if err != nil && r.Body != nil {
		// Allow id from query param as fallback for DELETE
	}
	_ = body // may be empty — extract id from query or body

	id := strings.TrimSpace(strField(body, "id"))
	if id == "" {
		id = strings.TrimSpace(r.URL.Query().Get("id"))
	}
	if id == "" {
		writeErrorJSON(w, r, 400, "id 不能为空")
		return
	}

	conn, err := featureParamsConn()
	if err != nil {
		writeErrorJSON(w, r, 502, "数据库连接失败")
		return
	}

	var existingID string
	err = conn.QueryRow(
		`SELECT id FROM cloud_recommended_llm_providers WHERE id = ?`, id,
	).Scan(&existingID)
	if err == sql.ErrNoRows {
		writeErrorJSON(w, r, 404, "供应商不存在")
		return
	}
	if err != nil {
		writeErrorJSON(w, r, 500, "查询供应商失败")
		return
	}

	_, err = conn.Exec(`DELETE FROM cloud_recommended_llm_providers WHERE id = ?`, id)
	if err != nil {
		writeErrorJSON(w, r, 500, "删除推荐供应商失败")
		return
	}

	writeJSON(w, 200, map[string]interface{}{
		"status": "deleted",
	})
}

// ensureRecommendedProvidersSeeded verifies that dataMigrate seeded the recommended
// LLM providers table. If no rows exist, it warns but does NOT insert — the seed is
// the single source of truth in dataMigrate/taskCloudService/011_seed_default_recommended_llm_providers.sql
// (fresh installs) plus 021_replace_openai_anthropic_with_xiaomi_mimo.sql (existing DBs).
func ensureRecommendedProvidersSeeded() error {
	conn, err := featureParamsConn()
	if err != nil {
		return err
	}
	var count int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM cloud_recommended_llm_providers`).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf(
			"cloud_recommended_llm_providers 表为空 — dataMigrate 可能未执行。" +
				"请运行: apply_datamigrate.sh task_cloud dataMigrate/taskCloudService",
		)
	}
	return nil
}
