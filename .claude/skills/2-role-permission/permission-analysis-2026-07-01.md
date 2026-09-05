# 角色权限分析：relay 直启预检 Token SSOT 消除双写

**日期：** 2026-07-01
**对应设计：** `docs/superpowers/specs/2026-07-01-relay-precheck-ssot-in-process-design.md`
**结论：** ✅ 绿灯 — 纯内部重构，权限模型不变

---

## 1. 权限影响矩阵

### 1.1 变更端点分析

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| `POST .../relay-to-trae/token-init` | 认证用户 | Task (Workspace) | read | `IsAuthenticated` (ViewSet级) + `workspace_id` URL参数 | ⚠️ 隐式 | 暂无硬缺失；函数内部不做 workspace 归属校验（依赖框架层）— 已有模式，非本次引入 |
| `POST .../relay-to-trae/repo-credentials-precheck` | 认证用户 | Task (Workspace) | read | `IsAuthenticated` (ViewSet级) | ⚠️ 隐式 | 同上；预检是读操作，且 token 通过 Go SSOT 签发后才有意义 |
| `POST .../relay-to-trae/start` | 认证用户 | Task (Workspace) | write | `IsAuthenticated` (ViewSet级) | ⚠️ 隐式 | 同上；启动操作应确保 workspace 成员身份，但这是已有模式 |
| `POST .../relay-to-trae/register` | 认证用户 | Task (Workspace) | write | `IsAuthenticated` (ViewSet级) | ⚠️ 隐式 | 同上 |
| `POST /v1/token/init` (Go, 新增 Django→Go 调用) | **Django 内部服务** | System | write | **无认证**（仅监听 127.0.0.1:8015） | ⚠️ 网络边界 | Go 的 `/v1/token/init` 无任何 internal secret 验证；当前仅靠 localhost 绑定保护。**建议后续 Phase 添加 `X-TaskGateway-Internal-Secret` header 验证** |

### 1.2 本次变更的权限差量：**零**

本次变更**不改变**任何端点的：
- URL 路径
- HTTP 方法
- 权限类 (`permission_classes`)
- 认证类 (`authentication_classes`)
- 主体/资源/操作语义

变更的本质是将 token 的**来源**从 `Django CloudServerConfig + SQLite hack` 切换为 `Go /v1/token/init HTTP 调用`。对 API 消费者（前端、容器）完全透明。

---

## 2. 新增角色/权限建模

**不适用。** 本次变更不引入新角色、新权限粒度或新跨资源访问模式。

---

## 3. 安全审查结论

| 检查项 | 状态 | 说明 |
|--------|------|------|
| **IDOR 风险** | ✅ 无新增 | URL 路径和参数不变；token 签发改为 Go SSOT 但 scope (tenant/workspace/task) 由 Django 传入，Django 端的 scope 构造来自已有 `_extract_relay_start_context()` |
| **权限提升** | ✅ 无新增 | 所有 endpoint 行为语义不变：token_init 返回 token 信息、precheck 返回凭证状态、start 发起启动 |
| **跨租户泄露** | ✅ 无新增 | Go `IssueToken` 按 `(tenant_id, task_id)` 查询，不会跨租户返回其他租户的 token；Django 传入的 tenant_id 来自请求上下文 |
| **403 vs 404** | ✅ 不变 | 现有模式继续 |
| **user_id 注入** | ✅ 不适用 | 接口不涉及 user_id 参数 |
| **敏感操作审计** | ✅ 保持 | `append_container_token_audit_event` 调用保留在 Django 侧；Go 内 `IssueToken` 也有审计事件 |
| **Go `/v1/token/init` 无认证** | ⚠️ 已有风险 | 设计标注为 "Internal token init endpoint for Django"，但无任何 secret/token 验证。仅靠 `127.0.0.1:8015` 绑定保护。**非本次引入，但建议独立 Phase 加固**（添加 `X-TaskGateway-Internal-Secret` header） |

### 3.1 Go `/v1/token/init` 无认证的风险评估

**风险等级：** 低（当前）
- Go 服务仅监听 `127.0.0.1:8015`，外部不可达
- 若攻击者已在宿主机获得 shell，可调用 `/v1/token/init` 为任意任务签发 token
- 但拥有宿主机 shell 的攻击者已有更多直接途径（读 SQLite、调其他服务）

**建议：** 作为独立 Phase，为 `/v1/token/init` 添加 `X-TaskGateway-Internal-Secret` header 验证（与 `_relay_headers()` 中 `X-Relay-To-Trae-Secret` 模式一致）。

---

## 4. 需新增的权限测试用例

**本次变更不引入新的权限测试需求。** 原因：

- 端点权限模型不变（`IsAuthenticated`）
- 所有 endpoint 行为语义不变
- 测试重点应在**功能正确性**（Go 调用成功/失败/超时场景），而非权限

如果后续 Phase 为 Go `/v1/token/init` 添加 internal secret 认证，则需新增：

| 测试场景 | 角色 | 操作 | 预期 |
|----------|------|------|------|
| 无 secret header 调 Go `/v1/token/init` | 未认证调用方 | POST /v1/token/init | 401/403 |
| 错误 secret header 调 Go | 伪造调用方 | POST /v1/token/init | 401/403 |
| 正确 secret header 调 Go | Django | POST /v1/token/init | 200 |

---

## 5. 风险评级

| 风险 | 等级 | 缓解 |
|------|------|------|
| Go `/v1/token/init` 无认证 | 🟡 低 | 仅 localhost 绑定；建议独立 Phase 加固 |
| Django→Go HTTP 调用失败导致预检/启动 502 | 🟡 低 | Go 是 runAll 的一部分，Django 启动前 Go 已就绪；设计已包含 error handling |
| 权限边界无变化 | 🟢 零 | 纯内部重构 |
| 新攻击面 | 🟢 零 | 不引入新端点 |

---

## 6. 总结

**结论：✅ 绿灯。** 本次变更是纯内部重构，将 token 来源从 Django 双写切换为 Go SSOT。所有 API 端点、权限模型、认证机制、URL 结构均不变。唯一标记的是 Go `/v1/token/init` 缺少内部认证（已有风险，非本次引入），建议独立 Phase 加固。
