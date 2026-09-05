# 实施计划: SSO 厂商门户原始 HTML 修复

## 任务清单

- [ ] **Task 1**: `vendor_auth_views.py` — 添加 `from .utils import _vendor_user, _SSO_ONLY_AUTH_BODY`
- [ ] **Task 2**: `staff_auth_views.py` — 添加 `from .utils import _staff_user, _SSO_ONLY_AUTH_BODY`
- [ ] **Task 3**: `vendor_image_views.py` — 添加 `from .utils import _vendor_user, _apply_userdata_template_to_vendor_csi, _get_cloud_credentials`
- [ ] **Task 4**: `admin_views.py` — 添加 `from .utils import _staff_user`
- [ ] **Task 5**: `public_views.py` — 添加 `from .utils import _public_container_image_payload, _vendor_cloud_server_image_runtime_userdata_body`
- [ ] **Task 6**: `misc_views.py` — 添加 `from .utils import _cloud_server_image_hardware_summary, _public_container_image_payload, _public_userdata_template_summary`
- [ ] **Task 7**: 语法验证 — `python -c "import apps.marketplace.views"` 确保无导入异常
- [ ] **Task 8**: Playwright 回归 — 重复 SSO 流程验证 `/api/vendor/auth/me/` 返回正确 JSON
- [ ] **Task 9**: 重启 ai-provider 服务使改动生效
