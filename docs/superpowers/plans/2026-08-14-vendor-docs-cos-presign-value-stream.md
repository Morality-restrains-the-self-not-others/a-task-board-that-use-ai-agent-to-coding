# Value Stream: 厂商证照 COS 预签名直传

> Derived from design: `docs/superpowers/specs/2026-08-14-vendor-docs-cos-presign-design.md`

## Value Summary

申请人把身份证/营业执照安全存到共享私有桶；运营仍能审核；多实例不再依赖本机磁盘。

## Related Value Streams

- **vendor-application-kyc-docs（2026-08-10 设计）**：modification — 上传从本机 multipart 改为预签名 PUT；申请/SMS/审核增量不变。
- `2026-08-10-vendor-review-toggle-value-stream.md`：无重叠（审核开关）。

## End-to-End Flow

选文件 → 签发预签名 → 浏览器 PUT COS → Head 确认 → 提交申请 → 运营 GetObject 审阅

## Value Increments

### Increment 1: COS 适配器 + 预签名 + complete（Thin Slice）

**Value to user:** 申请人可把证照放到 COS 并拿到合法 file_key。  
**Scope:** VendorDocStore 端口；local/cos 适配器；upload-url / upload-complete；假 COS 单测。  
**Business intents → events:** VendorDocumentUploaded  
**Depends on:** nothing

### Increment 2: 申请提交与运营下载走 COS

**Value to user:** 提交申请与 staff 看图走 Head/Get；存量本地 key 回退。  
**Scope:** 改 Resolve / handleAdminVendorDocument；旧 multipart 在 cos 模式 410。  
**Business intents → events:** 申请提交沿用无事件例外  
**Depends on:** Increment 1

### Increment 3: FE 直传 + 管理路径规则

**Value to user:** 表单直传；运营可改 pathRule 并立即生效。  
**Scope:** VendorApplicationForm；AdminVendorDocsStorage；写 vendor-docs-path.yaml。  
**Business intents → events:** VendorDocPathRuleUpdated  
**Depends on:** Increment 1–2
