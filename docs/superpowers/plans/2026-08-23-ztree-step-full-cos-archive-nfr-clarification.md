# NFR 澄清：ztree step_full COS 归档

- **Date:** 2026-08-23
- **Level default:** L2；资金路径不适用。对象存储可用性 L3（失败回退 023 摘要）。

## 质量场景

| 场景 | 度量 | 目标 |
|------|------|------|
| 关容器后刷新执行日志 | 有 COS 对象则 200 + source=saas_cos | L2 |
| COS 不可用 | 回退 023 或 local payload，不 5xx 挡整页 | L2 |
| 单对象大小 | 默认 cap 8MiB；超限拒绝并打 warn | L2 |
| 管理员改配置 | PATCH 热加载，密钥不落日志 | L3 安全 |

## 路径分片键强制审视

| 路径 | 已带 ID | 分片键判定 | 等级 | 动作 |
|------|---------|------------|------|------|
| POST job-step-full-push（compute path tenant/ws/task + comment） | tenant, workspace, task, comment, job | workspace_id 适合租户分库；job 为实体后缀 | L1 | 表 UNIQUE(ws,task,comment,job) |
| GET container-job-execution-log | 同上 | 禁止只靠 job_id | L1 | 已强制 workspace+task |
| COS key `workspace_{wid}/task_{tid}/comment_{cid}/...` | ws, task, comment | key 前缀即分区 | L1 | pathRule 必须含三键 |
| PATCH /api/system-admin/step-full-cos/ | 无租户键 | 平台级配置，L0 | L0 | 理由：全局基础设施，非租户数据面 |
| Kafka JobStepFullArchived | payload 含 ws/task/comment/job | 键 `ws:task:comment:job` | L1 | publish-only |

## 幂等性强制审视

| 路径 | 副作用 | 重复触发源 | 业务重复边界 | 幂等键 | 重放语义 | 键粒度 | 判定 |
|------|--------|------------|--------------|--------|----------|--------|------|
| job-step-full-push | 写 COS+表 | job close 重试 / at-least-once | 同一 job 全文 | (ws,task,comment,job) | 覆盖同 job；合并进 comment bundle | 同边界 | L2 |
| 管理员 PATCH COS | 写 conf | 双击保存 | 全局一份配置 | 平台单例 last-write-wins | 后写覆盖 | 配置级 | L1 |
| GET hydrate | 无 | — | — | — | — | — | L0 |

禁止用 company_id/user_id 作归档幂等键。
