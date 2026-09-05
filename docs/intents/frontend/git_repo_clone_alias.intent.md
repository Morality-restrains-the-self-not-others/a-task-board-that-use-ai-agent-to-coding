# 功能意图：Git 仓库克隆别名

## 用户故事

作为租户成员，我在创建或编辑项目时为每个 Git 仓库填写可选别名，以便容器克隆时目录使用该名称，而不是仅从 URL 推导。

## 验收标准

1. 创建项目页每个仓库行在地址旁有「别名」可选输入。
2. 编辑项目页同样支持别名的展示与保存。
3. 未填别名时行为与改前一致（目录名来自 URL）。
4. 填写别名后，task-detail → trae-agent 克隆目录名为该别名（经 sanitize）。
5. 同项目内重复别名有前端校验提示。

## 范围

- 前端 CreateProject / ProjectEdit
- taskProjectService `project_repos.clone_alias`
- taskTaskService container-snapshot / project 详情透传
- taskCredentialService task-detail
- trae-agent onlineServiceJS bootstrap / reclone


## 业务意图 → 事件对照

> 存量回填（自动）：对照 `.ai/08_prompt_management/01_intent_driven_development.md`。事件名若为启发式占位，可在后续迭代精修。

**无对应事件**：纯前端展示/交互或设计治理，无服务端业务状态变更意图。

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 功能意图：Git 仓库克隆别名 | — | — | — | 纯前端展示/交互或设计治理，无服务端业务状态变更意图 |
## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-07-14 | 初版 |
| 2026-07-14 | 补：详情页展示别名；后端同项目别名唯一；Playwright E2E |
