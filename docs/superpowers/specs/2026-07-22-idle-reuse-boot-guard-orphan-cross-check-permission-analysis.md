# 角色权限分析：闲置复用启动保护 + 孤儿交叉校验

- **关联设计**: `2026-07-22-idle-reuse-boot-guard-orphan-cross-check-design.md`
- **日期**: 2026-07-22

## 结论

无新 HTTP endpoint、无新权限点。变更均为 `taskCloudService` 内部门闩，沿用既有 `start-vm(-auto)` 与 workspace reconcile 鉴权。

| 改动点 | 服务 | 资源 | 操作 | 既有鉴权 | 风险 | 处置 |
|--------|------|------|------|----------|------|------|
| idle boot-guard | taskCloudService | CSC / instance | read filter | start-vm 用户 JWT + tenant/workspace | 低：仅收紧候选 | ✅ |
| orphan cross-check | taskCloudService | ECS DeleteInstance | write gate | internal reconcile（workspace CSC 范围） | 中：误跳过会留真孤儿 | ✅ 仅 skip 当他 CSC 持有；真孤儿仍删 |
| 跨 workspace | — | — | — | — | 禁止 | 查询仍限定 company+workspace |

## 审计要点

- 不得因 boot-guard 绕过 `enabled_authorization_ids` 白名单。
- `orphan_reconcile_skip_owned` 须打日志便于审计误杀防护。
