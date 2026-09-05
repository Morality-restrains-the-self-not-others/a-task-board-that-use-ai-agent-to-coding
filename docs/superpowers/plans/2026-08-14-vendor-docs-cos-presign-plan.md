# 实施计划：厂商证照 COS 预签名直传

> 设计 / 权限 / 价值流 / NFR / DDD 已齐。Goal-mode 自动进入 Build。

## 任务

- [x] T1 领域：PathRule 渲染/拒绝 + VendorDocStore/EventBus 端口（单测红绿）
- [x] T2 Config：vendorDocs + config.local + vendor-docs-path.yaml
- [x] T3 Local/COS 适配器 + fake COS（Presign/Head/Get）
- [x] T4 Handler：upload-url / upload-complete / 旧 upload 410 / 申请 Head / staff Get
- [x] T5 事件：VendorDocumentUploaded / VendorDocPathRuleUpdated
- [x] T6 Admin PATCH 写片段 + 热加载
- [x] T7 FE 直传；AdminVendorDocsStorage；AdminPortal 不继续膨胀
- [x] T8 Swagger/路由注册；回归单测全绿
