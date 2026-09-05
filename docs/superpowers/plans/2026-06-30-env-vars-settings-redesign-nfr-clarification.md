# NFR 澄清: 环境变量设置页面重设计

> 输入:
> - 设计文档: `docs/designs/env-vars-settings-redesign.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-30-env-vars-settings-redesign-value-stream.md`
>
> 输出使用者: `/7-plans-实施计划` (DDD 跳过 — 纯前端变更，无新领域概念)

## NFR 概览表

| 类别 | 等级 | 一句话量化 / 跳过理由 |
|------|------|---------------------|
| 性能 | L1 | 前端页面重布局，不引入新 API/数据流，现有性能已满足 |
| 可伸缩性 | L0 | 无新数据流，无新存储，无新外部依赖 |
| 可用性 | L0 | 复用现有 feature-params API，无新可用性要求 |
| 安全性 | L1 | 后端新增 env var key 校验（格式 + 保留字冲突），权限模型不变 |
| 数据一致性 | L0 | extra_env_vars 字段读写模式不变 |
| 容错机制 | L0 | 无新外部调用/消息队列/异步任务 |
| 可观测性 | L1 | 校验失败需记录 WARN 日志（非法 key 尝试） |
| 合规与隐私 | L0 | 不变更数据存储/传输方式 |
| 可维护性 | L1 | 组件拆分（EnvVarTableEditor / CollapsibleLLMConfigPanel / MergedEnvPreview），符合 ≤500 行规则 |

## 质量场景

### QS-01: 非法环境变量 key 被拒绝
| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L1 |
| 刺激源 | 用户通过前端表单提交 |
| 刺激 | POST 包含 `{"extra_env_vars": [{"key": "TASK_MY_VAR", "value": "x"}]}` |
| 制品 | POST /api/tenant/{id}/feature-params/ |
| 环境 | 正常 |
| 响应 | 400 Bad Request，message 说明 "TASK_* 为系统保留变量名前缀" |
| 响应度量 | 保留字冲突 100% 拒绝 |

### QS-02: 环境变量预览完整性
| 要素 | 内容 |
|------|------|
| 类别 | 可观测性 |
| 等级 | L1 |
| 刺激源 | 用户配置了 LLM 供应商 + 自定义 env vars |
| 刺激 | GET /api/tenant/{id}/feature-params/ |
| 制品 | env_preview 响应字段 |
| 环境 | 正常 |
| 响应 | env_preview 包含 TASK_* + extra_env_vars 合并后的完整映射 |
| 响应度量 | extra_env_vars 中的每个有效 key 均出现在 env_preview 中 |

## 领域模型影响

**无影响。** 纯前端变更 + 后端校验增强，不引入新的领域概念、实体、值对象或领域服务。DDD 步骤可跳过。

## 权衡与边界

### 取舍
- 选择在现有 `extra_env_vars` JSONField 上扩展 `description` 字段，而非新建独立表 — 牺牲规范化以换取零 migration、向后兼容

### 明确不做什么
- 不做独立 EnvironmentVariable 表和 CRUD API
- 不做环境变量模板市场/预设库
- 不做批量导入的异步处理（导入 .env 为前端纯客户端操作）
- 不做跨工作空间 env var 继承的可视化（V1 仅展示公司→工作空间→个人三层，不展示动态继承图）

### 升级触发条件
- 当单租户 env vars 超过 200 条时 → 考虑独立表 + 分页加载
- 当要求 env var 版本历史/回滚时 → 考虑独立模型 + 审计表
- 当需要 env var 跨租户模板共享时 → 考虑模板市场功能

## 跳过声明

以下 NFR 类别不适用于本次变更（纯前端重布局 + 后端校验增强，零新 API/零新数据流/零新外部依赖）：

- **可伸缩性**: 无新数据流，复用现有存储
- **可用性**: 复用现有 feature-params API 的可用性保障
- **数据一致性**: extra_env_vars 作为 JSONField 的读写模式不变
- **容错机制**: 无新外部调用（导入/导出 .env 为客户端操作）
- **合规与隐私**: 数据存储/传输方式不变
