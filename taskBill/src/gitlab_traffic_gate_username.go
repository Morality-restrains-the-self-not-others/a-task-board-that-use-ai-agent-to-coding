package main

import (
	"database/sql"
	"fmt"
	"strings"

	"log/slog"
)

// gitlab 用户名 → 租户解析（OPT-20260824-079：堵住个人命名空间仓的流量闸门绕过）。
//
// 背景：闸门按 project_path 正则 `^tenant-(\d+)` 归租户，个人命名空间
// （example-user/somanyad）与无 tenant-{id} 前缀的组仓一律 UNMAPPED → fail-open，
// 导致租户流量用尽/未预购时容器仍能公网克隆（容器证据：task_879620507262021632_cmt_879620511582154752）。
//
// 解析链（全部同库 MySQL 跨库只读，taskapp 有全局权限）：
//
//	git_oauth.git_oauth_appusercredential（provider=gitlab:{region}，remote_login=用户名）
//	  → task2app_user_id（平台用户）
//	task_tenant.tenant_company_member（is_active=1）→ 所属租户集合
//	  ∩ billing_tenant_gitlab_resource（region 且 provisioning_status='active'）→ 恰一个租户
//
// 恰好一个租户命中才返回；0/多个/查询失败均 (0, err)，调用方保持 UNMAPPED fail-open，
// 与「taskBill 不可达 fail-open」既有语义一致。测试通过换桩 resolveTenantIDFromGitlabUsername 覆盖。

// resolveTenantIDFromGitlabUsername 是包级钩子，测试可替换为桩。
var resolveTenantIDFromGitlabUsername = resolveTenantIDFromGitlabUsernameImpl

// resolveTenantByUsername 在 project_path 无 tenant-{id} 前缀时尝试归集租户：
// 优先仓所有者（命名空间顶层段），其次认证用户（组仓场景）。
func resolveTenantByUsername(req gitlabTrafficGateRequest, region string) int64 {
	region = strings.TrimSpace(region)
	if region == "" {
		return 0
	}
	tried := map[string]bool{}
	for _, name := range candidateGitlabUsernames(req) {
		if name == "" || tried[name] {
			continue
		}
		tried[name] = true
		tid, err := resolveTenantIDFromGitlabUsername(name, region)
		if err == nil && tid > 0 {
			return tid
		}
		if err != nil {
			slog.Warn("gitlab_traffic_gate_username_unresolved",
				"level", "warn",
				"username", name,
				"region", region,
				"project_path", strings.TrimSpace(req.ProjectPath),
				"error", err.Error(),
			)
		}
	}
	return 0
}

// candidateGitlabUsernames 返回 [仓所有者命名空间, 认证用户名]（去空）。
func candidateGitlabUsernames(req gitlabTrafficGateRequest) []string {
	var names []string
	if p := strings.Trim(strings.TrimSpace(req.ProjectPath), "/"); p != "" {
		if seg := strings.SplitN(p, "/", 2)[0]; seg != "" && !strings.ContainsAny(seg, " \t") {
			names = append(names, seg)
		}
	}
	if u := strings.TrimSpace(req.GitlabUsername); u != "" {
		names = append(names, u)
	}
	return names
}

// resolveTenantIDFromGitlabUsernameImpl 见文件头解析链说明。
func resolveTenantIDFromGitlabUsernameImpl(username, region string) (int64, error) {
	username = strings.TrimSpace(username)
	region = strings.TrimSpace(region)
	if username == "" || region == "" {
		return 0, fmt.Errorf("username/region required")
	}
	if db == nil {
		return 0, fmt.Errorf("db not ready")
	}
	userID, err := gitlabUsernameToPlatformUser(username, region)
	if err != nil {
		return 0, fmt.Errorf("no credential binding for %q on %q: %w", username, region, err)
	}
	return platformUserToTenantOnRegion(userID, region)
}

// gitlabUsernameToPlatformUser 精确匹配 gitlab:{region}（区域 slug == service_provider，
// 如 tencent-sh-1），未命中再按前缀 gitlab:% 取最近绑定（主 GitLab 区域
// tencent-shanghai-5 的 service_provider 是 daydaymoney-gitlab，slug ≠ SP）。
func gitlabUsernameToPlatformUser(username, region string) (string, error) {
	var uid string
	err := db.QueryRow(`
		SELECT task2app_user_id FROM git_oauth.git_oauth_appusercredential
		WHERE provider = ? AND remote_login = ? AND bind_status = 'active'
		ORDER BY updated_at DESC, id DESC LIMIT 1`, "gitlab:"+region, username).Scan(&uid)
	if err == nil {
		return uid, nil
	}
	if err != sql.ErrNoRows {
		return "", err
	}
	err = db.QueryRow(`
		SELECT task2app_user_id FROM git_oauth.git_oauth_appusercredential
		WHERE provider LIKE 'gitlab:%' AND remote_login = ? AND bind_status = 'active'
		ORDER BY updated_at DESC, id DESC LIMIT 1`, username).Scan(&uid)
	if err != nil {
		return "", err
	}
	return uid, nil
}

// platformUserToTenantOnRegion 平台用户 → 在 region 上拥有 active GitLab 资源的租户；
// 命中必须恰一个，多个视为歧义（保持 unresolved → fail-open）。
func platformUserToTenantOnRegion(userID, region string) (int64, error) {
	rows, err := db.Query(`
		SELECT m.company_id
		FROM task_tenant.tenant_company_member m
		JOIN task_bill.billing_tenant_gitlab_resource r
		  ON r.tenant_id = m.company_id
		 AND r.region = ?
		 AND r.provisioning_status = 'active'
		WHERE m.user_id = ? AND m.is_active = 1
		ORDER BY m.company_id`, region, userID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var tids []int64
	for rows.Next() {
		var tid int64
		if err := rows.Scan(&tid); err != nil {
			return 0, err
		}
		tids = append(tids, tid)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if len(tids) != 1 {
		return 0, fmt.Errorf("user %s resolves to %d tenants on region %s (want exactly 1)", userID, len(tids), region)
	}
	return tids[0], nil
}
