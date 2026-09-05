# 历史套餐管理 — 设计文档

- 日期：2026-07-16
- 状态：已采纳（goal-mode 自动确认）
- 范围：系统管理「价格管理」页补齐历史套餐管理能力

## 1. 目标与成功标准

| # | 标准 | 验收 |
|---|------|------|
| S1 | 页面区块标题为「历史套餐管理」，列表展示全部历史套餐 | UI |
| S2 | 列表含 GitLab 磁盘价目字段（与新建表单一致） | GET + UI |
| S3 | 超管可对在效期内套餐「结束在售」（设置 `valid_to`） | POST end |
| S4 | 已结束套餐不可再结束；`valid_to` ≥ `valid_from` | 校验 |
| S5 | 订户查看保持可用 | 既有 |
| S6 | 公网 SPA build + collectstatic 后可见 | 部署 |
| S7 | Go + Django 自动化测试覆盖结束在售 | 测试 |

**不做**：修改已创建套餐的单价字段（开户锁价不变）；删除套餐；租户侧换绑管理。

## 2. 方案对比（已选 A）

| 方案 | 说明 | 结论 |
|------|------|------|
| **A. 同页增强「历史套餐管理」** | 列表 + 结束在售 + 订户 | **采纳** — 与现网信息架构一致 |
| B. 独立路由/侧栏菜单 | 新页面 | 多余导航，拒绝 |
| C. 仅改标题不做管理操作 | 文案级 | 不满足「管理」 |

## 3. API

### 内部（taskBill）

`POST /api/internal/taskbill/pricing-packages/{id}/end/`

```json
{ "valid_to": "2026-07-16" }
```

- `valid_to` 可空，默认 UTC 当日。
- 成功：返回更新后的套餐 admin JSON。
- 失败：400（已结束 / 日期非法）、404（不存在）。

### 对外（Django bridge）

`POST /api/system-admin/pricing-packages/{package_id}/end/` — 仅 `is_superuser`。

同步修复：`GET/POST /api/system-admin/pricing-packages/` 透传 `gitlab_disk_points_per_gb_per_month`。

## 4. 权限与合规

- 仅超管；无新租户写接口。
- 结束在售只影响新开户/公开定价解析窗口，不改已锁价账户。
- 无支付/KYC 变更；日志不输出密钥。

## 5. 事件

- 价目窗口配置变更，无新聚合状态跃迁；**不新增 MQ 事件**（与创建套餐一致）。
- 例外理由：纯配置；扣费仍走既有锁价与用量事件。

## 6. 架构

- 无拓扑变更：仍在 taskBill 独占 `billing_pricing_package`；Django 透传。
- 不新增 ArchiMate 视图。
