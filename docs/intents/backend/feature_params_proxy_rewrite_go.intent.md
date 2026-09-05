# FeatureParams：proxy rewrite + TaskApiKeyUsage 迁 Go

## 意图

将 Django `rewrite_sub_token_providers_for_proxy` 与 `TaskApiKeyUsage` 写入迁入 `taskCloudService`，使 `feature-params-env` 热路径不再依赖 Django resolve HTTP。

## 验收

1. Go `feature_params_proxy_rewrite.go` 在 endpoint 启用时改写 `use_sub_token` providers。
2. Cloud 从 `conf/ai/task-ai-endpoint` 加载 `enabled` / `publicBaseUrl`（env 可覆盖），与 Django settings 对齐。
3. 成功 resolve 后尝试写 saas `projects_task_api_key_usage`；失败仅打日志，不阻断 env 响应。
4. Django `feature-params-env-resolve` compat HTTP **已删除**（404）。
5. 真实租户冒烟：`TASK_AI_ENDPOINT` 启用 + `use_sub_token` + `proxy_token` → providers 含 `proxy_mode=task_ai_endpoint` 与 gateway `base_url`。

## 变更日期

2026-07-10



## 业务意图 → 事件对照

> 精修（2026-07-15）：对照 `.ai/08_prompt_management/01_intent_driven_development.md`。

**无对应事件**：proxy rewrite 逻辑迁 Go（HTTP），无新增业务事件。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| FeatureParams：proxy rewrite + TaskApiKeyUsage 迁 Go | — | — | — | — | proxy rewrite 逻辑迁 Go（HTTP），无新增业务事件 |
## 变更记录

- 2026-07-10：Cloud `loadTaskAIEndpointConf` 对齐 conf；live smoke `PROXY_REWRITE_OK`。
- 2026-07-14：删除 Python `rewrite_sub_token_providers_for_proxy`；仅 Go 实现。
