# 镜像市场厂商申请 — 证照上传 + 联系方式验证 — 设计文档

> **superseded (2026-08-17)**: 认证申请入口已迁至厂商门户 `provider.*`（[2026-08-17-vendor-apply-on-provider-portal-design.md](./2026-08-17-vendor-apply-on-provider-portal-design.md)），镜像市场不再展示申请表单；证照上传/手机验证随 `VendorApplyPanel` 迁至厂商门户。本设计仅作历史记录。

- **日期**: 2026-08-10
- **作者**: claude
- **迭代**: vendor-application-kyc-docs
- **前置**: `2026-08-06-vendor-application-entry-design.md`（申请表单 + 审核流已交付）
- **状态**: superseded（入口迁厂商门户后仅作历史记录）

## 1. 问题背景

「申请成为厂商门户」表单目前仅有「公司名称 / 联系人」，运营无法核验主体资质与真实联系方式。产品要求：增加**身份证、营业执照**上传，并**验证用户联系方式（手机短信）**后方可提交。该表单随 2026-08-17 入口迁移现位于厂商门户「申请认证」卡片内，镜像市场不再承载。

## 2. 目标与成功标准

| # | 标准 | 验收 |
|---|------|------|
| S1 | 申请表单可上传身份证与营业执照（jpeg/png/webp/pdf，≤5MB） | FE 控件 + upload API + 单测 |
| S2 | 提交前须完成手机短信验证（复用 accounts 发码 + billing verify-phone-code / SMS gate） | PhoneVerificationGate + 后端 gate |
| S3 | 未上传双证或未验证手机时后端拒绝提交 | 400 + 文案 |
| S4 | 运营审核页可查看证照与脱敏手机号 | AdminPortal + staff download |
| S5 | 证照文件仅申请人上传目录可写、仅 staff 可读；日志不落 PII 内容 | 路径隔离 + 审计日志字段最小化 |
| S6 | ImageMarket.vue 行数 ≤500（表单抽离组件） | wc -l |

## 3. 方案决策（自主选定）

| # | 决策 | 理由 |
|---|------|------|
| D1 | 证照存 taskAiProvider 本地目录（`vendorDocsDir`），DB 存 `file_key` | 无对象存储基础设施；表属 ai-provider；运营审核闭环 |
| D2 | 联系方式验证复用既有 SMS gate（billing verify → taskAuth recharge-sms-gate） | 不新建验证体系；前端复用 PhoneVerificationGate |
| D3 | 申请单仍复用 `ai_provider_vendor` 行，新增列 | 与既有审核流一致，避免双表 |
| D4 | 架构视图不 bump（无新服务/边界） | 应用组件内增量；`No-architecture-bump` |

## 4. 数据模型

`dataMigrate/taskAiProvider/004_vendor_application_documents.sql`：

- `id_card_file_key` VARCHAR(512) NULL
- `business_license_file_key` VARCHAR(512) NULL
- `contact_phone` VARCHAR(32) NULL — E.164；对外 JSON 脱敏

## 5. API

### 5.1 `POST /api/ai-provider/vendor-application/upload/`（+ legacy `/api/vendor/application/upload/`）

- multipart: `kind`=`id_card`|`business_license`, `file`
- 鉴权：`X-User-Id` + 真实邮箱
- 响应：`{kind, file_key}`

### 5.2 `POST /api/ai-provider/vendor-application/`（扩展 body）

```json
{
  "company_name": "...",
  "contact_name": "...",
  "id_card_file_key": "...",
  "business_license_file_key": "...",
  "contact_phone": "+86138..."
}
```

校验：双 key 非空且属当前用户目录；contact_phone 非空；taskAuth SMS gate `sms_verified=true`。

### 5.3 `GET /api/ai-provider/admin-vendors/{id}/documents/{kind}/`

- staff JWT；流式返回文件

## 6. 前端

- 新组件 `VendorApplicationForm.vue`（从 ImageMarket 抽离）
- `PhoneVerificationGate`：新增 `forceRequired`、`descriptionText`
- AdminPortal 厂商表增加「手机 / 证照」列

## 7. 事件

无新领域事件（与既有申请流一致：无订阅方；状态闭环在 ai-provider）。

## 8. Trace / CRG

- Trace: `skipped_no_traceid`
- CRG: incremental update at pipeline start
