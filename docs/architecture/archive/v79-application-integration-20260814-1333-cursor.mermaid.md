# v79 — Vendor Docs COS Presign

```mermaid
flowchart LR
  FE[taskFE VendorApplicationForm MOD] -->|upload-url / complete| AIP[taskAiProvider MOD]
  FE -->|presigned PUT NEW| COS[Tencent COS ai-provider-1259712831 NEW]
  AIP -->|sign / Head / Get NEW| COS
  Admin[AdminPortal MOD] -->|GET/PATCH pathRule| AIP
  AIP --> Path[(vendor-docs-path.yaml NEW)]
  AIP --> Row[(ai_provider_vendor.file_key MOD)]
```

- **Version:** v79 target
- **Iteration:** vendor-docs-cos-presign-v79
- **Based on:** v78
- **Decisions:** ADR-0006；浏览器预签名 PUT；密钥仅 `config.local.yaml`；路径规则写回 conf 片段
