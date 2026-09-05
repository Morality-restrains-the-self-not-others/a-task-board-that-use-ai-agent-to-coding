---
name: 7-plans
description: "Step 7 - Writing Plans: produce a verifiable, checkbox-style task plan from the approved value stream and domain model."
---

# /7-plans — 实施计划

Invoke the Superpowers `writing-plans` skill: convert the approved value stream and domain model into a checklist of small, verifiable tasks with paths and commands.

## DDD Context (Backend Work)

The domain model from `/6-ddd` defines the contract for planning:

- Task ordering puts domain-layer files first (entities → value objects → aggregates → repository interfaces → domain events → domain services)
- Infrastructure implementations come after their domain interfaces are defined
- Repository implementations reference the ABC interfaces from `domain/repositories/`
- Event publishers/consumers are wired per the domain event contracts in `domain/events/`
- **Intent → event tasks (强制)**：计划中每个服务端业务意图须有显式任务：定义事件契约 → 应用服务发布 → MQ 适配器投递 →（若有）消费者；并更新 `docs/intents/` 对照表。禁止只写同步副作用而无事件投递任务。

Validate that the plan respects these dependencies. Do NOT invoke the DDD skill —
the domain model already exists; the plan references it.

To proceed, invoke the **writing-plans** skill via the Skill tool.

## 完成后 — 下一步选择

实施计划产出后，使用 `AskUserQuestion` 工具让用户一键选择下一步：

```
header: "下一步"
question: "实施计划已就绪。下一步做什么？"
multiSelect: false
options:
  1. label: "TDD 构建 (推荐)"
     description: "按计划逐任务执行测试驱动开发"
  2. label: "重新制定计划"
     description: "调整任务拆分、顺序或依赖关系"
```

- 用户选 1 → 调用 `/8-build`
- 用户选 2 → 重新执行本技能（实施计划）
