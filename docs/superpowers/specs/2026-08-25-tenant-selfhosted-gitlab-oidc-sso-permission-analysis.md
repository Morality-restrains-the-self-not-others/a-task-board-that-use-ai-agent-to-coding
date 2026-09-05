# 角色权限分析：租户自建 GitLab 平台 OIDC SSO

- **日期**: 2026-08-25
- **设计**: `docs/superpowers/specs/2026-08-25-tenant-selfhosted-gitlab-oidc-sso-design.md`
- **状态**: 已完成（goal 自动推进，零交互）

## 范围

覆盖链路 B（taskAuth 签发 `gitlab-tenant-{company_id}`，自建 GitLab 作 OmniAuth RP）的管理 API 与 `/api/oidc/authorize` 成员闸门。链路 A（Git 网站授权）权限不变。

## 角色矩阵

| 角色 | GET 签发状态 | PUT 签发 / POST rotate / DELETE 吊销 | OIDC authorize 获码 | 备注 |
|------|--------------|--------------------------------------|---------------------|------|
| 匿名 | 401 | 401 | 302 登录页 | 与现网 OIDC 一致 |
| 已登录非该租户成员 | 403（无权该租户） | 403 | `access_denied` 回 RP | **fail-closed**；平台超管若非成员同样拒绝 |
| 租户普通成员（有 `settings.gitlab.main` view） | 200，无 secret | 403 | 允许（成员闸门，**不**要求 gitlab 区域） | SSO 登录发生在客户 GitLab，不是设置页 |
| 租户普通成员（无 gitlab 区域） | 403 on GET 管理 API | 403 | 允许 | 管理面与登录面分离 |
| 租户管理员 / 持 `settings.gitlab.main` operate | 200 | 200（需 Path A `base_url` 已保存且匹配） | 允许 | 写按钮 `hasRegion('settings.gitlab.main')` |
| 平台超管（非成员） | 403 | 403 | `access_denied` | 禁止超管旁路进入客户 GitLab |

## 资源组

已有种子（`dataMigrate/taskAuth/032_logical_resource_groups.sql`）：

- page `settings.gitlab`（`rg-page-settings-gitlab`）
- ui_region `settings.gitlab.main`（`rg-reg-settings-gitlab-main`）

**本增量补** `auth_resource_member`：将下列 API 挂到 `settings.gitlab.main`：

| member_key | 方法 |
|------------|------|
| `GET /api/tenant/{tid}/gitlab-oidc-sso/` | view |
| `PUT /api/tenant/{tid}/gitlab-oidc-sso/` | operate |
| `POST /api/tenant/{tid}/gitlab-oidc-sso/rotate/` | operate |
| `DELETE /api/tenant/{tid}/gitlab-oidc-sso/` | operate |

Handler：**`authz.RequireRegionView`（GET）/ `RequireRegionOperate`（写）**，再叠加租户成员（GET）/ 租户管理员（写）。不以粗码 `company:manage` 为唯一门闩。

## IDOR / 越权

1. **路径 `{tid}` 必须等于 client `owner_company_id`**；禁止用路径 tid 操作他租户 client。
2. **client_id 固定** `gitlab-tenant-{tid}`，禁止请求体覆盖 client_id。
3. **跨租户探测**：他租户 GET 一律 403，不因「无配置」返回 404 泄露存在性。
4. **secret**：仅 PUT 首次签发与 POST rotate **响应体一次**返回；GET 永不返回；日志禁止打 secret / hash。
5. **OIDC authorize**：成员查询失败（taskTenant 不可达）→ `access_denied`（fail-closed），不得 fail-open。
6. **`oidc_region_gate`**：仅 `gitlab-git-service*`；`gitlab-tenant-*` **不得**进入区域 tester 闸门。

## 403 vs 404

| 情况 | 状态 |
|------|------|
| 未登录 | 401 |
| 已登录但非该 tid 成员，或无 region | 403 |
| 成员但未签发 SSO | GET 200 `configured: false` |
| Path A 未保存或 base_url 不匹配 | PUT 400 |
| 生产 HTTP redirect_uri | PUT 400 |

## 前端

- 写按钮：`hasRegion(tenantId, 'settings.gitlab.main')` + `createClickGuard` + `Idempotency-Key`
- 无权限：隐藏签发/轮换/吊销，不暴露 secret 输入框
- 链路 A 帮助文案对有 GET 权限的成员可见（只读 copy）
