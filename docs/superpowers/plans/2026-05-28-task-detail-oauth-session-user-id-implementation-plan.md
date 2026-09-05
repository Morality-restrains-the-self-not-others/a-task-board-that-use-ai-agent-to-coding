# 实施计划: 任务详情 OAuth session userId 解析

> 设计: `docs/superpowers/specs/2026-05-28-task-detail-oauth-session-user-id-design.md`  
> 价值流: Increment 1 only

## Tasks

- [x] **Task 1:** 新增 `sessionUserIdUtils.js` + 单元测试（cookie / profile 回退 / inflight）
- [x] **Task 2:** `TaskDetailLinkedProjectsPanel.vue` — `resolveOauthConnectionApiBase` 改为 async + 接入 resolver
- [x] **Task 3:** `TaskDetailLinkedProjectsPanel.test.js` — 无 cookie 场景回归
- [x] **Task 4:** 运行 Vitest 全绿
- [ ] **Task 5:** 手工验收任务详情 OAuth 绑定（可选 relayToTrae URL）

## 验证命令

```bash
cd task2app/front_project/app && npm test -- --run \
  src/utils/sessionUserIdUtils.test.js \
  src/components/task-detail/TaskDetailLinkedProjectsPanel.test.js
```
