# Feature-Params 访问控制 — DDD 摘要

## 限界上下文

SaaS Projects / Tenant Feature Params（现有）。新增实体：`FeatureParamsAccessAudit`（审计记录，非聚合根；从属访问用例）。

## 领域服务

`FeatureParamsAccessService`（应用服务）：
- `resolve_access(request, *, resource, allowed_contexts) -> AccessDecision`
- `redact_for_summary(payload) -> dict`
- `record_audit(decision, ...) -> None`

## 端口

- 审计仓储：Django ORM 适配器（同进程）

## 事件例外

无 MQ：审计落库为本域 SSOT。
