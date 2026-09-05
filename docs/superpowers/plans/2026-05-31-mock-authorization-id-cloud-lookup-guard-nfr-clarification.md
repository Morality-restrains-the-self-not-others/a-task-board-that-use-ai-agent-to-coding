# NFR 澄清: Mock authorization_id 云平台查库防护

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-05-31-mock-authorization-id-cloud-lookup-guard-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-05-31-mock-authorization-id-cloud-lookup-guard-value-stream.md`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L2 | 单次 helper 解析 O(1)，无额外 DB 查询 |
| 可用性 | L2 | mock-auth 路径 100% 返回 2xx/4xx，0% 500 |
| 安全性 | L2 | 不放宽 tenant/platform 过滤 |
| 数据一致性 | L1 | 只读路径，无写模型变更 |
| 可维护性 | L2 | 单一 helper，grep 可审计 |
| 可观测性 | L1 | 沿用现有 error 日志 |

## 质量场景

### QS-01: mock-auth 查上次配置
| 要素 | 内容 |
|------|------|
| 类别 | 可用性 |
| 等级 | L2 |
| 刺激源 | 任务详情 UI |
| 刺激 | GET previous-server-config，history.authorization_id=mock-auth |
| 制品 | get_previous_server_config |
| 环境 | 正常 |
| 响应 | HTTP 200，status=success，无 price/availability |
| 响应度量 | pytest `test_get_previous_server_config_accepts_mock_authorization_id` 通过 |

### QS-02: 数值 PK 行为不变
| 要素 | 内容 |
|------|------|
| 类别 | 可维护性 |
| 等级 | L2 |
| 刺激源 | 集成测试 |
| 刺激 | authorization_id 为真实 Snowflake PK 字符串 |
| 制品 | authorization_lookup.get_cloud_platform_authorization |
| 环境 | 正常 |
| 响应 | 返回 CloudPlatformAuthorization 或 None（tenant 不匹配） |
| 响应度量 | 单元测试覆盖 isdigit 正/负例 |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| L2 可用性 | 需显式区分 PK vs 占位符 | 新增 VO `CloudPlatformAuthorizationReference` |
| L2 安全 | tenant/platform 过滤保留在 infra lookup | helper 参数强制 company_id |

## 权衡与边界

### 明确不做什么
- 不迁移 CharField → FK
- 不在本次批量改 network/middleware 调用点（Phase 2 backlog）

### 升级触发条件
- 若 Phase 2 调用点 grep 清单 >10 处，提取 repository 接口

## 跳过声明
- 可伸缩性、合规：不适用（读路径防御性修复）
