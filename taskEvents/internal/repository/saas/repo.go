package saas

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Repository provides Go service clients for event-driven business operations.
// OPT-052: Django HTTP fallbacks removed (saas-backend retired 2026-07-30).
// All methods now call Go services directly.
type Repository struct {
	secret string
	client *http.Client
}

// New builds a Repository for Go service API calls.
func New(secret string) *Repository {
	return &Repository{
		secret: strings.TrimSpace(secret),
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func taskEventsSecret() string {
	return strings.TrimSpace(os.Getenv("TASK_EVENTS_INTERNAL_SECRET"))
}

// Close is a no-op for HTTP client.
func (r *Repository) Close() error { return nil }

func taskTenantServiceURL() string {
	if v := strings.TrimSpace(os.Getenv("TASK_TENANT_SERVICE_INTERNAL_URL")); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "http://127.0.0.1:8020"
}

func tenantServiceSecret() string {
	if v := strings.TrimSpace(os.Getenv("TENANT_SERVICE_INTERNAL_SECRET")); v != "" {
		return v
	}
	return ""
}

// CompanyByCreator returns existing company id and name if any.
// Calls taskTenantService directly (OPT-052: Django fallback removed — saas-backend retired 2026-07-30).
func (r *Repository) CompanyByCreator(creatorID string) (companyID int64, name string, ok bool, err error) {
	creatorID = strings.TrimSpace(creatorID)
	if creatorID == "" {
		return 0, "", false, nil
	}
	return r.companyByCreatorDirect(creatorID)
}

// companyByCreatorDirect calls taskTenantService /api/internal/tenant/companies/by-creator.
func (r *Repository) companyByCreatorDirect(creatorID string) (int64, string, bool, error) {
	tenantURL := taskTenantServiceURL()
	req, err := http.NewRequest(http.MethodGet,
		tenantURL+"/api/internal/tenant/companies/by-creator?creator_id="+creatorID, nil)
	if err != nil {
		return 0, "", false, err
	}
	req.Header.Set("X-Internal-Secret", tenantServiceSecret())
	resp, err := r.client.Do(req)
	if err != nil {
		return 0, "", false, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return 0, "", false, fmt.Errorf("taskTenant by-creator status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var list []map[string]interface{}
	if err := json.Unmarshal(raw, &list); err != nil {
		return 0, "", false, err
	}
	if len(list) == 0 {
		return 0, "", false, nil
	}
	first := list[0]
	id, err := asInt64(first["id"])
	if err != nil {
		return 0, "", false, err
	}
	n, _ := first["name"].(string)
	return id, n, true, nil
}

// CreatedCompany is the result of creating company + admin member.
type CreatedCompany struct {
	CompanyID int64
	Name      string
	CreatorID string
	CreatedAt time.Time
}

// CreateCompanyForUser creates company and admin member when none exists for creator.
// Calls taskTenantService directly (OPT-052: Django fallback removed — saas-backend retired 2026-07-30).
func (r *Repository) CreateCompanyForUser(userID string, username string) (*CreatedCompany, error) {
	return r.createCompanyDirect(userID, username)
}

// createCompanyDirect calls taskTenantService /api/internal/tenant/companies/upsert
// and then creates the creator as an admin company member.
func (r *Repository) createCompanyDirect(userID, username string) (*CreatedCompany, error) {
	nickname := r.fetchPersonalNickname(userID)
	companyName := ResolvePersonalCompanyName(username, nickname)
	log.Printf("[create_company_direct] event=company_name_resolved user_id=%s company_name=%s", userID, companyName)
	body := map[string]string{
		"name":       companyName,
		"creator_id": userID,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	tenantURL := taskTenantServiceURL()
	req, err := http.NewRequest(http.MethodPost,
		tenantURL+"/api/internal/tenant/companies/upsert", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Secret", tenantServiceSecret())
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respRaw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("taskTenant upsert status %d: %s", resp.StatusCode, strings.TrimSpace(string(respRaw)))
	}
	var result map[string]interface{}
	if err := json.Unmarshal(respRaw, &result); err != nil {
		return nil, err
	}
	companyIDStr := fmt.Sprint(result["id"])
	id, err := asInt64(result["id"])
	if err != nil {
		return nil, err
	}
	n, _ := result["name"].(string)
	cid, _ := result["creator_id"].(string)
	createdAt := time.Now().UTC()
	if s, ok := result["created_at"].(string); ok && s != "" {
		if t, e := time.Parse("2006-01-02T15:04:05", s); e == nil {
			createdAt = t
		}
	}
	// Create the creator as an admin company member.
	// The public onboarding API does this; the event-driven path must too.
	// Without this row, requireTenantMember checks fail for the new user
	// (「用户有公司无成员、权限全缺」且此前日志无法定位 — OPT-20260806-013)。
	// member_name 必须是个人昵称，绝不能写公司名（曾把「我的公司」写入成员列）。
	memberName := nickname
	if err := r.createCompanyMember(companyIDStr, userID, memberName, true); err != nil {
		log.Printf("[create_company_direct] event=create_company_member_failed company_id=%s user_id=%s err=%v",
			companyIDStr, userID, err)
		return nil, fmt.Errorf("company created but member row failed (company_id=%s user_id=%s): %w", companyIDStr, userID, err)
	}
	return &CreatedCompany{
		CompanyID: id,
		Name:      n,
		CreatorID: cid,
		CreatedAt: createdAt,
	}, nil
}

// UpsertUserProfileUsername upserts the username display cache.
// Calls taskAuth directly (OPT-052: Django fallback removed — saas-backend retired 2026-07-30).
func (r *Repository) UpsertUserProfileUsername(userID, username string) error {
	return r.upsertUserProfileDirect(userID, username)
}

// upsertUserProfileDirect calls taskAuth POST /api/internal/users/{user_id}/profile/.
func (r *Repository) upsertUserProfileDirect(userID, username string) error {
	taskAuthURL := strings.TrimSpace(os.Getenv("TASK_AUTH_INTERNAL_URL"))
	if taskAuthURL == "" {
		taskAuthURL = "http://127.0.0.1:8003"
	}
	taskAuthURL = strings.TrimRight(taskAuthURL, "/")
	body := map[string]string{"username": username}
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	url := taskAuthURL + "/api/internal/users/id/" + userID + "/profile/"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	sec := r.secret
	if sec == "" {
		sec = taskEventsSecret()
	}
	if sec != "" {
		req.Header.Set("X-TaskAuth-Internal-Secret", sec)
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respRaw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("taskAuth upsert-profile user=%s status %d: %s", userID, resp.StatusCode, strings.TrimSpace(string(respRaw)))
	}
	return nil
}

// UserPlatformRoles returns the user's platform-level roles (super_admin/employee).
// Calls taskAuth GET /api/internal/users/id/{uid}/platform-roles/.
// Errors are surfaced to callers: a failed lookup must NOT silently proceed
// with default behavior (e.g. auto-creating a company for a platform user).
func (r *Repository) UserPlatformRoles(userID string) ([]string, error) {
	taskAuthURL := strings.TrimSpace(os.Getenv("TASK_AUTH_INTERNAL_URL"))
	if taskAuthURL == "" {
		taskAuthURL = "http://127.0.0.1:8003"
	}
	taskAuthURL = strings.TrimRight(taskAuthURL, "/")
	url := taskAuthURL + "/api/internal/users/id/" + url.PathEscape(userID) + "/platform-roles/"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	sec := r.secret
	if sec == "" {
		sec = taskEventsSecret()
	}
	if sec != "" {
		req.Header.Set("X-TaskAuth-Internal-Secret", sec)
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("taskAuth platform-roles user=%s status %d: %s", userID, resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var out struct {
		Roles []string `json:"roles"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if out.Roles == nil {
		return []string{}, nil
	}
	return out.Roles, nil
}

// markUserTenantDirect calls taskAuth PATCH directly (no Django hop).
// Replaces the Django intent mark-user-tenant/ which just forwarded to taskAuth.
func (r *Repository) markUserTenantDirect(userID string) error {
	taskAuthURL := strings.TrimSpace(os.Getenv("TASK_AUTH_INTERNAL_URL"))
	if taskAuthURL == "" {
		taskAuthURL = "http://127.0.0.1:8003"
	}
	taskAuthURL = strings.TrimRight(taskAuthURL, "/")
	url := taskAuthURL + "/api/internal/users/id/" + userID + "/"
	body := map[string]bool{"is_tenant": true}
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	sec := r.secret
	if sec == "" {
		sec = taskEventsSecret()
	}
	if sec != "" {
		req.Header.Set("X-TaskAuth-Internal-Secret", sec)
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		respRaw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("taskAuth PATCH %s status %d: %s", userID, resp.StatusCode, strings.TrimSpace(string(respRaw)))
	}
	return nil
}

// grantInitialResourcesDirect calls taskBill directly for admin resource grants (no Django hop).
// Replaces the Django intent grant-initial-resources/ which called billing_bridge → taskBill.
func (r *Repository) grantInitialResourcesDirect(companyID, creatorID string) (map[string]interface{}, error) {
	taskBillURL := strings.TrimSpace(os.Getenv("TASK_BILL_INTERNAL_URL"))
	if taskBillURL == "" {
		taskBillURL = "http://127.0.0.1:8004"
	}
	taskBillURL = strings.TrimRight(taskBillURL, "/")

	// Resolve taskBill secret via fallback chain:
	//   TASK_BILL_INTERNAL_SECRET > SHARED_INTERNAL_SECRET > domain-events internalSecret > TASK_EVENTS_INTERNAL_SECRET
	sec := strings.TrimSpace(os.Getenv("TASK_BILL_INTERNAL_SECRET"))
	if sec == "" {
		sec = strings.TrimSpace(os.Getenv("SHARED_INTERNAL_SECRET"))
	}
	if sec == "" {
		sec = r.secret
	}
	if sec == "" {
		sec = taskEventsSecret()
	}
	grants := []map[string]interface{}{
		{
			"resource_type": "task_post",
			"quantity":      100,
			"expires_at":    time.Now().UTC().Add(180 * 24 * time.Hour).Format("2006-01-02 15:04:05.000000"),
			"reason":        "新用户礼包",
		},
	}
	body := map[string]interface{}{
		"tenant_id":       companyID,
		"grants":          grants,
		"user_id":         creatorID,
		"idempotency_key": "new_user_gift:" + companyID,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	url := taskBillURL + "/api/internal/taskbill/admin-grant-resources/"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if sec != "" {
		req.Header.Set("X-TaskBill-Internal-Secret", sec)
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respRaw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("taskBill admin-grant-resources status %d: %s", resp.StatusCode, strings.TrimSpace(string(respRaw)))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respRaw, &result); err != nil {
		result = map[string]interface{}{"ok": true}
	}
	return result, nil
}

func asInt64(v interface{}) (int64, error) {
	switch n := v.(type) {
	case float64:
		return int64(n), nil
	case int64:
		return n, nil
	case int:
		return int64(n), nil
	case json.Number:
		return n.Int64()
	case string:
		var out int64
		_, err := fmt.Sscan(n, &out)
		return out, err
	default:
		var out int64
		_, err := fmt.Sscan(fmt.Sprint(v), &out)
		return out, err
	}
}
