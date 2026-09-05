# 订单详情页 — 实施计划

## Tasks

- [ ] T0 文档：intent / design / permission / NFR / DDD（本文件）
- [ ] T1 Red：router.resolve create vs :orderId；创建成功 push 详情
- [ ] T2 Red：详情页渲染 items
- [ ] T3 Green：拆 router.js≤500；注册 `billing_order_detail`
- [ ] T4 Green：OrderDetail.vue（行项 + 迁入支付）；OrderCreate 成功后 push，删除内嵌详情
- [ ] T5 列表订单号真实 `<a href>` 进详情
- [ ] T6 支付轮询契约迁到 OrderDetail；OrderCreate 不再含「确认支付」
- [ ] T7 `wc -l` 触及文件 ≤500；vitest；`gofmt` N/A；eslint/py_compile 按类型
- [ ] T8 taskFE `npm run build`；登记精准重启 taskFE

无新 MQ 任务（查询例外）。
