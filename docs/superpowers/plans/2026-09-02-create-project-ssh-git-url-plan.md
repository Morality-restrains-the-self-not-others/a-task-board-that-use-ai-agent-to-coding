# 实施计划 — 创建项目 ssh:// Git URL

- **Date:** 2026-09-02

## Task 1 — 前端格式校验（Red → Green）

- [x] `gitRepoUrlUtils.test.js`：`ssh://git@github.com/owner/repo.git` 与带端口 URI 为 true；`ssh://host` 无 path、`ftp://` 为 false
- [x] `isValidGitRepoUrl` 接受 `ssh:`
- [x] 示例/hint/placeholder 含 `ssh://`
- [x] Playwright：`CreateProject.form-validation` 增加 ssh:// 用例

## Task 2 — 后端规范化（Red → Green）

- [x] `git_repo_url_normalize_test.go`：ssh:// → https（GitHub / 自建 website origin / 带端口）
- [x] `normalizeGitRepoURLForBranchLookup` 解析 ssh://
- [x] `project_repo_access`：ssh:// 未转成 HTTP 时仍视为无效 SSH 格式

## Task 3 — 举一反三

- [x] `looksLikeGitRepoRef` / `GIT_REF_IN_TEXT_RE` 识别 `ssh://`
- [x] `gitCloneRefMatchKey` 单测：ssh:// 与 git@ 同 key
- [x] ProjectEdit placeholder

## Task 4 — 意图文档

- [x] `docs/intents/frontend/create_project_ssh_git_url.intent.md` + test-intent
- [x] INDEX F-027

无新 MQ 事件任务：纯校验/规范化。
