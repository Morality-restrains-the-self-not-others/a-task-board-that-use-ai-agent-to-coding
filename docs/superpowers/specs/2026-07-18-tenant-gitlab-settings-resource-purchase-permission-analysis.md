# 权限分析 — tenant-gitlab-settings-resource-purchase

**基于：** `2026-07-18-tenant-gitlab-settings-resource-purchase-design.md`

## 角色

| 角色 | 说明 |
|------|------|
| tenant_member | 租户成员；可查看配额与连接（脱敏） |
| tenant_admin | 公司管理员；可保存/删除自建 OAuth；购买配额（产品期望） |
| authenticated | Django `IsAuthenticated`；billing 代理基线 |

## 改动点矩阵

| 接口/UI | 最小角色 | 资源范围 | 校验 | 备注 |
|---------|----------|----------|------|------|
| GET gitlab-resources | tenant_member | path tid = 会话租户 | 网关/登录 + parseTenantID | 只读 |
| POST gitlab-resources/purchase | authenticated（首期） | 同上 | 余额校验 | 与 switch_pricing 同级；后续可加 admin |
| GET/PUT/DELETE gitlab-oauth-connection | 既有 | 既有 | taskGitOauth ensureTenantAdmin（写） | 不变 |
| Sidebar「GitLab」 | tenant_member | — | 前端菜单 | 文案 |

## 风险与缓解

- **非管理员购买**：首期与换套餐一致；OPT 可加 Django 侧 company admin 门禁。
- **跨租户 IDOR**：路径 tid 必须与会话租户一致（既有网关/前端约定）；taskBill 以 path tid 记账。
- **Secret 泄露**：自建连接区块 secret 不回传（既有）。
