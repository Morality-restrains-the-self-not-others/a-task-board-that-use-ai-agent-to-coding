# 功能参数来源标识环境变量（TASK_FEATURE_PARAMS_SCOPE）设计

- **日期**: 2026-07-09
- **状态**: approved（goal-mode 自动采用）
- **作者**: claude
- **迭代**: feature-params-scope-env

## 1. 背景与目标

公司 / 工作空间 / 个人三套 feature-params 共享同一套环境变量体系。容器与脚本消费 env 时，无法从现有 `TASK_*` 系统变量判断「当前生效配置来自哪一层」。

**目标**：增加只读固定系统环境变量 `TASK_FEATURE_PARAMS_SCOPE`，取值随作用域变化，便于运行时区分来源。

## 2. 成功标准

1. 公司设置页预览含 `TASK_FEATURE_PARAMS_SCOPE=company`
2. 工作空间设置页：公司默认 Tab → `company`；自定义 Tab → `workspace`
3. 个人配置页预览含 `TASK_FEATURE_PARAMS_SCOPE=personal`
4. 容器 `feature-params-env` 拉取结果含该变量，值等于解析后的 `SnapshotMeta.source`（含回退到 company 的情形）
5. 用户无法通过 `extra_env_vars` 覆盖该键（序列化时系统值最后写入；保存校验仍拒绝 `TASK_*`）
6. 单元测试覆盖序列化与三作用域预览

## 3. 方案决策（已选定）

| 选项 | 结论 | 理由 |
|------|------|------|
| 变量名 | `TASK_FEATURE_PARAMS_SCOPE` | 与现有 `TASK_*` 系统契约一致；保留前缀已禁止用户自定义 |
| 取值 | `company` / `workspace` / `personal` | 复用 `FeatureParamsSource`，与任务来源选择器一致 |
| 注入点 | `FeatureParamsEnvSerializer` + 应用服务传入 scope | 单一真源；设置页预览与容器拉取共用 |
| 覆盖策略 | **系统值在用户变量之后写入** | 保证只读；即使脏数据进入 `extra_env_vars` 也无法覆盖 |
| 工作空间继承 | 选「公司默认」时预览为 `company`；运行时 workspace 源且继承公司时 meta.source=`company` | 与现有 resolver 语义一致：标识「实际生效配置来源」 |

## 4. 架构影响

**无新组件 / 无新持久化 / 无新 HTTP 路径。** 仅扩展既有 `FeatureParamsEnvSerializer` 契约与三页前端 `systemEnv`。

不新增 `docs/architecture/` v12 视图（非拓扑变更）。

## 5. 改动清单

### 后端

- `feature_params_env_serializer.py`：常量 + `serialize(..., scope=...)` / `serialize_from_fields(..., scope=...)`
- `tenant_feature_params_env.build_tenant_feature_params_env`：透传 `scope`
- `FeatureParamsApplicationService.resolve_and_snapshot`：用 `meta.source.value` 注入
- 公司 / 工作空间 / 个人 views 的 `env_preview`：传入对应 scope
- 测试：`test_tenant_feature_params_env.py` 等

### 前端

- `WorkspaceSettingsFeatureParams.vue`：`systemEnv` 加 `company`
- `WorkspaceFeatureParamsSettings.vue`：按 Tab 加 `company`/`workspace`
- `PersonalFeatureParamsConfigs.vue`：加 `personal`

### 文档

- `docs/intents/frontend/feature_params/002_*.intent.md` + `.test-intent.md`
- 价值流 / NFR / plan 制品（本流水线后续步骤）

## 6. 非目标

- 不改变任务来源选择器 UI
- 不新增用户可编辑的「来源」字段
- 不修改 `extra_env_vars` 存储结构
