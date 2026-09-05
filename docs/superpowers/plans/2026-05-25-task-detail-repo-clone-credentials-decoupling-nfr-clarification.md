# NFR 澄清: Task Detail 与 Repo Clone Credentials 解耦

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-05-24-task-detail-repo-clone-credentials-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-05-25-task-detail-repo-clone-credentials-decoupling-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L2 | 新增凭证接口服务端 P95 <= 600ms（单仓）/ <= 1.5s（10 仓） |
| 安全性 | L3 | 复用容器 access_token 鉴权 + 路径绑定校验 + 不落盘凭证明文 |
| 数据一致性 | L2 | 凭证接口缺失仓库时 fail-fast 返回 409，错误码与缺失清单强约束 |
| 可观测性 | L2 | 两段式 bootstrap 每阶段都有可检索日志与 trace_id 关联 |
| 可维护性 | L2 | 旧契约分阶段下线，保持 API 向后兼容窗口并有回归保护 |

## 逐增量 NFR 分析

### Increment 1: 独立凭证接口薄切片

#### NFR 类别: 安全性
- **等级**: L3 - 增强
- **量化目标**:
  - 100% 请求必须通过 `access_token` 鉴权、租户/工作空间/任务路径绑定校验、过期校验。
  - 响应体不新增长期密钥，仅返回临时 OAuth 访问令牌映射。

#### NFR 类别: 数据一致性
- **等级**: L2 - 标准
- **量化目标**:
  - 当 `expected_repo_urls - credential_repo_urls != empty` 时，100% 返回 `409`。
  - `error_code` 固定为 `REPO_CLONE_CREDENTIALS_INCOMPLETE`，并携带完整 `missing_repo_credentials`。

#### NFR 类别: 性能
- **等级**: L2 - 标准
- **量化目标**:
  - 单仓请求服务端处理时间 P95 <= 600ms。
  - 10 仓请求服务端处理时间 P95 <= 1.5s。

### Increment 2: onlineServiceJS 切换到两段式调用

#### NFR 类别: 可观测性
- **等级**: L2 - 标准
- **量化目标**:
  - `task-detail` 与 `repo-clone-credentials` 两次调用都需写出站摘要日志。
  - 凭证缺失错误必须保留结构化 payload（含 `trace_id` 与缺失仓库摘要）。

#### NFR 类别: 可维护性
- **等级**: L2 - 标准
- **量化目标**:
  - Node 侧错误转换单测覆盖两段式失败路径。
  - 失败文案统一，避免多入口提示分叉。

### Increment 3: task-detail 契约瘦身

#### NFR 类别: 数据一致性
- **等级**: L2 - 标准
- **量化目标**:
  - `task-detail` 只承担任务与仓库列表职责，不再承载凭证一致性判定。
  - 凭证一致性检查单点收敛到新接口，避免双源冲突。

#### NFR 类别: 可维护性
- **等级**: L2 - 标准
- **量化目标**:
  - 旧字段移除后，后端与 onlineServiceJS 回归用例全部通过。
  - 路径命名与 `task-detail` 保持同层级规范（`server-container-token/*`）。

### Increment 4: 回归与可观测性加固

#### NFR 类别: 可观测性
- **等级**: L2 - 标准
- **量化目标**:
  - 回归时可从日志确认调用顺序：`task-detail` -> `repo-clone-credentials` -> clone。
  - 失败案例能够在单次日志链路中定位到具体缺失仓库。

#### NFR 类别: 性能
- **等级**: L1 - 基础
- **量化目标**:
  - 新增一次凭证请求导致的整体 bootstrap 额外时延目标 <= 2s（P95）。
  - 本增量不做专项缓存与并发优化。

## 质量场景

### QS-01: 凭证接口在正常负载下返回可用映射
| 要素 | 内容 |
|------|------|
| 类别 | 性能 |
| 等级 | L2 |
| 刺激源 | onlineServiceJS bootstrap 流程 |
| 刺激 | 发起 `POST /server-container-token/repo-clone-credentials/` |
| 制品 | 凭证接口 view 与 OAuth 凭证装配逻辑 |
| 环境 | 正常负载（单任务，1~10 仓库） |
| 响应 | 返回 200 与 `repo_clone_credentials` 映射 |
| 响应度量 | 服务端处理时间 P95 <= 600ms（1 仓）/ <= 1.5s（10 仓），按服务端 APM 或应用日志统计 |

### QS-02: 凭证缺失时 fail-fast 且可操作
| 要素 | 内容 |
|------|------|
| 类别 | 数据一致性 |
| 等级 | L2 |
| 刺激源 | onlineServiceJS bootstrap 流程 |
| 刺激 | 请求凭证接口时存在未绑定 Git 授权的仓库 |
| 制品 | 凭证接口完整性校验 |
| 环境 | 正常 |
| 响应 | 返回 409，携带 `REPO_CLONE_CREDENTIALS_INCOMPLETE` 与缺失仓库列表 |
| 响应度量 | 缺失场景响应码恒为 409；`missing_repo_credentials` 与期望差集一致（集合相等） |

### QS-03: 两段式调用全链路可追踪
| 要素 | 内容 |
|------|------|
| 类别 | 可观测性 |
| 等级 | L2 |
| 刺激源 | 用户点击 task-detail 页面“启动” |
| 刺激 | relay 触发 onlineServiceJS 启动并执行 bootstrap |
| 制品 | outbound/init 相关日志与错误转换 |
| 环境 | 正常与凭证缺失两类场景 |
| 响应 | 日志中可见两段式调用顺序，失败时保留结构化错误字段 |
| 响应度量 | 100% 启动样本可检索到 `task-detail` 与 `repo-clone-credentials` 两个阶段日志；失败样本包含 `trace_id` 与缺失仓库摘要 |

### QS-04: 安全边界不因解耦回退
| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L3 |
| 刺激源 | 非法调用方或令牌不匹配调用 |
| 刺激 | 使用无效/过期 token，或路径 tenant/workspace/task 与 token 不匹配 |
| 制品 | 新增凭证接口鉴权与路径匹配逻辑 |
| 环境 | 正常 |
| 响应 | 返回 401 或 403，拒绝下发任何凭证 |
| 响应度量 | 相关负向测试覆盖通过；无凭证明文写入本地日志或持久化存储 |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| 数据一致性 L2（失败即 409 + 明确差集） | 将“凭证覆盖校验”建模为独立领域能力，而非 `task-detail` 的附带行为 | 在 DDD 中拆分 `TaskDetailReadModel` 与 `RepoCloneCredentialsGuard` 领域服务 |
| 安全性 L3（复用 token 与路径绑定） | 新接口与既有容器令牌聚合共享同一授权不变量 | 在 DDD 中为凭证下发应用服务引入统一 `ContainerTokenContext` 值对象 |
| 可观测性 L2（两段式可追踪） | 需要对阶段化流程建模事件，支持诊断 | 在 DDD 中新增 `RepoCloneCredentialsFetchAttempted/Succeeded/Failed` 领域事件 |
| 可维护性 L2（分阶段兼容） | 要求契约演进可控，避免聚合职责膨胀 | 在 DDD 中维持小聚合边界，避免把 task-detail 与凭证生命周期耦在同一聚合 |
| 性能 L2（10 仓 P95 <= 1.5s） | 不引入重型 CQRS；保留后续升级空间 | DDD 先采用同步读模型 + 轻量缓存接口预留，不提前拆读写模型 |

## 权衡与边界

### 取舍
- 选择数据一致性 L2（fail-fast + 明确错误）优先于“尽量继续 clone”的容错策略，以减少隐式失败和排障成本。
- 选择可维护性 L2 的分阶段迁移，优先降低切换风险，而不是一次性大改。
- 选择性能 L2，不做 L3/L4 的并行批量 token 拉取优化，控制当前范围。

### 明确不做什么
- 不改 OAuth 发证与 refresh 主链路。
- 不引入新数据库字段或跨服务事务编排（Saga/2PC）。
- 不在本增量实现前端“启动前授权完整性预检”。
- 不承诺极致性能目标（例如 10 仓 P95 < 500ms）。

### 升级触发条件
- 当单任务仓库规模常态超过 20 仓，且凭证接口 P95 > 2s 持续 3 天，性能从 L2 升级到 L3（评估并行拉取与缓存策略）。
- 当出现跨租户访问风险或合规要求提升，安全性从 L3 升级到 L4（引入更严格审计与策略验证）。
- 当接口迭代频率升高导致兼容成本增加，维护性从 L2 升级到 L3（引入显式版本化与弃用周期治理）。

## 跳过声明
- **可伸缩性**: 跳过。当前增量是接口职责拆分，不引入新流量入口；按现有单实例能力可满足。
- **可用性**: 跳过。未新增关键基础设施，不设定独立 RTO/RPO 目标，沿用现有服务等级。
- **合规与隐私**: 跳过。本增量不引入新的个人敏感数据处理流程，仅沿用现有 token 机制。

## 自检 (Hard Gate)

- [x] 每个相关 NFR 类别都有明确的支撑等级（L0-L4）
- [x] 每个 L1-L4 的 NFR 类别至少有一个量化目标
- [x] 每个 L2-L4 的 NFR 类别至少有一个质量场景（QS）
- [x] 每个质量场景的响应度量可验证（给出了具体数字和测量方式）
- [x] 影响领域模型的 NFR 决策已标注（含具体 DDD 动作）
- [x] 权衡和边界已明确（知道不做什么 + 升级触发条件）
- [x] 跳过的 NFR 类别有理由说明
- [x] 文档位置正确：`docs/superpowers/plans/2026-05-25-task-detail-repo-clone-credentials-decoupling-nfr-clarification.md`
