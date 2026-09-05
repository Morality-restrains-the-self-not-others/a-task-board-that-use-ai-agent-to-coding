# [运行时] Vue 3 嵌套 `ref` 不会自动解包 → 创建任务 OAuth「已绑定」不显示

- **日期**：2026-08-19
- **页面**：工作面板 → 创建/编辑任务 → 仓库行
- **data-testid**：`create-task-repo-oauth-bound`

## 现象

`useCreateTaskRepoOAuth()` 已返回 `boundByUrl` 且 `user-app-connection` 为 `connected:true`，提交门禁已放行，但仓库行仍不出现「已绑定 Git OAuth」，或绑定链接状态与门禁不一致。

## 根因

Vue 3 只对 **setup 顶层** `ref` 自动解包。把 composable 整包赋给嵌套对象后，模板里 `repoOAuth.boundByUrl[url]` 拿到的是 `Ref`，布尔比较恒为假：

```js
// 错误：嵌套 ref 在模板中不会解包
const repoOAuth = useCreateTaskRepoOAuth(...)
// 模板 :oauth-bound-by-url="repoOAuth.boundByUrl"

// 正确：解构为顶层 ref
const { boundByUrl: oauthBoundByUrl } = useCreateTaskRepoOAuth(...)
```

## 修复

`CreateTaskModal.vue` 解构 `boundByUrl` / `loadingByUrl` / `errorByUrl` 为顶层后再传入 `CreateTaskProjectBranchSection`。

## 验收

```bash
cd taskFE/app && npx vitest run src/components/CreateTaskModal.oauth-gate.test.js
```

已绑定用例须存在 `[data-testid=create-task-repo-oauth-bound]`。

## 预防

从 composable 取 `ref` 传给子组件时，必须在父组件 setup **解构到顶层**；禁止整包对象再点字段传 props。
