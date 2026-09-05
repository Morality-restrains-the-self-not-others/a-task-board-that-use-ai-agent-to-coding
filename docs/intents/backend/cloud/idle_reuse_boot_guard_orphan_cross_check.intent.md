# 意图：闲置复用启动保护 + 孤儿删除交叉校验

- **日期**: 2026-07-22
- **设计**: `docs/superpowers/specs/2026-07-22-idle-reuse-boot-guard-orphan-cross-check-design.md`
- **选型**: A + C
- **复用部分 Status:** superseded by [ADR-0013](../../../adr/0013-remove-idle-machine-reuse.md)
- **孤儿交叉校验:** 仍 accepted

## 背景与目标

Fork + `auto_run` 触发 `start-vm-auto` 时，工作区闲置复用曾把「刚启动、容器未就绪」的源任务机器当成闲置并抢绑；随后源任务孤儿对账按 `InstanceName` 误删已被新任务持有的实例。

2026-08-16 起跨任务闲置复用已拆除，启动保护（A）不再需要。孤儿删除不得误杀其他 CSC 仍持有的实例（C）仍然有效。

## 范围与边界

- **范围内**：`reconcileOrphanInstancesByName` 删除门闩（owned map / skip_owned）。
- **范围外**：已删除的 idle reuse 候选过滤；`prefer_idle_reuse` 策略 UI。

## 约束与风险

- 冷启动为默认路径；闲置机靠回收分钟数回收，不再被新评论领养。

## 业务意图 → 事件对照

**无对应新事件**：内部判定与删除门闩，不新增 MQ 契约。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 孤儿删除交叉校验 | — | — | — | — | 无对应事件：删除前门闩；删机仍走既有 orphan 路径 |

## 验收标准

1. ~~Running + 空 `server_url` + `idle_since IS NULL` → 不可 idle reuse。~~（复用已下线）
2. ~~全绑定有 `idle_since` 且无 busy → 可 reuse。~~
3. 实例已被他任务/评论 CSC 持有 → orphan reconcile 跳过 DeleteInstance。
4. 既有 orphan 单测不回归。

## 实施计划

1. ~~改 `findIdleMachineForReuseExcluding`（A）。~~ 已随 ADR-0013 删除。
2. 改 `reconcileOrphanInstancesByName`（C）— 已落地，保留。
3. 补单测 S3（skip_owned）。
4. 批准后写架构 v50 triad + VERSION_HISTORY。
