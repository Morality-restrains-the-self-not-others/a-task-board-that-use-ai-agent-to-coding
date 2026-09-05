# 评论启动日志「同一行为、不同时间」重复行

- **Status:** accepted
- **Date:** 2026-08-13
- **Iteration:** comment-startup-log-timezone-dedupe
- **Based on:** 任务详情启动日志重复（`p.text-gray-600.mb-1`）；Loki `trace_id=bb1157e1b936e3d8eb7e7b05`
- **Architecture impact:** **否** — 不增删服务/事件；修正既有 binding 日志时间解析与前端去重
- **ADR:** No-ADR: trivial tech choice, no architectural impact
- **python_api_approval:** not_applicable
- **Decision:** 方案 A — 写入侧 UTC 墙钟按 UTC 读回并 JSON 输出 RFC3339 `Z`；展示侧本地 `HH:mm:ss`；去重键忽略时间前缀与 `trace_id=` 后缀，优先保留带 TraceId 的行

---

## 0. 问题

页面启动日志同一 UserData 步骤出现两次，时钟差 8 小时：

| 展示 | 特征 | 来源 |
|------|------|------|
| `[23:30:59] [i-…、cmt_…] 容器运行时服务已就绪` | 无 `trace_id=` | 浏览器 SSE `toLocaleTimeString()`（CST） |
| `[15:30:58] [i-…、cmt_…] 容器运行时服务已就绪 trace_id=bb1157…` | 有后缀 | list API `created_at` 把 UTC 墙钟标成本地/带 `+08:00` |

Loki `{job=~".+"} | json | trace_id="bb1157e1b936e3d8eb7e7b05"`：事件 `ts` 为 `2026-08-13T23:27:27+08:00` … `23:31:00+08:00`。**真实时刻是 23:xx CST**；15:xx 是 UTC 墙钟未按 UTC 解释。

## 1. 根因

1. **双通道**：`publishTaskSSE` 推 SSE（前端立刻 `appendBindingLogLine`）同时 `logServerSchedulingToBindingBestEffort` 落库；`refreshBindings` → `mergeBackendBindingLogs` 再灌一遍。
2. **时区**：插入 `time.Now().UTC().Format("2006-01-02 15:04:05")`（无时区 DATETIME）；读回只 `Parse(RFC3339)`。驱动 `loc=Local` 常把同一墙钟标成 `+08:00`，JSON 后浏览器显示 15:xx；SSE 用本地时钟显示 23:xx。
3. **去重过严**：`line.includes(fullMessage)`。落库文案经 `serverSchedulingLogMessage` 追加 `trace_id=`，与 SSE 原文不成子串，两条都留下。

## 2. 决策（方案 A）

1. Go：`parseCloudUTCDateTime` 抽取 `YYYY-MM-DD HH:MM:SS` **一律当 UTC**；list `logs[].created_at` 输出 RFC3339 `…Z`。
2. FE：无时区字符串当 UTC；有 `Z`/offset 走 `Date`；展示本地 24h `HH:mm:ss`。
3. FE：`startupLogDedupeKey` = 去掉 `[时:分:秒]` 与末尾 `trace_id=\S+`；冲突时**保留带 `trace_id=` 的行**。

不新增 Kafka 事件。不把 `trace_id=` 从落库文案删掉（存量冷打开仍靠后缀回填 `start_trace_id`）。

## 3. 方案对比

| 方案 | 结论 |
|------|------|
| A. UTC 读回 + RFC3339 Z + 规范化去重 | **采用** |
| B. 只去重、不修时区 | 拒绝：冷打开仍显示 15:xx |
| C. 停写 binding 日志、只信 SSE | 拒绝：刷新丢失历史 |

## 4. 业务意图 → 事件对照

| 业务意图 | 事件名 | 例外理由 |
|---------|--------|---------|
| 启动日志按本地时钟去重展示 | — | 纯展示/既有 SSE 持久化修正，无新领域事实 |

## 5. 🕸️ Code Review Graph 分析

| 项 | 内容 |
|----|------|
| 图状态 | CRG unavailable for Go `taskCloudService`；以源码与 Loki 为准 |
| 爆炸半径 | `comment_container_bindings_store.go` 时间解析；`createBindingStatusLogs.js`；`bindingServerStartupLogs.js` |
| skip 理由 | Go 符号不在 CRG 图内，不阻断 |
