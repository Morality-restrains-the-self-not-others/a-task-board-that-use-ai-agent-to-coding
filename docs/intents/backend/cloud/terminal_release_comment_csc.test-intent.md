# 测试意图：任务终态按评论 CSC 释放运行资源

## 测试目标

证明 `release-servers-on-terminal` 在 v80 语义下：无评论运行态不 DLT；有评论运行态则按台发布停机事件。

## 测试分层

| 层 | 覆盖 |
|----|------|
| 单元 | `list-by-task` 过滤模板行；handler 三态（无资源 / 单评论 / 双评论）；Starting 无 instance no-op；Starting+instance 须停机 |
| 契约 | `CLOUD_SERVER_STOPPED` 含 `comment_id`/`instance_id`/`region_id` |
| 不测本轮 | Playwright 终态 UI；bootstrap 失败根因；sibling LoadForTask |

## 用例矩阵

| # | 预置 | 动作 | 期望 |
|---|------|------|------|
| T1 | 仅模板行，计数 0 | 终态 `TASK_STATUS_CHANGED` | success，0 条停机事件 |
| T2 | 评论 A Running `i-a` | 同上 | 1 条停机，`comment_id=A` `instance_id=i-a` |
| T3 | 评论 A、B 均 Running | 同上 | 2 条停机 |
| T4 | 评论 Starting、无 instance | 同上 | success，0 条停机 |
| T5 | 评论有 `server_url` 无 instance | 同上 | graceful/local stop，不因空 instance retry |
| T6 | 非终态列 | 事件 | success，不调 list |
| T7 | 评论 Starting **有** instance | 同上 | 1 条 `CLOUD_SERVER_STOPPED`（`TestDispatchStartingWithInstancePublishesStop`） |
| T8 | 任务 cancelled，CSC 仍 Running | heartbeat | 410 `TASK_TERMINAL`，`terminal_released=1`，instance 清空 |
| T9 | 任务进行中 | heartbeat | 200，不释放 |
| T10 | 任务 cancelled，无后续心跳 | `reconcileTerminalTaskCSCs` | cleaned=1，instance 清空 |
| T11 | terminal-kinds 查找失败 | heartbeat / 对账 | fail-open：不 410；对账返回 error 且不释放 |

## 数据与环境

- 不连真 Kafka / 真云；handler 用 mock loader/publisher。
- list-by-task 用内存 CSC 表或 httptest。

## 通过标准

- 上表 T1–T6 全绿。
- 不再存在「空模板 → DispatchRetryable / running resource not yet available」断言（旧测试须改期望）。
