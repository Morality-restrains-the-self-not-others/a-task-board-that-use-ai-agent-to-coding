# Value Stream: 修复 installed-images/dev-catalog 503 错误

> Derived from design: `.claude/plans/01-brainstorming-设计文档.md`

## Value Summary

开发者调用 `GET /api/tenant/{id}/installed-images/dev-catalog/` 时，当镜像市场服务（Saas_Ai_Provider）内部出错，不再收到无意义的 `"镜像服务返回错误: 500"`，而是获得包含 `error_type` 和 `trace_id` 的可排查错误响应，同时运维可通过主站日志直接看到镜像服务的响应体。

## Related Value Streams

- **cloud-integration / installed-image-api**: modification — 增强该步骤的错误处理路径，不改变正常流程。现有测试 `tests/test_installed_image.py` 已覆盖正常路径，需新增异常路径测试。

## End-to-End Flow

```
GET /api/tenant/{id}/installed-images/dev-catalog/
  → APISIX 转发 → Django get_dev_catalog()
    → fetch_vendor_development_catalog(user_id)
      → HTTP GET Saas_Ai_Provider /api/public/vendor-development-catalog/
        → [异常时] logger.exception 记录堆栈 + 返回结构化错误
      ← [异常时] 日志含响应体
    ← [异常时] Response(503, {detail, trace_id})
```

## Value Increments

### Increment 1: 纵深防御错误处理 (唯一增量)
**Value to user:** 错误可排查 — 日志含堆栈/响应体，API 响应含 trace_id
**Scope:**
- Saas_Ai_Provider: `_public_container_image_payload()` 空值防护 + `VendorDevelopmentCatalogView` try/except
- Saas_project: `fetch_vendor_development_catalog()` 响应体日志 + `get_dev_catalog()` 携带 trace_id
**Depends on:** nothing

## YAML Impact

**No new YAML entries needed.** 本次修复是现有 `installed-image-api` 步骤的错误处理增强：
- 测试文件不变: `tests/test_installed_image.py`
- 字段不变: `saas-backend.tenant_installed_images.image_url`
- 仅需在现有测试中新增 `ExternalAPIError → 503` 的异常路径用例
