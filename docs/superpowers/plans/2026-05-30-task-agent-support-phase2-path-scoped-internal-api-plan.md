# 实施计划: taskAgentSupport Phase 2 路径化 Internal API

> Design: `docs/superpowers/specs/2026-05-30-task-agent-support-phase2-path-scoped-internal-api-design.md`

## Task 1: Django scoped internal dispatch + 12 actions
- [ ] 重构 `internal_dispatch.py`：`dispatch_task_agent_support_internal_scoped`
- [ ] 注册 12 action → 现有 inbound 视图
- [ ] `urls_task_agent_support_internal.py` 单条 scoped path
- [ ] 更新 `test_task_agent_support_internal.py` 路径
- [ ] 新增 task-detail smoke test

## Task 2: Go forwardToDjango 路径化
- [ ] `django_client.go` scoped URL + 直接 body
- [ ] `handlers_test.go` 可选断言

## Task 3: 验证
- [ ] `pytest tests/test_task_agent_support_internal*.py -q`
- [ ] `cd taskAgentSupport && go test ./...`
