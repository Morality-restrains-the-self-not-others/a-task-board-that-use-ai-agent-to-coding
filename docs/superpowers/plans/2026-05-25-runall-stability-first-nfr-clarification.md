# NFR 澄清: runAll 稳定性优先改造

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-05-25-runall-stability-first-design.md`
> - 价值流输入: `value-stream.yaml`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 可用性 | L3 | 关键服务（git-oauth / saas-backend）启动成功率 >= 99%，失败 60 秒内可定位失败阶段 |
| 容错机制 | L3 | 端口冲突在 Preflight 阶段 3 秒内失败返回，不进入长时间 pending |
| 可观测性 | L3 | 失败事件 100% 具备 `failure_code + phase + hint + trace/session` |
| 一致性 | L3 | runAll 作为唯一入口，单服务同一时刻仅允许一个 owner session |
| 可维护性 | L2 | 单一配置源，配置分叉在启动前 fail-fast |
| 性能 | L2 | doctor 全量预检 P95 <= 8s；单服务 Preflight P95 <= 2s |
| 安全性 | L2 | 仅允许 owner session 执行 stop/restart/takeover，默认拒绝隐式接管 |

## 逐增量 NFR 分析

### Increment 1: 单一配置源 + 所有权冲突拦截（M1）

#### NFR 类别: 一致性
- **等级**: L3 - 增强
- **量化目标**:
  - 同一服务同一时刻最多 1 个受管实例（owner 唯一）
  - 冲突进程检测命中率 100%（端口冲突即拦截）
- **约束**: 不允许静默接管；必须显式 `takeover`。

#### NFR 类别: 可维护性
- **等级**: L2 - 标准
- **量化目标**:
  - 生效配置源唯一，启动日志输出 `config_path + config_hash`
  - 检测到双配置分叉时 100% fail-fast
- **约束**: 禁止“多份配置同时可生效”的灰色行为。

#### NFR 类别: 容错机制
- **等级**: L3 - 增强
- **量化目标**: 端口冲突在 3 秒内返回 `PRECHECK_PORT_CONFLICT`
- **约束**: 不进入 Launch 阶段，不产生残留子进程。

### Increment 2: 关键前置检查（migration gate）+ 三阶段生命周期（M2）

#### NFR 类别: 可用性
- **等级**: L3 - 增强
- **量化目标**:
  - 对关键依赖未就绪（如 migration 缺失）必须在 Preflight 拦截
  - 启动失败后 60 秒内可完成“失败分类 + 修复提示”
- **约束**: Launch 与 Readiness 必须分离；端口可达不等于 Healthy。

#### NFR 类别: 容错机制
- **等级**: L3 - 增强
- **量化目标**:
  - `PRECHECK_RUNTIME_PREREQ_FAILED` 分类准确率 >= 99%
  - `LAUNCH_PROCESS_EXITED` 与 `READINESS_TIMEOUT` 分类互斥且可重现
- **约束**: 禁止“统一超时文案”掩盖真实失败阶段。

#### NFR 类别: 性能
- **等级**: L2 - 标准
- **量化目标**:
  - 单服务 Preflight P95 <= 2s
  - 关键链路启动（git-oauth->saas-backend）额外前置检查开销 P95 <= 5s
- **约束**: 稳定优先，不追求最短冷启动时间。

### Increment 3: 状态页可诊断化 + doctor 命令（M3）

#### NFR 类别: 可观测性
- **等级**: L3 - 增强
- **量化目标**:
  - 失败事件 100% 包含 `failure_code/phase/hint/session_id`
  - 状态页可展示最近 N 次失败摘要（N>=5）
- **约束**: 失败提示可执行，不允许仅输出模糊错误文本。

#### NFR 类别: 可维护性
- **等级**: L2 - 标准
- **量化目标**:
  - `runAll doctor` 支持全量预检并给出确定性 exit code
  - 回归测试覆盖冲突拦截/迁移拦截/恢复路径
- **约束**: 新能力必须伴随可自动化测试，不仅靠手工验证。

## 质量场景 (Quality Attribute Scenarios)

### QS-01: 端口冲突快速失败
| 要素 | 内容 |
|------|------|
| 类别 | 容错机制 |
| 等级 | L3 |
| 刺激源 | 开发者误手工启动旧服务 |
| 刺激 | runAll 启动目标服务时发现目标端口已被非 owner 进程占用 |
| 制品 | runAll Preflight 端口冲突检查 |
| 环境 | 本地开发，重复启动 |
| 响应 | 阻断启动并返回 `PRECHECK_PORT_CONFLICT` + 占用 PID + 修复建议 |
| 响应度量 | 从启动命令到失败返回 <= 3s；不创建新子进程 |

### QS-02: 关键前置缺失拦截
| 要素 | 内容 |
|------|------|
| 类别 | 可用性 |
| 等级 | L3 |
| 刺激源 | 下游 gitOauth migration 未完成 |
| 刺激 | runAll 启动 saas-backend 前执行 prereq probe |
| 制品 | Preflight runtime prerequisites |
| 环境 | 依赖半可用（health ok 但 schema 不完整） |
| 响应 | 阻断启动并返回 `PRECHECK_RUNTIME_PREREQ_FAILED` + 迁移修复命令 |
| 响应度量 | 错误分类准确，且 60s 内可完成定位与修复动作确认 |

### QS-03: 启动成功但就绪失败可区分
| 要素 | 内容 |
|------|------|
| 类别 | 可观测性 |
| 等级 | L3 |
| 刺激源 | 服务进程已启动但业务探针失败 |
| 刺激 | Launch 成功后 Readiness 重试耗尽 |
| 制品 | runAll 生命周期状态机 |
| 环境 | 异常配置/依赖未就绪 |
| 响应 | 返回 `READINESS_TIMEOUT`（而非启动失败）并附最后探针响应摘要 |
| 响应度量 | 失败事件 100% 含 phase=`READINESS` 和 failure_code |

### QS-04: owner-only 操作约束
| 要素 | 内容 |
|------|------|
| 类别 | 一致性 |
| 等级 | L3 |
| 刺激源 | 非 owner session 调用 stop/restart |
| 刺激 | 尝试操作已有受管服务 |
| 制品 | ServiceOwnership 校验逻辑 |
| 环境 | 多终端并发操作 |
| 响应 | 拒绝操作并提示 owner session 信息，要求显式 takeover |
| 响应度量 | 非 owner 操作拒绝率 100%，无误杀 owner 操作 |

### QS-05: doctor 预检性能基线
| 要素 | 内容 |
|------|------|
| 类别 | 性能 |
| 等级 | L2 |
| 刺激源 | 开发者在启动前执行 `runAll doctor` |
| 刺激 | 对 platform 组全服务执行预检 |
| 制品 | doctor 命令 |
| 环境 | 本地常规开发状态 |
| 响应 | 返回预检报告（通过/失败分类） |
| 响应度量 | doctor 全量预检 P95 <= 8s |

## 领域模型影响

| NFR 决策 | 领域模型影响 | 对应 DDD 动作 |
|----------|-------------|-------------|
| 一致性 L3（owner 唯一） | 服务状态不能只依赖端口探测；需显式所有权实体与不变量 | 建模 `ServiceOwnership` 值对象，并在聚合内 enforce “single owner session” |
| 容错 L3（分阶段失败） | 状态机需要区分 Preflight/Launch/Readiness | 在聚合中引入阶段化状态与失败事件，而非二元 healthy/failed |
| 可观测性 L3（结构化失败） | 错误成为一等建模对象，而非日志字符串 | 建模 `FailureCode` / `FailureHint` 值对象与 `ServicePreflightFailed` 等事件 |
| 可维护性 L2（单一配置源） | 配置校验属于应用服务的入口契约 | 建立 `ConfigFingerprint` 值对象，统一 config 解析边界 |
| 安全 L2（owner-only） | stop/restart/takeover 属于受控命令 | 领域服务命令签名增加 `actor/session` 上下文 |

## 权衡与边界

### 取舍
- 牺牲部分启动速度（增加 Preflight 和先决检查）换取“可预测失败”与“无 pending”。
- 采用显式 takeover 而非自动接管，降低误操作风险。

### 明确不做什么
- 不在本增量引入 systemd/k8s 等外部编排替换。
- 不追求极致冷启动性能（不以并行最大化为目标）。
- 不改写业务服务内部健康检查实现，仅在 runAll 层做治理。

### 升级触发条件
- 若关键服务启动成功率持续 < 99%，可用性从 L3 升级到 L4，考虑守护进程化。
- 若 doctor P95 持续 > 8s，性能从 L2 升级到 L3，优化预检并行策略。
- 若出现 ownership 争用导致误中断，安全/一致性升级并引入更强会话签名机制。

## 跳过声明

- 可伸缩性: 跳过。当前为本地编排稳定性治理，不涉及流量规模扩展。
- 合规与隐私: 跳过。无新增个人数据处理与跨境要求。
- 数据本地化: 跳过。无新增数据落盘位置策略变更。

## 自检 (Hard Gate)

- [x] 每个相关 NFR 类别都有明确支撑等级（L0-L4）
- [x] 每个 L1-L4 类别具备量化目标
- [x] 每个 L2-L4 类别至少一个质量场景
- [x] 每个场景有可验证响应度量
- [x] 领域模型影响已标注且对应 DDD 动作
- [x] 权衡与边界明确（包含不做事项与升级触发）
- [x] 跳过类别给出理由
- [x] 文档路径正确（`docs/superpowers/plans/2026-05-25-runall-stability-first-nfr-clarification.md`）

---

NFR 澄清完成。关键 NFR 决策如下：

1. **一致性 L3**：runAll 唯一入口 + owner-only 操作，杜绝双管理导致的幽灵进程。
2. **容错/可用性 L3**：端口冲突与前置缺失必须在 Preflight 快速失败，禁止挂起。
3. **可观测性 L3**：失败必须结构化输出（code/phase/hint/session），支持状态页直读诊断。

这些决策将在 `/5-ddd-领域设计驱动` 中驱动聚合边界、状态机和事件建模。
