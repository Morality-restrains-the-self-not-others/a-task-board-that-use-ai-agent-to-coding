# runAll 稳定性优先改造设计

## 1. 背景与目标

当前环境在服务启动/关闭过程中存在以下高频不稳定问题：

- 端口被历史进程占用，但新的 runAll 会话未感知，导致“假启动成功”或请求长时间 pending。
- 服务健康检查通过了“端口可达”，但业务链路并未就绪（例如下游 schema/migration 未完成）。
- 同一服务可能被“手工命令 + runAll”混合管理，导致所有权不清晰、停止行为不可预测。

本设计采用**稳定性优先**策略，明确约束：

- 日常只允许通过 runAll 启停服务；
- 出现冲突时以“可诊断、可恢复、可审计”为第一优先级，不追求最快启动速度。

## 2. 设计范围

### In Scope

- runAll 的服务所有权治理（唯一入口）。
- 启动前预检（端口、依赖、关键运行前置条件）。
- 启动与就绪检查分离（Launch != Ready）。
- 失败分类与恢复动作标准化。
- 对现有 `runAll.yaml` / `runAll/config.yaml` 的配置统一治理方案。

### Out of Scope

- 替换 runAll 为全新编排器（如 systemd/k8s）。
- 重写各业务服务内部健康检查逻辑。
- 统一所有仓库的进程管理器（仅覆盖当前 runAll 管理范围）。

## 3. 关键约束与成功标准

### 约束

- 单一入口：runAll 是唯一允许的日常启停入口。
- 幂等：重复执行 start/stop/restart 不产生多实例残留。
- 可回溯：每次失败都能定位到阶段与原因类别。

### 成功标准

- S1：同一服务在任意时刻最多一个受管实例（runAll ownership）。
- S2：端口冲突时不进入“长时间 pending”，在预检阶段直接失败并给出修复建议。
- S3：关键依赖异常（例如 migration 未完成）在预检/就绪阶段被明确拦截。
- S4：runAll 状态页可直接展示“失败阶段 + 分类 + 建议动作”。

## 4. 方案对比与选型

### A. 软约束（文档约定）

- 成本低，但无法避免人工绕过导致的漂移。

### B. 硬约束单点编排（本次选型）

- runAll 实施所有权锁定、外来进程识别、分阶段健康验证。
- 在当前工程复杂度下，稳定性收益最高。

### C. 常驻守护进程

- 稳定性更高但改造面过大，超出当前最小可交付范围。

**结论**：采用 B，后续视运行数据再评估是否演进到 C。

## 5. 总体设计

### 5.1 统一配置源

现状存在 `runAll.yaml` 与 `runAll/config.yaml` 两套入口，容易产生配置分叉。

设计要求：

- 定义唯一生效配置（建议 `runAll/config.yaml`，并在根目录配置仅保留转发/提示）。
- runAll 启动时输出“配置指纹”（文件路径 + hash）到日志和状态页。
- 若检测到双配置且内容不一致，直接 fail-fast。

### 5.2 服务所有权模型（Ownership）

每个受管服务维护所有权记录：

- `service_name`
- `owner_session_id`
- `pid`
- `started_at`
- `config_hash`
- `health_endpoint`

规则：

- 只有当前 `owner_session_id` 可执行 stop/restart。
- 发现外来进程占用目标端口时，默认拒绝启动并提示 `takeover` 或手动清理。
- `takeover` 必须显式触发，不做静默接管。

### 5.3 三阶段生命周期

#### Phase 1: Preflight（启动前）

- 端口冲突检查（监听进程 + 所有权匹配）。
- 依赖服务可达性检查（按 `depends_on`）。
- 关键前置检查（可配置 probe）：
  - 示例：`gitOauth` migration 状态检查必须通过再启动依赖它的 `saas-backend`。

#### Phase 2: Launch（拉起进程）

- 仅在 Preflight 全通过后执行命令。
- 记录子进程 PID 与启动命令摘要。
- 对早退进程（启动即退出）立即判定失败，不进入健康重试环节。

#### Phase 3: Readiness（业务就绪）

- 读取 `health_check`，支持重试与回退参数。
- 仅在业务就绪通过后标记 `Healthy`，否则标记 `Failed(Readiness)`。
- 严禁“端口通即健康”作为最终状态。

## 6. 失败分类与恢复策略

标准错误分类：

- `PRECHECK_PORT_CONFLICT`
- `PRECHECK_DEPENDENCY_UNREADY`
- `PRECHECK_RUNTIME_PREREQ_FAILED`（如 migration/seed/schema）
- `LAUNCH_PROCESS_EXITED`
- `READINESS_TIMEOUT`
- `READINESS_BAD_STATUS`

每类错误绑定固定恢复建议（状态页展示）：

- 冲突类：给出占用 PID、命令、建议 `runAll stop --force` 或 `runAll takeover`。
- 前置类：给出具体前置命令，如 `python3 manage.py migrate api`。
- 就绪类：展示最后 N 次 health 响应摘要。

## 7. 关键流程

### 7.1 正常启动

1. 读取唯一配置并生成 `session_id`。
2. 逐服务执行 Preflight。
3. Preflight 通过后 Launch。
4. Readiness 通过后置为 `Healthy`。

### 7.2 检测到外来进程

1. 端口占用但 ownership 不匹配。
2. 直接 fail-fast（默认），记录 `PRECHECK_PORT_CONFLICT`。
3. 用户可选择：
   - 清理外来进程后重试；
   - 显式 `takeover`。

### 7.3 关键前置失败（例如 gitOauth migration）

1. Preflight 运行自定义 probe 失败。
2. 不启动上游依赖服务。
3. 直接返回修复命令并阻断后续链路。

## 8. 配置与实现改动建议

## 8.1 runAll 配置增强

新增可选字段（示意）：

- `preflight_checks`（命令型或 HTTP 型）
- `ownership`（strict / permissive，默认 strict）
- `readiness.mode`（http / command）
- `failure_hints`（错误码到修复建议映射）

### 8.2 服务分层策略

- `git-oauth`、`saas-backend`：启用严格 Preflight 与 Readiness。
- `taskFE`、`ai-monitor`：允许 `on_failure: skip`，但不影响核心链路状态。

### 8.3 CLI 行为

新增或增强子命令：

- `runAll doctor`：执行全量预检，不启动服务。
- `runAll takeover <service>`：显式接管外来进程。
- `runAll ps --verbose`：显示 ownership、PID、健康状态、失败分类。

## 9. 测试设计

### 单元测试（runAll）

- 端口冲突 + ownership 不匹配时返回 `PRECHECK_PORT_CONFLICT`。
- takeover 仅在显式指令下生效。
- Readiness 超时分类准确。

### 集成测试（编排）

- 场景1：`8001` 有外来进程 -> runAll 启动 backend 必须 fail-fast。
- 场景2：gitOauth migration 缺失 -> backend 启动前被阻断，提示修复命令。
- 场景3：修复 migration 后重新启动 -> 全链路健康。

### 回归测试

- 现有 `runAll/src/runner_test.go` 增补上述场景，保持停止/重启语义不回退。

## 10. Domain Concept Inventory（供 /5-ddd 输入）

### Bounded Contexts

- `orchestration-lifecycle`：服务编排生命周期管理
- `service-runtime-observability`：服务状态可观测与故障归因

### Key Entities

- `ManagedService`
- `ServiceOwnership`
- `StartupSession`
- `ReadinessSnapshot`

### Candidate Aggregates

- `ManagedServiceAggregate`（配置 + 所有权 + 当前状态）
- `StartupSessionAggregate`（一次编排会话内所有事件）

### Domain Events

- `ServiceOwnershipAcquired`
- `ServiceStartRejectedByForeignProcess`
- `ServicePreflightFailed`
- `ServiceReadinessFailed`
- `ServiceBecameHealthy`

## 11. Value Stream 影响分析

受影响对象：

- 顶层 `value-stream.yaml` 的 `runall_config` 生效链路。
- 跨域影响：`user-auth`、`cloud`、`project-workspace` 等流的开发环境稳定性。

建议新增 value stream：

- `runall-lifecycle-reliability`（建议 domain：平台工程/云平台与资源）
  - `single-config-source`
  - `ownership-and-port-guard`
  - `preflight-prereq-gates`
  - `readiness-failure-classification`

建议新增字段（runAll 状态持久层）：

- `runall.service_ownership.owner_session_id`
- `runall.service_ownership.pid`
- `runall.startup_attempt.phase`
- `runall.startup_attempt.failure_code`
- `runall.startup_attempt.hint`

## 12. 风险与回滚

主要风险：

- 过严拦截导致“本来能跑”的场景被拒绝启动。
- 旧脚本流程依赖手工启动，迁移期有学习成本。

缓解：

- 引入 `doctor` 先行，先看问题再启动。
- 首周提供 `strict=false` 临时逃生开关（默认仍 strict）。

回滚策略：

- 配置回退到旧行为（关闭 preflight gates 与 ownership strict）。
- 保留旧版本 runAll 二进制以便快速切换。

## 13. 里程碑（稳定性优先）

- M1（最小可交付）：唯一配置源 + ownership 冲突拦截 + 明确失败分类
- M2：preflight 关键前置检查（含 migration gate）
- M3：status UI 展示修复建议 + doctor 命令

---

该设计用于下一步 `/6-plans-实施计划` 细化任务与验收脚本。
