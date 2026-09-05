# v79 — Enterprise Landscape (Vendor Docs COS)

```mermaid
flowchart LR
  FE[taskFE MOD] -->|presigned PUT NEW| COS[Tencent COS ai-provider-1259712831 NEW]
  AIP[taskAiProvider MOD] -->|sign / Head / Get NEW| COS
  GW[API Gateway] --> AIP
  GW --> FE
  AIP --> DB[(taskAiProvider DB)]
```

- **Version:** v79 target
- **Iteration:** vendor-docs-cos-presign-v79
- **Based on:** v13 enterprise-landscape current
- **Decisions:** ADR-0006
