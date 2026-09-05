# 意图：ztree 层级落库与 step_full COS 归档

## 背景与目标

任务详情 ztree 树已有 `cloud_layer_graph_snapshot`。执行日志全文 `agent_step_full.json` 只在容器盘。关容器后无法复查。目标：保持层图 MySQL；job 终态把 step_full 归档到 COS；GET 优先 COS；管理员可配 COS。

## 范围与边界

- 范围内：快照继续 UPSERT；`job-step-full-push`；表 `cloud_job_step_full_object`；GET hydrate COS-first；系统管理 COS 页；事件 `JobStepFullArchived` / `StepFullCOSConfigUpdated`
- 范围外：克隆日志 COS；容器持有密钥；Python API；替换 023 表

## 约束与风险

- 表前缀 `cloud_`、utf8mb4、Snowflake 主键
- 查询必须带 workspace_id + task_id + comment_id
- 禁止业务 ticker；禁止环境 Proxy
- 密钥不进 git、不进 GET 响应

## 验收标准

1. 停止容器后硬刷新：ztree 来自快照；执行日志来自 COS（或 local 回退），`source=saas_cos`
2. 同 comment 第二次 job 归档不丢第一次 job 的 steps（bundle 按 job_id 合并）
3. 非平台员工 403 管理员 COS API
4. 无宿主机 JSON 文件 SSOT

## 业务意图 → 事件对照

| 业务意图 | 事件名 | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|--------|-----------|--------|--------------|---------|
| job 终态归档 step_full | JobStepFullArchived | Kafka；键 ws:task:comment:job | handleJobStepFullPush | publish-only | — |
| 管理员改 COS 配置 | StepFullCOSConfigUpdated | Kafka | admin PATCH | publish-only | — |
| 页面拉取历史日志 | — | — | GET Cloud | 读 COS/DB | 纯查询 |

## 实施计划

见 `docs/superpowers/plans/2026-08-23-ztree-step-full-cos-archive-plan.md`。
