# GitLab 磁盘购买可选时长 — 设计补充

- 日期：2026-07-18
- 状态：已采纳（goal-mode）
- 基于：`2026-07-18-tenant-gitlab-settings-resource-purchase-design.md`

## 成功标准

| # | 标准 | 验收 |
|---|------|------|
| S1 | 磁盘购买可选择时长（月） | UI select + API `disk_months` |
| S2 | 费用 = GB × 锁价(积分/GB/月) × 月数 | 单测 10×5×3+流量 |
| S3 | 记录 `disk_months` 与 `disk_expires_at` | GET/POST 回传 |
| S4 | 流量预购仍按 GB，无时长 | 不变 |

## 决策

- 时长单位：**月**（与价目单位一致）；可选 1/3/6/12，API 允许 1～36。
- `disk_gb>0` 时缺省 `disk_months=1`；`disk_gb=0` 时不收费时长。
- 新购覆盖配额与到期时间（自购买时刻起算 `now+months`），不叠加续期。
