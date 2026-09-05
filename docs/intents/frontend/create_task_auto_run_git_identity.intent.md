# 意图：创建任务勾选自动运行时须设置 Git 提交身份

- **日期**: 2026-08-21
- **状态**: 已落地

## 背景与目标

勾选「是否自动运行」后，创建任务会立刻写出【自动运行】评论并引导克隆/提交。评论 composer 的「Git 提交身份」来不及选。用户须在**创建任务**（工作台 work-panel 与 taskChromePlugin）时，对每个关联仓库选择 Git 提交身份。未勾选自动运行则不要求、不展示该字段。

身份仍落在自动运行评论的 `repo_identities_json`（v83 评论级），不写回 `task_repo_identities`。

## 范围与边界

- 范围内：work-panel `CreateTaskModal`（创建/编辑且 `auto_run=true`）；taskChromePlugin 浮窗、DevTools 单请求与批量创建。
- 范围内：所选项目每个非空 `git_repos` URL 必须有 `git_identity_id`；提交拦截文案含「Git 提交身份」。
- 范围内：POST/PUT 任务体可带 `repo_identities`；`auto_run=true` 且有关联仓时服务端校验；写入【自动运行】评论 JSON。
- 范围外：未勾选自动运行不拦截、不展示身份选择；不把身份改回任务级表；不把 GitHub App 账号选择下沉到创建表单（OAuth 门禁仍按既有 `create_task_oauth_bind`）；Fork 确认弹窗本增量不改（仍走详情评论身份）。

## 约束与风险

- 无关联仓库 URL 时不拦截（无可提交仓）。
- 身份列表来自当前用户 `/api/git-identities/user/{userId}/`；空列表时提示去账号中心创建（真实 `<a href>`）。
- 不得 `@click.prevent` 冒充链接。

## 验收标准

1. `auto_run=true` 且所选项目有 git 仓、未为每仓选择 Git 提交身份：提交 disabled / 插件报错，文案含「Git 提交身份」；每仓出现身份下拉 `create-task-repo-git-identity-select`（插件为 `data-git-identity`）。身份选择器紧挨「是否自动运行」下方，不挂在项目/仓行内。
2. `auto_run=false`：不展示身份下拉；不因缺身份拦截提交。
3. 已选齐身份且其它门禁满足：创建请求带 `repo_identities[{repo_url,git_identity_id}]`。
4. 服务端 `auto_run=true` 缺身份 → 400 `repo_identities_required`；未勾选 auto_run 不校验身份。
5. 【自动运行】评论 `repo_identities_json` 含创建时选择的身份；container-snapshot 按评论级读取。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 创建任务收集 Git 提交身份 | — | — | — | 写入【自动运行】评论 JSON，随既有 TASK_CREATED / start-vm 路径消费 | 无新领域事件；身份是自动运行评论的属性，不另发事件 |

## 路径分片键 / 幂等（NFR 摘要）

- 创建任务路径已含 `tenantId` / `workspaceId`。
- 写路径为既有 POST/PUT todos；幂等键仍为客户端 `Idempotency-Key` / fork 去重窗；`repo_identities` 随同一次创建写入，重复 POST 命中去重则不二次写评论。

## 变更记录

| 日期 | 相对旧版 | 原因 |
|------|----------|------|
| 2026-08-21 | 初版 | 自动运行在创建时立即克隆/提交，须提前选定 Git 提交身份 |
| 2026-08-21 | 身份选择器移到「是否自动运行」下方 | 创建弹窗原先把 Git 身份挂在项目仓行，用户先看到身份再看到自动运行 |
