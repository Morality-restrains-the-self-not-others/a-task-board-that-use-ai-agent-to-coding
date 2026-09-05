# DDD Domain Modeling: Health Check Proxy Bypass Fix

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-06-22-fix-health-check-proxy-bypass-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-22-fix-health-check-proxy-bypass-value-stream.md`
> - NFR 澄清: `docs/superpowers/plans/2026-06-22-fix-health-check-proxy-bypass-nfr-clarification.md`
>
> 输出使用者: `/6-plans-实施计划`

## 跳过声明

**DDD domain modeling is skipped for this change.**

理由:
1. 此修复仅修改 `runAll/src/health.go` 中的 HTTP 客户端实现（基础设施层）
2. 不引入新的限界上下文、实体、值对象、聚合、领域事件或仓储接口
3. 现有的领域概念（`HealthCheck` 配置模型、`Runner`、`StatusStore`）无需修改
4. NFR 澄清文档明确声明：「此修复仅改基础设施层 HTTP 客户端，不引入新领域概念」

## 受影响的基础设施组件

| 组件 | 变更 | 说明 |
|------|------|------|
| `healthHTTPClient` (新) | `http.Client{Transport: &http.Transport{Proxy: nil}}` | 新增包级变量，代理无关的 HTTP 客户端 |
| `checkHealth()` | `http.DefaultClient.Do` → `healthHTTPClient.Do` | 使用自定义客户端替代 DefaultClient |

## 领域层文件

无新增或修改。`runAll/domain/` 目录内容不变。
