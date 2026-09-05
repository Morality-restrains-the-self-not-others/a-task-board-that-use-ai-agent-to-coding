# Value Stream: 功能参数环境变量下发与智能体运行时解耦

> 设计：`docs/superpowers/specs/2026-05-29-feature-params-env-agent-agnostic-design.md`

## Related Value Streams

- **company-management / tenant-feature-params**（`value-stream.yaml`）：**修改** — 字段重命名、下发形态 YAML → env
- **task-detail-runtime / machine_container bootstrap**（文档）：**修改** — `feature-params-yaml` → `feature-params-env`
- **2026-05-28-task-agent-support-split-design**：**未来对齐** — 内部 task-agent-support API 可复用 env 契约

Greenfield 价值流条目：无独立新 stream，为既有 `tenant-feature-params` 的契约升级。

## Value Summary

租户管理员在功能参数页配置 LLM 供应商与模型后，可预览将下发给容器的 `TASK_*` 环境变量；容器 bootstrap 拉取 env 并在本地生成 Trae 配置，平台无需感知具体智能体运行时。

## End-to-End Flow

```text
[管理员打开功能参数页]
  → 填写 providers / agent_model / summary_model / agent_max_steps
  → 保存 → DB projects_tenant_feature_params
  → 页面展示 env 预览（env_preview）
[容器启动 bootstrap]
  → POST feature-params-env（access_token）
  → 收到 env map
  → onlineServiceJS featureParamsEnvToYaml → service_config.yaml
  → trae-cli 可运行
```

## Value Increments

### Increment 1: 后端 env 契约 + DB 重命名（Thin Slice）

**Value to user：** 容器可拉取 env 并启动（bootstrap 端到端）。

**Scope：**

- Migration：`trae_*` / `lakeview_*` → `agent_*` / `summary_*`
- `build_tenant_feature_params_env()`
- `feature-params-env/` 端点；删除 `feature-params-yaml/`
- 管理 API 字段重命名 + `env_preview`
- `test_manage_feature_params.py`、`test_container_runtime_tokens.py`

**Depends on：** 无

**Fields：**

- `saas-backend.projects_tenant_feature_params.agent_model`
- `saas-backend.projects_tenant_feature_params.agent_model_provider`
- `saas-backend.projects_tenant_feature_params.agent_max_steps`
- `saas-backend.projects_tenant_feature_params.summary_model`
- `saas-backend.projects_tenant_feature_params.summary_model_provider`
- `saas-backend.projects_tenant_feature_params.providers`

### Increment 2: onlineServiceJS Trae Adapter

**Value to user：** 与现网等价的 `service_config.yaml` 由 env 生成。

**Scope：**

- `featureParamsEnvToYaml.mjs` + 单测
- `bootstrap.mjs` 切换端点
- e2e mock URL 更新

**Depends on：** Increment 1

### Increment 3: 前端 env 预览 + job env 重命名

**Value to user：** 控制台预览与容器契约一致；任务级覆盖使用 `TASK_AGENT_*`。

**Scope：**

- `WorkspaceSettingsFeatureParams.vue`
- `useTaskDetail.js`
- `ai_instruct_stream_service_stream_fns.py`、`forward_container_*`
- `test_ai_task_comment.py`
- `machine_container.md` 等文档

**Depends on：** Increment 1

## 验收标准

| 增量 | 验收 |
|------|------|
| 1 | pytest 管理 API + 容器 env 端点通过；旧 yaml 端点不存在 |
| 2 | `featureParamsEnvToYaml.test.mjs` 通过；bootstrap 写入合法 yaml |
| 3 | 功能参数页显示环境变量预览；无 Trae/Lakeview/YAML 文案 |
