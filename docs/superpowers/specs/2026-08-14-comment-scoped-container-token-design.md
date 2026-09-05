# 评论级容器令牌

- **Date:** 2026-08-14
- **Status:** accepted（goal-mode）
- **ADR:** ADR-0005

## 目标

同一 Task 下两个评论各有独立 container access/refresh；先启动容器的 `exchange-refresh` 不再让后到评论 403。

## 一致性边界

`ContainerToken` 按 `(tenant, workspace, task, comment)` 一行。校验路径兼容无 comment 的旧容器。

## API

| 路径 | comment |
|------|---------|
| `POST /v1/token/init/tenant/{t}/workspace/{w}/task/{tk}/comment/{c}` | 路径必填 |
| 旧 9 段 init | query/body `comment_id`，缺则 400 |
| `GET /v1/token/by-scope` | query `comment_id`；缺且多行 → 须补 comment |
| `exchange-refresh` / `refresh-access` | body 可选 `comment_id` |

## 路径分片键审视

| 路径 | 已有 ID | 分片键判定 | 可伸缩性 | 动作 |
|------|---------|------------|----------|------|
| token init / by-scope / container-token callbacks | tenant + workspace + task + comment | **tenant_id** 为租户分片键；comment 为任务内隔离键 | L2 | comment 不替代 tenant 分片；库内 UNIQUE 含 comment |
| Cloud start-vm-auto → token init | 同上 | tenant 合适 | L2 | bootstrap URL 必须带 comment |
| JS exchange-refresh / register-reachability | URL 仍 task 级；body 补 comment | tenant 在 URL | L1 | body `comment_id` 防串票 |

升级触发：单租户 token 行数年增量 > 100 万再评估按 tenant 分库。

## 数据

`dataMigrate/taskCredentialService/002_comment_id.sql`：加列、去重、UNIQUE。utf8mb4。业务进程不 migrate。

## 验收

1. 两评论各 IssueToken → 两行；A exchange 不影响 B。
2. IssueToken 无 comment_id → 失败。
3. Cloud bootstrap URL 含 `/comment/{id}`。
4. JS exchange/reachability 带 `COMMENT_ID`。
