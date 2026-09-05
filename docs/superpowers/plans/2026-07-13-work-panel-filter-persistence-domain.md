# 领域模型：WorkPanelFilterPreference

- 日期：2026-07-13
- 限界上下文：Workspace Collaboration（taskProjectService）

## 聚合

**WorkPanelFilterPreference**
- 标识：`(UserId, CompanyId, WorkspaceId)`
- 属性：`Payload`（version + bars）、`UpdatedAt`
- 不变量：
  - 1 ≤ bars ≤ 20
  - 每 bar 有稳定 id + 非空 path
  - path 段 type ∈ {root, category, task}；category/task 须有 id

## 值对象

- `DeliverableFilterBar{Id, Path}`
- `PathSegment{Type, Id?, Label?}`

## 仓储接口

```
Get(user, company, workspace) → Preference | Default
Upsert(Preference) → error
```

## 领域服务

- `NormalizeFilterPayload(raw) → Payload | error`
- `DefaultFilterPayload() → Payload`

实现语言：Go（taskProjectService）；前端镜像 normalize 纯函数以便离线校验。
