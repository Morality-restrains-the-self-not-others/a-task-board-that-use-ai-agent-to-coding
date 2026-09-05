# 权限分析 — ImageMarket 恢复厂商申请入口

- **Date:** 2026-09-01
- **Design:** `docs/superpowers/specs/2026-09-01-image-market-sso-link-missing-design.md`
- **Verdict:** 绿灯 ✅

## 权限影响矩阵

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET `/api/ai-provider/vendor-status/` | 已登录主站用户 | Identity（saas_user） | read | APISIX `ai-provider-auth` + taskAuth forward-auth；handler 按 `X-User-Id` | ✅ 充分 | 不新增 |
| POST `/api/ai-provider/vendor-application/` | 已登录且可投递邮箱 | Identity | write | `requireVendorApplicant`：合成邮箱 400；按 saas_user_id 建档 | ✅ 充分 | 不新增 |
| POST upload-url / send-sms / verify-phone | 同上 | Identity | write | 同上用户级门禁 | ✅ 充分 | 不新增 |
| ImageMarket 申请 UI | 能打开租户镜像市场页的成员 | Tenant page 壳 + Identity 申请 | UI | 既有镜像市场页门禁；申请不按 tenant 隔离 | ✅ 充分 | 申请是账号级，不新增 tenant region |
| provider ApplyPanel 删除 | — | — | — | 门户未登录卡片不再提交申请 | ✅ | 减少未入租户申请面 |

## 角色建模

无新角色。厂商申请主体是主站账号，不是 tenant_admin / workspace 角色。

RBAC 元规则（页面组/ui_region）：**不触发**新租户页面或新租户 API。仅恢复既有 ImageMarket 页上的账号级表单。

## 安全审查

- [x] IDOR：申请按 gateway 注入的 `X-User-Id`，客户端不能指定他人 user_id
- [x] 权限提升：无 PATCH 他人厂商档案
- [x] 跨租户：vendor 行不按 tenant_id；任意租户页提交仍绑定同一 saas 用户（既有语义）
- [x] 合成邮箱：`@sso.invalid` 不能申请、不能 SSO
- [x] pending 不能 SSO（关闭自动建号后门）
- [x] 证照直传：预签名 COS，不经业务进程转文件

## 测试

| 场景 | 角色 | 操作 | 预期 |
|------|------|------|------|
| 无会话 | 匿名 | GET vendor-status | 401 |
| 合成邮箱 | 微信用户 | POST application | 400 |
| 真实邮箱 + 审核开 | 成员 | POST application | 200 pending |
| pending | 申请人 | 镜像市场 | 无 SSO href |
