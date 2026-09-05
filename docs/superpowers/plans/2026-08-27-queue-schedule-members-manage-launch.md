# Launch checklist — queue schedule members manage

- 日期：2026-08-27

## Pre-launch

- [x] 相关单测通过（QueueMembersCard 5 + composable 7 + page 6）
- [x] taskFE `npm run build`（releases/20260827214610-501574）
- [x] Anti-replay `--strict` 通过
- [x] 无新 DB 迁移
- [x] 无新 ADR（复用既有排队调度）
- [x] 安全清单见 review 文档
- [x] 可观测性：join/leave/patch 结构化 console 事件名

## Feature flag

无。增量挂在既有 `/queue-schedule/` 页，对未打开该页用户无影响。

## Rollback

1. 回滚 taskFE 该提交 / 切回上一 SPA release symlink（<5 分钟）
2. 无 DB 变更
3. 入队/出队后端行为未改；最坏为 UI 回到只读列表

## 触发回滚

- 排队卡加入/离开误伤其他任务（IDOR）— 未改鉴权，风险低
- SPA 构建后静态 404
