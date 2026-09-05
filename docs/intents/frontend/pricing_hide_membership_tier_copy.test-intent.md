# 定价页去掉会员等级体系描述 — 测试意图

## 对应功能意图

`docs/intents/frontend/pricing_hide_membership_tier_copy.intent.md`

## 测试目标

确认定价页不再渲染会员等级体系描述，同时商品价格与购买门槛仍可见。

## 测试分层

- 单元：挂载 `Pricing.vue`，mock `/api/public/resource-pricing/`。
- 端到端：`Home.nav-pricing.playwright.test.js` 从首页进入 `/pricing/`。

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 定价数据加载成功 | 文案不含「会员等级体系」「平台采用两级会员制度」「可购买资源：」「升级条件」「默认等级」「商品购买门槛」「以下各商品价格、所需会员等级及购买限制。」「与购买门槛」 |
| T2 | 同上 | 仍展示「资源收费标准」、任务 / GitLab 磁盘 / GitLab 流量费 |
| T3 | 从首页点「价格」进入定价页 | 会员等级体系与商品购买门槛标题不可见；三项资源与购买说明可见 |
| T4 | 定价数据加载成功 | `pricing-change-notice` 在 DOM 中位于「购买与扣费说明」之后 |
| T5 | 页头标题 | `h1` 含 `-mt-10`（上移 40px） |

## 数据与环境

- Mock `GET /api/public/resource-pricing/` 返回含 `tiers.normal` / `tiers.vip1` 的完整 payload（证明即使后端仍下发等级描述也不渲染）。

## 通过标准

- `taskFE/app/src/views/Pricing.membership-section.unit.test.js` 全绿
- `taskFE/tests/Home.nav-pricing.playwright.test.js` 全绿

## 实现位置

- `taskFE/app/src/views/Pricing.membership-section.unit.test.js`
- `taskFE/tests/Home.nav-pricing.playwright.test.js`
