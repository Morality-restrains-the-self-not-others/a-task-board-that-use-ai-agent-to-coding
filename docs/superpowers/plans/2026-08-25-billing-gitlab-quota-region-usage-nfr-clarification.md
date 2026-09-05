# NFR 澄清 — 账单页 GitLab 按区用量

- **日期**: 2026-08-25
- **价值流**: `docs/superpowers/plans/2026-08-25-billing-gitlab-quota-region-usage-value-stream.md`
- **默认等级**: L2（展示）；资金路径不在本增量

## 路径分片键强制审视

| 路径 | 分片 ID | 是否合适 | 可伸缩性 | 动作 |
|------|---------|----------|----------|------|
| `GET /api/tenant/{tid}/billing/quotas/` | `tid` | 是（租户账本） | L1 | 保持；`gitlab_resources` 按区行，查询已带 tenant_id |
| `/tenant/:tid/billing/` | `:tid` | 是 | L1 | 保持 |
| 区域 GitLab Web 外链 | 非平台分片路径 | — | L0 | 真实 href，不经本站写 |

无「缺分片 ID」的新路径。`region` 是区内二级键，不单独作为跨租户分片键。

升级触发：单租户区域数 > 50 时再分页（当前产品区域个位数）。

## 幂等性审视

| 路径 | 副作用 | 判定 | 理由 |
|------|--------|------|------|
| GET quotas | 无 | L0 | 只读 |
| 账单页渲染 | 无 | L0 | 无 POST |
| 购买资源 `<a href>` | 无（导航） | L0 | 真实链接；下单幂等属购买页既有设计 |

禁止把 `tenant_id` 当写路径幂等键 — 本增量无写路径。

## 类别等级

| 类别 | 等级 | 说明 |
|------|------|------|
| 可伸缩性 | L1 | tenant_id；区域列表短 |
| 数据一致性 | L1 | 展示最近一次同步的 used；不在本页刷新 GitLab |
| 安全 | L2 | 沿用 quotas 鉴权 |
| 可观测性 | L1 | 既有 quotas 请求 trace；不新增轮询 |
| 可用性 | L1 | used 缺省按 0 展示（合法：尚未上报） |

## 质量场景

1. 刺激：两区各 1GB 磁盘、不同 used。响应：两张磁盘卡，各自 used/quota，无跨区合计主卡。
2. 刺激：`gitlab_resources` 空、聚合 `gitlab_disk_used_gb=0.2`、`gitlab_disk_gb=1`。响应：回退绿卡显示 `0.2 / 1`。
3. 刺激：流量 used ≥ prepaid。响应：进度条 100%，文案仍显示真实 used（可超过配额数字）。

## 领域模型影响

不新增聚合。展示 VO：`GitlabRegionQuotaView{region, diskUsed, diskQuota, trafficUsed, trafficPrepaid, expiresAt}`。读取端口仍是既有 quotas 查询。
