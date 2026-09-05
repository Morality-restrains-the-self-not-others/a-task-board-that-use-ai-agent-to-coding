# GitLab 区域带宽共享展示 — 设计

- **Date**: 2026-08-22
- **Status**: accepted（/goal 自动采用）
- **Page**: `/system-admin/gitlab-resources`

## Context

区域卡片已有磁盘/流量容量条。运维还需要知道该 GitLab 分区是否使用云厂商**共享带宽包**，以及包的总带宽与剩余带宽。这是区域级基础设施属性，不是租户可购买商品。

## Decision

在 `billing_gitlab_region` 增加三列，经既有系统管理区域 API 读写，卡片按磁盘/流量同款进度条展示。

| 列 | JSON | 说明 |
|----|------|------|
| `bandwidth_shared` | bool | 带宽共享分区 / 独立带宽 |
| `total_bandwidth_mbps` | int64 | 总带宽 Mbps |
| `remaining_bandwidth_mbps` | int64 | 剩余带宽 Mbps（独立录入，clamp 到 `[0, total]`） |

不新增 HTTP 路径。`GET/POST /api/system-admin/gitlab-regions/` 与 `PUT .../{slug}/capacity/` 扩展字段。

## Alternatives Considered

1. **从腾讯云 API 实时拉带宽包** — 引入云凭证与轮询，超出本次展示需求。
2. **按租户汇总 allocated** — 当前无带宽 SKU，汇总恒为 0，无法表达共享包余量。
3. **只改前端写死文案** — 无法区分各区、无法改总量。

## Role / NFR / DDD（压缩）

- **权限**：沿用网关 system-admin；租户列表可带只读字段，不展示编辑。
- **路径分片**：`GET/PUT /api/system-admin/gitlab-regions/` 无 tenantId，L0（区域数十条；升级触发：>1 万再分页）。
- **幂等**：PUT 以 `slug` 为业务键，重放覆盖同三字段（L2）。GET L0 无副作用。
- **领域**：`GitlabRegion` 聚合扩展；不新增 Kafka。写成功打 `region_capacity_updated` 结构化日志。
- **架构**：无新服务/协议，不更新 ArchiMate。

## Consequences

- 管理员须在容量区手工维护带宽数字（与磁盘/流量总量同类）。
- 后续若售卖带宽配额，可再增加 `allocated_bandwidth_mbps` 并由租户资源回算。
