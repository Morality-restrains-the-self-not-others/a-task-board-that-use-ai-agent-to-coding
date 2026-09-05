# 实施计划: 修复 installed-images/dev-catalog 503 错误

> 输入:
> - 设计文档: `.claude/plans/01-brainstorming-设计文档.md`
> - 价值流: `docs/superpowers/plans/2026-06-30-installed-images-dev-catalog-503-fix-value-stream.md`
> - NFR: `docs/superpowers/plans/2026-06-30-installed-images-dev-catalog-503-fix-nfr-clarification.md`
> - DDD: `docs/superpowers/plans/2026-06-30-installed-images-dev-catalog-503-fix-ddd.md`

## 任务清单

### Task 1: Saas_Ai_Provider — `_public_container_image_payload()` 空值防护
- [ ] 文件: `task2app/Saas_Ai_Provider/apps/marketplace/views/utils.py:153-163`
- [ ] `o.image_group.name` → `o.image_group.name if o.image_group else ""`
- [ ] `o.image_group.description` → `o.image_group.description if o.image_group else ""`
- [ ] `o.vendor.id` → `o.vendor.id if o.vendor else 0`
- [ ] `o.vendor.company_name` → `o.vendor.company_name if o.vendor else ""`
- [ ] `o.updated_at.isoformat()` → `o.updated_at.isoformat() if o.updated_at else ""`

### Task 2: Saas_Ai_Provider — `VendorDevelopmentCatalogView.get()` 异常处理
- [ ] 文件: `task2app/Saas_Ai_Provider/apps/marketplace/views/misc_views.py:61-108`
- [ ] 在 vendor 查询 + 序列化逻辑外层包裹 `try: ... except Exception as e:`
- [ ] except 块中调用 `logger.exception(f"VendorDevelopmentCatalogView error: saas_user_id={saas_user_id}")`
- [ ] 返回 `Response({"detail": "获取开发中镜像列表失败", "error_type": type(e).__name__}, status=500)`
- [ ] 确认 logger 已在文件顶部导入

### Task 3: Saas_project — `fetch_vendor_development_catalog()` 增强错误日志
- [ ] 文件: `task2app/Saas_project/cloud/services/external_image_service.py:99-101`
- [ ] 在 `except requests.HTTPError as e:` 块中读取 `e.response.text[:500]`
- [ ] 将响应体包含在 `logger.error()` 中

### Task 4: Saas_project — `get_dev_catalog()` 错误响应携带 trace_id
- [ ] 文件: `task2app/Saas_project/cloud/views/installed_image_views.py:219-223`
- [ ] 在 `except ExternalAPIError` 的 Response 中增加 `'trace_id': request.META.get('HTTP_X_TRACE_ID', '')`

### Task 5: 测试 — Saas_project 异常路径
- [ ] 文件: `task2app/Saas_project/tests/test_installed_image.py`
- [ ] 新增测试: `test_dev_catalog_external_api_error` — mock `fetch_vendor_development_catalog` 抛出 `ExternalAPIError("镜像服务返回错误: 500")`，断言 status_code == 503，响应含 `detail` 和 `trace_id`

### Task 6: 验证
- [ ] 运行 `task2app/Saas_project/tests/test_installed_image.py` 确认全部通过
- [ ] 检查 Saas_Ai_Provider 无语法错误: `python -c "import apps.marketplace.views.utils"`
