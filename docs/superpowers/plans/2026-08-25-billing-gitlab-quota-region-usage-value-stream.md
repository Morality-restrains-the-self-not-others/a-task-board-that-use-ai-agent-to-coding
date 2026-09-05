# 价值流 — 账单页 GitLab 按区用量

- **日期**: 2026-08-25
- **设计**: `docs/superpowers/specs/2026-08-25-billing-gitlab-quota-region-usage-design.md`

## Related Value Streams

- `2026-08-18-tenant-purchase-gitlab-region-*`：购买按区入账 — 本增量只读账本。
- OPT-20260818-022 账单按区小字行：本增量将其升级为主卡片并补已用量。

## 端到端价值

租户管理员打开账单 → 看到每个 GitLab 区域的磁盘/流量 **已用 vs 配额** → 决定是否去购买页补购。

## 增量

| # | 增量 | 验收 | 依赖 |
|---|------|------|------|
| I1 | 按区卡片 + 已用/配额 | 多区 vitest；无列表不渲染按区块 | 既有 quotas |
| I2 | 无列表回退卡也显示已用 | 聚合字段 vitest | I1 |
| I3 | Go 断言列表含 used 字段 | `TestHandleResourceQuotasMultiRegion` 扩展 | 无 |

无独立 YAML valueStream 配置（纯前端展示 + 既有 GET；不引入新 runner 步骤）。
