# 实施计划: relay 直启预检内网 origin

## Task Checklist

### A. 后端预检 origin
- [x] **A1.** `_resolve_relay_precheck_task_api_origin()` + precheck 使用 internal URL
- [x] **A2.** `test_relay_to_trae_repo_credentials_precheck_uses_internal_task_api_origin`

### B. 前端错误分流
- [x] **B1.** 409 才展示 OAuth 引导；5xx 展示连通性提示

### C. 关联项目 UX
- [x] **C1.** 「克隆账号已保存」徽章

### D. 验证
- [ ] **D1.** `pytest tests/test_relay_to_trae_proxy.py -k precheck`
- [ ] **D2.** 前端单测（如有）
