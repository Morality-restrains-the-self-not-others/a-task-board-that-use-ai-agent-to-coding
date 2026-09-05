# DDD 轻量建模：导航栏任务搜索过滤

- **日期**: 2026-07-20
- **限界上下文**: Task（taskTaskService）+ Frontend Shell（Navbar）
- **架构变更**: 无（不写 v42 target）

## 领域概念

| 概念 | 类型 | 说明 |
|------|------|------|
| TaskSearchHit | Read Model | id, title, workspace_id, owner, assignees |
| TaskAssignee | 关联 | task_id ↔ company_member_id |
| NavbarTaskSearchQuery | 前端 VO | q + resolved assignee_ids |

## 命令 / 查询

| 名称 | 类型 | 事件 |
|------|------|------|
| SearchTasks | Query | 无（只读例外） |
| NavigateToWorkPanel / OpenTask | 前端导航 | 无 |

## 架构变更影响

- **迭代版本**: 无新版本（UI + 既有 API 扩展）
- **变更明细**: 🟡 taskTaskService search 匹配逻辑；🟡 Vue Navbar
- **伴生架构文件**: 不适用（不改拓扑）
