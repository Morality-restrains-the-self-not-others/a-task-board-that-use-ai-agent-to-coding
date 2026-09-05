# 实施计划 — TASK_FEATURE_PARAMS_SCOPE

- **日期**: 2026-07-09
- **设计**: `docs/superpowers/specs/2026-07-09-feature-params-scope-env-design.md`

## Tasks

### Task 1 — 红：序列化测试

- [x] 在 `test_tenant_feature_params_env.py` 增加：
  - `scope=company` 时输出 `TASK_FEATURE_PARAMS_SCOPE=company`
  - 用户 `extra_env_vars` 含同名键时仍被系统值覆盖
  - 未传 `scope` 时不注入该键（兼容旧调用）——**决定：未传则不注入**

### Task 2 — 绿：Serializer + build helper

- [x] 修改 `feature_params_env_serializer.py`
- [x] 修改 `tenant_feature_params_env.py` 透传 `scope`

### Task 3 — 应用服务与 API 预览

- [x] `FeatureParamsApplicationService` 传入 `meta.source.value`
- [x] 公司 / 个人 views：`scope=company` / `personal`
- [x] 工作空间 serialize：`use_company_default` → `company`，否则 `workspace`；继承公司时用公司字段 + `scope=company`
- [x] `cloud_compute_views` 个人预览：`scope=personal`

### Task 4 — 前端三页 systemEnv

- [x] 公司页固定 `company`
- [x] 工作空间页按 Tab
- [x] 个人页固定 `personal`

### Task 5 — 意图文档与索引

- [x] `002_功能参数来源标识环境变量.intent.md` + test-intent
- [x] 更新 `intent_index.md`

### Task 6 — 跑测试验证

- [x] `pytest` 相关文件通过（28 passed）
