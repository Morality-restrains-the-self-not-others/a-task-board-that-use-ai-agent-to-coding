# 设计：超管充值消费情况（价格管理）

- 日期：2026-07-22
- 状态：已采纳（goal-mode）
- 架构：v48
- `python_api_approval`: scoped（超管只读薄代理）

## 方案 A（采纳）

| 层 | 职责 |
|----|------|
| taskBill | `GET /api/internal/taskbill/admin/user-recharge-consumption/` 按 user_id 聚合 |
| Django | `GET /api/system-admin/user-recharge-consumption/` 超管闸门 + 补全用户资料 |
| Vue | 价格管理页 tabs：定价套餐 \| 充值消费情况；侧栏高亮 |

### 字段定义

| 列 | 定义 |
|----|------|
| 充值金额（元） | 用户充值类 `billing_transaction`（排除 `admin_grant`）积分合计 / 100 |
| 已消费积分 | `transaction_type='consumption'` SUM(amount) |
| 未消费积分 | 关联 ledger 的 `SUM(remaining_points)`；无 ledger 则 max(0, 充值积分−已消费) 近似 |
| 可分成引流分成积分 | **= 已消费积分**（引荐 1%+1% 基数） |

## 引荐页

`UserReferral.vue` 规则区增加醒目说明：**仅下线已消费积分可产生分成；未消费余额与充值未使用积分不计佣。**

## 非目标

证件 KYC、退款审批、改分成费率。
