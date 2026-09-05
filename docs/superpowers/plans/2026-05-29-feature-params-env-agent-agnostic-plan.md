# 实施计划: 功能参数环境变量下发与智能体运行时解耦

> 设计: `docs/superpowers/specs/2026-05-29-feature-params-env-agent-agnostic-design.md`  
> 价值流: `docs/superpowers/plans/2026-05-29-feature-params-env-agent-agnostic-value-stream.md`

## Task 1: DB migration + model

- [x] `0039_rename_tenant_feature_params_agent_fields.py`
- [x] `TenantFeatureParams` 字段重命名

## Task 2: env 序列化 + 容器 API

- [x] `projects/domain/tenant_llm_config/*`
- [x] `cloud/services/tenant_feature_params_env.py`
- [x] `fetch_tenant_feature_params_env_for_container`
- [x] 删除 `tenant_feature_params_yaml.py`
- [x] URL `feature-params-env/`

## Task 3: 管理 API + 前端

- [x] `feature_views.py` 新字段 + `env_preview` + 拒绝 legacy 字段
- [x] `WorkspaceSettingsFeatureParams.vue` 环境变量预览
- [x] `useTaskDetail.js`

## Task 4: onlineServiceJS

- [x] `featureParamsEnvToYaml.mjs` + 单测
- [x] `bootstrap.mjs` 切换
- [x] e2e mock 更新

## Task 5: job env 重命名

- [x] `ai_instruct_stream_service_stream_fns.py`
- [x] `forward_container_job_edit_run.py`
- [x] `forward_container_layer_command.py`

## Task 6: 测试与文档

- [x] `test_manage_feature_params.py`
- [x] `test_container_runtime_tokens.py`
- [x] `test_tenant_feature_params_env.py`
- [x] `test_ai_task_comment.py`
- [x] `normalizeJobCommandEnv.mjs` — job env `TASK_*` → `TRAE_*` 桥接
- [x] `machine_container.md` §4.3 env 响应文档
- [x] 前端 env 预览实时计算

## 验证命令

```bash
cd task2app/Saas_project && pytest tests/test_manage_feature_params.py tests/test_container_runtime_tokens.py tests/test_tenant_feature_params_env.py -q
cd trae-agent/onlineServiceJS && node --test src/featureParamsEnvToYaml.test.mjs
```
