# 权限分析 — 指令闲置回收

- **日期:** 2026-08-22
- **设计:** `docs/superpowers/specs/2026-08-22-instruction-idle-recycle-design.md`
- **结论:** 绿灯 ✅ — 无新角色；容器令牌 + internal secret；STS 不进评论

## 权限影响矩阵

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| POST `…/server-container-token/task-detail/` | 本评论容器 token | Workspace/Task/Comment | read | ValidateToken + comment scope | ✅ | 只读策略分钟数；不得把 CPA 密钥写入响应 |
| GET `/api/internal/cloud/workspace-machine-policy/` | Credential（internal） | Tenant+Workspace | read | `X-Internal-Secret` | ✅ | 禁止公网；query 必须带 company_id+workspace_id |
| heartbeat `instruction_idle` | 本评论容器 token | Comment CSC | write | inbound token + CSC 解析 | ✅ | 只改本 CSC 行；不得改他评论 |
| POST `request-machine-release` | 本评论容器 token | Instance | write | 已有 inbound + sole busy | ✅ | reason=`instruction_idle`；STS 不走此口 |
| POST recycle-idle-machines | taskEvents timer | Workspace CSC | write | internal secret | ✅ | 扩展扫描；无新公网 |
| task-detail `machine_release_sts` | 本评论容器 | 单实例 | 临时凭证 | 仅 CPA 配置 Role 时签发 | ✅ | session policy 锁 instance_id；禁止写评论/SSE |

无新 page/region。非租户控制台。

## 新增角色/权限建模

无。不新增角色。

## 安全审查结论

- [x] **IDOR**: task-detail / heartbeat / release 均绑定容器 token 的 tenant/workspace/task/comment
- [x] **权限提升**: 无 PATCH 策略；设置页闲置分钟已有 workspace admin
- [x] **跨租户**: internal GET 以 query company_id 为准，Credential 只传 token 所属租户
- [x] **403 vs 404**: token 失败沿用现有 401/403
- [x] **user_id 注入**: 无
- [x] **敏感操作**: DeleteInstance 仍走 CLOUD_SERVER_STOPPED 或锁实例 STS；交付失败禁止拆机

## 测试用例清单

| 测试场景 | 角色 | 操作 | 预期 |
|----------|------|------|------|
| 容器 token 拉 task-detail | 本评论容器 | POST task-detail | 200 + idle_recycle_minutes |
| 他任务 token | 其它容器 | POST 本任务 task-detail | 403 |
| 无 internal secret（生产非空时） | 匿名 | GET internal policy | 403 |
| 心跳 instruction_idle | 本评论 | heartbeat | 只写本 CSC |
| 无 Role | 容器 | task-detail | 无 machine_release_sts |

## 风险评级

| 项 | 级 | 缓解 |
|----|----|------|
| STS 泄露到评论/日志 | 高 | ADR-0031：不进评论/context_pack/SSE；日志脱敏 |
| 执行中误拆机 | 高 | 无 instruction_idle_since 不按 N 拆 |
| 跨评论拆机 | 中 | 沿用 request-machine-release sole busy |

**总结清单：**
- 主体：容器 token vs internal vs CPA Role — 三者分离
- STS：不签发 / 锁实例签发 — 采用后者且默认为空 Role
