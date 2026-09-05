# NFR 澄清: taskAuth 认证拆分

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-05-28-taskauth-split-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-05-28-taskauth-split-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L2 | 登录 P95 ≤ 500ms（taskAuth 处理，不含 Django enrich） |
| 可伸缩性 | L1 | 单实例 + 共享 SQLite，日活 < 1 万 |
| 可用性 | L3 | taskAuth 宕机时 Django fallback，RTO < 30s |
| 安全性 | L3 | internal secret + Token 不变 + 凭证不落日志 |
| 数据一致性 | L2 | 共享 SQLite 读己之写；副作用最终一致（邮件/Kafka） |
| 容错机制 | L3 | 桥接超时 30s + Django fallback |
| 可观测性 | L1 | health 端点 + 结构化日志 |
| 合规与隐私 | L2 | 隐私条款校验仍经 Django 真源 |
| 可维护性 | L2 | API 路径不变；TASKAUTH_ENABLED 特性开关 |

## 逐增量 NFR 分析

### Increment 1: 登录薄切片

#### 性能 — L2
- P50 < 200ms, P95 < 500ms（taskAuth SQLite 查询 + token 写入）
- 质量场景: QS-01

#### 可用性 — L3
- taskAuth 不可达时 Django 本地 login 100% 接管
- 质量场景: QS-02

#### 安全性 — L3
- `X-TaskAuth-Internal-Secret` 保护 internal API
- Token key 40 字符 hex，与 Django CustomToken 一致
- 质量场景: QS-03

#### 数据一致性 — L2
- user/login_method/token 同一 SQLite 事务边界
- enrich-login 失败时仍返回 token + minimal user（降级）

### Increment 2: 邮箱注册闭环

#### 数据一致性 — L2
- 注册写 user + login_method 同事务
- 邮件/Kafka 经 Django post-register 异步副作用

#### 容错 — L2
- post-register 失败不阻塞 201 响应（记录日志）

### Increment 3–5

- 手机 OTP：仍 L2，forward Django
- 密码重置：L1，暂留 Django
- runAll 编排：L2 标准依赖链

## 质量场景

### QS-01: 正常负载邮箱登录
| 要素 | 内容 |
|------|------|
| 类别 | 性能 |
| 等级 | L2 |
| 刺激源 | Web 客户端 |
| 刺激 | POST `/api/accounts/users/login/` |
| 制品 | taskAuth login handler |
| 环境 | 正常负载 |
| 响应 | 200 + token + user |
| 响应度量 | taskAuth 处理 P95 ≤ 500ms（不含网络） |

### QS-02: taskAuth 服务不可用
| 要素 | 内容 |
|------|------|
| 类别 | 可用性 |
| 等级 | L3 |
| 刺激源 | Django taskauth_bridge |
| 刺激 | taskAuth 连接拒绝/超时 |
| 制品 | UserViewSet.login delegate |
| 环境 | taskAuth 宕机 |
| 响应 | fallback 到 Django 原逻辑，用户仍可登录 |
| 响应度量 | pytest `UserViewSet_login_test` 在 TASKAUTH_ENABLED=false 全绿 |

### QS-03: internal API 未授权访问
| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L3 |
| 刺激源 | 外部攻击者 |
| 刺激 | POST `/api/internal/taskauth/enrich-login/` 无 secret |
| 制品 | taskauth_internal_views |
| 环境 | 生产配置 secret 已设 |
| 响应 | 401 unauthorized |
| 响应度量 | 集成测试或手工 curl 验证 |

## 领域模型影响

| NFR 决策 | 模型影响 | DDD 动作 |
|----------|---------|---------|
| 可用性 L3 fallback | AuthBridge 需「尝试远程 / 本地」策略 | AuthDelegationService 接口 |
| 一致性 L2 共享 DB | User/LoginMethod/Token 同聚合读写 | CredentialAggregate 根 |
| 副作用最终一致 | 注册/激活发事件经 Django | UserRegistered / UserActivated 领域事件 |
| 安全 L3 internal | SideEffectGateway 防腐层 | DjangoCallbackPort 接口 |
| 容错 L3 超时 | 基础设施层实现 timeout，领域层纯接口 | bridge client 在 infra |

## 权衡与边界

### 取舍
- 共享 SQLite 换取强一致写入 vs 独立 auth DB 的最终一致
- enrich-login 同步调用 Django 换取完整 UserSerializer vs Go 侧重复序列化逻辑

### 明确不做什么
- V1 不做 taskAuth 独立 PostgreSQL 副本
- 不做 P99 < 100ms 极致优化
- 不做 SuperAdmin 完整迁移（逐步）

### 升级触发条件
- 日活 > 1 万 → 性能/可伸缩性升至 L3，考虑独立 auth DB + 连接池
- 多区域部署 → 一致性升至 L3，SQLite 改为集中式 PG

## 跳过声明

- **合规 GDPR 遗忘权**: L0，沿用 Django 账户删除流程
- **可观测性链路追踪**: L1，仅 health + log，不做全链路 trace
