# 测试意图：创建任务勾选自动运行时须设置 Git 提交身份

## 对应功能意图

`docs/intents/frontend/create_task_auto_run_git_identity.intent.md`

## 用例

| ID | 场景 | 前置 | 步骤 | 期望 |
|----|------|------|------|------|
| T1 | 未勾选 auto_run | 所选项目有 git 仓 | 打开创建弹窗/插件表单 | 无身份下拉；提交不因身份拦截 |
| T2 | 勾选 auto_run 未选身份 | 有 git 仓，身份列表非空 | 勾选自动运行不选下拉 | 提交拦截，文案含「Git 提交身份」 |
| T3 | 勾选 auto_run 已选齐 | 每仓选定 `git_identity_id` | 提交创建 | payload 含 `repo_identities` |
| T4 | 无 git 仓 | 项目 `git_repos` 空 | `auto_run=true` | 不因身份拦截 |
| T5 | 服务端缺身份 | POST `auto_run=true` 无 identities | 创建 | 400 `repo_identities_required` |
| T6 | 服务端未勾选 | POST `auto_run=false` 无 identities | 创建 | 201，不校验身份 |
| T7 | 自动运行评论落库 | 创建 auto_run 且身份齐全 | 查【自动运行】评论 | `repo_identities_json` 含所选 `git_identity_id` |
| T8 | 插件未勾选 | Chrome 浮窗/单请求/批量 | 不勾选自动运行 | 不渲染身份编辑器；validate 通过 |
| T9 | 身份选择器在自动运行下方 | work-panel 创建弹窗 auto_run=true | 打开弹窗 | `create-task-git-identity-section` 在 `task-auto-run-field` 之后，不在项目仓行内 |

## 可执行测试

- `taskFE/app/src/utils/createTaskGitIdentityGate.test.js`
- `taskFE/app/src/components/CreateTaskModal.git-identity-gate.test.js`
- `taskChromePlugin/test/create-task-git-identity.test.js`
- `taskChromePlugin/test/create-task-payload.test.js`
- `taskTaskService/src/comment_repo_identities.go` 相关单测 / `auto_run_at_comment_test.go` / `create_task` auto_run 身份测例
