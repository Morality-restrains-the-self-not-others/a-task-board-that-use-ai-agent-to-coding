# 实施计划: relay 直启预检 Token SSOT 消除双写

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-07-01-relay-precheck-ssot-in-process-design.md`
> - 价值流: `docs/superpowers/plans/2026-07-01-relay-precheck-ssot-value-stream.md`
> - DDD 模型: `docs/superpowers/plans/2026-07-01-relay-precheck-ssot-ddd-model.md`
> - NFR 澄清: `docs/superpowers/plans/2026-07-01-relay-precheck-ssot-nfr-clarification.md`

## 任务清单

### Phase 1: 辅助函数与端口接口 (基础设施准备)

- [ ] **Task 1.1** 新增 `_task_credential_service_url()` 函数
  - 文件: `task2app/Saas_project/cloud/services/relay_to_trae_proxy.py`
  - 实现: 返回 `settings_manager.get_task_credential_service_url()` 或 fallback `http://127.0.0.1:8015`
  - 验证: 单测 `test_task_credential_service_url_returns_configured_value`

- [ ] **Task 1.2** 新增 `_credential_service_post()` 辅助函数
  - 文件: `task2app/Saas_project/cloud/services/relay_to_trae_proxy.py`
  - 实现: `requests.Session` + `trust_env=False` + JSON headers + timeout 参数
  - 验证: 单测 `test_credential_service_post_sends_correct_payload`

### Phase 2: token-init 改调 Go (Increment 1 — Thin Slice)

- [ ] **Task 2.1** 重构 `relay_to_trae_token_init()` — 改为调 Go `/v1/token/init`
  - 文件: `task2app/Saas_project/cloud/services/relay_to_trae_proxy.py` (line ~397)
  - 旧: `_build_runtime_env_for_relay()` → `build_relay_to_trae_runtime_env()` → `_replace_access_token_placeholder()` → Django 生成 token
  - 新: `_credential_service_post(f"{url}/v1/token/init", json={tenant_id, workspace_id, task_id})` → 返回 `{status:"ok", token_initialized:true, env_preview:{...}}`
  - 保留: `RelayTwoStepStartupService` 状态机调用、`append_container_token_audit_event` 审计调用
  - ⚠️ 日志: Go 调用失败时记录 ERROR 日志（含 status_code + body 摘要）
  - 验证: 更新 `test_relay_to_trae_token_init_*` 测试

- [ ] **Task 2.2** 更新 token_init 测试
  - 文件: `task2app/Saas_project/tests/test_relay_to_trae_proxy.py`
  - mock `_credential_service_post` 替代 mock `_build_runtime_env_for_relay`
  - 场景: Go 返回 200 → token_init 返回 200 + env_preview
  - 场景: Go 返回 500 → token_init 返回 502
  - 场景: Go 连接超时 → token_init 返回 502 + 明确错误消息
  - 运行: `cd task2app/Saas_project && python -m pytest tests/test_relay_to_trae_proxy.py::test_relay_to_trae_token_init* -xvs`

### Phase 3: precheck/start/register 改调 Go (Increment 2 — Core Value)

- [ ] **Task 3.1** 重构 `relay_to_trae_repo_credentials_precheck()` — 两阶段 Go 调用
  - 文件: `task2app/Saas_project/cloud/services/relay_to_trae_proxy.py` (line ~510)
  - 旧: `_issue_relay_access_token()` → 查 CloudServerConfig + SQLite hack → 调 Go repo-clone-credentials
  - 新: `_credential_service_post("/v1/token/init")` → 拿 token → `_credential_service_post("/repo-clone-credentials/")`
  - 保留: 200/409/502 响应分流逻辑（不变）
  - ⚠️ 日志: 两次 Go 调用均记录 TRACE 级别日志；失败记录 ERROR
  - 验证: 更新 `test_relay_to_trae_repo_credentials_precheck_*` 测试

- [ ] **Task 3.2** 重构 `relay_to_trae_start()` — token 从 Go 获取
  - 文件: `task2app/Saas_project/cloud/services/relay_to_trae_proxy.py` (line ~1015)
  - 旧: `_build_runtime_env_for_relay()` → Django 生成 token
  - 新: `_credential_service_post("/v1/token/init")` → 注入 `runtime_env["ACCESS_TOKEN"]`
  - 保留: `_dispatch_relay_start_async` 异步启动逻辑、`RelayTwoStepStartupService` 状态机
  - ⚠️ 日志: 记录 token 来源为 Go SSOT（INFO 级别）
  - 验证: 更新 `test_relay_to_trae_start_*` 测试

- [ ] **Task 3.3** 重构 `register_or_reuse` — token 从 Go 获取
  - 文件: `task2app/Saas_project/cloud/services/relay_to_trae_proxy.py` (line ~860)
  - `_issue_relay_access_token()` → `_credential_service_post("/v1/token/init")`
  - 保留: register_only 模式、token exchange in progress 降级逻辑
  - 验证: 更新 `test_relay_to_trae_register_*` 测试

- [ ] **Task 3.4** 更新 precheck/start/register 测试
  - 文件: `task2app/Saas_project/tests/test_relay_to_trae_proxy.py`
  - 新增: `test_precheck_two_phase_go_calls` — 验证先调 token-init 再调 repo-clone-credentials
  - 新增: `test_precheck_handles_go_token_init_failure` — Go 500 → 502
  - 新增: `test_start_calls_go_token_init` — 验证 start 调 /v1/token/init
  - 新增: `test_start_injects_go_token_into_runtime_env` — token 正确注入 ACCESS_TOKEN
  - 运行: `cd task2app/Saas_project && python -m pytest tests/test_relay_to_trae_proxy.py -xvs`

### Phase 4: 删除双写 hack (Increment 3 — Cleanup)

- [ ] **Task 4.1** 删除 `_sync_credential_service_token()`
  - 文件: `task2app/Saas_project/cloud/services/relay_to_trae_proxy.py` (line 209-231)
  - 确认: 无其他调用方 `grep -rn "_sync_credential_service_token" task2app/`

- [ ] **Task 4.2** 删除 `_issue_relay_access_token()`
  - 文件: `task2app/Saas_project/cloud/services/relay_to_trae_proxy.py` (line 234-292)
  - 确认: 无其他调用方（Phase 3 已全部替换）

- [ ] **Task 4.3** 删除 `_resolve_relay_precheck_task_api_origin()`
  - 文件: `task2app/Saas_project/cloud/services/relay_to_trae_proxy.py` (line 505-507)
  - 替换: 统一使用 `_task_credential_service_url()`

- [ ] **Task 4.4** 删除 `_build_runtime_env_for_relay()`
  - 文件: `task2app/Saas_project/cloud/services/relay_to_trae_proxy.py` (line 382-394)
  - 确认: Phase 2/3 已不再调用

- [ ] **Task 4.5** 删除 `_sync_token_to_credential_service()` 及其 import
  - 文件: `task2app/Saas_project/cloud/services/mock_run_container.py` (line 286-387)
  - 同时删除: `import sqlite3` / `import os` / `import uuid`（仅服务于 sync hack 的 import，确认无其他用途）
  - ⚠️ 确认: `os` 模块在 mock_run_container.py 中是否还被其他函数使用 → 若是则保留

- [ ] **Task 4.6** 清理 `_replace_access_token_placeholder()` 中的 relayToTrae 专属逻辑
  - 文件: `task2app/Saas_project/cloud/services/mock_run_container.py` (line 390-496)
  - 移除: `_sync_token_to_credential_service()` 调用 (line 486-493)
  - 保留: mockStart 路径的 token 生成逻辑（`generate_opaque_token()` + `cfg.save()`）
  - 确认: `grep -rn "_replace_access_token_placeholder" task2app/` 只有 mockStart 路径调用

- [ ] **Task 4.7** 删除过时测试
  - 文件: `task2app/Saas_project/tests/test_relay_to_trae_proxy.py`
  - 删除: `test_*_issue_relay_access_token_*`
  - 删除: `test_relay_to_trae_repo_credentials_precheck_uses_internal_task_api_origin`
  - 删除: 任何 mock `CloudServerConfig.objects` 的 relayToTrae 测试（如仍存在）
  - 确认: 所有 mock `_sync_credential_service_token` / `_sync_token_to_credential_service` 的测试已更新或删除

### Phase 5: 回归验证 (Increment 4 — Essential Support)

- [ ] **Task 5.1** 运行全量 relay 测试套件
  - 命令: `cd task2app/Saas_project && python -m pytest tests/test_relay_to_trae_proxy.py -xvs`
  - 目标: 所有测试通过（更新的 + 新增的 + 未修改的）

- [ ] **Task 5.2** 运行价值流测试
  - 命令: `cd valueStream && go test ./...`
  - 目标: `relay-precheck-token-ssot` 流通过

- [ ] **Task 5.3** 代码行数合规检查
  - 验证: `relay_to_trae_proxy.py` 变更后 ≤ 500 行（项目约束）
  - 验证: `mock_run_container.py` 删除 sync 函数后行数变化

- [ ] **Task 5.4** 手动冒烟测试
  - 启动: `./run.sh` 全栈启动
  - 验证: `curl -s http://127.0.0.1:8015/health` → `{"status":"ok"}`
  - 验证: `curl -s -X POST http://127.0.0.1:8015/v1/token/init -H 'Content-Type: application/json' -d '{"tenant_id":"test","workspace_id":"test","task_id":"test"}'` → 200 + access_token
  - 验证: 浏览器直启流程（token-init → precheck → start）正常

## 执行顺序

```
Phase 1 (辅助函数)
  ↓
Phase 2 (token-init → Go)  ← 最小可验证增量
  ↓
Phase 3 (precheck/start/register → Go)
  ↓
Phase 4 (删除双写 hack)  ← 必须在 Phase 2+3 之后
  ↓
Phase 5 (回归)
```

**依赖约束:**
- Phase 3 依赖 Phase 2（token-init 改为 Go 调用的模式被 precheck/start 复用）
- Phase 4 必须在 Phase 2+3 之后（删除的函数仍有调用方时不能删）
- 每个 Task 完成后运行其对应的测试验证

## 风险缓解

| 风险 | 缓解 |
|------|------|
| `mock_run_container.py` 的 `os` import 删除影响其他函数 | Task 4.5 执行前 grep 确认所有 `os.path` 调用 |
| `_replace_access_token_placeholder` mockStart 路径受影响 | Phase 4 只移除 `_sync_token_to_credential_service()` 调用，保留其余 |
| 测试文件行数增长 | 删旧测试 + 增新测试，净变化可控 |

## NFR 验收

| NFR | 验证方式 |
|-----|---------|
| 性能 L2: P95 ≤ 200ms (token-init) | Task 5.4 冒烟时手工计时 |
| 可用性 L2: Go 不可达 → 502 | Task 3.4 测试覆盖 |
| 容错 L2: 超时 5s/8s | `_credential_service_post` timeout 参数 |
| 可观测性 L2: 失败记录 ERROR 日志 | Task 2.1/3.1 实现时添加 logging |
| 数据一致性 L2: 无双写 | Task 4.1-4.6 后 `grep` 确认无 SQLite hack 残留 |
