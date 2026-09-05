# GitLab 流量费定价 — 实施计划

## Tasks

- [x] Migration `006_gitlab_traffic_pricing.sql`
- [x] Go：`PricingPackage` / `BillingAccount` / sync / create / switch / description / handlers / openapi
- [x] Go 测试：创建默认值、锁价切换、描述文案
- [x] Django：`pricing_store` / `pricing_views` / `taskbill_stub` / `taskbill_db` migrations 列表
- [x] 前端：价格管理表单/表格；展示 helper；计费看板与切换弹窗
- [x] 意图文档：`docs/intents/backend/gitlab-traffic-pricing.*`
- [x] 跑 taskBill 相关测试与前端 display 单测

## 事件例外

纯价目配置与查询扩展；无新业务意图成功路径需投递 MQ（书面例外：配置类写库，无跨服务业务事件消费方）。
