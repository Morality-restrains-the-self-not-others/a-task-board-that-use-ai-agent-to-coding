# 开放式邀请链接 — 角色权限分析

- **日期**: 2026-08-29
- **设计**: `docs/superpowers/specs/2026-08-29-open-invite-link-design.md`
- **结论**: 绿灯 ✅ — 无新角色、无新页面、无新公网 path

## 权限影响矩阵

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| POST .../members/invite/ | `member:manage`（租户管理员/被授人员管理） | Tenant `people.invite.main` | write | `requireCompanyAdmin` → `RequirePerm(PermMemberManage)`；FE `hasPerm(member:manage)` + `data-rg-key=people.invite.main` | ✅ | 不新增 RequireRegion（无新 path/region）；沿用既有登记 |
| GET validate-invite | 持 token 的任何人（含未入司） | Tenant invitation | read | 无登录强制；token + company_id | ✅ | token 即能力；禁止在错误中回显完整 token |
| POST join | 已登录用户 | Tenant membership | write | `requireAuthUser`；token 有效；非已成员 | ✅ | 不因开放链绕过「已在公司」 |
| GET pending-invitations | 租户成员 | Tenant | read | `requireCompanyMember` | ✅ | 开放链 use_count 非敏感超额 |
| POST revoke | `member:manage` | Tenant invitation | write | `requireCompanyAdmin` | ✅ | 开放链同样立即失效 |

**v72 对照**：页面 `people.invite` / region `people.invite.main` 已在 `dataMigrate/taskAuth/032_logical_resource_groups.sql`。本期不新增 page/region/API 叶子。Join 页为 guest 路由（未入司用户），与现网一致。

## 角色与权限建模

无新角色。开放链接不授予创建者以外的「无限拉人」权——仅持链接且未入司者可 join；角色/grants 仍取自邀请行（创建时由管理员设定）。

## 安全审查结论

- [x] **IDOR**: join/validate 以 `company_id`（URL tenant）+ token 双条件；跨租户同 token 失败
- [x] **权限提升**: 开放链 `is_admin` 仍由创建时 role 决定；普通成员不能创建（403）
- [x] **跨租户泄露**: 所有 SQL 带 `company_id=?`
- [x] **403 vs 404**: 无效 token 统一 400「无效或已过期」，不暴露是否存在
- [x] **user_id 注入**: join 只用网关 `X-User-Id`，不接受 body user_id
- [x] **敏感操作**: 链接即入职凭证；token 32 字节 CSPRNG；日志禁止完整 token
- [x] **并发超额**: `UPDATE ... use_count < max_uses` 行锁
- [x] **密钥**: 无新密钥

## 权限测试清单

| 场景 | 角色 | 操作 | 预期 |
|------|------|------|------|
| 管理员创建 open | admin1 | POST invite link_kind=open | 201 |
| 普通成员创建 | u7 | POST invite | 403 |
| 跨租户 join | 用户 + 他司 token | POST join | 400 |
| 无登录 join | 无头 | POST join | 401 |

## 风险评级

| 项 | 级 | 缓解 |
|----|-----|------|
| 链接泄露导致陌生人入司 | 中 | 有效期 + 可选 max_uses + 可撤销；默认仍为单次 |
| 开放管理员链接 | 中 | 与现单次管理员链接相同风险；产品可选；UI 不默认 admin |

## 对设计文档回写

见设计文档「成功标准」与 API 契约；本分析不改变落点。
