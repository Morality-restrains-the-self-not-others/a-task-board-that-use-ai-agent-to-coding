# [运行时] 容器日志缺少 TASK_FEATURE_PARAMS_SCOPE

## 失败现象

任务详情页 `relayToTrae=true` 启动后，启动日志 / `feature-params-env pulled` 快照中看不到 `TASK_FEATURE_PARAMS_SCOPE`（及 CONFIG_ID/NAME）。

## 环境与上下文

- 代码已合入 `task2app`（`9825540b` / `a3aa764d`）
- 运行时为 gunicorn（`runall-saas-backend.sh`，无 `--reload`）
- 公司可能尚无 `TenantFeatureParams` 记录，容器 API 走 `_env_from_params` 回退

## 排查过程

1. 确认序列化器与 ApplicationService 已注入三键（单元测试通过）
2. 对照 `onlineProject_state/logs/feature-params-env.log`：拉取结果无 SCOPE
3. 对照进程启动时间与提交时间：gunicorn 早于代码合入，未重载
4. 确认该公司 `TenantFeatureParams` 为空 → 走回退路径；回退路径在新代码中已传 `scope=company`

## 根因

1. **主因**：gunicorn 未在代码合入后重启，运行中的 worker 仍是旧字节码，回退/序列化均不写 SCOPE
2. **次因（体验）**：`persistFeatureParamsEnv` 原先只写 YAML + 日志，未写入 `process.env`；若用户用 `printenv`/子进程检查会误判「未注入」

## 解决方案

1. 重启 `saas-backend`（runAll `/api/restart` 或等价脚本）使新代码生效
2. `persistFeatureParamsEnv` 在落盘前 `applyFeatureParamsEnvToProcess`
3. 容器 API 回归：有/无公司配置两种路径均断言 SCOPE 三键

## 预防措施

- 合入影响 gunicorn 加载的 Python 改动后，必须重启 `saas-backend`
- 容器 env 契约变更须有 `test_container_runtime_tokens` 级断言，避免仅测序列化器
- 文档明确：init.log 早于 feature-params 拉取；以 `feature-params-env pulled` 行为准
