# Relay Startup Stability Runbook

## Scope

用于 task-detail 页面 `relayToTrae=true` 的启动异常排查、回归验证与回滚顺序说明。

`RelayStartupSession` 现由 Django 侧 Redis 仓储共享（`relay:startup:wf:*` / `relay:startup:scope:*`，TTL 7200s），多 worker 下 status converge 不再依赖进程内存亲和；测试或无 Redis 时可回退 InMemory。

## Quick Health Checks

```bash
curl -s -o /dev/null -w "%{http_code}\n" "http://127.0.0.1:8797/health"
curl -s -o /dev/null -w "%{http_code}\n" "http://localhost:8001/api/core/test-number-serialization/"
curl -s -o /dev/null -w "%{http_code}\n" "http://localhost:4000/"
```

预期结果：全部返回 `200`。

## Backend Contract Regression

```bash
cd task2app/Saas_project
python3 -m pytest "tests/test_container_runtime_tokens.py" "tests/test_relay_to_trae_proxy.py" -q
```

预期结果：测试通过，且 token/relay 失败路径返回结构包含 `error_code` 与 `trace_id`。

## Frontend Relay Regression

```bash
cd task2app/playwright
npx playwright test -c front_project/playwright.config.local.js ../taskFE/tests/TaskDetail.relay-to-trae-start-stop-button.playwright.test.js --project=chromium-local
```

预期结果：3 条用例全部通过：

1. 启动后停止按钮持续可见
2. 启动中 -> 停止后按钮状态正确回收
3. 按 `error_code` 映射显示错误提示

## Production Triage Notes

当用户反馈“启动失败”时，优先确认：

1. 后端响应是否带 `error_code`（非仅 message）
2. 是否能拿到 `trace_id`
3. `relay_to_trae_proxy` 是否透传了下游 `error_code`/`trace_id`
4. 前端是否展示映射后的用户文案（而不是原始服务错误）

## Rollback Order

按影响面从小到大回滚：

1. 前端错误码映射与文案层（`ServerConfig.logic.vue` + Playwright 断言）
2. relay 代理层错误透传（`relay_to_trae_proxy.py`）
3. token 视图统一错误契约（`container_runtime_token_views.py`）

回滚后必须复跑：

- 后端契约回归（pytest）
- 前端 relay 回归（Playwright）

避免只回滚一层导致契约和展示不一致。
