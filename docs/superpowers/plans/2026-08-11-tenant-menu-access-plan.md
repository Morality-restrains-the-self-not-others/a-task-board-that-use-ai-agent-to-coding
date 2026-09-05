# Step 7 — Plan：租户菜单访问管理

## 切片

- [x] **S1** `tenantConsoleNav.js` + 单测（映射/可见性）
- [x] **S2** `usePermissions` 增加 `invalidate`/`reload` + 单测
- [x] **S3** `PeopleAccess.vue` 页面 + 路由 + Sidebar 入口
- [x] **S4** Sidebar 按权限过滤菜单项 + 更新 Sidebar 单测
- [x] **S5** 修复 PeopleGroups 权限码；访问页保存编排单测
- [x] **S6** 文档 intents/design 对齐；OPT 落盘

## 事件任务

- 无新事件契约；保存调用既有 publish 路径。

## 完成定义

全部 TI 对应单测绿；Sidebar 含「访问管理」；无权限隐藏菜单。
