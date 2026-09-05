# NFR 澄清: gitOauth 绑定失败留痕与状态化凭据

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-05-22-oauth-token-fetch-timeout-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-05-26-gitoauth-bind-failure-persistence-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L1 | OAuth callback 增量写库不引入额外远程调用，单次额外 DB 写入开销目标 P95 ≤ 30ms（服务端） |
| 安全性 | L2 | 失败原因字段禁止写入 token 明文；审计与响应中敏感键统一脱敏 |
| 数据一致性 | L2 | 凭据状态从 `pending -> active/failed` 单调迁移；`access-for-user` 仅消费 `active` |
| 可观测性 | L2 | bind 失败 100% 可在 `bind_status/bind_error` 查询到；回调失败链路可关联日志定位 |
| 可维护性 | L2 | 保持既有接口向后兼容（新增字段为扩展项），并有回归测试覆盖失败留痕场景 |
| 可用性 | L1 | 失败时可重试绑定，不新增 HA 架构或跨实例一致性机制 |

## 逐增量 NFR 分析

### Increment 1: 回调先落库薄切片

#### NFR 类别: 数据一致性
- **等级**: L2 - 标准
- **量化目标**:
  - OAuth callback 在换票与用户信息确认后，必须先写入凭据行，`bind_status=pending`
  - 不允许“后续失败直接删行”导致观测空洞
- **约束**: 不要求跨系统分布式事务，仅保证 gitOauth 本地事务内一致性

#### NFR 类别: 性能
- **等级**: L1 - 基础
- **量化目标**: 单次 callback 相比原流程新增写库步骤 P95 增量开销 ≤ 30ms（服务端）
- **约束**: 不做索引重构与批量写优化

### Increment 2: 绑定结果状态化

#### NFR 类别: 可观测性
- **等级**: L2 - 标准
- **量化目标**:
  - bind HTTP 非 200 时，`bind_status=failed` 与 `bind_error` 必须同事务写入
  - 失败原因编码统一前缀（如 `bind_http_*`、`bind_request_exception:*`）
- **约束**: 失败原因字符串上限 512，不做复杂错误 taxonomy 扩展

#### NFR 类别: 安全性
- **等级**: L2 - 标准
- **量化目标**: `bind_error`、审计 `detail`、响应字段均不得出现 access/refresh token 明文
- **约束**: 保持现有脱敏规则（key 含 token/secret/authorization 统一 redacted）

### Increment 3: 消费侧契约收敛

#### NFR 类别: 数据一致性
- **等级**: L2 - 标准
- **量化目标**:
  - `access-for-user` 查询条件强制 `bind_status=active`
  - `summary-for-user` 返回 `bind_status/bind_error`，让消费者可判定可用性
- **约束**: 允许短暂最终一致窗口（同一请求周期内无跨服务强一致保证）

#### NFR 类别: 可维护性
- **等级**: L2 - 标准
- **量化目标**:
  - 旧客户端不识别新字段时不影响既有行为（兼容扩展）
  - 关键路径测试覆盖：callback 失败留痕、summary 字段扩展、active 过滤
- **约束**: 本次不引入 API 版本分支，仅做向后兼容扩展

### Increment 4: 回归与可观测加固

#### NFR 类别: 可维护性
- **等级**: L2 - 标准
- **量化目标**:
  - `gitOauth` API 测试必须覆盖 GitHub/GitLab 两种 bind 失败留痕
  - 迁移执行后 schema 与测试通过率维持 100%
- **约束**: 不增加独立压测基线，本轮以功能回归为主

#### NFR 类别: 可用性
- **等级**: L1 - 基础
- **量化目标**: 失败后允许用户重新触发绑定流程，不新增自动补偿队列
- **约束**: 不承诺跨实例重试去重与幂等令牌

## 质量场景

### QS-01: 回调失败留痕可见
| 要素 | 内容 |
|------|------|
| 类别 | 可观测性 |
| 等级 | L2 |
| 刺激源 | OAuth 回调链路 |
| 刺激 | 主站 bind 返回 HTTP 5xx |
| 制品 | `api_githubappusercredential` 记录 |
| 环境 | 正常网络，主站短暂异常 |
| 响应 | 记录保留，`bind_status=failed`，`bind_error=bind_http_<code>` |
| 响应度量 | 集成测试断言同一 `provider/task2app_user_id/github_user_id` 行存在且字段匹配 |

### QS-02: 失败记录不可被换票消费
| 要素 | 内容 |
|------|------|
| 类别 | 数据一致性 |
| 等级 | L2 |
| 刺激源 | task2app 内部 `access-for-user` 调用 |
| 刺激 | 用户仅存在 `failed` 状态凭据 |
| 制品 | `/api/internal/*/oauth/access-for-user/` |
| 环境 | 正常 |
| 响应 | 不返回 access token，按未绑定语义返回 404 |
| 响应度量 | 单测/集成测试断言查询包含 `bind_status=active`，失败行不会被命中 |

### QS-03: 敏感信息不泄漏
| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L2 |
| 刺激源 | 日志/响应写入逻辑 |
| 刺激 | audit/detail 中包含 token 相关键 |
| 制品 | 审计落库、错误响应、`bind_error` |
| 环境 | 正常 |
| 响应 | token/secret/authorization 值被脱敏或拒绝入库 |
| 响应度量 | 测试断言敏感键值为 `<redacted>`，DB 中无明文 token |

### QS-04: 向后兼容摘要契约
| 要素 | 内容 |
|------|------|
| 类别 | 可维护性 |
| 等级 | L2 |
| 刺激源 | 旧版调用方 |
| 刺激 | 调用 `summary-for-user` 且忽略新增字段 |
| 制品 | `/api/internal/*/oauth/user-credential/summary-for-user/` |
| 环境 | 正常 |
| 响应 | 原有字段保持可用，新增字段仅扩展不破坏 |
| 响应度量 | 回归测试通过，旧字段断言不变，新字段可选读取 |

### QS-05: 回调写库路径基础性能
| 要素 | 内容 |
|------|------|
| 类别 | 性能 |
| 等级 | L1 |
| 刺激源 | OAuth 回调请求 |
| 刺激 | 单用户完成一次 callback |
| 制品 | callback 处理函数中的 `update_or_create + 状态更新` |
| 环境 | 正常负载 |
| 响应 | 仅新增有限 DB 操作，不引入额外外部依赖 |
| 响应度量 | 本地/测试环境观测 callback 处理时延无明显回归（P95 增量 ≤ 30ms） |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| 一致性 L2（状态单调迁移） | 凭据实体由“静态凭据”升级为“带生命周期状态的实体” | 在 OAuth Token Broker Context 中把 `GithubAppUserCredential` 作为聚合根，显式建模状态迁移规则 |
| 可观测性 L2（失败可追踪） | 失败原因成为领域内可查询属性，而非仅日志副产物 | 在领域服务中补充 `mark_bind_failed(reason)` 语义方法 |
| 安全性 L2（敏感字段脱敏） | 错误信息属于受控值对象，不能任意拼接原始异常 | 增加 `BindError` 值对象规范（code + safe_detail） |
| 可维护性 L2（兼容扩展） | 查询模型需容忍新增字段、老字段共存 | DDD 阶段区分命令模型（状态迁移）与查询 DTO（兼容输出） |

## 权衡与边界

### 取舍
- 选择 L2 一致性（状态机+可见失败）而不是 L3 分布式强一致，避免引入跨系统事务复杂度。
- 选择 L2 可观测（字段留痕）而不是独立事件流水服务，优先解决“记录消失”核心痛点。

### 明确不做什么
- 不在本次实现自动重试队列与失败补偿工作流。
- 不引入新的 OAuth 协议扩展或 provider 统一抽象重构。
- 不做跨服务全链路性能压测与容量规划升级。

### 升级触发条件
- 当 `bind_failed` 占比连续 7 天 > 3%：可观测从 L2 升级到 L3，需引入失败分类看板与告警。
- 当 callback P95 相比基线回退 > 20%：性能从 L1 升级到 L2，需做索引与写路径优化。
- 当出现多实例并发写冲突导致状态反转：一致性从 L2 升级，需增加版本号/乐观锁策略。

## 跳过声明

- 可伸缩性: 跳过。当前改动为字段与状态迁移逻辑，流量模型无新增入口。
- 容错机制: 跳过。仅记录失败并允许人工重试，不引入断路器/重试编排。
- 合规与隐私: 跳过。未新增个人敏感数据类型，仅增强已有凭据元数据状态。

## 自检 (Hard Gate)

- [x] 每个相关 NFR 类别都有明确的支撑等级（L0-L4）
- [x] 每个 L1-L4 的 NFR 类别至少有一个量化目标
- [x] 每个 L2-L4 的 NFR 类别至少有一个质量场景（QS）
- [x] 每个质量场景的响应度量可验证（给出了具体数字和测量方式）
- [x] 影响领域模型的 NFR 决策已标注（含具体 DDD 动作）
- [x] 权衡和边界已明确（知道不做什么 + 升级触发条件）
- [x] 跳过的 NFR 类别有理由说明
- [x] 文档位置正确：`docs/superpowers/plans/2026-05-26-gitoauth-binding-state-persistence-nfr-clarification.md`

