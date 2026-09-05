# taskAuth Increment 7 — Value Stream

> 设计: `docs/superpowers/specs/2026-05-28-taskauth-increment7-auth-table-cleanup-design.md`

## 增量切片

| # | 增量 | 用户价值 | 依赖 |
|---|------|----------|------|
| 7.1 | send_verification_code → taskAuth | 验证码发送统一入口 | — |
| 7.2 | taskAuth 不可用时 503（禁 default auth 写） | 生产不写共享 auth 表 | 7.1 |
| 7.3 | drop 共享 auth 表 + 验证脚本 | 共享库瘦身、单一真源 | 7.2 + 数据已迁移 |
| 7.4 | value-stream 字段收口至 task-auth.* | 可观测性一致 | 7.3 |

## 影响流

- **user-auth**：verification-code、全部 auth 写步骤
- **新增 auth-table-cleanup**：验证 default 无 auth 表

## YAML 变更

见 `value-stream.yaml`：`auth-table-cleanup` 步骤；`verification-code` 增补 `task-auth.*` 字段。
