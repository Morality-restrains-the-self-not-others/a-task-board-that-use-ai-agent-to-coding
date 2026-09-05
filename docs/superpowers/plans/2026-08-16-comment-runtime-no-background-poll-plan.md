# 实施计划 — 评论运行态推送同步

- **日期**: 2026-08-16
- **设计**: `docs/superpowers/specs/2026-08-16-comment-runtime-no-background-poll-design.md`

## Tasks

- [x] 删除 taskFE runtime/startup/binding UI `setInterval` Describe
- [x] 挂载与 SSE 成功路径禁止自动 GET Describe；只 apply snapshot
- [x] 「刷新状态」为唯一 Describe，带面板 comment_id
- [x] 启动成功 SSE 带 `runtime_status=Running`
- [x] 停止命令 SSE 走 `publishTaskSSE` + Stopping
- [x] `CLOUD_SERVER_STOPPED` envelope 带 `comment_id`；完成 SSE 带 `Stopped`
- [x] 遗留看板 `startLegacyStartupStatusPoll` 改为 no-op
- [x] 架构 v84 四类伴生；OPT-018 取消
- [x] 测例：runtimeStatusRefreshPolicy / RuntimeStatusSection / cloudserverstopped / stop event / legacy poll

## Intent → Event

对照 `docs/intents/frontend/comment_runtime_no_background_poll.intent.md`：刷新为纯查询例外；停机完成走既有 `CLOUD_SERVER_STOPPED` + SSE。

## Step 9 Review（2026-08-16）

| 轴 | 结论 |
|---|---|
| Correctness | 直播只 apply `comment_id`+`runtime_status`；Describe 仅按钮；停机完成 SSE 带 `Stopped`；无 comment_id 不回落 `forPoll()` |
| Readability | startup/legacy poll 保留 no-op 入口，避免残留调用方再引入 timer |
| Architecture | v84 current；ADR-0011；周期工作仍走既有 Kafka/SSE，未新增 HTTP |
| Security | 刷新 path 必须带该面板 comment_id；无密钥入日志 |
| Performance | 去掉 5s/30s Describe；uptime `setInterval` 仅改本地文案 |
| Logs | 复用 `publishTaskSSE` / cloudserverstopped 既有结构化路径；前端无周期请求可观测点 |

Critical/Required：无。积压：OPT-022（存量 ticker）、OPT-024（其它 SPA API 轮询）。
