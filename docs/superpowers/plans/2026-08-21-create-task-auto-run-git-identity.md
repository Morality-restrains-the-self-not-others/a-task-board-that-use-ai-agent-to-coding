# 实施计划：创建任务自动运行 Git 提交身份

## 任务

- [x] T1 纯函数 `createTaskGitIdentityGate.js` + vitest（T1–T4）
- [x] T2 Chrome `create-task-payload.js` 校验/payload/HTML 构建器 + node:test
- [x] T3 Go `ValidateGitIdentitiesForCreateAutoRun` 红绿 + create/update 接线
- [x] T4 `ensureAutoRunAtComment` 写入/复用更新 JSON
- [x] T5 CreateTaskModal / ProjectBranchSection UI + 提交 payload
- [x] T6 Chrome 浮窗 + panel 单请求/批量 + API listGitIdentities + 使用说明
- [x] T7 价值流图注记 + 意图对照已落地

每切片：红 → 绿 → 提交。
