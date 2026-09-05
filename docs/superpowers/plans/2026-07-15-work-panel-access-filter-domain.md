# 领域模型：工作面板按可访问人或小组过滤

- 日期：2026-07-15
- 说明：本迭代为**前端视图域**增强；无新服务端聚合根落库。

## 1. 限界上下文

| 上下文 | 关系 |
|--------|------|
| Project（workspace access） | 只读：permissions / collaborators |
| Accounts（groups） | 只读：group members |
| Task | 只读：todo.owner / assignees |
| WorkPanel UI | 新建 AccessFilter 值对象与过滤服务（JS 模块） |

## 2. 值对象

```text
AccessFilter {
  kind: null | 'person' | 'group'
  id: string          // company_member_id 或 group_id
  label: string
  memberIds: string[] // company_member_id 集合
}
```

## 3. 领域服务（前端）

- `buildPersonAccessFilter(userInfo) → AccessFilter`
- `buildGroupAccessFilter(groupInfo, memberCompanyMemberIds) → AccessFilter`
- `filterTodosByAccess(todos, AccessFilter) → todos`
- `resolveCompanyMemberIdsFromGroupMembers(groupMembers, collaborators) → string[]`

## 4. 领域事件

**无**。书面例外：纯前端过滤，无业务状态变更，不投递 MQ。

## 5. 架构变更影响

无需新架构版本（见设计文档 §9）。
