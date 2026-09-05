package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"snowflake"
)

// saasHTTPStore holds in-memory saas membership/git data for unit tests.
// Feature-params tables live in local task_cloud.db (see putTenant/putWorkspace/putPersonal).

type saasHTTPStore struct {
	mu sync.Mutex

	members      map[string]memberStoreRow // key: tenantID+"|"+userID
	gitIDs       map[string]gitIDStoreRow
	creators     map[string]string          // companyID -> creator userID
	groupMembers map[string]map[string]bool // groupID -> set of userIDs
}

type memberStoreRow struct {
	ID      string
	IsAdmin bool
	UserID  string
}

type gitIDStoreRow struct {
	UserID       string
	CompanyID    string
	GitUserName  string
	GitUserEmail string
}

func newSaasHTTPStore() *saasHTTPStore {
	return &saasHTTPStore{
		members:      map[string]memberStoreRow{},
		gitIDs:       map[string]gitIDStoreRow{},
		creators:     map[string]string{},
		groupMembers: map[string]map[string]bool{},
	}
}

func (s *saasHTTPStore) putTenant(row *tenantFeatureParamsRow) {
	if row == nil || db == nil {
		return
	}
	now := fpNowUTC()
	llm := 0
	if row.LLMBudgetEnabled {
		llm = 1
	}
	id := row.ID
	if id == "" {
		id = snowflake.GenerateIDString()
	}
	_, _ = db.Exec(`
		INSERT INTO cloud_tenant_feature_params (
			id, company_id, providers, agent_model, agent_model_provider, agent_max_steps,
			summary_model, summary_model_provider, llm_budget_enabled, extra_env_vars, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			providers = VALUES(providers), agent_model = VALUES(agent_model),
			agent_model_provider = VALUES(agent_model_provider), agent_max_steps = VALUES(agent_max_steps),
			summary_model = VALUES(summary_model), summary_model_provider = VALUES(summary_model_provider),
			llm_budget_enabled = VALUES(llm_budget_enabled), extra_env_vars = VALUES(extra_env_vars),
			updated_at = VALUES(updated_at)`,
		id, row.CompanyID, row.ProvidersJSON, row.AgentModel, row.AgentProvider, row.AgentMaxSteps,
		row.SummaryModel, row.SummaryProvider, llm, row.ExtraEnvJSON, now, now,
	)
}

func (s *saasHTTPStore) putWorkspace(workspaceID string, useCompany bool, cfgRow *featureParamsConfig) {
	s.putWorkspaceForCompany("c1", workspaceID, useCompany, cfgRow)
}

func (s *saasHTTPStore) putWorkspaceForCompany(companyID, workspaceID string, useCompany bool, cfgRow *featureParamsConfig) {
	if db == nil {
		return
	}
	if cfgRow == nil {
		cfgRow = &featureParamsConfig{ProvidersJSON: "[]", ExtraEnvJSON: "[]", AgentMaxSteps: 200}
	}
	now := fpNowUTC()
	id := cfgRow.ID
	if id == "" {
		id = snowflake.GenerateIDString()
	}
	useCo := 0
	if useCompany {
		useCo = 1
	}
	_, _ = db.Exec(`
		INSERT INTO cloud_workspace_feature_params (
			id, workspace_id, company_id, use_company_default, providers, agent_model,
			agent_model_provider, agent_max_steps, summary_model, summary_model_provider,
			llm_budget_enabled, extra_env_vars, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			company_id = VALUES(company_id), use_company_default = VALUES(use_company_default),
			providers = VALUES(providers), agent_model = VALUES(agent_model),
			agent_model_provider = VALUES(agent_model_provider), agent_max_steps = VALUES(agent_max_steps),
			summary_model = VALUES(summary_model), summary_model_provider = VALUES(summary_model_provider),
			extra_env_vars = VALUES(extra_env_vars), updated_at = VALUES(updated_at)`,
		id, workspaceID, companyID, useCo, cfgRow.ProvidersJSON, cfgRow.AgentModel, cfgRow.AgentProvider,
		cfgRow.AgentMaxSteps, cfgRow.SummaryModel, cfgRow.SummaryProvider, cfgRow.ExtraEnvJSON, now, now,
	)
}

func (s *saasHTTPStore) putPersonal(cfgRow *featureParamsConfig, userID string) {
	s.putPersonalForCompany("c1", cfgRow, userID)
}

func (s *saasHTTPStore) putPersonalForCompany(companyID string, cfgRow *featureParamsConfig, userID string) {
	if cfgRow == nil || db == nil {
		return
	}
	name := cfgRow.DisplayName
	if strings.HasPrefix(name, "个人配置:") {
		name = strings.TrimPrefix(name, "个人配置:")
	}
	if name == "" {
		name = "default"
	}
	now := fpNowUTC()
	id := cfgRow.ID
	if id == "" {
		id = snowflake.GenerateIDString()
	}
	_, _ = db.Exec(`
		INSERT INTO cloud_personal_feature_params_config (
			id, user_id, company_id, name, providers, agent_model, agent_model_provider,
			agent_max_steps, summary_model, summary_model_provider, llm_budget_enabled,
			extra_env_vars, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			id = VALUES(id), providers = VALUES(providers), agent_model = VALUES(agent_model),
			agent_model_provider = VALUES(agent_model_provider), agent_max_steps = VALUES(agent_max_steps),
			summary_model = VALUES(summary_model), summary_model_provider = VALUES(summary_model_provider),
			extra_env_vars = VALUES(extra_env_vars), updated_at = VALUES(updated_at)`,
		id, userID, companyID, name, cfgRow.ProvidersJSON, cfgRow.AgentModel, cfgRow.AgentProvider,
		cfgRow.AgentMaxSteps, cfgRow.SummaryModel, cfgRow.SummaryProvider, cfgRow.ExtraEnvJSON, now, now,
	)
}

func (s *saasHTTPStore) putMember(tenantID, userID, memberID string, isAdmin bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.members[tenantID+"|"+userID] = memberStoreRow{ID: memberID, IsAdmin: isAdmin, UserID: userID}
	s.members["byid|"+tenantID+"|"+memberID] = memberStoreRow{ID: memberID, IsAdmin: isAdmin, UserID: userID}
}

func (s *saasHTTPStore) putCreator(companyID, userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.creators[companyID] = userID
}

func (s *saasHTTPStore) putGroupMember(groupID, userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.groupMembers[groupID] == nil {
		s.groupMembers[groupID] = map[string]bool{}
	}
	s.groupMembers[groupID][userID] = true
}

func (s *saasHTTPStore) putGitIdentity(id, userID, companyID, name, email string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gitIDs[id] = gitIDStoreRow{
		UserID: userID, CompanyID: companyID, GitUserName: name, GitUserEmail: email,
	}
}

// storeLookupGitIdentity implements fetchUserCompanyGitIdentityFn using the
// in-memory store, avoiding cross-database queries in unit tests.
func storeLookupGitIdentity(s *saasHTTPStore, identityID, userID, companyID string) (*userCompanyGitIdentityRow, error) {
	s.mu.Lock()
	row, ok := s.gitIDs[identityID]
	s.mu.Unlock()
	if !ok {
		return nil, nil
	}
	if row.UserID != userID {
		return nil, nil
	}
	if row.CompanyID != "" && row.CompanyID != companyID {
		return nil, nil
	}
	return &userCompanyGitIdentityRow{
		GitUserName:  row.GitUserName,
		GitUserEmail: row.GitUserEmail,
	}, nil
}

func (s *saasHTTPStore) snapshotCount(taskID string) int {
	n, err := countFeatureParamsSnapshots(taskID)
	if err != nil {
		return 0
	}
	return n
}

func (s *saasHTTPStore) apiKeyUsageCount(taskID string) int {
	n, err := countTaskApiKeyUsage(taskID)
	if err != nil {
		return 0
	}
	return n
}

func (s *saasHTTPStore) apiKeyUsageRows(taskID string) []map[string]any {
	if db == nil {
		return nil
	}
	rows, err := db.Query(`
		SELECT id, task_id, workspace_id, tenant_id, provider_name, api_key_hash, key_type
		FROM cloud_task_api_key_usage WHERE task_id = ?`, taskID,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, tid, wid, tenant, provider, hash, keyType string
		if err := rows.Scan(&id, &tid, &wid, &tenant, &provider, &hash, &keyType); err != nil {
			continue
		}
		out = append(out, map[string]any{
			"id": id, "task_id": tid, "workspace_id": wid, "tenant_id": tenant,
			"provider_name": provider, "api_key_hash": hash, "key_type": keyType,
		})
	}
	return out
}

func startSaasInternalMock(t *testing.T, store *saasHTTPStore) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case path == "/api/internal/tenant/companies/creator" || path == "/api/internal/tenant/companies/company-creator":
			companyID := strings.TrimSpace(r.URL.Query().Get("company_id"))
			store.mu.Lock()
			creatorID, known := store.creators[companyID]
			hasAnyMember := known
			if !hasAnyMember {
				for k := range store.members {
					if strings.HasPrefix(k, companyID+"|") {
						hasAnyMember = true
						break
					}
				}
			}
			store.mu.Unlock()
			if !hasAnyMember {
				writeJSON(w, 200, map[string]any{"found": false})
				return
			}
			writeJSON(w, 200, map[string]any{"found": true, "creator_id": creatorID})
		case strings.HasPrefix(path, "/api/internal/feature-params/resolve-owner-user"):
			body, _ := io.ReadAll(r.Body)
			var req map[string]any
			_ = json.Unmarshal(body, &req)
			ownerKey := strField(req, "owner_key")
			tenantID := strField(req, "tenant_id")
			if db != nil && ownerKey != "" {
				var n int
				_ = db.QueryRow(
					`SELECT COUNT(*) FROM cloud_personal_feature_params_config WHERE user_id = ?`, ownerKey,
				).Scan(&n)
				if n > 0 {
					writeJSON(w, 200, map[string]any{"user_id": ownerKey})
					return
				}
			}
			store.mu.Lock()
			if m, ok := store.members["byid|"+tenantID+"|"+ownerKey]; ok {
				store.mu.Unlock()
				writeJSON(w, 200, map[string]any{"user_id": m.UserID})
				return
			}
			store.mu.Unlock()
			writeJSON(w, 200, map[string]any{"user_id": ownerKey})
		case strings.HasPrefix(path, "/api/internal/taskproject/resolve-user-member"):
			body, _ := io.ReadAll(r.Body)
			var req map[string]any
			_ = json.Unmarshal(body, &req)
			tenantID := strField(req, "tenant_id")
			userID := strField(req, "user_id")
			key := tenantID + "|" + userID
			store.mu.Lock()
			m, ok := store.members[key]
			creatorID := store.creators[tenantID]
			store.mu.Unlock()
			if !ok {
				writeJSON(w, 404, map[string]any{"error": "member not found"})
				return
			}
			writeJSON(w, 200, map[string]any{
				"company_member_id": m.ID,
				"user_id":           m.UserID,
				"is_admin":          m.IsAdmin,
				"is_creator":        creatorID != "" && creatorID == userID,
			})
		case strings.HasPrefix(path, "/api/internal/taskproject/user-in-group"):
			body, _ := io.ReadAll(r.Body)
			var req map[string]any
			_ = json.Unmarshal(body, &req)
			groupID := strField(req, "group_id")
			userID := strField(req, "user_id")
			store.mu.Lock()
			inGroup := store.groupMembers[groupID] != nil && store.groupMembers[groupID][userID]
			store.mu.Unlock()
			writeJSON(w, 200, map[string]any{"in_group": inGroup})
		case strings.HasPrefix(path, "/api/internal/tenant/members/resolve"):
				companyID := strings.TrimSpace(r.URL.Query().Get("company_id"))
				userID := strings.TrimSpace(r.URL.Query().Get("user_id"))
				key := companyID + "|" + userID
				store.mu.Lock()
				m, ok := store.members[key]
				store.mu.Unlock()
				if !ok {
					writeJSON(w, 404, map[string]any{"detail": "member not found"})
					return
				}
				writeJSON(w, 200, map[string]any{
					"id":       m.ID,
					"user_id":  m.UserID,
					"is_admin": m.IsAdmin,
				})
			case strings.HasPrefix(path, "/api/internal/tenant/members/by-id"):
				memberID := strings.TrimSpace(r.URL.Query().Get("member_id"))
				cid := strings.TrimSpace(r.URL.Query().Get("company_id"))
				store.mu.Lock()
				byidKey := "byid|" + cid + "|" + memberID
				m, ok := store.members[byidKey]
				store.mu.Unlock()
				if !ok {
					writeJSON(w, 404, map[string]any{"detail": "member not found"})
					return
				}
				writeJSON(w, 200, map[string]any{
					"id":       m.ID,
					"user_id":  m.UserID,
					"is_admin": m.IsAdmin,
				})
			case strings.HasPrefix(path, "/api/internal/git-identities/lookup"):
			body, _ := io.ReadAll(r.Body)
			var req map[string]any
			_ = json.Unmarshal(body, &req)
			ids, _ := req["ids"].([]any)
			idents := []map[string]any{}
			store.mu.Lock()
			for _, raw := range ids {
				id := strings.TrimSpace(fmt.Sprintf("%v", raw))
				row, ok := store.gitIDs[id]
				if !ok {
					continue
				}
				idents = append(idents, map[string]any{
					"id": id, "user_id": row.UserID, "company_id": row.CompanyID,
					"git_user_name": row.GitUserName, "git_user_email": row.GitUserEmail,
				})
			}
			store.mu.Unlock()
			writeJSON(w, 200, map[string]any{"identities": idents})
		default:
			writeJSON(w, 404, map[string]any{"detail": "not found", "path": path})
		}
	}))
	t.Cleanup(srv.Close)
	prevTenant := cfg.TaskTenantURL
	cfg.TaskTenantURL = srv.URL
	t.Cleanup(func() {
		cfg.TaskTenantURL = prevTenant
	})
	return srv
}

var testSaasStore *saasHTTPStore

func setupBudgetTestDB(t *testing.T) *saasHTTPStore {
	t.Helper()
	setupCloudTestDB(t)
	if err := setupTestBudgetDB(t); err != nil {
		t.Fatalf("open budget: %v", err)
	}
	t.Cleanup(closeBudgetDB)
	if err := ensureBudgetLedgerSchema(getBudgetDB()); err != nil {
		t.Fatalf("budget schema: %v", err)
	}
	store := newSaasHTTPStore()
	startSaasInternalMock(t, store)
	testSaasStore = store
	t.Cleanup(func() { testSaasStore = nil })
	return store
}

func seedCompanyFP(t *testing.T, companyID, agentModel string) {
	t.Helper()
	if testSaasStore == nil {
		t.Fatal("testSaasStore nil; call setupBudgetTestDB first")
	}
	testSaasStore.putTenant(&tenantFeatureParamsRow{
		ID: "1", CompanyID: companyID, ProvidersJSON: "[]",
		AgentModel: agentModel, AgentProvider: "p1", AgentMaxSteps: 200, ExtraEnvJSON: "[]",
	})
}

func seedBudgetTenant(t *testing.T, companyID string) {
	t.Helper()
	if testSaasStore == nil {
		t.Fatal("testSaasStore nil")
	}
	providers := `[{"provider":"openai","api_key":"sk","base_url":"https://api.openai.com/v1","supported_models":["gpt-4.1"],"use_sub_token":true,"budget_enabled":true}]`
	testSaasStore.putTenant(&tenantFeatureParamsRow{
		ID: "1", CompanyID: companyID, ProvidersJSON: providers, LLMBudgetEnabled: true,
		AgentModel: "gpt-4.1", AgentProvider: "openai", AgentMaxSteps: 200, ExtraEnvJSON: "[]",
	})
	_, err := getBudgetDB().Exec(`
		INSERT INTO cloud_workspace_model_budget_default
		(id, workspace_id, company_id, provider, base_url, model_name, input_price_per_1m, output_price_per_1m, budget_limit)
		VALUES (1, 'w1', ?, 'openai', 'https://api.openai.com/v1', 'gpt-4.1', '18', '72', '50')
	`, companyID)
	if err != nil {
		t.Fatalf("seed default: %v", err)
	}
}

func countSnapshots(t *testing.T, taskID string) int {
	t.Helper()
	if testSaasStore == nil {
		return 0
	}
	return testSaasStore.snapshotCount(taskID)
}
