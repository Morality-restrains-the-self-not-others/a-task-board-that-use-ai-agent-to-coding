# 启动风暴缓解（startup storm）

**日期：** 2026-05-29  
**状态：** 已落地（Phase 1）

## 问题定义

relay 直启后约 1～3 秒内，多条 inbound 同时打向 Django/SQLite：

| 来源 | 典型请求 |
|------|----------|
| go_relayToTrae | `exchange-refresh`、`refresh-access`（容器 bootstrap 已 `TRAE_SKIP_CONTAINER_TOKEN_EXCHANGE=1` 跳过重复换票） |
| onlineServiceJS listen 后 | `register-reachability`、**立即** `heartbeat`、后台 `task-detail` → `repo-clone-credentials` → `feature-params-yaml`、`layer-graph-push` |
| 浏览器 / relay | `relay-to-trae/status-push`、`cloud/compute/*` |

**不是高 QPS**：SQLite 仅需 **2 个重叠写事务** 即可 `database is locked`。

## 已实施缓解

| 层 | 改动 | 环境变量 |
|----|------|----------|
| Django | `exchange-refresh` 审计写入移出 `transaction.atomic()` | — |
| go_relay | 换票两步之间 `150ms` 间隔 | — |
| onlineServiceJS | 首跳心跳延迟默认 5s（原 listen 后立即 tick） | `TRAE_SAAS_HEARTBEAT_INITIAL_DELAY_SEC`（默认 `5`） |
| onlineServiceJS | bootstrap 连续 SaaS POST 间隔 200ms | `TASK_API_BOOTSTRAP_SAAS_STAGGER_MS`（默认 `200`） |
| 架构（上轮） | taskAgentSupport :8011 承接 inbound | `taskAgentSupport.enabled` |

## 验收

1. relay 直启：日志中 `exchange-refresh` 不再与同秒 `heartbeat` + `feature-params-yaml` 叠成连续 `database is locked`。
2. 任务详情：启动后 5s 内心跳可能仍显示 connecting，之后正常（可设 `TRAE_SAAS_HEARTBEAT_INITIAL_DELAY_SEC=2` 加快）。
3. `pytest tests/test_container_runtime_tokens.py::test_exchange_refresh_then_refresh_access_updates_db` 仍绿。

## 未做（Phase 2）

- 合并 bootstrap 为单次 internal batch API
- dev 默认 PostgreSQL
- git-push 完全异步
