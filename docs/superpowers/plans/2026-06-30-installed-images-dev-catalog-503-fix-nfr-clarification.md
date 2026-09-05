# NFR 澄清: 修复 installed-images/dev-catalog 503 错误

> 输入:
> - 设计文档: `.claude/plans/01-brainstorming-设计文档.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-30-installed-images-dev-catalog-503-fix-value-stream.md`
>
> 输出使用者: `/6-ddd-领域设计驱动`, `/7-plans-实施计划`, `/8-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 可观测性 | L2 | 异常时记录完整堆栈 + 响应体到日志；错误响应携带 trace_id |
| 容错机制 | L2 | 空值防护防止 AttributeError；try/except 防止未捕获异常导致 500 |
| 性能 | L0 | 不适用 — 异常路径，不影响正常请求性能 |
| 可伸缩性 | L0 | 不适用 — 无数据量或并发变化 |
| 可用性 | L0（间接改善） | 修复后减少 503，但无专项可用性目标 |
| 安全性 | L0 | 不适用 — 无认证/授权/加密变更 |
| 数据一致性 | L0 | 不适用 — 无数据写入 |
| 合规与隐私 | L0 | 不适用 |
| 可维护性 | L0 | 不适用 — 无部署或 API 版本变更 |

## 逐增量 NFR 分析

### Increment 1: 纵深防御错误处理（唯一增量）

#### NFR 类别: 可观测性
- **等级**: L2 - 标准
- **量化目标**: 
  - 异常发生时 100% 记录完整堆栈（`logger.exception`）
  - 上游 HTTP 错误时 100% 记录下游响应体（截断至 500 字符）
  - 错误响应 100% 携带 `trace_id` 字段
- **质量场景**: QS-01 (见下方)

#### NFR 类别: 容错机制
- **等级**: L2 - 标准
- **量化目标**:
  - `image_group`/`vendor`/`updated_at` 为 None 时返回空字符串而非崩溃
  - 任何未预期异常被 try/except 捕获并返回结构化错误（含 `error_type`）
- **质量场景**: QS-02 (见下方)

## 质量场景

### QS-01: 异常可追溯
| 要素 | 内容 |
|------|------|
| 类别 | 可观测性 |
| 等级 | L2 |
| 刺激源 | Saas_Ai_Provider 内部发生未预期异常（如 DB 连接断开） |
| 刺激 | VendorDevelopmentCatalogView.get() 执行过程中抛出 Exception |
| 制品 | Saas_Ai_Provider 日志文件 + API 响应 |
| 环境 | 正常 |
| 响应 | 日志记录完整堆栈（含 saas_user_id）；API 返回 `{detail, error_type}` |
| 响应度量 | `grep "VendorDevelopmentCatalogView error" logs/ai-provider.log` 可见完整 traceback |

### QS-02: 空值不崩溃
| 要素 | 内容 |
|------|------|
| 类别 | 容错机制 |
| 等级 | L2 |
| 刺激源 | 数据库中存在 image_group FK 断裂的 VendorContainerImage 记录 |
| 刺激 | `_public_container_image_payload(o, ...)` 访问 `o.image_group.name` |
| 制品 | `_public_container_image_payload()` 函数 |
| 环境 | 正常 |
| 响应 | 返回 `""` 而非抛出 AttributeError |
| 响应度量 | 单元测试：传入 image_group=None 的对象，函数不抛异常，返回 dict 中 name 为空字符串 |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| 可观测性 L2 — 异常全量记录 | 无模型影响 — 纯基础设施层变更 | 无需 DDD 调整 |
| 容错 L2 — 空值防护 | 无模型影响 — 防御性编程，不改变领域逻辑 | 无需 DDD 调整 |

## 权衡与边界

### 取舍
- 选择在异常时返回 `error_type`（如 `AttributeError`）给调用方，可能暴露内部实现细节，但换取了可排查性
- 响应体截断至 500 字符：牺牲完整错误信息换取日志存储可控

### 明确不做什么
- 不引入结构化错误码体系（那是新功能，非 bug fix 范围）
- 不修改正常请求路径的任何逻辑
- 不为 Saas_Ai_Provider 新增 Sentry/APM 集成

### 升级触发条件
- 当同一异常类型在 24h 内出现 >10 次时，升级为 bug 修复任务
- 当需要跨服务错误码统一时，引入结构化错误响应标准（新 feature）

## 跳过声明

- **性能**: 跳过。仅修改异常路径，正常请求性能不变。
- **可伸缩性**: 跳过。无数据量或并发变化。
- **可用性**: 跳过。修复间接改善可用性（减少 503），但无专项可用性目标。
- **安全性**: 跳过。无认证/授权/加密变更。
- **数据一致性**: 跳过。无数据写入操作。
- **合规与隐私**: 跳过。不涉及。
- **可维护性**: 跳过。无部署或 API 版本变更。
