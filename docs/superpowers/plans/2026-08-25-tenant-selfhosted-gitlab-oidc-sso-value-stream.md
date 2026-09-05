# 价值流：租户自建 GitLab 平台 OIDC SSO

- **日期**: 2026-08-25
- **设计**: `docs/superpowers/specs/2026-08-25-tenant-selfhosted-gitlab-oidc-sso-design.md`
- **权限**: `docs/superpowers/specs/2026-08-25-tenant-selfhosted-gitlab-oidc-sso-permission-analysis.md`

Mapping the approved design into a value stream.

## 受影响的既有流

- `租户设置 · GitLab 资源购买与自建连接`（TGR / 链路 A）— 本流在同一页增加 SSO 区块，不替换 OAuth Application。
- 平台 gitService OIDC（ADR-0016）— **对照物**，client 前缀与区域闸门不得复用。

## 增量切片（按用户价值排序）

| # | 增量 | 用户可感知价值 | 范围 |
|---|------|----------------|------|
| V1 | 成员闸门 | 非成员无法用平台账号进入客户 GitLab | `handleOidcAuthorize` + `gitlab-tenant-*` |
| V2 | 签发 client | 管理员拿到 client_id / 一次性 secret / OmniAuth 片段 | GET/PUT SSO API + DDL |
| V3 | 轮换/吊销 | 泄露后可换密或关闭 | POST rotate / DELETE + 事件 |
| V4 | 设置页 | 同页完成配置与链路 A 帮助 | taskFE + Path A copy |

最小可交付：**V1+V2+V3+V4 同一次交付**（无闸门的签发不可上线；无 UI 则管理员无法落地 OmniAuth）。

## YAML 字段

- `task-auth.auth_oidc_client.client_id`
- `task-auth.auth_oidc_client.managed_by`
- `task-auth.auth_oidc_client.owner_company_id`
- `task-auth.auth_oidc_client.purpose`
- `task-auth.auth_oidc_client.redirect_uris`

## 测试文件

- `taskAuth/domain/tenant_gitlab_oidc_sso_test.go`
- `taskAuth/src/oidc_tenant_sso_gate_test.go`
- `taskAuth/src/handlers_tenant_gitlab_oidc_sso_test.go`
- `taskFE/app/src/views/WorkspaceSettingsGitlabOidcSso.test.js`
- `taskFE/app/src/views/WorkspaceSettingsGitlabConnection.test.js`

## 测试点（同步 value-stream-test-integration.wsd）

- TP-sso-member-code / TP-sso-nonmember-denied / TP-sso-superuser-nonmember-denied
- TP-sso-region-gate-skipped
- TP-sso-issue-secret-once / TP-sso-get-no-secret / TP-sso-rotate / TP-sso-disable
- TP-sso-base-url-mismatch-400 / TP-sso-http-redirect-400
- TP-sso-fe-enable-guard / TP-sso-path-a-help
