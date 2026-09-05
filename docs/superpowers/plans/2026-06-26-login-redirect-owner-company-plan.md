# Implementation Plan: Login Redirect to Owner Company

> Inputs:
> - Value Stream: `docs/superpowers/plans/2026-06-26-login-redirect-owner-company-value-stream.md`
> - NFR: `docs/superpowers/plans/2026-06-26-login-redirect-owner-company-nfr-clarification.md`

## Tasks

### Task 1: Fix `_login_redirect_url` — prioritize creator_id

- [ ] **File:** `task2app/Saas_project/accounts/taskauth_internal_views.py`
- [ ] **Change:** In `_login_redirect_url`, replace `CompanyMember.objects.filter(user_id=user_id).first()` with `Company.objects.filter(creator_id=user_id).first()` as primary lookup, keep `CompanyMember` as fallback
- [ ] **Test:** `python -m pytest task2app/Saas_project/tests/test_enrich_login_policy_consent.py -v`
- [ ] **Test:** `python -m pytest task2app/Saas_project/tests/test_auth_principal_inc4.py -v`
- [ ] **Expected:** Login redirect returns `/tenant/<user_owned_company_id>/projects/`

### Task 2: Frontend 403 redirect resilience

- [ ] **File:** `task2app/front_project/app/src/views/Projects.vue`
- [ ] **Change:** In `loadProjects`, add `else if (response.status === 403)` branch that fetches user's actual tenant and redirects
- [ ] **Test:** `cd task2app/front_project/app && npx vitest run src/views/ProjectDetail.test.js`
- [ ] **Expected:** 403 on wrong tenant → auto-redirect to user's actual tenant

### Task 3: Verify end-to-end

- [ ] Start the app: `cd task2app && ./run.sh`
- [ ] Login as `contact@daydaymoney.com`
- [ ] Verify redirect goes to `/tenant/850256677331562496/projects/` (not `/projects/`)
- [ ] Navigate to `/tenant/857901920809938944/projects` — should redirect to own tenant
