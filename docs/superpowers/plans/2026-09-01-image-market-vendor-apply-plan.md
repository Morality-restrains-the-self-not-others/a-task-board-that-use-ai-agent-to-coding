# 实施计划 — ImageMarket 恢复厂商申请入口

## 约束

- Logic-Rollback-OK: restore ImageMarket vendor apply form; remove provider ApplyPanel; SSO still only after qualified
- 不新增 Python/Go HTTP 接口
- 不恢复 30s vendor-status 轮询
- ImageMarket.vue ≤500 行：表单拆组件

## Tasks

- [x] T1 红：`imageMarketVendorCta.js` 四态纯函数单测（SSO / 绑邮箱 / pending / 可打开表单）
- [x] T2 绿：实现 CTA 纯函数
- [x] T3 红：改 `ImageMarket.vendorStatus.test.js` 为四态
- [x] T4 绿：`ImageMarketVendorApply.vue` + `vendorDocsDirectUpload.js`
- [x] T5 红：portal 测试改为 **不**挂 ApplyPanel
- [x] T6 绿：删除 `VendorApplyPanel.vue`；改 `VendorPortal.vue` 文案
- [x] T7 事件：不新增 MQ；对照表保持 VendorApplicationSubmitted 日志例外
- [x] T8 跑 vitest + node:test；`taskFE/app` `npm run build`
