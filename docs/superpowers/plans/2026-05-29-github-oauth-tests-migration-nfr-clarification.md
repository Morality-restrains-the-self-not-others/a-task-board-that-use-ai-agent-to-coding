# NFR 澄清: GitHub OAuth 测例迁移

> 设计：`docs/superpowers/specs/2026-05-29-github-oauth-tests-migration-to-gitoauth-design.md`  
> 价值流：`docs/superpowers/plans/2026-05-29-github-oauth-tests-migration-value-stream.md`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L0 | 不适用（仅测例搬迁） |
| 可维护性 | L2 | 测例与真源服务同仓；删除重复 mock |
| 数据一致性 | L2 | summary JSON 与 DB 行 bind_status/bind_error 一致 |
| 安全性 | L2 | 测例不泄露 refresh 明文；沿用既有 cipher 字段 |
| 可观测性 | L0 | 不适用 |

## 质量场景

### QS-01: summary 失败态契约
| 要素 | 内容 |
|------|------|
| 类别 | 数据一致性 |
| 等级 | L2 |
| 刺激源 | task2app 桥接（间接） |
| 刺激 | DB 行 `bind_status=failed` + `bind_error` |
| 制品 | POST `/api/internal/github/oauth/user-credential/summary-for-user/` |
| 环境 | 单元测试（gitOauth pytest） |
| 响应 | JSON 顶层与 connections[] 均含 failed/error |
| 响应度量 | `cd gitOauth && python -m pytest api/tests.py::GithubOAuthCredentialSummaryForUserViewTests::test_summary_exposes_failed_bind_status_and_error -q` 通过 |

## 领域模型影响

| NFR 决策 | 模型影响 | DDD 动作 |
|----------|---------|----------|
| L2 一致性 | 无新聚合；沿用 `GithubAppUserCredential` | 不新增 domain 文件，仅测 HTTP 视图 |

## 权衡与边界

- 不在 task2app 保留重复 HTTP 契约 mock
- valueStream 全局 `runner` 仍为 `task2app/Saas_project`；gitOauth 测例路径为 `../../gitOauth/api/tests.py`，执行时需 gitOauth 的 Django settings（本地可 `cd gitOauth && pytest`）

## 跳过声明

- 性能/可伸缩性/合规：测例迁移无运行时 NFR 变更
