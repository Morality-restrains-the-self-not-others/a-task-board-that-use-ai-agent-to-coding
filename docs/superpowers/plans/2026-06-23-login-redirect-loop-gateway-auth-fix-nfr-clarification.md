# NFR 澄清: 登录跳转循环修复 — 网关统一认证

> 输入:
> - 设计文档: `docs/design/login-redirect-loop-fix.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-23-login-redirect-loop-gateway-auth-fix-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 安全性 | L3 | 网关 forward-auth 统一认证，secret 校验防 spoofing |
| 可用性 | L2 | 登录成功率 ≥ 99.5%，本地 token fallback 降级可用 |
| 容错机制 | L2 | taskAuth 不可达时本地 DB 回退，不阻塞用户登录后流程 |

## 逐增量 NFR 分析

### Increment 1: 网关统一认证修复

#### NFR 类别: 安全性
- **等级**: L3 - 增强（auth 领域强制）
- **量化目标**:
  - 网关 forward-auth 必须校验 `X-TaskAuth-Internal-Secret` 后才注入 `X-User-Id`
  - Django 侧必须校验 `X-TaskGateway-Internal-Secret` 匹配后才信任网关头
  - 客户端不得通过伪造 `X-User-Id` / `X-Gateway-Auth-Verified` 绕过认证（transformer 插件剥离）
- **质量场景**: QS-01

#### NFR 类别: 可用性
- **等级**: L2 - 标准
- **量化目标**: 登录成功后 `/api/accounts/users/profile/` 首次调用成功率 ≥ 99.5%
- **质量场景**: QS-02

#### NFR 类别: 容错机制
- **等级**: L2 - 标准
- **量化目标**: taskAuth HTTP 不可达时，本地 token fallback 在 50ms 内完成解析
- **质量场景**: QS-03

## 质量场景

### QS-01: 网关反 spoofing
| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L3 |
| 刺激源 | 恶意客户端 |
| 刺激 | 发送请求时伪造 `X-User-Id: admin-id` 和 `X-Gateway-Auth-Verified: 1` |
| 制品 | APISIX transformer 插件 + Django CustomTokenAuthentication |
| 环境 | 正常 |
| 响应 | transformer 剥离客户端伪造头；Django 校验 `X-TaskGateway-Internal-Secret` 失败 → 拒绝 |
| 响应度量 | 无网关头或 secret 不匹配时返回 401/403，不通过认证 |

### QS-02: 登录后认证成功
| 要素 | 内容 |
|------|------|
| 类别 | 可用性 |
| 等级 | L2 |
| 刺激源 | 已登录用户 |
| 刺激 | `window.location` 跳转后 router guard 触发 profile 请求 |
| 制品 | APISIX forward-auth → taskAuth → Django profile endpoint |
| 环境 | 正常负载 |
| 响应 | 200 + profile JSON |
| 响应度量 | 端到端成功率 ≥ 99.5%，服务端处理 P95 ≤ 500ms |

### QS-03: taskAuth 不可达时本地回退
| 要素 | 内容 |
|------|------|
| 类别 | 容错机制 |
| 等级 | L2 |
| 刺激源 | 系统运维事件 / taskAuth 重启 |
| 刺激 | taskAuth HTTP resolve 超时或连接拒绝 |
| 制品 | `load_principal_from_token` fallback 路径 |
| 环境 | 降级 |
| 响应 | 查本地 `accounts_customtoken` 表 → 返回 principal |
| 响应度量 | 解析耗时 ≤ 50ms，不抛出 503 |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| 网关统一认证 (L3 安全) | 认证决策点从 Django 上移到网关；Django 侧 Token Auth 变为降级路径 | `CustomTokenAuthentication` 保持现有网关头信任逻辑，不新增领域概念 |
| 本地 token fallback (L2 容错) | token 需在 Django DB 有副本；`token_registry` 增加 DB 实现 | 新增 `resolve_local_token` / `upsert_local_token` 仓储方法 |
| 响应头传播 (L2 可用性) | delegate 层不再丢弃 taskAuth 响应元数据 | `delegate_post` 返回值从 `(status, data)` 扩展为 `(status, data, headers)` |

## 权衡与边界

### 取舍
- 选择本地 DB fallback 而非纯内存（内存丢失重启 → 所有用户需重新登录），接受一次 DB 写操作的开销（~5ms）
- delegate 传播 Set-Cookie 保持向后兼容，即使 token 认证不依赖 session

### 明确不做什么
- 不修改 taskAuth Go 服务代码（token 创建/解析逻辑不变）
- 不修改 APISIX 路由配置（forward-auth 插件配置已就绪）
- 不修改前端代码
- 不做 token 轮转或过期机制（保持现有策略）

### 升级触发条件
- 当本地 token fallback 命中率 > 10% → taskAuth 可用性告警，需排查
- 当网关 forward-auth 拒绝率 > 1% → secret 配置或 token 签发异常

## 跳过声明
- **性能**: 跳过。本次不引入新数据流，现有性能基线不变。
- **可伸缩性**: 跳过。token 本地表日增行数较低（用户数级别），无需特殊伸缩设计。
- **数据一致性**: 跳过。token 本地副本与 taskAuth 主本短暂不一致可接受（登录成功即写入，毫秒级窗口）。
- **可观测性**: 跳过。现有 trace 链路已覆盖认证路径。
- **合规与隐私**: 跳过。不涉及新数据类型或跨境场景。
- **可维护性**: 跳过。代码改动量小（3 文件），无 API 接口变更。
