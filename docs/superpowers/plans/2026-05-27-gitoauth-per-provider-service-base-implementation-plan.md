# Implementation Plan: gitOauth per-provider service base

**Goal:** 移除 DJANGO_GITOAUTH_BASE，内部 API 按 gitOauth 各 service_provider 的 service.allowedHost 路由。

- [x] **Step 1:** `resolve_gitoauth_service_base` + `list_distinct_gitoauth_service_bases` in `git_oauth_providers.py`
- [x] **Step 2:** Refactor `github_app_tokens.py` (6 call sites)
- [x] **Step 3:** Fix `GitlabAppConnectionView.delete` provider_key
- [x] **Step 4:** Remove `DJANGO_GITOAUTH_BASE` from settings + `django.gitoauth` from port_config
- [x] **Step 5:** Tests: `test_gitoauth_per_provider_service_base.py`, update existing pytest
- [ ] **Step 6:** Run targeted pytest + create PR
