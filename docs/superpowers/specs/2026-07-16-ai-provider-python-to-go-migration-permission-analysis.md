# 权限分析：ai-provider Python → Go

**日期：** 2026-07-16  
**依赖设计：** `2026-07-16-ai-provider-python-to-go-migration-design.md`  
**迭代：** ai-provider-go-migration  

## 角色

| 角色 | 标识 | 鉴权 |
|------|------|------|
| Anonymous | 无 | public / health / OIDC start / SSO exchange / userdata-verify |
| Vendor | JWT `typ=vendor` | vendor CRUD、credentials proxy |
| Staff | JWT `typ=staff` | admin CRUD、审批 |
| Internal consumer | 无（public） | taskCloudService 拉 catalog |

## 端点权限矩阵（保持原语义）

| 路径前缀 | Anonymous | Vendor | Staff |
|----------|-----------|--------|-------|
| `/api/health/` | ✅ | ✅ | ✅ |
| `/api/public/*` | ✅ | ✅ | ✅ |
| `/api/auth/sso/exchange/` | ✅（校验 bridge JWT） | — | — |
| `/api/auth/oidc/*` | ✅ | — | — |
| `/api/vendor/auth/me/` | ❌ | ✅ | ❌ |
| `/api/vendor/**`（业务） | ❌ | ✅（本 vendor 作用域） | ❌ |
| `/api/admin/auth/me/` | ❌ | ❌ | ✅ |
| `/api/admin/**` | ❌ | ❌ | ✅ |
| login/register | 恒 403 | — | — |

## 作用域规则

- Vendor 只能读写 `vendor_id = self` 的 image group / container image / cloud server image
- Staff 可审批任意 pending 镜像；可管理 userdata templates / vendors 只读列表
- Proxy 到 taskCloudService：转发原始 Bearer，不做本服务 JWT 再校验（与 Python 一致）

## 审计要求

- 登录成功/失败结构化日志（含 role、saas_user_id，禁止密码）
- 审批动作写 `marketplace_containerimagereviewhistory`
- 出站 proxy/OCI 请求脱敏落盘
