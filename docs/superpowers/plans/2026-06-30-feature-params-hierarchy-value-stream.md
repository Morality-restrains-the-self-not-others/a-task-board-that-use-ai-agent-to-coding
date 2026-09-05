# Value Stream: 功能参数多层级配置（公司 / 工作空间 / 个人）

> 设计文档: `docs/designs/feature-params-hierarchy.md`
> 权限分析: `docs/designs/feature-params-hierarchy-permission-analysis.md`

## Value Summary

公司管理员、工作空间管理员、普通成员各自在自己权限范围内配置 LLM 功能参数；创建任务时可选择公司默认 / 工作空间覆盖 / 个人自定义配置；每次容器运行自动记录使用的配置快照，可追溯可复现。

## Related Value Streams

- **project-workspace / tenant-feature-params** (`conf/value-stream.yaml` L525): **扩展** — 在现有公司级 CRUD 基础上，增加 workspace + personal 两个层级
- **task-llm-budget-governance / workspace-model-budget-defaults** (`conf/value-stream.yaml` L1724): **协调** — workspace feature params 的 provider 选择与 workspace budget defaults 共享 `workspace_id` 维度
- **2026-05-29-feature-params-env-agent-agnostic**: **修改** — 容器 env 拉取接口不再仅读 `TenantFeatureParams`，改为调用 `FeatureParamsResolver` 多层级解析

## End-to-End Flow

```text
[管理员] 配置公司级 feature params (已有)
  → [工作空间管理员] 可选覆盖工作空间级配置，设置 allow_personal_feature_params
    → [普通成员] 可选创建个人配置（多份，仅允许个人配置的工作空间可用）
      → [任务创建者] 创建/编辑任务时选择配置来源：company / workspace / personal
        → [容器启动] POST feature-params-env → resolver 多层级解析 → 返回 env
          → [审计] 同步写入 TaskFeatureParamsSnapshot（来源 + 完整 env）
            → [用户] 在任务详情可查看历史运行记录（脱敏摘要）
```

## Value Increments

### Increment 0: 修复现有权限缺陷（前置）

**Value to user:** 公司级配置不被非管理员篡改

**Scope:**
- `manage_feature_params` POST 增加 `CompanyMember.is_admin` 强制校验
- 测试: `tests/test_manage_feature_params.py` 增加 403 测试用例

**Depends on:** 无

### Increment 1: 数据模型 + 配置解析服务（Thin Slice）

**Value to user:** 后端基础设施就绪，但尚无 UI 入口

**Scope:**
- Workspace 增加 `allow_personal_feature_params` 字段 + migration
- 创建 `WorkspaceFeatureParams` 模型 + migration
- 创建 `PersonalFeatureParamsConfig` 模型 + migration
- 创建 `TaskFeatureParamsSnapshot` 模型 + migration
- Todo 增加 `feature_params_source` + `personal_feature_params_config_id` 字段 + migration
- 实现 `FeatureParamsResolver` 领域服务（含工作空间治理检查 + 回退逻辑）
- 修改容器 `feature-params-env/` 接口：调用 resolver → 写快照 → 返回 env
- 单元测试: resolver 6 条路径 + 快照写入

**Depends on:** Increment 0 (admin 校验先修复)

**Fields:**
- `saas-backend.projects_workspace.allow_personal_feature_params`
- `saas-backend.projects_workspace_feature_params.use_company_default`
- `saas-backend.projects_workspace_feature_params.providers`
- `saas-backend.projects_workspace_feature_params.agent_model`
- `saas-backend.projects_personal_feature_params_config.name`
- `saas-backend.projects_personal_feature_params_config.user_id`
- `saas-backend.projects_personal_feature_params_config.providers`
- `saas-backend.projects_task_feature_params_snapshot.task_id`
- `saas-backend.projects_task_feature_params_snapshot.source`
- `saas-backend.projects_task_feature_params_snapshot.resolved_env`
- `saas-backend.projects_todo.feature_params_source`
- `saas-backend.projects_todo.personal_feature_params_config_id`

### Increment 2: 管理 API

**Value to user:** 通过 API 管理工作空间和个人配置

**Scope:**
- 工作空间 feature params CRUD API（含权限：workspace admin）
- Workspace `allow_personal_feature_params` 读写
- 个人配置 CRUD API（含 `IsPersonalConfigOwner` 权限类 + user_id 注入防护）
- 任务 feature params binding PATCH API（含三重校验：workspace 允许 + config 归属 + company 一致）
- 任务创建/列表接口扩展 feature_params 字段
- 快照查询 API（脱敏摘要）
- API 测试: 28 条权限用例 + CRUD 功能用例

**Depends on:** Increment 1

**Fields:**
- `saas-backend.projects_workspace_feature_params.agent_max_steps`
- `saas-backend.projects_workspace_feature_params.summary_model`
- `saas-backend.projects_personal_feature_params_config.agent_model`
- `saas-backend.projects_personal_feature_params_config.agent_max_steps`
- `saas-backend.projects_task_feature_params_snapshot.providers_summary`
- `saas-backend.projects_task_feature_params_snapshot.source_display_name`

### Increment 3: 前端 UI

**Value to user:** 在界面上完整使用多层级配置功能

**Scope:**
- 工作空间配置页面（含"允许个人配置"开关 + 公司配置参考）
- 个人配置管理页面（CRUD + 复制）
- 任务创建/编辑表单增加配置选择器（动态过滤个人配置可用性）
- 任务详情展示当前使用的配置 + 历史运行记录面板

**Depends on:** Increment 2

### Increment 4: E2E 测试 + 数据兼容

**Value to user:** 生产就绪，向后兼容

**Scope:**
- 现有任务 `feature_params_source='company'` 零迁移
- Playwright E2E: 完整流程（管理员配 workspace → 用户创建 personal config → 创建任务选择 → 查看快照）
- 删除个人配置/关闭 workspace 开关的回退路径验证

**Depends on:** Increment 3
