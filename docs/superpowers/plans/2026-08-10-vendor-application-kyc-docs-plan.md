# Plan: vendor-application-kyc-docs

## Role / NFR / DDD（压缩）

- **角色**: 申请人（tenant user）写申请+上传；staff 读证照；禁止跨用户读文件
- **NFR**: L2 常规；PII L3（证照不进日志；手机脱敏）
- **DDD**: Vendor Application Aggregate 扩展 Documents + ContactPhone；端口 `DocumentStore`（本地 FS）；`ContactVerificationPort` → taskAuth SMS gate
- **事件**: 无（例外已文档化）

## 切片

- [x] S1 DDL 004 + domain/Vendor 字段
- [x] S2 DocumentStore + upload handler + 单测
- [x] S3 ApplyVendorApplication 扩展 + SMS gate + 更新既有 SSO 测例
- [x] S4 Admin documents GET + AdminPortal 列
- [x] S5 PhoneVerificationGate props + VendorApplicationForm + ImageMarket
- [x] S6 FE vitest 更新 + 行数门禁
- [x] S7 精准重启登记 + OPT
