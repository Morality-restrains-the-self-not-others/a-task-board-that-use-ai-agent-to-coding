# NFR 澄清: 功能参数环境变量下发与智能体运行时解耦

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-05-29-feature-params-env-agent-agnostic-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-05-29-feature-params-env-agent-agnostic-value-stream.md`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L2 | env 端点 P95 ≤ 500ms |
| 可伸缩性 | L0 | 租户级配置，无扩展需求 |
| 可用性 | L1 | bootstrap 失败可重试 |
| 安全性 | L3 | access_token + 租户成员鉴权；api_key 仅授权路径 |
| 数据一致性 | L2 | 单聚合强一致（TenantFeatureParams） |
| 容错机制 | L2 | env JSON 解析失败 bootstrap 明确失败 |
| 可观测性 | L1 | bootstrap 日志保留 outbound 记录 |
| 合规与隐私 | L1 | api_key 与现 YAML 预览等价暴露面 |
| 可维护性 | L2 | 破坏性 API 切换；无旧字段别名 |

## 质量场景

### QS-01: 容器 bootstrap 拉取 env

| 要素 | 内容 |
|------|------|
| 类别 | 性能 |
| 等级 | L2 |
| 刺激源 | onlineServiceJS bootstrap |
| 刺激 | POST feature-params-env |
| 制品 | container_feature_params_views |
| 环境 | 正常负载 |
| 响应 | 200 + env map |
| 响应度量 | 服务端 P95 ≤ 500ms |

### QS-02: 无效 token 拒绝

| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L3 |
| 刺激源 | 未授权调用方 |
| 刺激 | 错误 access_token |
| 制品 | feature-params-env |
| 环境 | 正常 |
| 响应 | 401 |
| 响应度量 | pytest 断言 status=401 |

### QS-03: env 解析失败不写入半份配置

| 要素 | 内容 |
|------|------|
| 类别 | 容错机制 |
| 等级 | L2 |
| 刺激源 | bootstrap |
| 刺激 | TASK_LLM_PROVIDERS_JSON 非法 |
| 制品 | featureParamsEnvToYaml |
| 环境 | 正常 |
| 响应 | bootstrap 抛错，不写 service_config.yaml |
| 响应度量 | 单测 + bootstrap 集成测试 |

## 领域模型影响

| NFR 决策 | 模型影响 | DDD 动作 |
|----------|---------|----------|
| 一致性 L2 | TenantFeatureParams 单聚合 | 聚合根封装字段重命名与 env 序列化 |
| 安全 L3 | env 含 api_key | 序列化服务纯函数，view 层鉴权 |
| 可维护 L2 破坏性切换 | 无双写 | 删除 yaml 生成器 |

## 权衡与边界

### 取舍

- 破坏性 API 切换换取单一契约，避免长期双维护。

### 明确不做什么

- 不在 task2app 配置 AGENT_RUNTIME。
- 不实现 Cursor adapter。

### 升级触发条件

- 多运行时并存时需扩展 adapter 注册表（容器侧）。

## 跳过声明

- 可伸缩性：租户级单行配置，不适用。
- 合规：与现网等价，无新增跨境要求。
