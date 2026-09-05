# NFR 澄清: runAll 链式启停（Cascade Lifecycle）

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-05-27-runall-cascade-lifecycle-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-05-27-runall-cascade-lifecycle-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L1 | API 在计划生成后同步返回；整链耗时由健康检查主导，platform 三跳 P95 ≤ 8min |
| 可用性 | L2 | 默认链式启停下，单点启动 `taskFE` 一次点击成功率 ≥ 90%（依赖已就绪环境） |
| 容错机制 | L2 | 链中任一步失败 5s 内返回；fail-fast，不无限重试链 |
| 数据一致性 | L2 | 链式计划与执行状态在单次请求内可观测；无跨请求分布式事务 |
| 安全性 | L2 | 链中每步复用 `session_id` 所有权；foreign ownership fail-fast |
| 可观测性 | L2 | 失败响应 100% 含 `failed_at`；日志 100% 含计划顺序 |
| 可维护性 | L2 | `cascade=false` 保留旧语义；Go 测试覆盖计划生成与 API 默认行为 |
| 可伸缩性 | L0 | 不适用（本地单实例 runAll） |
| 合规与隐私 | L0 | 不适用（无用户数据持久化变更） |

## 逐增量 NFR 分析

### Increment 1: 启动链式最小闭环（Thin Slice）

#### NFR 类别: 性能
- **等级**: L1 - 基础
- **量化目标**:
  - `planCascadeStart` 计算 P95 ≤ 50ms（≤ 30 个服务节点）
  - HTTP `POST /api/start` 在**启动链开始执行后**即挂起直至链结束或失败（同步语义），不额外引入 job 队列
- **约束**: 整链 wall-clock 由最慢服务健康检查决定（如 `ai-provider` 可达 180s timeout），不在本增量优化并行

#### NFR 类别: 可用性
- **等级**: L2 - 标准
- **量化目标**: 在 `git-oauth`、`saas-backend` 均为 `stopped` 时，仅点击 `taskFE` 启动，三服务最终 `healthy` 比例 ≥ 90%（本地开发环境，Docker/venv 已就绪）
- **约束**: 上游 preflight 失败（端口冲突等）计入失败，不单独降级链式行为

#### NFR 类别: 容错机制
- **等级**: L2 - 标准
- **量化目标**:
  - 链中第 N 步失败：不执行 N+1..end；已启动的 1..N-1 保持运行
  - 错误在失败步骤完成后 5s 内返回客户端
- **约束**: 不对整链做自动 rollback（停止已启动上游）

#### NFR 类别: 数据一致性
- **等级**: L2 - 标准
- **量化目标**:
  - 计划顺序满足：对所有 `depends_on` 边，上游在拓扑序中严格位于下游之前
  - 已 `healthy` 的上游跳过启动（幂等），不重复 `CompareAndSwapStatus` 冲突
- **约束**: 一致性范围限于单 runAll 进程内 `StatusStore`，无跨进程协调

#### NFR 类别: 安全性
- **等级**: L2 - 标准
- **量化目标**: 链中每一步 `StartServiceWithActor` 使用同一 `session_id`；遇到非本 session 拥有的运行中服务时 fail-fast 并中止链
- **约束**: 不自动 `takeover` 外来进程

#### NFR 类别: 可维护性
- **等级**: L2 - 标准
- **量化目标**: `runner_test` 含至少 1 个三节点链用例；`ui_test` 验证 `cascade` 默认 true

---

### Increment 2: 关闭链式

#### NFR 类别: 可用性
- **等级**: L2 - 标准
- **量化目标**: 当 `git-oauth` 有 `healthy` 下游 `taskFE` / `ai-provider` 时，点击关闭 `git-oauth`，三服务均在单次请求内变为 `stopped` 比例 ≥ 95%
- **约束**: 关闭顺序为下游先于上游

#### NFR 类别: 容错机制
- **等级**: L2 - 标准
- **量化目标**:
  - 下游 `StopService` 失败则中止链，上游保持原状态
  - 与 Increment 1 相同 fail-fast，无部分回滚补偿事务
- **约束**: `cascade=false` 时仍返回「active downstream」错误（回归）

#### NFR 类别: 数据一致性
- **等级**: L2 - 标准
- **量化目标**: `stopOrder` 中所有传递下游（`isBlockingStatus`）先于目标服务；`StopGroup` 既有顺序测试不回归
- **约束**: 仅停止阻塞状态下游，`stopped` / `failed` 下游不纳入计划

#### NFR 类别: 性能
- **等级**: L1 - 基础
- **量化目标**: 单服务 SIGTERM 等待上限沿用现网 5s；三服务链关闭 wall-clock P95 ≤ 20s（不含慢健康检查）
- **约束**: 关闭链不等待 health URL 超时

---

### Increment 3: 启动本组

#### NFR 类别: 可用性
- **等级**: L2 - 标准
- **量化目标**: `platform` 组「启动本组」在组内服务均为 `stopped` 时，组内服务按 `git-oauth → saas-backend → (taskFE, ai-provider)` 依赖顺序进入 `healthy` 或明确失败于某步
- **约束**: 组外 stopped 上游（若未来出现跨组依赖）纳入计划

#### NFR 类别: 性能
- **等级**: L1 - 基础
- **量化目标**: `StartGroup` 与单服务链式启动相同串行模型；不引入组内并行 level
- **约束**: `container-stack` 等与 platform 无依赖的组互不影响

#### NFR 类别: 可维护性
- **等级**: L2 - 标准
- **量化目标**: `POST /api/start-group` 与 `stop-group` 对称测试（参数校验、404 group、session_id 必填）

---

### Increment 4: 级联失败可诊断 + 旧语义回归

#### NFR 类别: 可观测性
- **等级**: L2 - 标准
- **量化目标**:
  - 链失败时 HTTP 4xx 响应体 100% 包含 `cascade.failed_at` 与 `cascade.completed` 数组
  - 服务日志行 `[cascade] start plan: a -> b -> c` 在每次链式操作前输出
- **约束**: 成功响应保持 `{ "status": "ok" }`，不强制返回 plan

#### NFR 类别: 可维护性
- **等级**: L2 - 标准
- **量化目标**:
  - `cascade=false` 单点关闭阻断下游：回归测试 1 例通过
  - UI `alert` 展示 `failed_at` 与 `completed`（人工验收 + 可选 HTML 片段测试）
- **约束**: 不新增 WebSocket 推送

#### NFR 类别: 性能
- **等级**: L1 - 基础
- **量化目标**: 错误 JSON 序列化增量字段 ≤ 2KB
- **约束**: 不记录完整 stderr 于 `cascade` 字段

---

### Increment 5: 全局启停（Future，仅登记）

- **性能 / 可用性**: 待二期单独澄清；预期 L1 串行全图，wall-clock 可达数十分钟
- **本 MVP 不实现，不写入 DDD/plan 硬门禁**

## 质量场景

### QS-01: 一键启动 taskFE 依赖链

| 要素 | 内容 |
|------|------|
| 场景 ID | QS-01 |
| 类别 | 可用性 |
| 等级 | L2 |
| 刺激源 | 开发者 |
| 刺激 | 在 `http://localhost:9999/` 对 `stopped` 的 `taskFE` 点击「启动」（`cascade` 默认 true） |
| 制品 | `POST /api/start` → `StartServiceCascade` |
| 环境 | 本地；`git-oauth`、`saas-backend` 均为 stopped；Docker/venv 就绪 |
| 响应 | `git-oauth`、`saas-backend`、`taskFE` 依次 `healthy`，或在中途返回含 `failed_at` 的错误 |
| 响应度量 | 手动验收 + `runner_test` 三节点链；成功路径 1 次点击内完成（不要求 wall-clock 上限，但不得需二次点击补上游） |

### QS-02: 关闭 git-oauth 自动清理下游

| 要素 | 内容 |
|------|------|
| 场景 ID | QS-02 |
| 类别 | 可用性 |
| 等级 | L2 |
| 刺激源 | 开发者 |
| 刺激 | `taskFE`、`ai-provider`、`saas-backend`、`git-oauth` 均为 healthy 时，点击 `git-oauth`「关闭」 |
| 制品 | `POST /api/stop` → `StopServiceCascade` |
| 环境 | 本地正常负载 |
| 响应 | 下游先 `stopped`，最后 `git-oauth` `stopped` |
| 响应度量 | `runner_test` 断言 stop 调用顺序；状态页 10s 内四轮询可见全链 stopped |

### QS-03: 链中失败可定位

| 要素 | 内容 |
|------|------|
| 场景 ID | QS-03 |
| 类别 | 可观测性 |
| 等级 | L2 |
| 刺激源 | 开发者 |
| 刺激 | 链中第 2 个服务启动失败（测试注入 preflight 错误） |
| 制品 | `POST /api/start` 错误响应 + runAll 日志 |
| 环境 | 正常 |
| 响应 | HTTP 400 含 `cascade.failed_at` = 失败服务名；`cascade.completed` 列出已成功步骤；日志含 `[cascade] start plan:` |
| 响应度量 | `ui_test` 解析 JSON；日志 grep 集成测试或 runner 测试断言 |

### QS-04: 非本 session 服务阻断链

| 要素 | 内容 |
|------|------|
| 场景 ID | QS-04 |
| 类别 | 安全性 |
| 等级 | L2 |
| 刺激源 | 另一 runAll session 已拥有上游服务 |
| 刺激 | 当前 session 对依赖该上游的下游发起链式启动 |
| 制品 | `EnsureOperableBySession` |
| 环境 | 双 session 模拟（`runner_test`） |
| 响应 | 链在冲突点 fail-fast，错误提示 ownership |
| 响应度量 | 测试断言未启动后续服务 |

### QS-05: 保留单点关闭语义（回归）

| 要素 | 内容 |
|------|------|
| 场景 ID | QS-05 |
| 类别 | 可维护性 |
| 等级 | L2 |
| 刺激源 | API 调用方 |
| 刺激 | `POST /api/stop` body `{ "name": "git-oauth", "cascade": false }`，下游仍 healthy |
| 制品 | `StopServiceWithActor` |
| 环境 | 正常 |
| 响应 | 400 + 「active downstream」类错误，不关闭任何服务 |
| 响应度量 | `runner_test` / `ui_test` 1 例 |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| 链式计划纯函数、无持久化（一致性 L2） | 编排逻辑适合 **领域服务** + **值对象** `ServiceLifecyclePlan`，非新聚合根 | 新增 `ServiceCascadeOrchestrationService.PlanStart/PlanStop` |
| fail-fast、无 saga 补偿（容错 L2） | 不建模「补偿事务」实体；失败用 `CascadeExecutionReport` 值对象 | 事件 `ServiceCascadeStepFailed` 携带 `completed[]`、`failed_at` |
| 每步复用 ownership（安全 L2） | `StartServiceWithActor` 不变；编排服务不绕过 guard | 应用层串行调用既有用例 |
| 幂等跳过已 healthy 上游（一致性 L2） | 计划生成阶段过滤状态，非运行时重试 | `ServiceLifecyclePlan` 构造时注入 `StatusSnapshot` 端口 |
| 串行执行（性能 L1） | 无「并行 level」领域概念 | 不引入 `ExecutionLevel` 在 UI 链路的复用 |
| `cascade` 开关（可维护性 L2） | API 层 **策略选择**：Cascade vs Single，领域计划与单点用例分离 | UI 仅传 flag；领域服务不被 `cascade` 污染 |

## 权衡与边界

### 取舍

- **选择串行链式** 以换取可预测顺序与简单失败语义，接受 platform 全链启动可达数分钟 wall-clock。
- **选择 fail-fast 无自动回滚**：上游已启动在失败时保持运行，避免误杀用户已就绪服务。
- **选择同步 HTTP** 而非后台 job：实现与调试简单，UI 依赖 2s 轮询展示中间态。

### 明确不做什么

- 不在 MVP 实现链式 **restart**、**build**。
- 不做链内并行启动（即使 `BuildDAG` 支持 level 并行）。
- 不做全局 `StartAll`/`StopAll`（Increment 5 / 二期）。
- 不在 `cascade` 错误体中嵌入完整日志 tail。
- 不跨 runAll 实例协调链式状态。

### 升级触发条件

- 当 platform 服务数 > 15 且全链 P95 > 15min：性能升至 L2，评估组内并行 level 或异步 job + 轮询 API。
- 当需要「失败自动回滚已启动上游」：一致性升至 L3，引入 Saga / 补偿停止计划。
- 当多开发者共享一台机器频繁 ownership 冲突：安全升至 L3，UI 显式展示 takeover 流程。

## 跳过声明

| 类别 | 理由 |
|------|------|
| 可伸缩性 | 本地单用户 runAll，无水平扩展需求 |
| 合规与隐私 | 无个人数据落库变更 |
| 分布式一致性 | 单进程内存状态，无跨 DB 事务 |
| 断路器 / 舱壁 | 链为本地子进程启停，不适用外部下游熔断 |
| 链路追踪 | L1 日志文本足够；不强制 OpenTelemetry |

## 自检

- [x] 各相关 NFR 类别有 L0–L4 等级
- [x] L1+ 类别有量化目标
- [x] L2+ 类别有质量场景（QS-01..05）
- [x] 响应度量可验证
- [x] 领域模型影响已标注
- [x] 权衡、边界、升级条件已写
- [x] 跳过类别有理由
- [x] 文档路径正确
