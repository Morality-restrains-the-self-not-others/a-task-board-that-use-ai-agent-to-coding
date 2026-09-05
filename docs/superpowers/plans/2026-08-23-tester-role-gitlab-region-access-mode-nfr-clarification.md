# NFR 澄清：测试角色 + GitLab 区域访问模式

- **日期**: 2026-08-23
- **价值流**: `2026-08-23-tester-role-gitlab-region-access-mode-value-stream.md`
- **默认等级**: L2；认证相关门禁 L3

## 质量场景

| 场景 | 等级 | 说明 |
|------|------|------|
| 设置 is_tester | L2 | 单行 UPDATE，延迟不敏感 |
| 区域目录过滤 | L2 | 区域行数极少（个位数） |
| 开发模式门禁 | L3 | 错误开放会导致预发 GitLab 暴露给全体租户 |
| 事件投递 | L2 | 审计用，失败打 error 不回滚已提交标志 |

## 路径分片键强制审视

| 路径 | 是否携带分片 ID | 判定 | 动作 |
|------|-----------------|------|------|
| GET /api/system-admin/users/?role=tester | 否 | L0 | 平台全局用户目录，本就按全库分页；升级触发：用户表 >1000 万再按 id 范围 |
| PATCH /api/system-admin/users/{id}/ is_tester | 是 user_id | 合适 | 账号主键即边界 |
| GET /api/billing/gitlab-regions/ | 否 | L0 | 区域元数据全局小表；非租户膨胀 |
| PUT /api/system_admin/gitlab-regions/{slug}/ | 是 slug | 合适 | 区域自然键 |
| POST gitlab-resources purchase | 是 tenant_id + region | 合适 | 租户+区域即配额分片 |

## 幂等性强制审视

| 路径 | 副作用 | 等级 | 重复触发源 | 业务重复边界 | 幂等键 | 重放语义 |
|------|--------|------|------------|--------------|--------|----------|
| GET 用户列表 / 区域目录 | 无 | L0 | — | — | — | 纯查询 |
| PATCH is_tester | 有 | L2 | 双击保存 | 同一 user_id + 目标布尔值 | 行主键 user_id | 重复 PATCH 同值空操作；变值则更新并再发事件 |
| PUT 区域 access_mode | 有 | L2 | 双击保存 | 同一 slug + 目标 mode | slug | 同值空操作（仍 200）；变值更新+事件 |
| POST 购买（已有） | 有 | L3 | 订单幂等既有 | tenant+region+订单 | 沿用既有订单幂等 | 本增量只加 403 前置，不改资金幂等 |
| UserTesterFlagChanged 消费 | 无自动消费者 | L0 | — | — | 事件 user_id | 人工审计 |
| GitlabRegionAccessModeChanged 消费 | 无自动消费者 | L0 | — | — | 事件 region slug | 人工审计 |

前端保存：沿用既有 saving/disabled；PATCH/PUT 非资金路径，同步 busy 即可，不强制新 Idempotency-Key（元规则 52：写操作须门闩；本页已有 saving 状态）。

## 领域模型影响

- 值对象 `GitlabRegionAccessMode`：`release` \| `development`
- 领域服务 `CanUseGitlabRegion(mode, isTester) bool`
- 聚合：`AuthUser` 增标志；`GitlabRegion` 增模式。不新增限界上下文。
