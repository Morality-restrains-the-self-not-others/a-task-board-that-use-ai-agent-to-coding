# NFR 澄清: taskAuth 错误透明度 + 测试覆盖提升

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-06-29-taskauth-error-transparency-and-test-coverage-design.md`
> - 价值流: `docs/superpowers/plans/2026-06-29-taskauth-error-transparency-and-test-coverage-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`

## 跳过声明

本次改进属于**纯可观测性增强 + 测试覆盖补充**，不引入新数据流、外部依赖、或架构变更。根据 NFR 澄清步骤跳过条件「增量不引入新的数据流或外部依赖」，完整 NFR 分析可跳过。

具体理由：
- **任务 1（日志透明化）**: 仅在现有 catch-all 错误分支添加 `log.Printf`，零行为变更
- **任务 2（Django 日志增强）**: 在现有 warning 日志中补充 status code 上下文，不改变异常类型或 HTTP 响应
- **任务 3（测试覆盖）**: 新增 Go 测试函数，不涉及生产代码路径

## NFR 概览表

| 类别 | 等级 | 说明 |
|------|------|------|
| 可观测性 | L2 (标准) | 本次增强正是提升可观测性：使 taskAuth 错误日志可定位具体 DB 错误 |
| 可维护性 | L2 (标准) | 新增测试覆盖防止 username upsert 回归 |
| 其他所有类别 | L0 | 不适用——无新增数据流、无架构变更 |

## 领域模型影响

**无影响。** 本次改进不改动任何领域模型、聚合边界、接口签名或数据流向。

## 权衡与边界

- **不做什么**: 不改造 taskAuth 所有 handler 的错误日志（仅改两个 upsert handler）
- **不做什么**: 不新增异常类型（保持 `AuthenticationServiceUnavailable` 语义不变）
- **不做什么**: 不改造 SQLite schema

## 自检

- [x] 跳过理由已明确声明
- [x] 文档位置正确
- [x] 领域模型影响已确认（无影响）
