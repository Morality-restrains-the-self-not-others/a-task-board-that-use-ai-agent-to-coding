# 测试意图：前端 GitLab 磁盘目录价与起购 10 GB

## 对应功能意图

`docs/intents/frontend/gitlab_disk_catalog_price_and_min_gb.intent.md`

## 测试目标

管理页展示 API 单价；购买页拦截不足 10 GB；定价页展示起购 10 GB。

## 测试分层

- 单元：`OrderCreate.contract.test.js`、`Pricing.membership-section.unit.test.js`
- Playwright：`SystemAdmin.price-management-defaults.playwright.test.js`

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| F1 | 管理页 mock 4.00 | 可见 `4.00` 与 `元/GB/月` |
| F2 | 购买页磁盘填 5 | 不 POST，文案含起购 10 |
| F3 | 购买页磁盘填 10 | POST `quantity` 为 10 |
| F4 | 定价页 | 最低购买数量 10 GB，不含累计限购 1 GB |

## 通过标准

上述前端测例全绿。
