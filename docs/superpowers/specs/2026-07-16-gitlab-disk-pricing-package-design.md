# GitLab 磁盘价格套餐项 — 设计文档

- 日期：2026-07-16
- 状态：已采纳（goal-mode 自动确认）
- 范围：系统管理价格套餐增加「GitLab 磁盘」价目（积分/GB/月）

## 1. 目标与成功标准

| # | 标准 | 验收 |
|---|------|------|
| S1 | 超管在 `/system-admin/price-management` 创建套餐时可填写「GitLab 磁盘」单价 | UI + POST |
| S2 | 历史套餐列表展示该字段 | GET list |
| S3 | 字段语义为「积分 / GB / 月」 | 文案与 `unit` |
| S4 | 新开户/换套餐时锁价到 BillingAccount | locked 字段 |
| S5 | 公开定价与租户账单套餐展示包含该项 | Pricing / Dashboard |
| S6 | 种子 `billing_unit.unit_type=gitlab_disk`（价目标签） | 迁移 |
| S7 | 自动化测试覆盖创建与回传 | Go + Django |

**本迭代不做**：按实际占用 GB 的月结扣费任务（用量采集另案）；仅完成价目配置与锁价链路。

## 2. 方案对比（已选 A）

| 方案 | 说明 | 结论 |
|------|------|------|
| **A. 套餐宽表加列**（与续存同级） | `gitlab_disk_points_per_gb_per_month` + 账户锁价 | **采纳** — 与现网一致、管理页零重构 |
| B. 仅 `billing_unit` 全局价 | 不进套餐锁价 | 管理页不会出现，不符需求 |
| C. 套餐行项目子表 | 灵活 SKU | 超范围重构 |

## 3. 数据模型

```
billing_pricing_package.gitlab_disk_points_per_gb_per_month  BIGINT NOT NULL DEFAULT 1
billing_account.locked_gitlab_disk_points_per_gb_per_month   BIGINT NOT NULL DEFAULT 1
billing_unit: unit_type='gitlab_disk', name='GitLab 磁盘', unit='GB/月', price=<list price>
```

API 字段名：`gitlab_disk_points_per_gb_per_month`（创建/列表/公开定价/套餐 JSON）。

## 4. 权限

- 创建/列表：仅 `is_superuser`（既有 `system_admin_pricing_packages`）。
- 租户只读展示锁价/套餐价；换套餐沿用既有规则。
- 无新对外 Django 路由；扩展既有 taskBill 内部与租户 JSON。

## 5. 合规

- 价目为平台积分标价，非第三方收单；不引入卡数据/KYC 变更。
- 日志禁止输出支付密钥；沿用既有脱敏。

## 6. 架构影响

- **无拓扑变更**：仍在 taskBill 独占 `billing_*` 表内加列；Django bridge 透传。
- 不新增 ArchiMate 视图（非组件/边界变更）。

## 7. 事件

- 纯价目配置扩展；创建套餐不新增业务 MQ 事件（与现网续存字段一致）。
- 例外理由：无新聚合状态跃迁；扣费事件待用量计量落地后再投递。
