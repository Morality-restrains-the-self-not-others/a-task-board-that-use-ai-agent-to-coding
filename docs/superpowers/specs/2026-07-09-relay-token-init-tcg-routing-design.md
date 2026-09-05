# 设计文档：relay token-init / env-prepare / precheck 迁 taskContainerGateway

**日期：** 2026-07-09  
**状态：** 采用（goal-mode 自动批准）  
**动机：** 点击「启动」时 `POST .../relay-to-trae/token-init/` 经 APISIX 落入 `taskCloudService`，返回 501 `compute action not yet ported`，阻断后续 precheck → start → 容器克隆。

## 根因

| 层 | 现状 |
|----|------|
| APISIX `relay-to-trae-proxy` | 仅 health/status/register/start/stop |
| 未匹配 URI | 落入 `task-cloud-service`（priority 860） |
| taskCloudService | `isContainerOutboundComputeSub` 不含 relay；`isDjangoProxiedComputeSub` 恒 false → **501** |
| TCG | 已有 `credentialTokenInit`（供 start/register 内联），**无独立** token-init / env-prepare / precheck HTTP 处理 |

## 方案（采用）

与既有 lifecycle 设计一致：`Browser → taskGateway → taskContainerGateway → taskCredentialService`。

| action | 方法 | TCG 行为 |
|--------|------|----------|
| `env-prepare` | GET | validate_session → 返回 cfg.relayRuntime origins + ACCESS_TOKEN 占位符 |
| `token-init` | POST | validate_session → CRED `/v1/token/init` → 审计事件（可选）→ `{status:ok, token_initialized, env_preview}` |
| `repo-credentials-precheck` | POST | validate_session → token-init → CRED `repo-clone-credentials` → 映射 200/409/502（含 token 刷新降级） |

**不采用：** 在 taskCloudService 代理回 Django（与 Go 拆分方向相反）。

## 契约兼容

前端 `ServerConfig.logic.vue` 成功判据不变：

- token-init：`ok` 且 `status !== 'error'`
- precheck：`ok` 且非 409/502 阻断码
- start：已走 TCG，返回 `accepted`

## 变更清单

1. `taskGateway/routes/routes.yaml` + 再生 `apisix.yaml`
2. `taskContainerGateway` 新增三 handler + 单测
3. Django `@require_django_forward` 对三 action 返回 410
4. Playwright：点启动 → token-init≠501 → precheck/start 成功 → 观测克隆完成信号

## 连带修复（同会话发现）

1. **validate_session**：`_relay_probe_path` 仅放行 health/status；本地直启无 CloudServerConfig → token-init/start 403。改为 `/relay-to-trae/` 全路径仅校验租户成员。
2. **CRED 业务库**：`projects_taskrepoidentity` / `projects_todo` 已迁 Go；改为读 `data/task_task.db` + `data/task_project.db` + saas git identity。
3. **SCP URL provider**：`git@host:path` 未被 `extractHostNetloc` 解析 → precheck 409；已支持 SCP 风格。

## 后续优化（2026-07-09 同日落地）

1. **GitLab oauth/token 400 → precheck 降级**：`access-for-user` 在 provider 前缀回退命中其它 `service_provider` 时，仍用请求里的 `provider_key` 取 client → 与 refresh 签发应用不一致。改为用凭证行自身 `row.provider` 换票，并按 `service_provider` 精确传入 `client_id`/`client_secret`（避免同 website 多 OAuth App 串用）。
2. **env-prepare origin 对齐**：TCG `relayRuntime.businessApiEndpointOrigin` 由 `http://127.0.0.1:8011/api` 改为 `http://127.0.0.1:8765`（与 Django `prepare_relay_to_trae_env` / 前端面板默认一致）；`taskApiEndpointOrigin` 保持 `http://127.0.0.1:8011`（与 `get_relay_task_api_base_url` 同源）。

## 架构影响

增量路由扩展，不新增组件；不强制新 ArchiMate 大版本（沿用 v4/v5 relay 公网入口 Plateau）。
