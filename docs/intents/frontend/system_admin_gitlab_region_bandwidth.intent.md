# 功能意图：系统管理 GitLab 区域卡片展示带宽共享与剩余带宽

- **日期**: 2026-08-22
- **状态**: 已实施
- **页面**: `/system-admin/gitlab-resources`

## 背景与目标

系统管理「GitLab 资源」区域卡片已展示磁盘/流量总量与剩余，但运维无法从卡片判断该分区是否走**云厂商共享带宽包**，也无法看到分区总带宽与剩余带宽。腾讯上海一区等实例需要在卡片上直接读到这三项。

## 范围与边界

- 范围内：`GET /api/system-admin/gitlab-regions/` 区域对象新增字段；卡片展示；创建/更新容量时可填写。
- 范围外：租户购买带宽商品、按租户扣减带宽、从云厂商 API 实时拉取带宽包用量。

## 字段口径

| 字段 | 含义 | 单位 |
|------|------|------|
| `bandwidth_shared` | 是否带宽共享分区 | bool |
| `total_bandwidth_mbps` | 分区总带宽（共享包或独享上限） | Mbps |
| `remaining_bandwidth_mbps` | 剩余可用带宽（管理员录入；不得大于总量） | Mbps |

剩余为独立存储值（对齐运维在云控制台看到的包余量），不是租户配额汇总。

## 验收标准

1. 每张区域卡片展示「带宽共享分区」或「独立带宽」（`data-testid="gitlab-region-bandwidth-share"`）。
2. 卡片展示总带宽与剩余带宽（Mbps），`data-testid` 分别为 `gitlab-region-bandwidth-total`、`gitlab-region-bandwidth-remaining`。
3. 管理员可在容量区更新上述三项；保存后刷新仍正确。
4. 未配置时总量/剩余为 0，不报错。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 超管更新区域带宽容量 | GITLAB_REGION_CAPACITY_UPDATED | 日志（待 Kafka） | taskBill `handleSystemAdminUpdateRegionCapacity` | 卡片与后续容量校验读 `billing_gitlab_region` | 证据豁免：与既有磁盘/流量容量更新同档，首期仅结构化日志 |

## 变更记录

| 日期 | 变更 |
|------|------|
| 2026-08-22 | 区域卡片增加带宽共享标记、总带宽、剩余带宽 |
