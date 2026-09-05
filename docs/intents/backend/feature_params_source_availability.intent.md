# 意图：智能体资源配置来源可用性含 LLM 配置

## 背景与目标

任务详情「智能体资源配置」选择器的空状态由后端 `env_var_sources_available` 驱动。旧口径只统计 `extra_env_vars` 的非空 key。公司设置页 `/settings/feature-params/` 的主配置是 LLM provider/模型（系统会生成 `TASK_LLM_*` 环境变量），自定义变量常为空。结果：设置页已有公司智能体资源，任务详情仍提示「暂无可用环境变量」。

## 范围与边界

- 范围内：`taskCloudService` 公司/工作空间 feature-params GET 的 `env_var_sources_available`；前端读取 `data` 内标志（兼容顶层）。
- 范围外：不改设置页表单；不改选择器选项集合；不改任务绑定 PATCH 契约。

## 约束与风险

- summary 视图仍脱敏 extra_env_vars / API key，标志必须在脱敏前计算。
- 空 provider 占位（`provider=""`）与空模型不视为已配置。
- 工作空间可用性只看本级配置，不把公司 LLM 算成 workspace=true。

## 验收标准

1. 公司已配 named LLM provider 或 agent_model/provider、即使 extra_env_vars=[] → `company=true`。
2. 公司/工作空间均无自定义变量且无 LLM 配置 → 两级 false。
3. 工作空间自有 LLM 配置、公司为空 → `workspace=true`。
4. 任务详情在上述 (1) 下不展示「暂无可用」提示，选择器可点。

## 实施计划

1. `featureParamsSourceConfigured`：extra key **或** named provider **或** 非空 agent 模型/厂商。
2. 前端 `readEnvVarSourcesAvailableFlag` 统一读 `data.env_var_sources_available`。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---|---|---|---|---|---|
| 查询智能体资源配置来源可用性 | — | — | — | — | 纯查询 GET 标志，无状态变更 |
