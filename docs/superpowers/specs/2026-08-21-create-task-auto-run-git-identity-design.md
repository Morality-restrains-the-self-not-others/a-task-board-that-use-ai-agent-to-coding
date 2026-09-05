# 设计文档：创建任务自动运行须设置 Git 提交身份

- **日期**: 2026-08-21
- **作者**: cursor
- **迭代**: create-task-auto-run-git-identity
- **状态**: accepted（/goal 自动采纳）
- **python_api_approval**: n/a（零新增 Python 接口）
- **架构**: 不写新 target（无组件增删；沿用 v83 评论级 `repo_identities_json`）

## 0. 目标与成功标准

| # | 标准 | 验收 |
|---|------|------|
| S1 | work-panel 创建/编辑：`auto_run=true` 须为每个关联仓选择 Git 提交身份 | 下拉 + 提交拦截 |
| S2 | `auto_run=false` 不要求、不展示 | 无 `create-task-repo-git-identity-select` |
| S3 | Chrome 插件三入口对齐 | 浮窗 / 单请求 / 批量 |
| S4 | 服务端 `auto_run=true` 校验并写入【自动运行】评论 JSON | 400 / 落库单测 |
| S5 | 不写 `task_repo_identities` | 符合 v83 D7 |

## 1. 对当前架构的理解

- 应用层：taskFE CreateTaskModal；taskChromePlugin 浮窗与 DevTools。
- 自动运行：创建后 `ensureAutoRunAtComment` 写【自动运行】评论再 start-vm。
- Git 提交身份已在评论 composer（v83）；OAuth 已在创建时按 auto_run 门禁。
- 缺口：自动运行评论当前 INSERT 不含 `repo_identities_json`。

本次无新服务，不更新 ArchiMate 四件套。

## 2. 方案（采纳）

| 决策 | 内容 |
|------|------|
| D1 | UI 紧挨「是否自动运行」下方（独立区块，按仓列出），仅 `editingTask.auto_run===true` 显示 |
| D2 | 门禁只要求 `git_identity_id`（不要求 `github_user_id`）；GitHub App 仍走评论/详情 |
| D3 | 创建/更新 payload `repo_identities`；auto_run 触发时写入【自动运行】评论 |
| D4 | 复用评论 JSON 校验的仓列表（关联项目 git_repos）；无仓则放行 |
| D5 | 身份列表 GET 既有 `/api/git-identities/user/{userId}/`；空列表用真实 href 链到 `/tenant/:id/profile/git-identities/` |
| D6 | 强制重启复用评论时 UPDATE `repo_identities_json` |

拒绝：把身份写回任务级表；未勾选 auto_run 也强制选择。

## 3. 🕸️ Code Review Graph

| 项 | 内容 |
|----|------|
| skip 理由 | `unavailable` — codegraph MCP 无匹配；改用源码检索 CreateTaskModal / ensureAutoRunAtComment |

## 4. 业务意图 → 事件

见意图文档对照表。无新事件名。
