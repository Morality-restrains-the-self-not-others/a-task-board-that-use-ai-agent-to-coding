# 测试意图：创建项目 OAuth 回流后写入项目 L2

## 覆盖点

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 创建页 bootstrap `location.search` 含 `grant_ticket` + `repo_url` | sessionStorage 记住该 gitsite 的 ticket |
| T2 | 创建 POST body | 含 `grant_ticket` 与 `grant_tickets` |
| T3 | `handleCreateProject` 消费 ticket 成功 | `project_git_oauth_grant` 存在 |
| T4 | consume 失败 | 仍 201，不写 L2 |

## 可执行

- `taskFE/app/src/utils/grantTicketSession.test.js`
- `taskFE/app/src/composables/useCreateProjectGitRepoRows.batch.test.js`
- `taskFE/app/src/views/CreateProject.submitForm.test.js`
- `taskProjectService/src/git_oauth_grant_ticket_test.go`

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-09-02 | 初版 |
