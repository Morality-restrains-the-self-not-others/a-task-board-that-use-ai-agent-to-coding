# 领域模型: relay register 换票窗口保护

> 输入:
> - 设计: `docs/superpowers/specs/2026-05-29-relay-register-token-protection-design.md`
> - 价值流: `docs/superpowers/plans/2026-05-29-relay-register-token-protection-value-stream.md`
> - NFR: `docs/superpowers/plans/2026-05-29-relay-register-token-protection-nfr-clarification.md`
>
> 输出使用者: `/6-plans-实施计划`, `/7-build-构建`

## 限界上下文

| 上下文 | 职责 | 本增量变更 |
|--------|------|-----------|
| **Container Token Lifecycle** | access/refresh 换票、bootstrap | 新增 TEIP 相位判定与 bootstrap 阻断规则 |
| **Relay Startup Orchestration** | token-init → start → register 编排 | register 与 token 签发解耦 |
| **Relay Observability** | token 审计事件 | 新增 `token_bootstrap_blocked` 等事件类型 |

上下文间通信：Relay Startup 调用 Token Lifecycle 的 **只读相位查询** + **条件 bootstrap**；阻断时发布领域事件供审计上下文消费。

## 值对象

### `TokenExchangePhase`（新增）

```python
@dataclass(frozen=True)
class TokenExchangePhase:
    BOOTSTRAPPED = "bootstrapped"       # 有 access，无 refresh
    EXCHANGE_IN_PROGRESS = "teip"       # 有 refresh，无 access（TEIP）
    ACTIVE = "active"                   # 有 access + refresh
    UNINITIALIZED = "uninitialized"     # 皆空
```

**派生规则（纯函数，放在 VO 或聚合方法）：**

```text
derive_phase(access, refresh, expires_at) -> TokenExchangePhase
  refresh 非空 AND access 空 AND expires_at 空 → EXCHANGE_IN_PROGRESS
  access 非空 AND refresh 空 → BOOTSTRAPPED
  access 非空 AND refresh 非空 → ACTIVE
  else → UNINITIALIZED
```

与 NFR QS-01 对齐：`EXCHANGE_IN_PROGRESS` 等价 TEIP。

### `RelayRegisterIntent`（新增）

```python
@dataclass(frozen=True)
class RelayRegisterIntent:
    REGISTER_ONLY = "register_only"           # 仅登记 scope 到 go_relay
    ISSUE_ACCESS_AND_REGISTER = "issue"     # 占位符 → bootstrap → register
```

应用层根据请求体 `register_only` 与当前 `TokenExchangePhase` 选择意图。

### 已有 VO 复用

- `TaskScope`, `AccessToken`, `RefreshToken`, `BusinessApiEndpoint`
- `RelayStartPhase` — 补充语义：当 `START_DISPATCHING` 且 TEIP 时，禁止 `ISSUE_ACCESS`

## 实体与聚合

### 聚合根：`ContainerTokenSession`（扩展既有实体）

**文件：** `cloud/domain/entities/container_token_session.py`

**新增方法：**

| 方法 | 行为 |
|------|------|
| `exchange_phase() -> TokenExchangePhase` | 由当前 token 字段派生 |
| `is_exchange_in_progress() -> bool` | `exchange_phase() == EXCHANGE_IN_PROGRESS` |
| `assert_bootstrap_allowed() -> None` | TEIP 时 `raise TokenBootstrapBlocked` |
| `bootstrap_access(...)` | **重命名/包装**现有 bootstrap 路径；入口先 `assert_bootstrap_allowed()` |

**不变量（新增）：**

1. `EXCHANGE_IN_PROGRESS` 状态下，禁止 `exchange_refresh` 以外的写操作清空 `refresh`。
2. `bootstrap_access` 仅在 `BOOTSTRAPPED` 或 `UNINITIALIZED`（且无 refresh）时允许。

既有 `exchange_refresh` / `refresh_access` 方法不变。

### 聚合根：`RelayStartupSession`（扩展既有实体）

**文件：** `cloud/domain/entities/relay_startup_session.py`

**规则扩展：**

- 当 `phase in (START_DISPATCHING, START_ACCEPTED)` 且关联 `ContainerTokenSession.is_exchange_in_progress()`：
  - `RelayRegisterIntent` 强制为 `REGISTER_ONLY`
- register 不得触发 `build_relay_to_trae_runtime_env` 的 bootstrap 分支

### 实体：`RelayTaskRegistration`（既有，语义收紧）

**文件：** `cloud/domain/entities/relay_task_registration.py`

- 登记记录 **task scope + task_api_origin**，可选关联 `access_token_suffix`（审计用）
- **不拥有** token 签发职责；token 引用只读快照

## 领域服务

### `ContainerTokenBootstrapPolicy`（新增）

```python
class ContainerTokenBootstrapPolicy:
    def resolve_bootstrap_action(
        self,
        *,
        session: ContainerTokenSession,
        intent: RelayRegisterIntent,
    ) -> BootstrapAction:  # ALLOW | BLOCK | REGISTER_ONLY
```

| session.phase | intent | 结果 |
|---------------|--------|------|
| EXCHANGE_IN_PROGRESS | ISSUE_ACCESS | BLOCK → 事件 `TokenBootstrapBlocked` |
| EXCHANGE_IN_PROGRESS | REGISTER_ONLY | REGISTER_ONLY |
| BOOTSTRAPPED | ISSUE_ACCESS | ALLOW（复用现有 access 或 regenerate 按现有规则） |
| ACTIVE | ISSUE_ACCESS | ALLOW 复用现有 access（`_issue_relay_access_token` 现有逻辑） |

应用服务 `relay_to_trae_register` / `_replace_access_token_placeholder` 调用此策略，而非内联 if。

### 既有服务扩展

**`ContainerTokenLifecycleService`**

- `refresh_access_token` 不变
- 新增：`get_exchange_phase(scope) -> TokenExchangePhase`（读模型）

**`RelayTwoStepStartupService`**

- `accept_async_start` 后标记 `START_DISPATCHING`；与 TEIP 窗口对齐（NFR 关联）

## 仓储接口

### 既有 — 无新表

- `ContainerTokenSessionRepository` — 增加 `find_by_scope_for_update(scope)`（应用层 TEIP 写保护用，基础设施 `select_for_update`）
- `ContainerTokenAuditEventRepository` — 写入新事件类型
- `RelayTaskRegistrationRepository` — 不变

### 读模型缓存（应用层，非聚合）

**`StatusPushCfgCache`**（`relay_to_trae_status.py`）

- 领域规则：token 轮换事件（`RefreshAccess` / `ExchangeRefresh`）触发 `invalidate(access_token_digest)`
- 实现留在基础设施/应用层，领域事件 `AccessTokenRotated` 作为契约

## 领域事件

| 事件 | 触发 | 消费者 |
|------|------|--------|
| `TokenBootstrapBlocked`（新增） | TEIP 下尝试 bootstrap | 审计 `token_bootstrap_blocked` |
| `RelayRegisterReusedState`（新增） | register-only 成功 | 审计 `relay_register_reused_state` |
| `RefreshTokenExchanged`（既有） | exchange-refresh | 审计 + **缓存失效** |
| `AccessTokenRefreshed`（既有） | refresh-access | 审计 + **缓存失效** |

**`TokenAuditEventTypes` 扩展：**

```python
TOKEN_BOOTSTRAP_BLOCKED = "token_bootstrap_blocked"
RELAY_REGISTER_REUSED_STATE = "relay_register_reused_state"
```

## 应用层编排（Increment 1 映射）

```text
relay_to_trae_register(request):
  scope = extract_scope(request)
  intent = resolve_register_intent(request)  # register_only flag
  session = repo.find_by_scope(scope)
  action = bootstrap_policy.resolve(session, intent)
  match action:
    BLOCK → emit TokenBootstrapBlocked; return 409 or register-only fallback
    REGISTER_ONLY → forward /v1/register without access
    ALLOW → existing _issue_relay_access_token path
```

```text
_replace_access_token_placeholder(..., caller=...):
  session = load_cfg_as_session()
  if session.is_exchange_in_progress():
    session.assert_bootstrap_allowed()  # raises → caught → no DB mutation
```

## 与现有代码映射

| 领域概念 | 现有实现 | 变更 |
|----------|----------|------|
| `ContainerTokenSession` | `container_token_session.py` + `DjangoContainerTokenSessionRepository` | 加 phase 方法与 assert |
| TEIP 判定 | 无（隐式字段组合） | 显式 `TokenExchangePhase` |
| register 签发 | `relay_to_trae_register` → `_issue_relay_access_token` | 经 `ContainerTokenBootstrapPolicy` |
| bootstrap 清空 refresh | `mock_run_container._replace_access_token_placeholder` | TEIP guard |
| 审计 | `append_container_token_audit_event` | 新 event_type |

## DDD 自检

- [x] 领域层无 Django ORM 导入（策略/相位在 entity + domain service）
- [x] TEIP 不变量在聚合内表达（NFR L3 强一致）
- [x] register 与 bootstrap 分离（Relay vs Token 上下文）
- [x] 事件驱动审计与缓存失效
- [x] 复用既有 `ContainerTokenSession` / `RelayStartupSession`，最小增量

## 实施顺序建议（供 `/6-plans`）

1. `TokenExchangePhase` VO + `ContainerTokenSession.is_exchange_in_progress()`
2. `ContainerTokenBootstrapPolicy` + 单测
3. 接入 `_replace_access_token_placeholder` / `relay_to_trae_register`
4. 审计事件类型 + pytest QS-01~04
5. 前端 Increment 2 + go_relay Increment 3
