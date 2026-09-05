# Value Stream: 任务详情关联项目名称可点击跳转

> Derived from design: `docs/superpowers/specs/2026-05-28-task-detail-linked-project-name-link-design.md`

## Related Value Streams

- `2026-05-27-task-detail-stale-repo-sync-button-value-stream.md` — 同面板「关联项目」UX 增强，无冲突。
- `2026-05-21-task-detail-api-ddd-value-stream.md` — task-detail 主链路；本增量为只读导航，不触及 API。

## Value Summary

任务协作者从任务详情「关联项目」一键进入项目详情，查看仓库/OAuth/配置，减少复制项目名或回项目列表查找的步骤。

## Capability Classification

- **Core value**: 项目名称 → 项目详情 SPA 导航
- **Essential support**: 既有 `project_id`、`tenantId`、路由 `project_detail`
- **Enhancement**: hover 样式、`data-testid`
- **Future**: 无

## End-to-End Flow

[Trigger] 用户打开任务详情，看到关联项目  
→ [Stage 1] 项目名称渲染为 `router-link`  
→ [Stage 2] 点击导航至 `/tenant/{tenant}/projects/{id}/`  
→ [Delivery] 项目详情页加载

## Value Increments

### Increment 1: 关联项目名称链接（唯一增量）

**Value to user:** 点击项目名进入项目详情。  
**Scope:** `TaskDetailLinkedProjectsPanel.vue` + Playwright 回归。  
**Depends on:** nothing.
