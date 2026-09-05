# 测试意图 — 租户邀请有效期上限 365 天（后端）

- **对应意图:** `people_invite_expiration.intent.md`

## 测试目标

规范化函数与 HTTP 创建/重发路径覆盖默认 90、上限 365、越界拒绝。

## 测试分层

- 领域：`invite_expiration_test.go`
- HTTP：`invite_handlers_test.go`

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 缺省/0 且允许默认 | 90 |
| T2 | 365 | 365，无错 |
| T3 | 366 | 错误，含「365」 |
| T4 | 重发 0 | 不允许默认，错误 |
| T5 | HTTP 创建 365 | 201，`expiration_days=365` |
| T6 | HTTP 创建 366 | 400 |

## 通过标准

上述 Go 单测全绿。
