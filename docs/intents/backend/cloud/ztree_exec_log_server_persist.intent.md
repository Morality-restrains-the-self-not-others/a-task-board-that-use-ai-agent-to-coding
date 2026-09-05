# 意图：ztree 执行日志服务端持久化（容器关闭后仍可访问）

## 背景与目标

任务详情 ztree 层图今日只经 `layer-graph-push` 打 SSE，Cloud 不落库；容器关闭后刷新树空。Agent 步骤已有 `cloud_job_execution_event`。**用户锁定：不存克隆日志。**

目标：以 taskCloudService 为 owner，按 **workspace_id / task_id** 访问，把 **层图快照** 落到 MySQL JSON 列（**不是**宿主机 JSON 文件）；关容器后 ztree 与 job 步骤仍可读。克隆进度/clone-log 仍仅容器内存。

## 范围与边界

- 范围内：`cloud_layer_graph_snapshot`；layer-graph-push UPSERT；GET `container-layer-graph` Cloud hydrate；FE 在 endpoint 未就绪时仍展示树与 job 步骤；事件 `LayerGraphSnapshotPersisted`；023 job hydrate 保持
- 范围外：**克隆日志 / exec-stream 落库**；本地 JSON 文件 SSOT；COS；stdout chunk；文件树内容；Loki 采集；替换 023 表

## 约束与风险

- 表前缀 `cloud_`、utf8mb4
- 查询必须带 workspace_id + task_id
- 禁止业务进程 ticker；禁止新增 Python 接口
- 旧容器从未 PUSH 的树不可还原

## 验收标准

1. 停止容器后硬刷新：ztree 来自快照，job 步骤来自 023
2. 克隆区关容器后允许空白
3. 层图 UPSERT 不插出行；无宿主机 JSON SSOT

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 容器上报层图 | LayerGraphSnapshotPersisted | Kafka/SSE 契约含 workspace_id/task_id/comment_id | taskCloudService `handleLayerGraphPush` | SSE 直播；表 UPSERT 在发布前同聚合完成 | — |
| 页面拉取历史层图/步骤 | — | — | GET Cloud | 读库 | 纯查询 |
| 页面拉取克隆日志 | — | — | GET Gateway→容器 | — | 纯查询；不落库（用户锁定） |

## 实施计划

见 `docs/superpowers/specs/2026-08-20-ztree-exec-log-server-persist-design.md`。

## 变更记录

- 2026-08-20：初版含克隆日志落库。
- 2026-08-20：用户锁定不存克隆日志；范围收窄为层图快照 + 既有 job 步骤。
