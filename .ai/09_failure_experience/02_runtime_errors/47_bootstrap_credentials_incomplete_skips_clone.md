# [运行时] repo-clone-credentials 409 导致引导永不克隆（空 /app）

## 基本信息

- 版本：1.0.0
- 创建日期：2026-07-18
- 最后修改：2026-07-18
- 维护者：Trae AI 团队

## 现象

- code-server（如 `http://<ip>:8888/?folder=/app`）可打开，但 `/app` 下无关联仓库。
- `GET /api/repos/bootstrap-clone-log` 返回 `{"layer_id":null,"text":""}`；`/layers` 的 `bootstrap_layer_id` 为 null。
- 容器 HTTP（8765）与心跳正常，看似「服务已就绪」。

复现任务示例：`task_13478486238904149865` @ `47.239.98.3`（2026-07-18）。

## 根因

1. 容器 listen 后 bootstrap：`task-detail` 200 → `repo-clone-credentials` **409** `REPO_CLONE_CREDENTIALS_INCOMPLETE`（关联仓 Git 授权尚未齐）。
2. `fetchBootstrapRepoInputs` **一次性失败即抛错**，`runBootstrapAfterListen` 在 `clone_begin` 之前退出；**从不调用** `cloneReposIntoSharedLayer`。
3. 无退避重试、无延迟恢复；用户稍后补绑后 credentials 已 200，但进程不会再跑引导克隆。
4. 失败时 clone-log 仍为空，前端/8888 难以区分「未克隆」与「进行中」。

## 解决方案

`trae-agent/onlineServiceJS`：

1. `postRepoCloneCredentialsWithRetry`：对 `REPO_CLONE_CREDENTIALS_INCOMPLETE` 按 env 退避重试（`TASK_API_REPO_CLONE_CREDENTIALS_RETRIES` / `_BACKOFF_MS`）。
2. `noteBootstrapFailure` + `bootstrapCloneLogFailurePayload`：`GET bootstrap-clone-log` 在无 layer 时仍返回可读失败摘要。
3. `scheduleBootstrapCredentialsRecovery`：首轮仍失败后周期性再跑引导（间隔/轮次可配），成功后注册 clone job。

## 预防

- 凭证类 409 不得「失败一次永久跳过克隆」。
- 引导失败必须可经 clone-log / 日志检索（`BOOTSTRAP_FAILED` / `REPO_CLONE_CREDENTIALS_INCOMPLETE`）。
- 补绑 Git 后应能在同一容器生命周期内恢复克隆，而不必依赖人工重启。

## 验证

```bash
cd trae-agent/onlineServiceJS && node --test src/bootstrap.cloneCredentials.test.mjs
```

公网生效需重建并推送 onlineServiceJS 镜像（见 `onlineServiceJS/ai.md`）。
