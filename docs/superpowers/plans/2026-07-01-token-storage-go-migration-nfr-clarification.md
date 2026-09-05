# NFR 澄清: Container Token 存储迁移至 Go taskCredentialService

> 输入:
> - 设计文档: brainstorming session (token exchange failed: TOKEN_ACCESS_INVALID)
> - 价值流文档: `docs/superpowers/plans/2026-07-01-token-storage-go-migration-value-stream.md`
>
> 输出使用者: `/6-ddd-领域设计驱动`, `/7-plans-实施计划`, `/8-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L2 | Go validate-token 本地调用 P95 ≤ 10ms，不劣于当前 SQLite 直查 |
| 可用性 | L1 | Go 不可达时 Django 返回 503（非静默失败），无 HA 要求 |
| 安全性 | L2 | `/v1/token/validate` 用 internal secret 保护，127.0.0.1 绑定 |
| 数据一致性 | L2 | Go tokens.sqlite3 为 token SSOT；Django CloudServerConfig 仅存业务字段 |
| 容错机制 | L1 | Go 不可达时 Django 下游 view 返回 503；exchange-refresh 本身无降级 |
| 可观测性 | L2 | trace_id 在 taskAgentSupport → Go → Django 全链路传递 |
| 可维护性 | L2 | 向后兼容——Django CloudServerConfig token 字段保留（仅停写不停读） |
| 可伸缩性 | L0 | 不适用——单实例本地部署，无水平扩展需求 |
| 合规与隐私 | L0 | 不适用——无 PII 变更 |

## 逐增量 NFR 分析

### Increment 1: Go exchange-refresh + refresh-access

**性能**: L2 — 本地 SQLite 操作，P95 ≤ 5ms
**安全性**: L2 — AllowAny + access_token 校验不变；新增 internal secret check 防止绕过 taskAgentSupport
**数据一致性**: L2 — Go tokens.sqlite3 为 token SSOT。exchange-refresh 使用 `UpdateAccessToken` + `UpdateRefreshToken` 两个独立 UPDATE（非原子事务），但业务上可接受——失败重试即可

### Increment 2: taskAgentSupport 路由分叉

**性能**: L2 — 路由判断在内存中完成（`if action == "exchange-refresh"`），零额外延迟
**可观测性**: L2 — `forwardToCredentialService` 必须传递 trace_id header，确保链路不断

### Increment 3: Go /v1/token/validate

**性能**: L2 — 本地 SQLite `SELECT ... WHERE container_access_token = ?`，P95 ≤ 3ms
**安全性**: L2 — 仅绑定 127.0.0.1 + internal secret header 校验；不允许外部网络可达
**容错**: L1 — 无断路器（本地调用失败即 503），不重试

### Increment 4: Django 下游 view 切换到 Go validate

**性能**: L2 — 增加一次 localhost HTTP roundtrip（~2-5ms），替换原有 SQLite 查询（~1ms）。净增 ~3ms，在心跳/回调场景可接受
**可用性**: L1 — Go 不可达时返回 503 + `error_code: "TOKEN_VALIDATE_SERVICE_UNAVAILABLE"`，不静默失败

### Increment 5: Django internal_dispatch 清理

**可维护性**: L2 — 移除死代码，无功能影响

### Increment 6: 测试更新

**可观测性**: L2 — 测试覆盖 Go 调用失败场景

## 质量场景

### QS-01: exchange-refresh 性能不退化
| 要素 | 内容 |
|------|------|
| 类别 | 性能 |
| 等级 | L2 |
| 刺激源 | go_relayToTrae 发起 token exchange |
| 刺激 | POST exchange-refresh（含 access_token） |
| 制品 | taskCredentialService exchange-refresh handler |
| 环境 | 正常负载 |
| 响应 | 200 + refresh_token |
| 响应度量 | 服务端处理时间 P95 ≤ 50ms（含 2 次 SQLite UPDATE） |

### QS-02: Django validate-token 调用性能
| 要素 | 内容 |
|------|------|
| 类别 | 性能 |
| 等级 | L2 |
| 刺激源 | Django heartbeat view |
| 刺激 | 调用 Go /v1/token/validate |
| 制品 | Go taskCredentialService |
| 环境 | 正常负载 |
| 响应 | 200 + {valid, company_id, workspace_id, task_id} |
| 响应度量 | 端到端（Django HTTP 调用 → Go 响应）P95 ≤ 15ms |

### QS-03: Go 不可达时 Django 不静默失败
| 要素 | 内容 |
|------|------|
| 类别 | 容错机制 |
| 等级 | L1 |
| 刺激源 | Go taskCredentialService 进程宕机 |
| 刺激 | Django 调用 /v1/token/validate 连接拒绝 |
| 制品 | Django `_validate_token_via_go()` |
| 环境 | 故障 |
| 响应 | 返回 503 + `error_code: "TOKEN_VALIDATE_SERVICE_UNAVAILABLE"` |
| 响应度量 | 日志记录 ERROR 级别，trace_id 保留 |

### QS-04: 未授权访问 /v1/token/validate
| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L2 |
| 刺激源 | 外部攻击者（绕过 APISIX 直接访问 8015 端口） |
| 刺激 | POST /v1/token/validate 不带 internal secret |
| 制品 | Go validate-token handler |
| 环境 | 正常 |
| 响应 | 403 Forbidden |
| 响应度量 | 不泄露 token 是否存在的信息 |

### QS-05: trace_id 全链路传递
| 要素 | 内容 |
|------|------|
| 类别 | 可观测性 |
| 等级 | L2 |
| 刺激源 | 容器进程发起 exchange-refresh |
| 刺激 | POST 请求带 X-Trace-Id header |
| 制品 | taskAgentSupport → Go taskCredentialService |
| 环境 | 正常 |
| 响应 | Go 日志中记录 trace_id |
| 响应度量 | Loki 可按 trace_id 串联 taskAgentSupport + Go + Django 三端日志 |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| Go 为 token SSOT (L2 一致性) | `ContainerToken` 聚合在 Go 侧，Django 仅持有 `CloudServerConfig`（业务数据） | Go domain: 新增 `ExchangeRefresh`, `RefreshAccess` 领域服务方法 |
| Django 通过 RPC 获取 scope | `CloudServerConfig` 与 `ContainerToken` 属于不同限界上下文，通过 `/v1/token/validate` 通信 | Django: `_resolve_cfg_by_access_token_for_callback` 改为两步（RPC 验证 + scope 查本地） |
| 内部端点需 secret 保护 | Go handler 层新增认证中间件 | Infrastructure: `InternalSecretMiddleware` 包装 HTTP handler |

## 权衡与边界

### 取舍
- 选择 Django → Go HTTP RPC 调用（~3ms overhead）而非直接 SQLite 跨语言读，以换取清晰的 BC 边界
- exchange-refresh 的两个 SQLite UPDATE 非原子（无事务包裹），接受极低概率的中间态，换取实现简洁性
- Django CloudServerConfig token 字段保留不删（Phase 2 清理），接受短期数据冗余，降低回滚风险

### 明确不做什么
- 不在 Go 和 Django 之间实现分布式事务（无 2PC/Saga）
- 不做 Go 的 HA/多实例部署（单实例满足当前规模）
- 不做 token 验证结果缓存（每次调用都查 Go，避免 cache invalidation 复杂度）
- Django CloudServerConfig 的 token 字段不立即删除（保留以支持快速回滚）

### 升级触发条件
- 日活任务启动 > 1000 时：Go validate-token 考虑增加内存缓存（L1→L2 性能）
- 需要多实例部署时：Go SQLite → PostgreSQL 迁移（L1→L2 可用性）
- 客户要求 SOC2 时：审计日志从 Go token_audit_events 扩展到全量操作记录（L2→L3 安全性）

## 跳过声明
- **可伸缩性**: 跳过。单实例本地部署，无水平扩展需求。
- **合规与隐私**: 跳过。Token 为 opaque random string，不含 PII。
