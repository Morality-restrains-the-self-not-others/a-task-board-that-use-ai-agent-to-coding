# NFR 澄清: relay register 换票窗口保护

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-05-29-relay-register-token-protection-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-05-29-relay-register-token-protection-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 数据一致性 | L3 | TEIP 窗口内 refresh 不得被并发 bootstrap 清空；换票两步原子语义可验证 |
| 容错机制 | L2 | register 在 TEIP 降级为 register-only，不 500；refresh-access 失败 go_relay 快速失败 |
| 可观测性 | L2 | 每次阻断写审计 `token_bootstrap_blocked` + `error_code` + `trace_id` |
| 性能 | L1 | 本地 dev SQLite，TEIP 判定为 O(1) 字段检查，不引入分布式锁 |
| 可用性 | L1 | 无专项 SLA；换票失败用户重试直启即可 |
| 安全性 | L2 | 不在审计/日志落盘明文 token；register-only 不扩大鉴权面 |
| 可维护性 | L2 | API 向后兼容：旧客户端 register+占位符在 TEIP 自动降级 |
| 可伸缩性 | L0 | 不适用（单任务单 SQLite 行） |
| 合规与隐私 | L0 | 不适用 |

## 逐增量 NFR 分析

### Increment 1: Django TEIP 硬保护

#### 数据一致性 — L3
- **量化目标：** `exchange_refresh` 成功后 5s 内任意 `relay_register` / `_issue_relay_access_token` 不得使 `container_refresh_token` 变空，除非同一请求完成 `refresh_access`。
- **质量场景：** QS-01、QS-02

#### 容错机制 — L2
- **量化目标：** TEIP 下 register 返回 200（register-only）或 409（需 access 的 precheck），不得 500。
- **质量场景：** QS-03

#### 可观测性 — L2
- **量化目标：** 100% TEIP 阻断写 `token_bootstrap_blocked` 或 `relay_register_reused_state`，含 `trace_id`。
- **质量场景：** QS-04

#### 性能 — L1
- TEIP 判定仅读 `CloudServerConfig` 三字段，无额外查询；acceptable for dev 直启路径。

### Increment 2: 前端 register 语义收敛

#### 可维护性 — L2
- 删除冗余 register 调用，不改变用户可见启动步骤数。
- Playwright 回归：直启日志不得含 `无效的 access_token`。

#### 数据一致性 — L2（辅助）
- 减少竞态触发频率，但 **不替代** Increment 1 的后端硬保护。

### Increment 3: 关联加固（Enhancement）

#### 容错机制 — L3（本增量内）
- go_relay：`exchange-refresh` 成功后 `refresh-access` 失败 → 启动失败，禁止 fallback 旧 access。
- **质量场景：** QS-05

#### 数据一致性 — L2
- status-push cfg 缓存在 token 轮换后失效，缓存 TTL 不得掩盖 401 超过 5s。

## 质量场景

### QS-01: TEIP 期间 register 不得清空 refresh
| 要素 | 内容 |
|------|------|
| 类别 | 数据一致性 |
| 等级 | L3 |
| 刺激源 | 前端 `start` 202 后并发 `relay register`（issueTokenOnServer） |
| 刺激 | `exchange_refresh` 已写入 refresh，`access` 已清空 |
| 制品 | `_replace_access_token_placeholder` / `relay_to_trae_register` |
| 环境 | 本地 runAll，单任务单 worker |
| 响应 | DB `container_refresh_token` 保持不变；后续 `refresh_access` 可 200 |
| 响应度量 | pytest：`exchange_refresh` → `relay_register(placeholder)` → assert refresh 未变 → `refresh_access` 200 |

### QS-02: 完整换票后 access 有效
| 要素 | 内容 |
|------|------|
| 类别 | 数据一致性 |
| 等级 | L3 |
| 刺激源 | go_relay `performTokenExchange` |
| 刺激 | 正常两步换票 + 期间一次 register-only |
| 制品 | `CloudServerConfig` + `register-reachability` |
| 环境 | 正常直启 |
| 响应 | 新 `container_access_token` 非空；onlineServiceJS 不报 `TOKEN_ACCESS_INVALID` |
| 响应度量 | Playwright 直启用例 + `test_exchange_refresh_then_refresh_access_updates_db` 回归绿 |

### QS-03: TEIP 下 register 可恢复
| 要素 | 内容 |
|------|------|
| 类别 | 容错机制 |
| 等级 | L2 |
| 刺激源 | 前端 health/register |
| 刺激 | TEIP 状态下 register 带占位符 |
| 制品 | `relay_to_trae_register` |
| 环境 | 换票窗口内 |
| 响应 | HTTP 200 register-only 或 409 `TOKEN_EXCHANGE_IN_PROGRESS`（precheck 需 token 时） |
| 响应度量 | 无 500；响应 body 含 `error_code`（409 时） |

### QS-04: 阻断可审计追溯
| 要素 | 内容 |
|------|------|
| 类别 | 可观测性 |
| 等级 | L2 |
| 刺激源 | Django bootstrap 阻断逻辑 |
| 刺激 | TEIP 下尝试 server-issue |
| 制品 | `cloud_container_token_audit_event` |
| 环境 | 任意 |
| 响应 | 写入 `event_type=token_bootstrap_blocked`，`error_code=TOKEN_EXCHANGE_IN_PROGRESS` |
| 响应度量 | pytest 查审计表；Grafana trace 可按 `trace_id` 串联 exchange_refresh → blocked |

### QS-05: 换票半失败不得启动子进程
| 要素 | 内容 |
|------|------|
| 类别 | 容错机制 |
| 等级 | L3 |
| 刺激源 | go_relay `startOnlineService` |
| 刺激 | `exchange-refresh` 成功、`refresh-access` 401 |
| 制品 | `go_relayToTrae/src/process.go` |
| 环境 | 模拟 refresh 被清空的故障（Increment 1 未部署时的回归） |
| 响应 | 不向 onlineServiceJS 传递已作废 access；启动返回 error |
| 响应度量 | go 单测或集成日志无 `using original token` |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| TEIP 强一致 (L3) | `ContainerTokenSession` 需显式生命周期态（Bootstrap / Exchanging / Active） | 在聚合上增加 `TokenExchangePhase` 值对象或派生属性 `is_exchange_in_progress()` |
| 审计 L2 | 阻断为领域事件 | 新增 `TokenBootstrapBlocked` 领域事件，由应用服务写入审计 |
| register-only L2 | Relay 登记与 token 签发分离 | `RelayStartupSession` 与 `ContainerTokenSession` 限界上下文接口分离：`register_task(scope)` 不调用 `bootstrap_access()` |
| 幂等 register L2 | 重复 register 不改变 token 态 | `register_task` 用幂等语义：TEIP 时 no-op on token fields |
| 乐观锁（可选 follow-up） | 并发写 `CloudServerConfig` | 实体 `version` 字段；本增量可先用 `select_for_update` 已有路径 |

## 权衡与边界

### 取舍
- 选择 **TEIP 期间禁止 bootstrap**（L3 一致性），换取直启路径正确性；接受 register 在换票窗口内不能重签 access。
- 选择 **register-only 自动降级**（L2 容错），而非一律 409，避免 health 轮询打断 UI 在线态。
- Increment 3 go_relay fail-fast **优先于** 长时间 status-push 误报在线（缓存失效）。

### 明确不做什么
- **V1 不做 TEIP 超时自动清理**（Future）：go_relay 崩溃后 refresh 残留需用户「停止 → 重试」；不在本增量引入定时任务清库。
- **不做分布式锁 / Redis**：本地 SQLite 单进程 dev 场景，L0 可伸缩性。
- **不合并 token-init 与 start API**（startup-storm Phase 2）。
- **不在 NFR 层要求 P99 换票延迟**：性能 L1，功能正确优先。

### 升级触发条件
- 若生产环境改为 **PostgreSQL + 多 worker**：TEIP 保护需升级为行级锁或 `version` 乐观锁 + 重试（一致性升至 L4）。
- 若 TEIP 残留导致支持工单 > 1 次/周：实现 TEIP TTL（设计文档开放问题 #1）。
- 若需跨 region relay：register-only 需携带 token 版本向量（因果一致 L3）。

## 跳过声明

- **可伸缩性 L0：** 单任务 `CloudServerConfig` 行级状态，无水平扩展需求。
- **合规与隐私 L0：** 无新个人数据字段；审计仍仅 hash/suffix。
- **可用性 L1：** 开发工具链特性，非 7×24 对外 SLA。
