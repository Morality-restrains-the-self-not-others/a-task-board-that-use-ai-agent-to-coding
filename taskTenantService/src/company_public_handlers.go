package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"snowflake"
	"strings"
)

// handleCompaniesRoute serves public (gateway-authenticated) company endpoints under
// /api/tenant/<tid>/accounts/companies.  Members and groups were already migrated;
// this adds the company read/upsert surface to complete OPT-044.
func handleCompaniesRoute(w http.ResponseWriter, r *http.Request, tenantID string, rest []string) {
	userID, ok := requireAuthUser(w, r)
	if !ok {
		return
	}

	sub := strings.Trim(strings.Join(rest, "/"), "/")

	switch {
	case r.Method == http.MethodPost && sub == "":
		// POST /api/tenant/<tid>/accounts/companies — create a company (onboarding)
		body, err := readJSONBody(r)
		if err != nil {
			writeError(w, r, 400, "invalid json")
			return
		}
		name := strings.TrimSpace(strField(body, "name"))
		if name == "" {
			writeError(w, r, 400, "公司名称不能为空")
			return
		}
		companyID := snowflake.GenerateIDString()
		if err := upsertCompanyRow(companyID, name, userID, ""); err != nil {
			writeError(w, r, 500, "创建公司失败: "+err.Error())
			return
		}
		// Add creator as admin member.
		// member_name 必须是租户个人昵称，绝不能写公司名（曾把「我的公司」写进成员列）。
		memberName := ""
		if nicks := fetchPersonalNicknamesFn([]string{userID}); nicks != nil {
			memberName = strings.TrimSpace(nicks[userID])
		}
		memberID := snowflake.GenerateIDString()
		if _, err := db.Exec(
			`INSERT INTO tenant_company_member
				(id, user_id, company_id, is_admin, is_active, workspace_id, member_name)
				VALUES (?,?,?,1,1,'',?)`,
			memberID, userID, companyID, memberName,
		); err != nil {
			// Log but don't fail — company exists, member can be fixed later
			logInfo("create company member failed: "+err.Error(), "company_id="+companyID)
		} else {
			// v63 硬切换: 创建者 → tenant_admin 角色行（判定不再读 is_admin 列）
			ensureMemberRole(memberID, "tenant_admin", companyID, "system")
			// v77: 创建者成员行 → MEMBER_JOINED（自动默认 Git 身份）
			go publishMemberJoined(memberID, userID, companyID, memberName, "admin", "", "company_create", nil)
		}
		// 事件失效: 创建者 PDP 缓存带 rev 快照，此处 incr 使 taskAuth 侧
		// 立即失效新公司权限缓存（否则新建公司后最多延迟标准 TTL 30s）
		incrMembershipRev(userID)
		writeJSON(w, 201, map[string]interface{}{
			"company_id": companyID,
			"name":       name,
		})

		// Publish COMPANY_CREATED event so downstream intents create
		// the default workspace, deliverable system, progress system, etc.
		go publishEvent("COMPANY_CREATED", map[string]interface{}{
			"company_id":   companyID,
			"company_name": name,
			"creator_id":   userID,
		})

		// Mark creator as tenant immediately on company creation (no payment gate).
		// Previously is_tenant=true was only set on billing recharge; now company
		// creation alone qualifies a user as a tenant.
		go markUserTenant(userID)

	case r.Method == http.MethodPatch && sub == "":
		// PATCH /api/tenant/<tid>/accounts/companies — update company name
		body, err := readJSONBody(r)
		if err != nil {
			writeError(w, r, 400, "invalid json")
			return
		}
		cid := strings.TrimSpace(strField(body, "company_id"))
		name := strings.TrimSpace(strField(body, "name"))
		if cid == "" || name == "" {
			writeError(w, r, 400, "company_id and name required")
			return
		}
		// Verify the requester is admin of the company
		if isAdmin, _ := checkCompanyAdmin(r, userID, cid); !isAdmin {
			writeError(w, r, 403, "仅公司管理员可修改公司名称")
			return
		}
		if err := updateCompanyName(cid, name); err != nil {
			writeError(w, r, 500, "更新公司名称失败: "+err.Error())
			return
		}
		writeJSON(w, 200, map[string]interface{}{
			"company_id": cid,
			"name":       name,
		})

	case r.Method == http.MethodGet && sub == "current":
		// GET /api/tenant/<tid>/accounts/companies/current — return current company
		// info with admin/creator flags for the TenantCompanySettings page.
		c, err := getCompanyByID(tenantID)
		if err != nil {
			writeError(w, r, 500, err.Error())
			return
		}
		if c == nil {
			writeError(w, r, 404, "company not found")
			return
		}
		isAdmin, _ := checkCompanyAdmin(r, userID, tenantID)
		isCreatorFlag := isCreator(tenantID, userID)
		writeJSON(w, 200, map[string]interface{}{
			"name":              c.Name,
			"member_is_admin":   isAdmin,
			"member_is_creator": isCreatorFlag,
		})

	case r.Method == http.MethodPatch && sub == "current":
		// PATCH /api/tenant/<tid>/accounts/companies/current — update current company name
		body, err := readJSONBody(r)
		if err != nil {
			writeError(w, r, 400, "invalid json")
			return
		}
		name := strings.TrimSpace(strField(body, "name"))
		if name == "" {
			writeError(w, r, 400, "公司名称不能为空")
			return
		}
		// Verify the requester is admin or creator of the company
		if isAdmin, _ := checkCompanyAdmin(r, userID, tenantID); !isAdmin {
			writeError(w, r, 403, "仅公司管理员可修改公司名称")
			return
		}
		if err := updateCompanyName(tenantID, name); err != nil {
			writeError(w, r, 500, "更新公司名称失败: "+err.Error())
			return
		}
		writeJSON(w, 200, map[string]interface{}{
			"name": name,
		})

	case r.Method == http.MethodGet && sub == "":
		// GET /api/tenant/<tid>/accounts/companies?company_id=...
		cid := strings.TrimSpace(r.URL.Query().Get("company_id"))
		if cid == "" {
			writeError(w, r, 400, "company_id required")
			return
		}
		c, err := getCompanyByID(cid)
		if err != nil {
			writeError(w, r, 500, err.Error())
			return
		}
		if c == nil {
			writeError(w, r, 404, "not found")
			return
		}
		writeJSON(w, 200, companyToJSON(c))

	case r.Method == http.MethodGet && sub == "by-name":
		name := strings.TrimSpace(r.URL.Query().Get("name"))
		if name == "" {
			writeError(w, r, 400, "name required")
			return
		}
		c, err := getCompanyByName(name)
		if err != nil {
			writeError(w, r, 500, err.Error())
			return
		}
		if c == nil {
			writeError(w, r, 404, "not found")
			return
		}
		writeJSON(w, 200, companyToJSON(c))

	case r.Method == http.MethodGet && sub == "by-creator":
		uid := strings.TrimSpace(r.URL.Query().Get("creator_id"))
		if uid == "" {
			writeError(w, r, 400, "creator_id required")
			return
		}
		list, err := listCompaniesByCreator(uid)
		if err != nil {
			writeError(w, r, 500, err.Error())
			return
		}
		items := make([]map[string]interface{}, 0, len(list))
		for i := range list {
			items = append(items, companyToJSON(&list[i]))
		}
		writeJSON(w, 200, items)

	case r.Method == http.MethodGet && sub == "name-taken":
		taken, err := nameTaken(r.URL.Query().Get("name"), r.URL.Query().Get("exclude_id"))
		if err != nil {
			writeError(w, r, 500, err.Error())
			return
		}
		writeJSON(w, 200, map[string]interface{}{"taken": taken})

	default:
		writeError(w, r, 404, "not found")
	}
}

// markUserTenant calls taskAuth PATCH to set is_tenant=true on the user.
// This replaces the old billing/recharge gate — company creation alone now
// qualifies a user as a tenant. Fire-and-forget (best-effort, logged).
func markUserTenant(userID string) {
	taskAuthURL := cfg.TaskAuthURL
	if taskAuthURL == "" {
		taskAuthURL = "http://127.0.0.1:8003"
	}
	body := map[string]bool{"is_tenant": true}
	raw, err := json.Marshal(body)
	if err != nil {
		logInfo("markUserTenant marshal failed user_id="+userID+" err="+err.Error(), "")
		return
	}
	url := taskAuthURL + "/api/internal/users/id/" + userID + "/"
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(raw))
	if err != nil {
		logInfo("markUserTenant new request failed user_id="+userID+" err="+err.Error(), "")
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.InternalSecret != "" {
		req.Header.Set("X-TaskAuth-Internal-Secret", cfg.InternalSecret)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logInfo("markUserTenant call failed user_id="+userID+" err="+err.Error(), "")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		logInfo("markUserTenant non-OK user_id="+userID+" status="+itoa(resp.StatusCode), "")
		return
	}
	logInfo("markUserTenant ok user_id="+userID, "")
}
