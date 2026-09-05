# 角色权限分析 — auto_run 交付 push/PR 修复

- **日期**: 2026-07-21
- **设计**: `2026-07-21-auto-run-delivery-push-pr-fix-design.md`
- **结论**: **SKIP 实质变更** — 无新端点、无新角色；仍复用容器 `ACCESS_TOKEN` + 既有 `layer-github-oauth-access-tokens` / oauth-access-push。

| 触点 | 鉴权 | 变更 |
|------|------|------|
| job close → 交付 | 容器内进程 | 逻辑修复 |
| oauth-refresh-push | 容器 token → Credential/Django | 无 |
| 启动补跑 | 同容器 | 无对外 API |

CRG: skipped（无权限边界 diff）。
