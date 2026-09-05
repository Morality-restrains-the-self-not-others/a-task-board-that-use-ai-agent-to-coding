# NFR 澄清: runAll failed 下游级联关闭

> 输入:
> - 设计: `docs/superpowers/specs/2026-05-27-runall-cascade-failed-downstream-stop-design.md`
> - 价值流: `docs/superpowers/plans/2026-05-27-runall-cascade-failed-downstream-stop-value-stream.md`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L1 | 本地 dev 工具，级联关闭 < 30s |
| 可用性 | L1 | 无 SLA；失败应可观测 |
| 安全性 | L2 | 保留 cascade=false 严格 ownership |
| 数据一致性 | L0 | 无持久化状态机 |
| 容错机制 | L2 | 单点 stop 仍 blocked；cascade 含 failed |
| 可观测性 | L2 | async 失败写 server log；Playwright 防回归 |
| 可维护性 | L2 | 领域函数单测 + Playwright |

## 质量场景

### QS-01: failed 下游场景 git-oauth 级联关闭

| 要素 | 内容 |
|------|------|
| 类别 | 容错机制 |
| 等级 | L2 |
| 刺激源 | 开发者 |
| 刺激 | saas-backend=failed+pid>0 时对 git-oauth 点关闭 |
| 制品 | runAll StopServiceCascade |
| 环境 | 本地 platform 链 |
| 响应 | ai-provider → saas-backend → git-oauth 均 stopped |
| 响应度量 | `go test` + Playwright 通过 |

### QS-02: 单点关闭仍 blocked

| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L2 |
| 刺激源 | API 调用方 |
| 刺激 | cascade=false 停 git-oauth，下游 failed+pid>0 |
| 制品 | StopServiceWithActor |
| 环境 | 单元测试 |
| 响应 | 返回 active downstream 错误 |
| 响应度量 | `TestRunner_StopService_BlocksFailedDependentWithRunningPID` 保持绿色 |

## 领域模型影响

| NFR 决策 | 模型影响 | DDD 动作 |
|----------|---------|---------|
| cascade 含 failed (L2) | 扩展 `isCascadeStopCandidateStatus` | 与 `isBlockingStatus` 并列 |
| 单点 strict (L2) | 不改 `runningDependents` | 保持 CanStop 安全边界 |

## 权衡与边界

- **取舍:** 仅扩展 cascade 计划，不放宽单点 stop。
- **不做:** 同步 API、修复 saas-backend health 根因。
- **升级触发:** 若 async silent fail 仍困扰用户 → 后续 increment 做 cascade 失败 UI toast。

## 跳过声明

- 可伸缩性 / 合规: 本地 dev 工具，不适用。
