# 功能参数：环境变量下发与智能体运行时解耦设计

**日期：** 2026-05-29  
**状态：** 已评审（方案 B：DB 字段一步到位重命名）  
**页面：** `http://127.0.0.1:4000/tenant/<tenant>/settings/feature-params/`

## 背景与问题

当前「功能参数」链路存在三层 Trae 耦合：

1. **数据模型**：`TenantFeatureParams` 字段名为 `trae_model`、`lakeview_model`、`trae_max_steps` 等。
2. **YAML 双份实现**：Vue `yamlContent` 与 Python `build_tenant_feature_params_yaml()` 各维护一份 Trae 专用 `service_config.yaml` 模板（含 `agents.trae_agent`、`mcp_servers.playwright` 等）。
3. **容器 bootstrap**：`feature-params-yaml/` 由 Django 拼好整段 YAML 返回，`onlineServiceJS` 仅落盘。

后果：

- 平台层与 Trae 运行时结构绑定，无法在不改 Django 的情况下切换 Cursor 等智能体。
- YAML 模板变更需同步改前后端，易漂移。
- 控制台「YAML 预览」展示的是 Trae 配置，而非容器实际接收的契约。

## 目标

| 目标 | 说明 |
|------|------|
| **环境变量契约** | 平台只下发扁平 `Record<string, string>`，不再下发 YAML |
| **容器侧生成配置** | `onlineServiceJS` 拉取 env 后，由 Agent Runtime Adapter 生成 `service_config.yaml`（默认 Trae） |
| **平台对运行时无感知** | task2app 不感知 Trae / Cursor / 其他智能体；不携带 tools、MCP、agent 段落 |
| **字段一步到位重命名（方案 B）** | DB、API、前端、job env 覆盖统一改为通用命名，不保留 `trae_*` / `lakeview_*` 别名 |

## 非目标（本期不做）

- 在 task2app 中选择或配置「Trae vs Cursor」（由容器镜像 / `AGENT_RUNTIME` 环境变量决定）。
- 实现 Cursor adapter（仅预留 adapter 接口，默认 Trae）。
- `command_kind: trae|shell` job API 语义重命名（属于 job 调度契约，与 feature-params 独立）。
- task-agent-support 内部 API 拆分（见 `2026-05-28-task-agent-support-split-design.md`，可后续对齐）。

## 分层架构

```text
┌─────────────────────────────────────────────────────────────┐
│ task2app（租户 LLM 配置，运行时无感知）                        │
│  - TenantFeatureParams（agent_model, summary_model, …）      │
│  - GET/POST /api/tenant/{id}/feature-params/                 │
│  - POST …/feature-params-env/  → { env: { TASK_*: "..." } }  │
└───────────────────────────┬─────────────────────────────────┘
                            │ env map
┌───────────────────────────▼─────────────────────────────────┐
│ onlineServiceJS（Agent Runtime 层）                            │
│  - bootstrap 拉取 feature-params-env                          │
│  - featureParamsEnvToYaml.mjs（Trae adapter，默认）            │
│  - 未来：AGENT_RUNTIME=cursor → envToCursorConfig()          │
│  - 写入 service_config.yaml                                  │
└─────────────────────────────────────────────────────────────┘
```

`AGENT_RUNTIME` 由**容器镜像 / relay 启动 env** 设置，**不由 task2app 下发或存储**。

## 数据模型变更（方案 B）

### 表 `projects_tenant_feature_params`

| 旧字段 | 新字段 | verbose_name |
|--------|--------|--------------|
| `trae_model` | `agent_model` | 智能体模型名 |
| `trae_model_provider` | `agent_model_provider` | 智能体模型供应商 |
| `trae_max_steps` | `agent_max_steps` | 智能体最大迭代次数 |
| `lakeview_model` | `summary_model` | 摘要模型名 |
| `lakeview_model_provider` | `summary_model_provider` | 摘要模型供应商 |
| `providers` | `providers` | 不变 |

实现：Django migration `RenameField` × 5；模型类名保持 `TenantFeatureParams`（表名不变，减少迁移面）。

### 管理 API 契约

`GET/POST /api/tenant/{tenant}/feature-params/` 请求/响应字段同步为新名：

```json
{
  "data": {
    "providers": [...],
    "agent_model": "gpt-4.1",
    "agent_model_provider": "openai",
    "agent_max_steps": "200",
    "summary_model": "gpt-4.1-mini",
    "summary_model_provider": "openai"
  }
}
```

**不兼容旧字段名**：`trae_model` 等 POST 字段返回 400 或忽略（实施时选一种并在测试中固定；推荐 **400 + 明确错误**，避免静默丢配置）。

## 环境变量契约

### 容器拉取 API

- **路径（新）**：`POST …/cloud/server-container-token/feature-params-env/`
- **鉴权**：与现 `feature-params-yaml/` 相同（`access_token`、路径三元组校验、过期检查）。
- **响应**：

```json
{
  "company_id": "...",
  "workspace_id": "...",
  "task_id": "...",
  "env": {
    "TASK_LLM_PROVIDERS_JSON": "[{\"provider\":\"openai\",\"api_key\":\"sk-...\",\"base_url\":\"https://api.openai.com/v1\",\"supported_models\":[\"gpt-4.1\"],\"use_sub_token\":false}]",
    "TASK_AGENT_MODEL": "gpt-4.1",
    "TASK_AGENT_MODEL_PROVIDER": "openai",
    "TASK_AGENT_MAX_STEPS": "200",
    "TASK_SUMMARY_MODEL": "gpt-4.1-mini",
    "TASK_SUMMARY_MODEL_PROVIDER": "openai"
  }
}
```

### Key 规范

| Key | 来源 | 说明 |
|-----|------|------|
| `TASK_LLM_PROVIDERS_JSON` | `providers` JSONField | JSON 数组字符串；元素含 `provider`, `api_key`, `base_url`, `supported_models`, `use_sub_token` |
| `TASK_AGENT_MODEL` | `agent_model` | 空串允许 |
| `TASK_AGENT_MODEL_PROVIDER` | `agent_model_provider` | 空串允许 |
| `TASK_AGENT_MAX_STEPS` | `agent_max_steps` | 字符串十进制整数，默认 `"200"` |
| `TASK_SUMMARY_MODEL` | `summary_model` | 空串允许 |
| `TASK_SUMMARY_MODEL_PROVIDER` | `summary_model_provider` | 空串允许 |

所有值为 **string**；复杂结构仅通过 `TASK_LLM_PROVIDERS_JSON` 承载。

### 后端唯一序列化函数

新增 `cloud/services/tenant_feature_params_env.py`：

```python
def build_tenant_feature_params_env(
    *,
    providers: list,
    agent_model: str,
    agent_model_provider: str,
    agent_max_steps: str,
    summary_model: str,
    summary_model_provider: str,
) -> dict[str, str]: ...
```

容器 view 与管理端 env 预览**必须**调用此函数，禁止散落拼装逻辑。

删除 `tenant_feature_params_yaml.py` 及 `feature-params-yaml/` 端点（无过渡期双写；调用方仅 `bootstrap.mjs` 与测试）。

## onlineServiceJS 变更

### bootstrap

`bootstrap.mjs` `runBootstrapAfterListen`：

1. `POST …/feature-params-env/`（替换 `feature-params-yaml/`）
2. 校验 `env` 为 object 且含必需 key（`TASK_AGENT_MAX_STEPS` 至少存在）
3. `featureParamsEnvToYaml(env)` → YAML 文本
4. `YAML.parse` 校验后写入 `service_config.yaml`

### Trae Adapter

新文件 `trae-agent/onlineServiceJS/src/featureParamsEnvToYaml.mjs`：

- 输入：`TASK_*` env map
- 输出：与当前 `trae_config.yaml.example` / 原 Python 生成器**行为等价**的 YAML
- 静态默认值留在 adapter：`agents.trae_agent.tools`、`mcp_servers.playwright`、`models.*.max_tokens` 等
- 单元测试：`featureParamsEnvToYaml.test.mjs` 覆盖 providers JSON、空 provider、max_steps 等

### 未来 Adapter 扩展

```javascript
function resolveAgentConfigFromEnv(env) {
  const runtime = String(process.env.AGENT_RUNTIME || 'trae').toLowerCase();
  if (runtime === 'cursor') throw new Error('AGENT_RUNTIME=cursor not implemented');
  return featureParamsEnvToYaml(env);
}
```

## 控制台 UI 变更

文件：`WorkspaceSettingsFeatureParams.vue`

| 项 | 变更 |
|----|------|
| 表单字段 | `agent_model`, `agent_model_provider`, `agent_max_steps`, `summary_model`, `summary_model_provider` |
| 预览区标题 | 「YAML 预览」→「环境变量预览」 |
| 预览内容 | `KEY=VALUE` 行列表（只读 textarea）；数据来自本地调用与后端一致的序列化（推荐 GET 时后端返回 `env_preview` 字段，或前端 POST 前用共享逻辑——**实施优先：GET `/feature-params/` 增加 `env_preview` 字段由后端计算**） |
| 按钮 | 「复制 YAML」→「复制环境变量」 |
| 删除 | 前端 `yamlContent` computed 及 Trae YAML 模板字符串 |

`useTaskDetail.js` 中读取 `trae_model_provider` / `trae_model` 的代码改为 `agent_model_provider` / `agent_model`。

## Job 级 env 覆盖（一并重命名）

以下路径当前写入 `TRAE_*`，本期改为 `TASK_AGENT_*` 以与 feature-params 契约一致：

| 文件 | 变更 |
|------|------|
| `ai_instruct_stream_service_stream_fns.py` | `TRAE_MAX_STEPS` → `TASK_AGENT_MAX_STEPS`；`TRAE_MODEL` → `TASK_AGENT_MODEL`；`TRAE_MODEL_PROVIDER` → `TASK_AGENT_MODEL_PROVIDER` |
| `forward_container_job_edit_run.py` | `TRAE_MAX_STEPS` → `TASK_AGENT_MAX_STEPS` |
| `forward_container_layer_command.py` | 同上 + model/provider key 重命名 |
| `test_ai_task_comment.py` | 断言同步 |

**onlineServiceJS** 在执行 `command_kind=trae` 的 job 时，若 job payload 带 `env.TASK_AGENT_*`，应合并进运行时（若已有合并逻辑则更新 key 名；若无则在 adapter 或 job 启动路径文档化「job env 覆盖 tenant env」优先级）。

## 领域概念清单（供 `/5-ddd`）

| 类型 | 名称 | 说明 |
|------|------|------|
| Bounded Context | Tenant LLM Configuration | 租户级模型与供应商配置 |
| Bounded Context | Container Bootstrap | 容器启动拉取 env、写本地配置 |
| Entity | `TenantFeatureParams` | 聚合根；语义为租户 LLM 配置 |
| Domain Service | `FeatureParamsEnvSerializer` | `build_tenant_feature_params_env` |
| Value Object | `TaskLlmProviderEntry` | providers 数组元素（可选后续提取） |
| 跨上下文接口 | `feature-params-env` API | 平台 → 容器的稳定契约 |

本期不引入 Domain Event。

## 价值流影响

### 受影响步骤

| Stream | Step | 影响 |
|--------|------|------|
| `company-management` | `tenant-feature-params` | 字段重命名；测试 `test_manage_feature_params.py` 全面更新 |
| 容器 bootstrap（文档化于 `machine_container.md`） | 拉取形态 YAML → env | `test_container_runtime_tokens.py`、bootstrap e2e |

### `value-stream.yaml` 字段更新（供 `/3-value-stream`）

```yaml
- name: saas-backend.projects_tenant_feature_params.agent_max_steps
# 可选补充：agent_model, summary_model, providers
```

### 测试影响

| 文件 | 变更类型 |
|------|----------|
| `tests/test_manage_feature_params.py` | 字段名、错误文案 |
| `tests/test_container_runtime_tokens.py` | 新端点 `feature-params-env`、响应 `env` |
| `trae-agent/onlineServiceJS/src/featureParamsEnvToYaml.test.mjs` | 新增 |
| `trae-agent/onlineServiceJS/e2e/*.spec.mjs` | mock URL `feature-params-env` |
| `tests/test_ai_task_comment.py` | env key 断言 |

### 跨流依赖

- 容器 bootstrap 依赖本契约；与 relay 换票、heartbeat 无逻辑耦合，仅 URL 变更。
- `machine_container.md`、`machine_container_ai_reference.md`、`onlineServiceJS/skill.md` 需同步。

## 错误处理

| 场景 | 行为 |
|------|------|
| 无效/过期 `access_token` | 401，与现 yaml 端点一致 |
| 路径与 token 不匹配 | 403 |
| 租户无配置 | 返回默认 env（空 providers JSON `[]`，`TASK_AGENT_MAX_STEPS=200`） |
| `TASK_LLM_PROVIDERS_JSON` 解析失败（容器侧） | bootstrap 失败，日志明确；不写入半份 yaml |
| YAML 校验失败 | bootstrap 失败，与现 `YAML.parse` 行为一致 |

## 安全

- `api_key` 仍出现在 env 预览与容器响应中（与现 YAML 预览等价）；页面仅租户管理员可访问。
- `TaskApiKeyUsage` 记录逻辑保留在容器 env 端点 view 中（与现 yaml view 相同）。

## 实施顺序建议

1. Django migration + model + `build_tenant_feature_params_env` + 管理 API 字段重命名  
2. 容器 `feature-params-env` view + 删除 yaml 端点  
3. `featureParamsEnvToYaml.mjs` + bootstrap 切换 + 单测  
4. 前端功能参数页 env 预览 + 字段重命名  
5. job env `TASK_AGENT_*` 重命名 + 相关测试  
6. 文档与 value-stream 字段更新  

## 验收标准

- [ ] 功能参数页展示环境变量预览，无 Trae/Lakeview/YAML 字样  
- [ ] DB 列名为 `agent_*` / `summary_*`，无 `trae_*` / `lakeview_*`  
- [ ] 容器 bootstrap 从 `feature-params-env` 拉取并生成与现网等价的 `service_config.yaml`（Trae 默认运行时）  
- [ ] `tenant_feature_params_yaml.py` 与 `feature-params-yaml/` 已移除  
- [ ] task2app 代码库中 feature-params 路径无 `agents.trae_agent` / `mcp_servers` 模板字符串  
- [ ] 相关 pytest 与 onlineServiceJS 单测通过  

## 决策记录

| 决策 | 选择 | 理由 |
|------|------|------|
| providers 编码 | `TASK_LLM_PROVIDERS_JSON` 单字段 | 与 DB JSONField 对齐，可变数量 provider |
| DB 重命名时机 | **B：本期 migration 一步到位** | 用户确认，避免长期双命名 |
| YAML 过渡 | 无双写，直接切换端点 | 唯一消费者为 bootstrap + 测试 |
| 运行时选择 | 容器 `AGENT_RUNTIME`，非 task2app | 平台对智能体工具无感知 |
