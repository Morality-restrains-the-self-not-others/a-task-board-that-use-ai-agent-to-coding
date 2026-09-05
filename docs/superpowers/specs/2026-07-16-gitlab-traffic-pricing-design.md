# GitLab 流量费定价 — 设计文档

- 日期：2026-07-16
- 状态：已自动采纳（goal-mode / 0-auto-flow，跳过用户确认）
- 架构影响：**无**新服务/新边界；扩展既有 taskBill 套餐价目与 `billing_unit`

## 目标

在系统管理「价格管理」中增加计费项 **GitLab 流量费**：

| 项 | 值 |
|---|---|
| `unit_type` | `gitlab_traffic` |
| 展示名 | GitLab 流量费 |
| 计量单位 | `GB`（按每 GB 计费，非 GB/月） |
| 套餐列 | `gitlab_traffic_points_per_gb` |
| 账户锁价列 | `locked_gitlab_traffic_points_per_gb` |
| 默认单价 | 1 积分/GB |
| 说明 | 同区域内网不计费 |

## 方案（唯一采纳）

完全对齐既有 `gitlab_disk` 模式：

1. Migration `006_gitlab_traffic_pricing.sql`
2. Go `taskBill`：结构体、同步 `billing_unit`、创建/列表/锁价/切换、用户描述文案、OpenAPI
3. Django bridge / stub / views 透传字段
4. 前端价格管理表单与历史表列；租户侧展示与说明

扣费入口：`POST /api/internal/taskbill/charge-gitlab-traffic/`（`gb`/`bytes` + `is_intranet`；同区域内网跳过）。GitLab 侧采集与区域判定由调用方负责。

## 成功标准

1. 超管可在价格管理创建含「GitLab 流量费」的套餐
2. `billing_unit.unit_type=gitlab_traffic`，`unit=GB`
3. 套餐/锁价 JSON 含 `gitlab_traffic_points_per_gb`
4. 用户可见说明含「同区域内网不计费」
5. Go/前端相关单测通过
