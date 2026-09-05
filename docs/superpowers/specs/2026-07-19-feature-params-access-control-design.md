# Feature-Params 访问控制与审计 — 设计

- **日期**: 2026-07-19
- **状态**: 已采用（goal-mode 自动采用，跳过 USER GATE）
- **范围**: `GET/POST /api/tenant/{id}/feature-params/` 与 workspace 同构接口；前端调用方迁移

## 1. 问题

公司级 feature-params GET 对任意活跃成员返回完整 `providers[].api_key` / `extra_env_vars` / `env_preview`。任务详情页（模型下拉、预算开关）也调用该接口，导致密钥在非设置场景泄露。

## 2. 成功标准

1. 公司成员仅在**公司参数设置页**或**工作空间参数设置页**可获取含密钥的完整配置。
2. 任务详情等其它页面不得获取明文密钥；可用脱敏摘要（模型列表、`llm_budget_enabled`）。
3. 每次访问（含拒绝）记录：谁、何时、何种方式（auth + access_context + view）、路径与结果。
4. 既有设置页读写与容器内部 resolve 路径不受影响。

## 3. 方案（选定）

**不**使用 Referer 鉴权（可伪造且仓库无先例）。

| 模式 | 条件 | 响应 |
|------|------|------|
| **full** | 活跃成员 + 请求头 `X-Feature-Params-Access-Context` ∈ {`company_settings`,`workspace_settings`} | 完整配置（含密钥） |
| **summary** | 活跃成员 + `?view=summary` | 脱敏：`api_key` 清空、无 `extra_env_vars` 值、无含密钥的 `env_preview` |
| **拒绝** | 成员但既无合法 context 也非 summary | **403** |

约束：
- 公司接口 full 仅接受 `company_settings`；工作空间接口 full 仅接受 `workspace_settings`（页面与资源对齐）。
- POST 写操作同样要求对应 settings context（防脚本绕过页面）。
- 服务间/容器 `feature-params-env` / internal API **不在本变更范围**。

### Python 例外说明

本变更为**既有 Django 路由行为收紧 + 查询投影**，不新增公网业务 endpoint。符合「修改存量 Django 面」而非「新增接口默认落 Go」。审计表仍属 saas-backend / projects 数据所有权。

### 架构图

**不新增** ArchiMate/PUML 视图：无新服务、无拓扑变更，仅权限与审计落库。

## 4. 审计模型

`FeatureParamsAccessAudit`（表 `projects_feature_params_access_audit`）：

- `user_id`, `company_id`, `workspace_id`（可空）
- `resource` ∈ {company, workspace}
- `access_context`（请求声明或空）
- `view_mode` ∈ {full, summary, denied}
- `auth_method` ∈ {token, session, unknown}
- `http_method`, `path`, `status_code`
- `client_ip`, `user_agent`, `referer`（元数据，不作鉴权）
- `trace_id`, `created_at`

结构化日志同步一条 `info`/`warn`（拒绝为 warn）。

## 5. 前端

| 页面 | 变更 |
|------|------|
| `WorkspaceSettingsFeatureParams.vue` | 请求带 `X-Feature-Params-Access-Context: company_settings` |
| `WorkspaceFeatureParamsSettings.vue` | 请求带 `…: workspace_settings` |
| `taskDetailLayerGraphModelOptions.js` | company/workspace 拉配置改 `?view=summary` |
| `TaskDetailLlmBudgetPanel.vue` | 改 `?view=summary` |

## 6. 事件

访问审计以 DB 行 + 结构化日志为 SSOT；**无 MQ 领域事件**（非跨服务编排意图；书面例外）。

## 7. 测试

- GET 无 context → 403，写审计 denied
- GET + company_settings → 200 含 api_key
- GET `?view=summary` → 200 且 api_key 为空
- 错误 context（workspace_settings 打公司接口）→ 403
- 前端单测更新 URL / header
