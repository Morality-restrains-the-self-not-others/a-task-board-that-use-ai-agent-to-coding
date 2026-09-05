# NFR 澄清: 容器启动时功能参数来源选择

> 输入:
> - 设计文档: `docs/designs/env-var-preset-switching.md`
> - 价值流文档: `docs/superpowers/plans/2026-07-01-feature-params-source-selector-value-stream.md`
>
> 输出使用者: `/6-ddd-领域设计驱动`, `/7-plans-实施计划`, `/8-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L2 | 预览端点 P95 ≤ 500ms (服务端) |
| 安全性 | L3 | IDOR 防护 + config 归属校验 + env 中 API key 脱敏 |
| 数据一致性 | L1 | Todo 字段更新无特殊事务要求 |
| 可观测性 | L2 | 预览/启动操作记录 info 日志 + IDOR 拒绝记录 warn |
| 可维护性 | L2 | 新参数 optional，完全向后兼容 |

跳过的类别：
- **可伸缩性**: 不适用。预览端点为低频操作（用户主动点击），启动端点已有吞吐量基线。
- **可用性**: 不适用。无新增关键路径，依赖的 FeatureParamsResolver 已有可用性保障。
- **容错机制**: 不适用。无外部依赖调用，纯内部 DB 查询 + 序列化。
- **合规与隐私**: 不适用。无跨境数据、无新数据留存。

---

## 逐增量 NFR 分析

### Increment 1: 后端预览端点 + 启动参数扩展

#### NFR 类别: 性能

- **等级**: L2 - 标准
- **量化目标**: 预览端点 P95 ≤ 500ms（服务端处理时间，不含网络）；启动端点无额外延迟（已有基线）
- **理由**: 预览是用户在启动前的手动操作，500ms 内返回可以接受；非高频操作（每次启动点一次）

#### NFR 类别: 安全性

- **等级**: L3 - 增强
- **量化目标**:
  - `personal_config_id` 归属校验不可绕过（100% 拦截跨用户访问）
  - `source=company` / `source=workspace` 请求预览返回 403（不可绕过）
  - 预览返回的 `extra_env_vars` 含 API key 时前端遮罩
- **理由**: 设计文档权限分析发现 IDOR 风险（🔴高），必须加固

#### NFR 类别: 数据一致性

- **等级**: L1 - 基础
- **量化目标**: `Todo.feature_params_source` 写入为单字段 UPDATE，无分布式事务需求
- **理由**: 单表单字段更新，Django ORM 默认行为即可满足

#### NFR 类别: 可观测性

- **等级**: L2 - 标准
- **量化目标**:
  - 预览端点: info 日志记录 `user_id + personal_config_id + source`
  - IDOR 拒绝: warn 日志（复用 `personal_feature_params_views.py:60` 已有格式）
  - 启动端点: info 日志记录 `task_id + feature_params_source + personal_config_id`
- **理由**: 安全敏感操作需要可追溯

### Increment 2: 前端来源选择器 + 预览集成

#### NFR 类别: 可维护性

- **等级**: L2 - 标准
- **量化目标**: 新增组件不影响现有 relayToTrae 启动流程（不选来源时行为与当前完全一致）
- **理由**: `feature_params_source` 和 `personal_feature_params_config_id` 均为 optional 参数

---

## 质量场景

### QS-01: 预览个人配置的正常响应

| 要素 | 内容 |
|------|------|
| 类别 | 性能 |
| 等级 | L2 |
| 刺激源 | Web 前端（用户点击 [预览] 按钮） |
| 刺激 | GET .../feature-params-env-preview/?source=personal&personal_config_id={id} |
| 制品 | 预览端点 `get_feature_params_env_preview` |
| 环境 | 正常负载（单用户操作） |
| 响应 | 200 + JSON `{source, config_name, env: {...}}` |
| 响应度量 | 服务端 P95 ≤ 500ms，由 Django request logging middleware 测量 |

### QS-02: IDOR 防护 — 拒绝跨用户访问

| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L3 |
| 刺激源 | 恶意用户（已认证，workspace member） |
| 刺激 | 传入他人的 `personal_config_id` 调用预览端点 |
| 制品 | 预览端点 `get_feature_params_env_preview` |
| 环境 | 正常 |
| 响应 | 403 + `{message: "无权访问该配置"}` |
| 响应度量 | `config.user_id != request.user.id` 时 100% 返回 403；warn 日志含 user_id、target_config_id、owner_id |

### QS-03: 禁止预览公司/工作空间配置

| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L3 |
| 刺激源 | 已认证用户 |
| 刺激 | GET .../preview/?source=company 或 ?source=workspace |
| 制品 | 预览端点 |
| 环境 | 正常 |
| 响应 | 403 + `{message: "仅支持预览个人配置"}` |
| 响应度量 | 任何 `source != personal` 的请求 100% 返回 403 |

### QS-04: 启动时校验个人配置归属

| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L3 |
| 刺激源 | 恶意用户 |
| 刺激 | POST .../relay-to-trae/start/ + `feature_params_source=personal` + 他人的 `personal_feature_params_config_id` |
| 制品 | `relay_to_trae_start` 服务函数 |
| 环境 | 正常 |
| 响应 | 403 — 拒绝启动 |
| 响应度量 | `config.user_id != request.user.id` 时 100% 拒绝 |

### QS-05: 向后兼容 — 不传新参数行为不变

| 要素 | 内容 |
|------|------|
| 类别 | 可维护性 |
| 等级 | L2 |
| 刺激源 | 现有前端（未升级）或 API 调用方 |
| 刺激 | POST .../relay-to-trae/start/ 不传 `feature_params_source` |
| 制品 | `relay_to_trae_start` 服务函数 |
| 环境 | 正常 |
| 响应 | 正常启动，默认 `feature_params_source=company` |
| 响应度量 | 现有 Playwright 测试全部通过 |

---

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| 安全性 L3 — 归属校验 | 预览端点领域服务需要 `actor` (user_id) 参数 | 服务函数签名增加 `user_id` 参数，调用前注入 `request.user.id` |
| 安全性 L3 — 禁止预览 company/workspace | 预览端点需 source 过滤逻辑 | 在 view 层做 source 判断 + 403（非领域层） |
| 可观测性 L2 — 操作日志 | 无需建模新事件 | 使用 Django logging，不引入领域事件 |

**本次改动不引入新领域概念**——所有实体（`PersonalFeatureParamsConfig`、`Todo`、`FeatureParamsResolver`）已存在。NFR 决策仅影响 view 层的权限校验位置和服务函数签名，不影响领域模型结构。

---

## 权衡与边界

### 取舍

- 选择在 **view 层** 做 IDOR 校验（而非领域层），与现有 `_check_idor()` 模式保持一致
- 选择 **不新增领域事件** 用于预览操作（预览是纯读操作，无副作用）

### 明确不做什么

- 不引入新的权限角色或模型——复用 `IsAuthenticated` + 归属校验
- 不对预览端点做缓存（个人配置低频访问，实时性优先）
- 不做 company/workspace 环境变量预览（按设计文档 v3 的明确要求）

### 升级触发条件

- 当个人配置数量 > 100 时 → 个人配置下拉增加搜索过滤（当前预期 < 20）
- 当预览端点被滥用（> 10 req/s） → 增加频率限制

---

## 自检

- [x] 每个相关 NFR 类别都有明确的支撑等级（L0-L4）
- [x] 每个 L1-L4 的 NFR 类别至少有一个量化目标
- [x] 每个 L2-L4 的 NFR 类别至少有一个质量场景（QS-01 ~ QS-05）
- [x] 每个质量场景的响应度量可验证
- [x] 影响领域模型的 NFR 决策已标注
- [x] 权衡和边界已明确
- [x] 跳过的 NFR 类别有理由说明
- [x] 文档位置正确
