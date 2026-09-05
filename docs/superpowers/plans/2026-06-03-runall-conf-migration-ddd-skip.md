# DDD 领域建模: runAll + conf/ 配置统一 (跳过)

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-06-03-runall-conf-migration-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-03-runall-conf-migration-value-stream.md`
> - NFR 澄清: `docs/superpowers/plans/2026-06-03-runall-conf-migration-nfr-clarification.md`

## 跳过声明

**本次变更跳过 DDD 领域建模。**

理由：
- 这是一次**配置迁移**任务，不引入新的限界上下文、实体、值对象、聚合或领域事件
- 变更仅涉及：YAML 配置格式调整 + Go 程序配置加载逻辑修改
- `runAll` 程序的 `Config`、`Service`、`HealthCheck` 等结构体是**数据传输对象**（DTO），不属于领域模型
- 无新增业务逻辑或业务规则

## 受影响的结构体（非领域模型，仅 DTO）

以下 Go 结构体已有明确定义，此次只需增加字段，无需 DDD 建模：

| 结构体 | 变更 |
|--------|------|
| `Service` | 新增 `ConfApp`、`HealthPath`、`LivenessPath` 字段 |
| `Config` | 加载流程新增 `resolveConfApps()` 步骤 |

这些是基础设施层的数据结构变更，不涉及领域层。
