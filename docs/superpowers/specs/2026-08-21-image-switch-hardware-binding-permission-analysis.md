# 切换镜像 × 硬件硬拦截 — 角色权限分析

- **Date:** 2026-08-21
- **Status:** accepted（/goal 自动采用）
- **Design:** `docs/superpowers/specs/2026-08-21-image-switch-hardware-binding-design.md`

## 权限影响矩阵

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| PATCH `/api/projects/{id}/tenant_id/{tid}/` 镜像+模版校验 | 已登录且可写该租户项目的成员 | Tenant / Project | write | 网关鉴权 + `X-Auth-Tenant-Id` 与项目 `company_id` | ✅ 充分 | 校验在写前；不新增公开 path |
| GET `/api/internal/tenant-installed-images/lookup` | 服务间（project→cloud） | Tenant | read | `X-Internal-Secret` | ✅ 充分 | 沿用既有 lookup |
| start-vm 架构不匹配 | 任务协作者 | Workspace / Task | write（开云资源） | 既有 start-vm 鉴权 | ✅ | 不放宽 |
| 前端保存镜像/模版按钮 | 项目页可编辑用户 | Project | write | 现有项目详情写权限 | ✅ | 防重放沿用 busy |

无新角色、无新 region ACL。租户隔离仍以 `tenant_id` / `company_id` 为准。

## 角色与权限建模

不引入新角色。校验是同一 PATCH 上的业务不变量，不是新权限码。

## 安全审查结论

- [x] **IDOR**: 仍按 `X-Resource-Id` + 租户项目行更新；lookup 带 `tenant_id`
- [x] **权限提升**: 无新 PATCH path
- [x] **跨租户**: lookup 与项目 `company_id` 一致
- [x] **403 vs 404**: 项目不存在仍 404；架构不匹配 400（业务校验，非权限）
- [x] **user_id 注入**: 无
- [x] **敏感操作**: 非资金；400 打 `trace_id`，不写部分字段

**风险评级:** 低。lookup 失败拒绝写库，避免跨架构漏网（可用性换正确性）。

## 权限测试用例

| 测试场景 | 角色 | 操作 | 预期 |
|----------|------|------|------|
| 同租户成员跨架构只改镜像 | tenant member | PATCH image only | 400，库不变 |
| 同租户成员同单匹配模版 | tenant member | PATCH image+template | 200 |
| 内部 lookup 无 secret | 未授权 | GET lookup | 401/403（既有） |

绿灯 ✅
