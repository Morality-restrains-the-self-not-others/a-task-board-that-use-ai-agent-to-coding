# 实施计划：启动日志默认折叠 + 推送并创建PR

- **日期**: 2026-07-12
- **状态**: approved (auto)

## Task 1 — 意图文档

- [ ] 更新 `019_relay_startup_logs_collapse.intent.md` / `.test-intent.md`：默认折叠
- [ ] 新建 `ztree_push_and_create_pr.intent.md` + test-intent

## Task 2 — 启动日志默认折叠（TDD）

- [ ] 红：单测期望默认折叠
- [ ] 绿：`ref(false)` + prop default false
- [ ] 验证 vitest

## Task 3 — 后端 wait_for_pr（TDD）

- [ ] 红：`wait_for_pr=true` 时响应含解析后的 `github_pull_request`（mock job succeeded → html_url）
- [ ] 绿：`forward_container_layer_git_push` + PR 模块辅助函数轮询 job
- [ ] 保留无 flag 时异步行为单测

## Task 4 — 前端按钮与跳转（TDD）

- [ ] 文案「推送并创建PR」
- [ ] 请求体 `wait_for_pr: true`
- [ ] 成功打开 `html_url`（既有逻辑恢复可用）

## Task 5 — 回归

- [ ] 相关 vitest / Django 单测通过
- [ ] 价值流测试点勾选
