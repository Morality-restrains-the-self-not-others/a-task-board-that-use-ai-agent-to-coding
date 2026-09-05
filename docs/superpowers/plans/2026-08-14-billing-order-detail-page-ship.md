# 订单详情页 — Ship checklist

- [x] 单测：OrderCreate.contract / OrderDetail.contract / router.billingOrderDetail / BillingOrders.billingUrl — 9 passed
- [x] `node --check` 拆分后的 router JS
- [x] 行数：OrderCreate 266、OrderDetail 320、BillingOrders 474、router.js 58、tenantRoutes 468（均 ≤500）
- [x] `cd taskFE/app && npm run build` 成功（atomic-vite-build）
- [x] 已登记精准编译重启：`taskFE`
- [x] 意图无新 MQ 例外已写
- [x] 架构：无新组件，未升 ArchiMate 版本
- [ ] 公网生效：需在 http://10.2.150.68:9999/ 点「精准编译重启」
- [ ] PR：未创建（用户未要求合入/开 PR）

回滚：还原 taskFE 本次前端提交即可；无 DDL。
