# Feature-Params 访问控制与审计

## 意图

对公司/工作空间 feature-params 完整配置（含 API Key）实施访问门禁：仅设置页上下文可取 full；其它场景仅 summary；并审计谁、何时、以何种方式访问。

## 验收

1. 无 `X-Feature-Params-Access-Context` 且非 `view=summary` → 403
2. `company_settings` / `workspace_settings` 与资源匹配时可取 full
3. `view=summary` 返回脱敏配置（api_key 空）
4. 访问写入 `FeatureParamsAccessAudit`（含 auth_method、access_context、view_mode）
5. 任务详情/预算面板改走 summary；设置页带正确 header

## 变更日期

2026-07-19

## 业务意图 → 事件对照

| 业务意图 | 事件名 | MQ | 发布点 | 例外理由 |
|---------|--------|-----|--------|---------|
| 记录 feature-params 访问 | — | — | — | 审计落库为本域 SSOT，非跨服务编排 |
