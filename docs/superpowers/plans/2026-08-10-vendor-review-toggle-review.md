# Step 9 — Review：vendor-review-toggle-sso-entry

## 五轴

| 轴 | 结论 |
|----|------|
| Correctness | Go SSO 套件全绿；FE 10/10；审核关自动建档/激活；默认审核开兼容 |
| Readability | 设置读写集中 marketplace_settings_handlers；桥接逻辑注释标明双模式 |
| Architecture | 设置归属 taskAiProvider；无跨服务直连表 |
| Security | admin PATCH 需 platform staff 或 staff JWT；公开 GET 仅布尔；失败默认审核开 |
| Performance | 单行表；无 N+1 |

## Intent→Event

配置变更书面无 MQ（设计文档 §7）。SSO 沿用既有 exchange。

## CRG / CodeGraph

`UpsertVendorFromBridge` 影响面：`ExchangeBridge`（已测 auto-create / activate / 审核开拒绝）。

## 残留

- Archi loadModel 未头验证 → OPT-20260810-029
- 公网验收 → OPT-20260810-028
