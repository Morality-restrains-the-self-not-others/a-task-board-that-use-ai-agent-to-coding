# 测试意图：创建项目支持 ssh:// Git URL

## 覆盖点

| ID | 场景 | 期望 |
|----|------|------|
| T1 | `isValidGitRepoUrl('ssh://git@github.com/owner/repo.git')` | true |
| T2 | `ssh://git@gitlab.daydaymoney.com:2222/g/p.git` | true |
| T3 | `ssh://git@host` 无 path | false |
| T4 | `git@` / `https://` | 仍 true |
| T5 | Playwright 创建页填 ssh:// | 无 format-hint；按钮不因格式禁用 |
| T6 | `normalizeGitRepoURLForBranchLookup(ssh://git@github.com/org/repo.git)` | `https://github.com/org/repo` |
| T7 | 自建 GitLab website origin | ssh:// 继承 origin scheme |
| T8 | `looksLikeGitRepoRef('ssh://git@h/a.git')` | true |
| T9 | `gitCloneRefMatchKey` ssh:// 与 git@ 同仓 | 相同 key |
| T10 | 编辑页 `gitRepoRowsFormatError` / `useProjectGitRepoRows` | `ssh://` 无格式错误；`ftp://` 为 INVALID_REPO_URL_MSG |

## 可执行

- `taskFE/app/src/utils/gitRepoUrlUtils.test.js`
- `taskFE/app/src/composables/useProjectGitRepoRows.test.js`
- `taskFE/tests/CreateProject.form-validation.playwright.test.js`
- `taskProjectService/src/git_repo_url_normalize_test.go`

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-09-02 | 初版 |
| 2026-09-02 | T10 编辑页复用格式校验（OPT-20260902-004） |
