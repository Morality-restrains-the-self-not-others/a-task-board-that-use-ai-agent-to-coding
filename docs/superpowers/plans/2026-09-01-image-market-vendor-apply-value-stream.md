# 价值流 — ImageMarket 恢复厂商申请入口

Mapping the approved design into a value stream.

## Related Value Streams

- `ai-provider-oidc-login`（`conf/value-stream.yaml`）：OIDC 换票与 Vendor.email 绑定。本流**不改** OIDC；申请入口从门户回到镜像市场。
- `vendor-cloud-test-credentials`：qualified 之后的门户能力，本流之后才可达。

本迭代是对 2026-08-17「申请在门户」的**回退修改**（`Logic-Rollback-OK`），不是 greenfield。

## Increments（按价值顺序）

1. **漏斗 CTA（镜像市场）** — none/pending/rejected/qualified/无邮箱 四态可见；SSO 仅 qualified 或审核关+真实邮箱。
2. **提交申请（镜像市场表单）** — 复用既有 Go vendor-application / 证照直传 / 用户级短信。
3. **门户撤面板** — `VendorPortal` 不再挂 `VendorApplyPanel`。

## YAML

见 `conf/value-stream.yaml` 流 `image-market-vendor-apply`。
