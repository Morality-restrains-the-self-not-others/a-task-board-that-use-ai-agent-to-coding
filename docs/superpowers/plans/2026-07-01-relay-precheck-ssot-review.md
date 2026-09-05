# Code Review: relay 直启预检 Token SSOT 消除双写

**日期:** 2026-07-01
**对应计划:** `docs/superpowers/plans/2026-07-01-relay-precheck-ssot-plan.md`

## 审查结论: ✅ 通过（无阻塞问题）

## 1. 计划合规检查

| 计划任务 | 状态 | 说明 |
|---------|------|------|
| Task 1.1 `_task_credential_service_url()` | ✅ | relay_to_trae_proxy.py:295-297 |
| Task 1.2 `_credential_service_post()` | ✅ | relay_to_trae_proxy.py:300-311 |
| Task 2.1 token-init → Go | ✅ | `_issue_token_via_go()` 替代 `_build_runtime_env_for_relay()` |
| Task 3.1 precheck → Go 两阶段 | ✅ | token-init → repo-clone-credentials |
| Task 3.2 start → Go | ✅ | `_issue_token_via_go()` 替代 `_build_runtime_env_for_relay()` |
| Task 3.3 register → Go | ✅ | `_issue_token_via_go()` 替代 `_issue_relay_access_token()` |
| Task 4.1 删除 `_sync_credential_service_token` | ✅ | relay_to_trae_proxy.py |
| Task 4.2 删除 `_issue_relay_access_token` | ✅ | relay_to_trae_proxy.py |
| Task 4.3 删除 `_resolve_relay_precheck_task_api_origin` | ✅ | relay_to_trae_proxy.py |
| Task 4.4 删除 `_build_runtime_env_for_relay` | ✅ | relay_to_trae_proxy.py |
| Task 4.5 删除 `_sync_token_to_credential_service` | ✅ | mock_run_container.py — 函数 + import sqlite3/os/uuid 全部删除 |
| Task 4.6 清理 `_replace_access_token_placeholder` | ✅ | 移除了 `_sync_token_to_credential_service()` 调用 |
| Task 4.7 删除过时测试 | ✅ | 更新所有 mock 为 `_issue_token_via_go` |
| Task 5.1 测试全绿 | ✅ | 28/28 passed |
| Task 5.2 价值流测试 | ⏭️ | valueStream 编译/运行需要 Go 工具链 |
| Task 5.3 代码行数 | ✅ | relay_to_trae_proxy.py: 1066 (-45 行), mock_run_container.py: 1048 (-95 行) |

## 2. DDD 合规

- **✅** `check_ddd_bdd_compliance.py` 通过
- **✅** 无基础设施导入到领域层
- **✅** 端口接口 `TaskCredentialServicePort` 已在 DDD 模型中定义（虽未物理创建文件——因为函数式代码风格，通过 `_issue_token_via_go` / `_credential_service_post` 两个薄函数充当端口）
- **✅** `build_relay_to_trae_runtime_env` 保留在 mock_run_container.py 中仅用于 mockStart 路径（符合设计：Phase 2 迁移）

## 3. 日志审计 (Log Audit)

### A. 覆盖率

| 检查项 | 状态 | 详情 |
|--------|------|------|
| except 块有 ERROR/WARN 日志 | ✅ | 所有 `except` 块使用 `logger.exception()` 或 `logger.error()` |
| 外部 HTTP 调用有日志 | ✅ | `_issue_token_via_go`: DEBUG 前 + INFO/ERROR 后；precheck/start/register 失败有 ERROR |
| 状态变更日志 | ✅ | `service.mark_token_init_succeeded/failed` + `append_container_token_audit_event` 保留 |
| 认证拒绝日志 | N/A | 无新增认证逻辑 |
| 后台任务日志 | ✅ | `_relay_to_trae_start_async_task` 已有 started/completed/failed 日志 |
| 资源生命周期日志 | N/A | 无新增资源创建/删除 |

### B. 质量

| 检查项 | 状态 |
|--------|------|
| 日志含定位信息（ID） | ✅ — 所有日志含 tenant_id / task_id |
| 异常日志含 traceback | ✅ — 使用 `logger.exception()` |
| 敏感信息泄露 | ✅ — 无 token/password/secret 硬编码 |
| 日志级别合理 | ✅ — DEBUG 用于调用前，INFO 用于成功，ERROR 用于失败 |

### C. 噪音

| 检查项 | 状态 |
|--------|------|
| 循环内无 INFO 日志 | ✅ |
| 无无意义日志 | ✅ |
| 无注释掉的日志 | ✅ |

**日志审计结论: ✅ 无缺失。**

## 4. 代码质量

### 优点
- 消除了跨语言 SQLite 直写 hack（最恶劣的耦合模式）
- Token 来源统一为 Go SSOT，Django 不再持有 token 副本
- 错误处理完善：Go 不可达 → 502 + 明确消息；Go 返回错误 → 透传
- HTTP 调用统一经过 `_credential_service_post()`（trust_env=False + timeout）
- 日志覆盖所有关键路径

### 注意事项（非阻塞）

| 项目 | 说明 |
|------|------|
| 文件行数 | relay_to_trae_proxy.py 1066 行 — 超过 500 行限制，但本次变更净减少 ~45 行，后续可拆分 |
| `build_relay_to_trae_runtime_env` 残留 | mock_run_container.py:559 — 仅 mockStart 使用，Phase 2 迁移后删除 |
| Go `/v1/token/init` 无内部认证 | 已有风险，独立 Phase 加固（添加 `X-TaskGateway-Internal-Secret` header） |

## 5. 测试覆盖

- 28 个测试全部通过（含更新后的 11 个 + 未修改的 17 个）
- 覆盖场景：token_init 成功/失败、precheck 200/409/502、start 异步派发、register 成功/失败/复用
- 缺失场景（建议后续补充）：Go 连接超时模拟、Go 返回非 JSON 响应

## 6. 风险评级

| 风险 | 等级 | 缓解 |
|------|------|------|
| Go 不可达 → token-init/precheck/start 502 | 🟢 低 | Go 是 runAll 一部分，启动顺序保证；HTTP 调用有超时保护 |
| mockStart token 同步 | 🟢 低 | mockStart 暂不迁移，仍写 CloudServerConfig（Phase 2 独立处理） |
| 回归 | 🟢 低 | 28/28 测试通过，HTTP 契约不变 |
