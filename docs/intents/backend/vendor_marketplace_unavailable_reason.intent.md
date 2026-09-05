# 意图：厂商门户版本「不可用」须展示具体原因

## 背景与目标

厂商门户（`https://provider.daydaymoney.com/`，标题「AI 容器镜像市场」）版本行上有 `badge-unavailable` 文案「不可用」，但未说明具体原因，厂商无法据此补齐运行环境配置。

根因：公开目录 `ListApprovedCatalog` 已计算 `is_marketplace_available` / `unavailable_reason`，但厂商/管理端列表 `scanContainerImageRows` 未附带这两字段；前端对缺失字段使用 `!im.is_marketplace_available`，导致恒显「不可用」且原因区不渲染。

## 范围与边界

- **纳入**：`GET /api/vendor/container-images/`、`GET /api/admin/container-images/`（及同源列表）响应附带可用性字段；厂商门户版本行徽章展示原因文案。
- **不纳入**：改变可用性判定规则本身；公开目录行为（已具备字段）。

## 约束与风险

- 可用性判定沿用 `marketplaceAvailability`：无区域关联 →「未设置任何区域运行环境…」；有关联但无 UserData 模板 →「区域运行环境未选择 UserData 模板…」。
- 前端仅在 `is_marketplace_available === false` 时展示不可用徽章（避免字段缺失误判）。

## 验收标准

1. 厂商列表 API 每条镜像含 `is_marketplace_available`（bool）与 `unavailable_reason`（string；可用时为空串）。
2. 无区域运行环境时，`unavailable_reason` 非空且徽章可见文本含该原因（不仅 hover title）。
3. 可用镜像不展示「不可用」徽章。
4. 单元/接口测覆盖上述契约。

## 实施计划

1. 列表扫描后 `attachMarketplaceAvailability`。
2. 前端 `formatUnavailableBadge` + 严格相等判断。
3. Go 接口测 + 前端 unit 测。

## 业务意图 → 事件对照

**无对应事件**：只读列表字段补齐与 UI 展示，无服务端状态变更。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 厂商门户展示镜像市场不可用原因 | — | — | — | — | 只读查询/展示，无对应事件 |

## 变更记录

| 日期 | 变更 | 原因 |
|------|------|------|
| 2026-07-18 | 初版 | 页面反馈：不可用无具体原因 |
