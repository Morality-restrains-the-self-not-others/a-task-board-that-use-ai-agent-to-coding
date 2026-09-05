# Value Stream: runAll 链式启停（Cascade Lifecycle）

> Derived from design: `docs/superpowers/specs/2026-05-27-runall-cascade-lifecycle-design.md`

## Value Summary

本地开发者在 `http://localhost:9999/` 上**一次点击**即可按 `depends_on` 依赖顺序启动或关闭服务链，无需手工逐个处理上游/下游，与 runAll CLI 批量编排语义一致。

## End-to-End Flow

```text
[用户在 runAll 状态页点击「启动」或「关闭」/「启动本组」]
  → [UI 携带 session_id，cascade 默认 true]
  → [Runner 生成传递闭包 + 拓扑计划]
  → [串行执行 Start/Stop（含 preflight、ownership、健康检查）]
  → [状态页 2s 轮询展示各步 starting/stopping/healthy/stopped]
  → [用户获得目标服务就绪或整链安全关闭]
```

## Value Stages

| 分类 | 阶段 | 说明 |
|------|------|------|
| **Trigger** | 单行「启动/关闭」或组头「启动本组」 | 用户意图：一键完成依赖链 |
| **Core** | 启动链式计划 + 串行拉起 stopped 上游 | 点 `taskFE` 即可间接启动 `git-oauth` → `saas-backend` |
| **Core** | 关闭链式计划 + 逆序关闭活跃下游 | 点 `git-oauth` 不再被下游阻断 |
| **Core** | 启动本组 | `platform` 等组一键正序启动 |
| **Essential support** | `cascade` API 契约 + 失败 `completed`/`failed_at` | 可诊断、可回归旧语义 |
| **Essential support** | 所有权 / preflight 每步复用 | 与稳定性设计一致，非独立用户价值 |
| **Enhancement** | 失败 alert 展示级联进度 | 降低「卡在哪一步」的困惑 |
| **Future** | 全局「启动全部 / 关闭全部」 | 二期，非 MVP |
| **Delivery point** | 目标服务 healthy 或整链 stopped | 见设计 §11 验收标准 |

## Wait / Dependency Points

- 链中每一步依赖前一步 **healthy / stopped** 判定完成才继续。
- 启动链依赖上游 preflight（含 `git-oauth` 端口冲突恢复，若已实现 `runall-log-copy-gitoauth-port-conflict-recovery`）。
- 关闭链依赖 `ServiceStopPolicyService` 对下游阻塞状态的识别。
- UI 反馈依赖 `/api/status` 轮询，无 WebSocket。

## Value Increments

### Increment 1: 启动链式最小闭环（Thin Slice）

**Value to user:** 仅点击某一服务的「启动」，即可按依赖顺序自动拉起所有已停止的上游，最终启动目标服务。

**Scope:**

- `planCascadeStart` + `StartServiceCascadeWithActor`（串行复用现有 `StartServiceWithActor`）
- `POST /api/start` 默认 `cascade: true`
- `status.html` 启动请求带 `cascade: true`
- Go 测试：`runner_test` 三节点链 `a→b→c`，仅启动 `c` 验证顺序

**Depends on:** nothing

---

### Increment 2: 关闭链式

**Value to user:** 点击「关闭」时自动先关闭所有仍在运行的传递下游，再关闭目标，不再出现「有活跃下游依赖」阻断。

**Scope:**

- `planCascadeStop` + `StopServiceCascadeWithActor`
- `POST /api/stop` 默认 `cascade: true`
- UI 关闭请求默认链式
- Go 测试：关闭链顶端服务应先停下游

**Depends on:** Increment 1（共享计划生成基础设施与 API 壳）

---

### Increment 3: 启动本组

**Value to user:** 在组标题旁点击「启动本组」，按 DAG 正序拉起组内（及必要组外上游）所有 stopped 服务。

**Scope:**

- `StartGroup` / `StartGroupWithActor`
- `POST /api/start-group`
- UI「启动本组」按钮
- 回归：`StopGroup` 行为不变

**Depends on:** Increment 1

---

### Increment 4: 级联失败可诊断 + 旧语义回归

**Value to user:** 链式中途失败时能看到失败步骤与已完成步骤；需要时仍可通过 `cascade: false` 使用单点启停。

**Scope:**

- API 错误体扩展 `cascade.completed` / `cascade.failed_at`
- UI `alert` 解析展示
- `cascade=false` 关闭仍阻断下游（回归测试）
- `ui_test.go` 覆盖默认 cascade 与错误体

**Depends on:** Increment 2、3

---

### Increment 5: 全局启停（Future）

**Value to user:** 页面顶部一键启动/关闭全部服务（跨 group 全 DAG）。

**Scope:** `StartAll` / `StopAll` + 标题栏按钮。

**Depends on:** Increment 4

**Status:** 设计明确为二期，YAML 步骤可标 `planned`，不阻塞 MVP 交付。

## Existing Value Stream Impact

| 维度 | 评估 |
|------|------|
| **受影响现有流** | `runall-log-copy-gitoauth-port-conflict-recovery` — 启动链会串联触发 `git-oauth` preflight/恢复；级联失败诊断与其错误展示互补 |
| **新流** | `runall-cascade-lifecycle`（domain: `云平台与资源`） |
| **字段** | 运行态登记于各受管服务 `*.runtime.cascade_*`（无 DB schema 变更） |
| **测试** | 主验证在 `runAll` Go 测试；价值流工具以 `view_test/*.md`（`planned`）登记验收场景 |
| **交叉依赖** | 无新业务 API 依赖；环境稳定性受益于 Increment 1 自动拉起 `git-oauth` |

## YAML 草案（待写入用户确认的配置文件）

```yaml
- name: runall-cascade-lifecycle
  domain: 云平台与资源
  description: runAll Web UI 依赖链一键启停（启动/关闭链式 + 启动本组）
  steps:
    - name: runall-start-cascade-thin-slice
      status: planned
      test_file: view_test/runall-start-cascade-thin-slice.md
      fields:
        - name: git-oauth.runtime.cascade_plan_order
          description: 启动链计划中的服务顺序（上游在前）
        - name: saas-backend.runtime.lifecycle_status
          description: 链式启动过程中 saas-backend 状态迁移
        - name: taskFE.runtime.lifecycle_status
          description: 目标服务最终是否 healthy
    - name: runall-stop-cascade
      status: planned
      test_file: view_test/runall-stop-cascade.md
      fields:
        - name: taskFE.runtime.lifecycle_status
          description: 关闭链中下游先变为 stopped
        - name: ai-provider.runtime.lifecycle_status
          description: 并行下游关闭顺序与状态
        - name: git-oauth.runtime.lifecycle_status
          description: 上游目标服务最后关闭
    - name: runall-start-group
      status: planned
      test_file: view_test/runall-start-group.md
      fields:
        - name: git-oauth.runtime.cascade_plan_order
          description: platform 组启动本组计划顺序
        - name: saas-backend.runtime.lifecycle_status
          description: 组内依赖服务启动结果
    - name: runall-cascade-failure-feedback
      status: planned
      test_file: view_test/runall-cascade-failure-feedback.md
      fields:
        - name: saas-backend.runtime.cascade_failed_at
          description: 级联失败时停在哪一步
        - name: saas-backend.runtime.cascade_completed_steps
          description: 失败前已成功完成的步骤列表
    - name: runall-global-start-stop-all
      status: planned
      test_file: view_test/runall-global-start-stop-all.md
      fields:
        - name: docker-infra.runtime.lifecycle_status
          description: 二期全局启停验收（infrastructure 组）
```

## Self-Review Checklist

- [x] 每个 MVP 增量均有用户可见价值（Increment 1–4）
- [x] Thin slice 端到端：UI 点击 → API → Runner 链式启动 → 状态页可见
- [x] 依赖顺序：1 → 2；3 依赖 1；4 依赖 2+3；5 为 Future
- [x] 字段名为三段式，服务名来自 `runAll/config.yaml`
- [x] YAML 语法在草案中可解析（待写入目标文件后做工具校验）
