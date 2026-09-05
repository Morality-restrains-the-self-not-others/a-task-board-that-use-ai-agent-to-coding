# Value Stream: 任务详情陈旧仓库地址同步按钮修复

> Derived from: `docs/superpowers/specs/2026-05-27-task-detail-stale-repo-sync-button-design.md`

## Value Summary

用户在任务详情看到「仓库地址已变更」时，可一键 PATCH 同步任务记录，并在 relay 直接启动前被正确阻断或确认。

## End-to-End Flow

[Trigger: 任务 projects 中 repo_address_mismatch=true] → [关联项目显示徽章 + 同步按钮] → [用户点击同步] → [PATCH todos/projects] → [徽章消失、relay 横幅消失] → [可正常启动 relay]

## Value Increments

### Increment 1: relay 面板接线（Essential）
- `ServerConfig.logic.vue` 接入 `taskRepoAddressMismatch.js`
- 横幅、阻断启动、确认不更新

### Increment 2: 关联项目同步按钮（Core UX）
- 徽章旁「同步仓库地址」按钮
- 复用 PATCH 逻辑

## Test Files

- `task2app/front_project/app/src/utils/taskRepoAddressMismatch.test.js`
- `task2app/playwright/front_project/tests/TaskDetail.relay-to-trae-stale-repo-address.playwright.test.js`
