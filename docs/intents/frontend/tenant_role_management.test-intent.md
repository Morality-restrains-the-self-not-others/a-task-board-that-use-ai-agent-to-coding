# 测试意图：租户角色管理

## 覆盖设计

`docs/superpowers/specs/2026-08-11-tenant-role-management-v75-design.md`

## 用例清单

| # | 场景 | 期望 |
|---|------|------|
| 1 | 有权限用户打开 `/people/roles/` | 看到角色列表（系统 + 自定义） |
| 2 | 无 `page:people.roles` 且无 member:manage | 无权限空态 |
| 3 | 创建自定义角色并保存 region view/operate | GET resource-groups 一致；PDP 注入对应码 |
| 4 | 删除仍被成员引用的角色 | 422（或明确错误），角色仍在 |
| 5 | 访问管理为成员勾选两个自定义角色并保存 | list 显示两角色；权限为并集 |
| 6 | 移除其中一个角色 | 并集缩小；另一角色仍在 |
| 7 | 访问管理保存路径 | **不**新建 `访问·…` 角色 |
| 8 | 组多角色 | 组成员继承组角色并集 |
| 9 | 角色页不展示粗码勾选 UI | 仅 page/region 双档 |
| 10 | API 错误 | 展示带 `data-traceId` |

## 回归

- 既有 PeopleAccess 目录加载、orphan 清理仍可用
- 邀请 pending_grants（v74）兼容不破；选角色为增强项
