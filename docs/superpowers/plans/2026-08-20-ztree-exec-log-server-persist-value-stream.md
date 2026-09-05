# 价值流 — ztree 执行日志服务端持久化

- **Date:** 2026-08-20
- **Design:** `docs/superpowers/specs/2026-08-20-ztree-exec-log-server-persist-design.md`

## Related Value Streams

扩展 `conf/value-stream.yaml` 中 **`task-container-gateway`** 已有步骤，不是绿场：

| 既有步骤 | 已交付 | 本增量 |
|----------|--------|--------|
| `container-job-stream-push-kafka` | 容器 job-stream → SSE | 不改 |
| `job-stream-persist-from-kafka` | 023 步骤落库 | 不改 |
| `job-stream-hydrate-from-db` | GET job-execution-log 读库 | 保持 |
| `frontend-exec-log-hydrate-then-sse` | 无 endpoint 仍拉 job 步骤 | **扩展**：无 endpoint 仍拉层图快照；克隆区仍允许空 |

不撤销 clone-log 代理容器。无冲突。

## 价值增量

| # | 增量 | 用户可见结果 | 验证 |
|---|------|--------------|------|
| 1 | 层图 PUSH 落库 | 同评论后写覆盖，一行 | Cloud UPSERT 单测 T2/T3 |
| 2 | GET 层图 Cloud hydrate | 关容器/无 Gateway 仍 200 出树 | GET 单测 T4；跨 ws 空图 |
| 3 | FE 无 endpoint 仍拉树 | 硬刷新 ztree 有节点；克隆区可空 | `taskDetailContainerFns.commentId.test.js` |
| 4 | 事件 LayerGraphSnapshotPersisted | 成功落库后投递 | T8 spy publish |

触发用户：任务详情打开评论执行细节（容器可关）。

## YAML 字段（三段式）

- `task-cloud-service.cloud_layer_graph_snapshot.graph_json`
- 已有 `taskCloudService.job_execution_log.read_db`
