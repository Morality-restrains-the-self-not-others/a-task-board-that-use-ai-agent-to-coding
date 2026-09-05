# 实施计划：评论级仓库身份

> 设计：`docs/superpowers/specs/2026-08-16-comment-level-repo-identity-design.md`

## 事件任务（强制）

- [x] 契约：`TASK_COMMENT_IMAGE_MENTIONED.data.repo_identities` 数组
- [x] publish：handleCreateComment 在 mentions 非空时写入 eventData
- [x] 消费者：taskEvents 透传到 start-vm（本增量至少 payload 不丢；Cloud 读 snapshot comment_id）
- [x] 意图对照表已写在 `034_comment_level_repo_identity.intent.md`

## I1 关联项目只读

- [x] 红：`TaskDetailLinkedProjectsPanel.test.js` / ViewMode / Toolbar 断言无 OAuth 文案、无进度、无身份下拉；仍有项目名与分支
- [x] 绿：改 Toolbar/ViewMode/Panel props，去掉身份/OAuth/进度/reclone/拉身份
- [x] 保留 EditMode 与地址失配同步

## I2 composer 身份

- [x] 红：`CommentComposerRepoIdentity.test.js` + Composer：无 run config 不渲染；有则渲染；校验函数单测
- [x] 绿：新组件 + `commentRepoIdentityDraft.js`；submitComment 带 `repo_identities`
- [ ] 预填：最近评论 JSON → task_repo_identities（OPT）

## I3 持久化

- [x] DDL `dataMigrate/taskTaskService/011_comment_repo_identities.sql`
- [x] 红：comment create 有 mention 无 identities → 400；有则落库；list 回读
- [x] 红：snapshot `comment_id` 优先 JSON，否则回退
- [x] 绿：handler + `validateRepoIdentities` + 日志 `event=comment_repo_identities_saved`
- [x] 事件 payload 单测

## I4 对齐

- [x] 更新 Playwright 中关联项目 OAuth 入口（改为 composer 或 skip 说明改入口）
- [x] 引导文案：`commentLayerZtreeUiState` / Relay 面板不再指向「上方关联项目」完成身份
- [x] 更新 `comment_clone_progress_under_trace.intent.md` 范围外句（本流已移除关联项目进度）

每切片 <100 行业务 + 对应测试。TDD。
