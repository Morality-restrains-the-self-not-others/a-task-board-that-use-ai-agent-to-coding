# 价值流：创建任务自动运行 Git 提交身份

- **日期**: 2026-08-21
- **设计**: `docs/superpowers/specs/2026-08-21-create-task-auto-run-git-identity-design.md`

## 端到端增量

用户打开 work-panel 或 Chrome 插件 → 选项目/仓 → 勾选自动运行 → 为每仓选 Git 提交身份 → 创建任务 → 服务端校验 → 【自动运行】评论带 `repo_identities_json` → 容器按评论级身份提交。

未勾选自动运行：跳过身份步骤，直接创建。

## 切片（垂直，按交付顺序）

1. 纯函数门禁（收集仓 URL、auto_run 才校验 `git_identity_id`）
2. work-panel UI + 提交 payload
3. taskTaskService 创建/更新 + 自动运行评论落库
4. Chrome 插件表单/payload/使用说明

## 测试点

见 `docs/intents/frontend/create_task_auto_run_git_identity.test-intent.md` T1–T8。
索引图 `docs/flows/value-stream-test-integration.wsd` CRUD 注记追加本意图。
