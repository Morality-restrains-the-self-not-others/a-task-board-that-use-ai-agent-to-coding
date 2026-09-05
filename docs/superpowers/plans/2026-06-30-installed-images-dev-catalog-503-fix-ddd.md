# DDD 领域建模: 修复 installed-images/dev-catalog 503 错误

> 输入:
> - 设计文档: `.claude/plans/01-brainstorming-设计文档.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-30-installed-images-dev-catalog-503-fix-value-stream.md`
> - NFR 澄清: `docs/superpowers/plans/2026-06-30-installed-images-dev-catalog-503-fix-nfr-clarification.md`

## 跳过声明

本次修复无新领域概念。变更内容为：

| 文件 | 改动 | 领域影响 |
|------|------|---------|
| `Saas_Ai_Provider/.../utils.py` | `_public_container_image_payload()` 空值防护 | 无 — 防御性编程 |
| `Saas_Ai_Provider/.../misc_views.py` | `VendorDevelopmentCatalogView` try/except | 无 — 异常处理 |
| `Saas_project/.../external_image_service.py` | HTTP 错误日志含响应体 | 无 — 可观测性 |
| `Saas_project/.../installed_image_views.py` | 错误响应携带 trace_id | 无 — 可观测性 |

**不涉及**：新实体、新值对象、新聚合、新领域服务、新端口接口、新领域事件。

## 已有领域概念（不变）

| 概念 | 类型 | 上下文 |
|------|------|--------|
| `VendorContainerImage` | Entity | 镜像市场 (Saas_Ai_Provider) |
| `Vendor` | Entity | 镜像市场 |
| `ContainerImageGroup` | Entity | 镜像市场 |
| `TenantInstalledImage` | Entity | 租户管理 (Saas_project) |
| `fetch_vendor_development_catalog()` | Domain Service | 跨上下文 — 主站调用镜像市场 |

## 依赖反转验证

不适用 — 无新增端口接口或适配器。
