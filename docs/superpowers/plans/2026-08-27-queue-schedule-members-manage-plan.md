# 自动调度安排 · 排队任务卡可管理 — 实施计划

- 日期：2026-08-27

## 任务

- [ ] **T1 意图文档**  
  `docs/intents/frontend/queue_schedule_members_manage.intent.md` + `.test-intent.md`；INDEX；value-stream.yaml；flows wsd。

- [ ] **T2 Red：composable PATCH**  
  `useWorkspaceQueueSchedule.test.js`：`patchQueuedAutoRun` 发 PATCH + Idempotency-Key，成功后再 GET。

- [ ] **T3 Green：composable**  
  实现 `patchQueuedAutoRun`；修正 `request()` headers 被 `...options` 覆盖导致丢失 Accept。

- [ ] **T4 Red：QueueMembersCard**  
  未启用不 PATCH；搜索入队；离开确认/取消；错误 data-traceId；刷新 L1。

- [ ] **T5 Green：QueueMembersCard.vue**  
  抽出卡片；搜索 debounce；modalService；clickGuard。

- [ ] **T6 接入页面**  
  `WorkspaceQueueSchedule.vue` 使用卡片；刷新走 `reload`（修 OPT-041）；save 补 clickGuard。

- [ ] **T7 页面测例**  
  刷新点击再 GET；加入队列按钮可见。

- [ ] **T8 跑测 + SPA build**  
  vitest 相关文件；`taskFE/app` `npm run build`；登记精准重启 taskFE。
