# NFR 澄清: Fix Password Reset Datetime Parsing

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-06-22-fix-password-reset-datetime-parsing.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-22-fix-password-reset-datetime-parsing-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 安全性 | L3 | 密码重置 token 校验必须正确（auth domain） |
| 数据一致性 | L1 | 纯代码修复，无分布式事务 |
| 性能 | L1 | 无性能变化（单次 time.Parse → 最多 4 次，差异可忽略） |

## 逐增量 NFR 分析

### Increment 1: Fix Datetime Parsing

#### NFR 类别: 安全性
- **等级**: L3 - 增强（auth 域默认升级）
- **量化目标**: Token 过期校验必须在 token 实际过期时正确拒止，在 token 未过期时正确通过
- **质量场景**: QS-01

#### NFR 类别: 数据一致性
- **等级**: L1 - 基础
- **量化目标**: 修复后 token 过期判定结果与存储值一致
- **质量场景**: 无需独立场景（现有测试覆盖）

#### NFR 类别: 性能
- **等级**: L1 - 基础
- **量化目标**: parseDateTime() 最多尝试 4 种格式，单次调用 < 1µs（内存操作，无 I/O）
- **质量场景**: 无需独立场景（time.Parse 为纯 CPU 操作）

## 质量场景

### QS-01: Token 过期校验正确性
| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L3 |
| 刺激源 | 用户点击密码重置邮件中的链接 |
| 刺激 | GET /api/accounts/users/get-reset-user-info/{valid-token}/ |
| 制品 | taskAuth: isPasswordResetTokenValid() |
| 环境 | 正常运行时 |
| 响应 | 有效未过期 token → 200 + identifier；过期 token → 400 "无效的重置链接" |
| 响应度量 | 单元测试：给定有效未过期 token，返回 `valid=true`；给定过期 token，返回 `valid=false` |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| 无 | 纯基础设施修复，不改变领域模型 | 无需 DDD 建模 |

## 权衡与边界

### 取舍
- 无。此修复不涉及设计取舍。

### 明确不做什么
- 不引入新的时间库（如 `dateparse`）——使用 Go 标准库 `time.Parse` 足够
- 不增加缓存或性能优化——datetime 解析仅在 token 校验时调用，频率低
- 不修改数据库 schema 或存储格式

### 升级触发条件
- 无。修复完成后功能正常，无后续升级计划。

## 跳过声明
- **可伸缩性**: 跳过。无新增负载或数据量变化。
- **可用性**: 跳过。无架构变更，可用性不变。
- **可观测性**: 跳过。无新增日志/指标需求（调试日志已移除）。
- **合规与隐私**: 跳过。无新数据处理。
- **可维护性**: 跳过。parseDateTime() 为纯函数，单一职责。
- **容错机制**: 跳过。无外部调用或网络依赖。
