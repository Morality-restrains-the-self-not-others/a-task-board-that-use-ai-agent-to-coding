package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"gatewayauth"
	"snowflake"
)

// --- Sub-Token Providers ---
//
// Admin CRUD at /api/system-admin/sub-token-providers/
// Tenant read-only at /api/sub-token-providers/
//
// These are AI providers that support deriving sub-keys from a master key.
// The list is used by the feature-params UI to auto-check use_sub_token
// for known providers, and by the AI endpoint proxy to rewrite base_url/api_key.

func handleSystemAdminSubTokenProviders(w http.ResponseWriter, r *http.Request) {
	gatewayauth.ApplyGatewayUser(r, cfg.GatewayInternalSecret)

	switch r.Method {
	case http.MethodGet:
		listSubTokenProviders(w, r)
	case http.MethodPost:
		createSubTokenProvider(w, r)
	case http.MethodPut:
		updateSubTokenProvider(w, r)
	case http.MethodDelete:
		deleteSubTokenProvider(w, r)
	default:
		writeErrorJSON(w, r, 405, "method not allowed")
	}
}

func handleTenantSubTokenProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, 405, "method not allowed")
		return
	}
	listSubTokenProviders(w, r)
}

type subTokenProvider struct {
	ID                 string          `json:"id"`
	ProviderName       string          `json:"provider_name"`
	BaseURL            string          `json:"base_url"`
	DeriveEndpoint     string          `json:"derive_endpoint"`
	DeriveMethod       string          `json:"derive_method"`
	DeriveParams       json.RawMessage `json:"derive_params"`
	ResponseTokenField string          `json:"response_token_field"`
	ExpiresInField     string          `json:"expires_in_field"`
	Description        string          `json:"description"`
}

type subTokenProviderRow struct {
	ID                 string
	ProviderName       string
	BaseURL            string
	DeriveEndpoint     string
	DeriveMethod       string
	DeriveParams       string
	ResponseTokenField string
	ExpiresInField     string
	Description        string
}

func (row subTokenProviderRow) toItem() subTokenProvider {
	var deriveParams json.RawMessage
	if row.DeriveParams != "" {
		deriveParams = json.RawMessage(row.DeriveParams)
	} else {
		deriveParams = json.RawMessage("{}")
	}
	return subTokenProvider{
		ID:                 row.ID,
		ProviderName:       row.ProviderName,
		BaseURL:            row.BaseURL,
		DeriveEndpoint:     row.DeriveEndpoint,
		DeriveMethod:       row.DeriveMethod,
		DeriveParams:       deriveParams,
		ResponseTokenField: row.ResponseTokenField,
		ExpiresInField:     row.ExpiresInField,
		Description:        row.Description,
	}
}

func listSubTokenProviders(w http.ResponseWriter, r *http.Request) {
	conn, err := featureParamsConn()
	if err != nil {
		writeErrorJSON(w, r, 502, "数据库连接失败")
		return
	}
	rows, err := conn.Query(
		`SELECT id, provider_name, base_url, derive_endpoint, derive_method,
			derive_params, response_token_field, expires_in_field, description
		 FROM cloud_sub_token_providers ORDER BY provider_name ASC`,
	)
	if err != nil {
		writeErrorJSON(w, r, 500, "查询派生Token供应商失败")
		return
	}
	defer rows.Close()

	items := make([]subTokenProvider, 0)
	for rows.Next() {
		var row subTokenProviderRow
		if err := rows.Scan(
			&row.ID, &row.ProviderName, &row.BaseURL, &row.DeriveEndpoint,
			&row.DeriveMethod, &row.DeriveParams, &row.ResponseTokenField,
			&row.ExpiresInField, &row.Description,
		); err != nil {
			writeErrorJSON(w, r, 500, "读取派生Token供应商数据失败")
			return
		}
		items = append(items, row.toItem())
	}
	if err := rows.Err(); err != nil {
		writeErrorJSON(w, r, 500, "遍历派生Token供应商数据失败")
		return
	}

	writeJSON(w, 200, map[string]interface{}{
		"items": items,
	})
}

func createSubTokenProvider(w http.ResponseWriter, r *http.Request) {
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, r, 400, "请求体解析失败")
		return
	}

	providerName := strings.TrimSpace(strField(body, "provider_name"))
	if providerName == "" {
		writeErrorJSON(w, r, 400, "供应商名称不能为空")
		return
	}
	baseURL := strings.TrimSpace(strField(body, "base_url"))
	if baseURL == "" {
		writeErrorJSON(w, r, 400, "API基础地址不能为空")
		return
	}

	conn, err := featureParamsConn()
	if err != nil {
		writeErrorJSON(w, r, 502, "数据库连接失败")
		return
	}

	deriveEndpoint := strings.TrimSpace(strField(body, "derive_endpoint"))
	if deriveEndpoint == "" {
		deriveEndpoint = "/api/token/derive"
	}
	deriveMethod := strings.TrimSpace(strField(body, "derive_method"))
	if deriveMethod == "" {
		deriveMethod = "POST"
	}
	deriveParams := "{}"
	if raw, ok := body["derive_params"]; ok && raw != nil {
		if b, err := json.Marshal(raw); err == nil {
			deriveParams = string(b)
		}
	}
	responseTokenField := strings.TrimSpace(strField(body, "response_token_field"))
	if responseTokenField == "" {
		responseTokenField = "token"
	}
	expiresInField := strings.TrimSpace(strField(body, "expires_in_field"))
	if expiresInField == "" {
		expiresInField = "expires_in"
	}
	description := strings.TrimSpace(strField(body, "description"))
	now := fpNowUTC()
	id := snowflake.GenerateIDString()

	_, err = conn.Exec(
		`INSERT INTO cloud_sub_token_providers
			(id, provider_name, base_url, derive_endpoint, derive_method,
			 derive_params, response_token_field, expires_in_field, description,
			 created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, providerName, baseURL, deriveEndpoint, deriveMethod,
		deriveParams, responseTokenField, expiresInField, description,
		now, now,
	)
	if err != nil {
		writeErrorJSON(w, r, 500, "创建派生Token供应商失败")
		return
	}

	var row subTokenProviderRow
	err = conn.QueryRow(
		`SELECT id, provider_name, base_url, derive_endpoint, derive_method,
			derive_params, response_token_field, expires_in_field, description
		 FROM cloud_sub_token_providers WHERE id = ?`, id,
	).Scan(
		&row.ID, &row.ProviderName, &row.BaseURL, &row.DeriveEndpoint,
		&row.DeriveMethod, &row.DeriveParams, &row.ResponseTokenField,
		&row.ExpiresInField, &row.Description,
	)
	if err != nil {
		writeErrorJSON(w, r, 500, "读取新建供应商失败")
		return
	}

	writeJSON(w, 201, map[string]interface{}{
		"item": row.toItem(),
	})
}

func updateSubTokenProvider(w http.ResponseWriter, r *http.Request) {
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
		`SELECT id FROM cloud_sub_token_providers WHERE id = ?`, id,
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

	if v, ok := body["provider_name"]; ok {
		setClauses = append(setClauses, "provider_name = ?")
		args = append(args, strings.TrimSpace(fmt.Sprintf("%v", v)))
	}
	if v, ok := body["base_url"]; ok {
		setClauses = append(setClauses, "base_url = ?")
		args = append(args, strings.TrimSpace(fmt.Sprintf("%v", v)))
	}
	if v, ok := body["derive_endpoint"]; ok {
		setClauses = append(setClauses, "derive_endpoint = ?")
		args = append(args, strings.TrimSpace(fmt.Sprintf("%v", v)))
	}
	if v, ok := body["derive_method"]; ok {
		setClauses = append(setClauses, "derive_method = ?")
		args = append(args, strings.TrimSpace(fmt.Sprintf("%v", v)))
	}
	if v, ok := body["derive_params"]; ok {
		if b, err := json.Marshal(v); err == nil {
			setClauses = append(setClauses, "derive_params = ?")
			args = append(args, string(b))
		}
	}
	if v, ok := body["response_token_field"]; ok {
		setClauses = append(setClauses, "response_token_field = ?")
		args = append(args, strings.TrimSpace(fmt.Sprintf("%v", v)))
	}
	if v, ok := body["expires_in_field"]; ok {
		setClauses = append(setClauses, "expires_in_field = ?")
		args = append(args, strings.TrimSpace(fmt.Sprintf("%v", v)))
	}
	if v, ok := body["description"]; ok {
		setClauses = append(setClauses, "description = ?")
		args = append(args, strings.TrimSpace(fmt.Sprintf("%v", v)))
	}

	if len(setClauses) == 1 {
		writeErrorJSON(w, r, 400, "没有需要更新的字段")
		return
	}

	args = append(args, id)
	query := fmt.Sprintf(
		"UPDATE cloud_sub_token_providers SET %s WHERE id = ?",
		strings.Join(setClauses, ", "),
	)
	_, err = conn.Exec(query, args...)
	if err != nil {
		writeErrorJSON(w, r, 500, "更新派生Token供应商失败")
		return
	}

	var row subTokenProviderRow
	err = conn.QueryRow(
		`SELECT id, provider_name, base_url, derive_endpoint, derive_method,
			derive_params, response_token_field, expires_in_field, description
		 FROM cloud_sub_token_providers WHERE id = ?`, id,
	).Scan(
		&row.ID, &row.ProviderName, &row.BaseURL, &row.DeriveEndpoint,
		&row.DeriveMethod, &row.DeriveParams, &row.ResponseTokenField,
		&row.ExpiresInField, &row.Description,
	)
	if err != nil {
		writeErrorJSON(w, r, 500, "读取更新后供应商失败")
		return
	}

	writeJSON(w, 200, map[string]interface{}{
		"item": row.toItem(),
	})
}

func deleteSubTokenProvider(w http.ResponseWriter, r *http.Request) {
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
		`SELECT id FROM cloud_sub_token_providers WHERE id = ?`, id,
	).Scan(&existingID)
	if err == sql.ErrNoRows {
		writeErrorJSON(w, r, 404, "供应商不存在")
		return
	}
	if err != nil {
		writeErrorJSON(w, r, 500, "查询供应商失败")
		return
	}

	_, err = conn.Exec(`DELETE FROM cloud_sub_token_providers WHERE id = ?`, id)
	if err != nil {
		writeErrorJSON(w, r, 500, "删除派生Token供应商失败")
		return
	}

	writeJSON(w, 200, map[string]interface{}{
		"status": "deleted",
	})
}
