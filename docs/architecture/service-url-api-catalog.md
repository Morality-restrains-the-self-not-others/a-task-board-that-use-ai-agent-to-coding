# Service URL / API catalog

> Intended topology (config SSOT). Grafana `:3000` overlays **runtime**
> traffic and Tempo service map — it is not the source of website URLs.

- Grafana dashboard: `http://10.2.150.68:3000/d/service-url-api-catalog/service-url-api-catalog`
- API routes: **312** (core 195, supporting 92, ops 2, internal 21, orphaned 2)
- SPA pages: **111** (core 34)

## How to read roles

| role | meaning |
|---|---|
| core | Declared value-stream path (auth/billing/task/project/compute/kyc) |
| supporting | Needed but not a primary user journey |
| ops | Health, swagger, schema, admin chrome |
| internal | Machine/`/api/internal` — not a browser URL |
| orphaned | deny or null-upstream (dead public path) |

## By upstream service

| service | core | supporting | ops | internal | orphaned |
|---|---:|---:|---:|---:|---:|
| `(none)` | 0 | 0 | 0 | 0 | 1 |
| `gitOauth` | 4 | 1 | 0 | 0 | 0 |
| `null-upstream` | 0 | 0 | 0 | 0 | 1 |
| `taskAIComment` | 0 | 1 | 0 | 0 | 0 |
| `taskAgentSupport` | 0 | 0 | 0 | 6 | 0 |
| `taskAiProvider` | 0 | 6 | 0 | 0 | 0 |
| `taskAuth` | 73 | 43 | 2 | 0 | 0 |
| `taskBill` | 68 | 10 | 0 | 14 | 0 |
| `taskCloudService` | 18 | 1 | 0 | 1 | 0 |
| `taskFE` | 0 | 1 | 0 | 0 | 0 |
| `taskProjectService` | 7 | 12 | 0 | 0 | 0 |
| `taskReferral` | 16 | 0 | 0 | 0 | 0 |
| `taskSse` | 1 | 0 | 0 | 0 | 0 |
| `taskTaskService` | 7 | 5 | 0 | 0 | 0 |
| `taskTenantService` | 1 | 12 | 0 | 0 | 0 |

## Core API routes

| uri | upstream | owner | stream | auth |
|---|---|---|---|---|
| `/api/workspace/settings` | `taskProjectService` | `` | project | token |
| `/api/workspace/settings/` | `taskProjectService` | `` | project | token |
| `/api/workspace/settings/*` | `taskProjectService` | `` | project | token |
| `/api/workspace/info` | `taskProjectService` | `` | project | token |
| `/api/workspace/info/` | `taskProjectService` | `` | project | token |
| `/api/workspace/info/*` | `taskProjectService` | `` | project | token |
| `/api/sse/*` | `taskSse` | `` | task | token |
| `/api/git-oauth/github-callback/*` | `gitOauth` | `` | git-oauth | public |
| `/api/git-oauth/gitlab-callback/*` | `gitOauth` | `` | git-oauth | public |
| `/api/git-oauth/*` | `gitOauth` | `git-oauth` | git-oauth | token |
| `/api/accounts/*/oauth/callback/` | `gitOauth` | `saas-backend` | user-auth | public |
| `/api/accounts/users/logout/` | `taskAuth` | `saas-backend` | user-auth | token |
| `/api/accounts/users/logout/*` | `taskAuth` | `saas-backend` | user-auth | token |
| `/api/accounts/users/bind_phone/` | `taskAuth` | `saas-backend` | user-auth | token |
| `/api/accounts/users/bind_phone/*` | `taskAuth` | `saas-backend` | user-auth | token |
| `/api/accounts/users/bind_email/` | `taskAuth` | `saas-backend` | user-auth | token |
| `/api/accounts/users/bind_email/*` | `taskAuth` | `saas-backend` | user-auth | token |
| `/api/accounts/sso` | `taskAuth` | `saas-backend` | user-auth | public |
| `/api/accounts/sso/` | `taskAuth` | `saas-backend` | user-auth | public |
| `/api/accounts/sso/*` | `taskAuth` | `saas-backend` | user-auth | public |
| `/api/auth/user-roles/` | `taskAuth` | `task-auth` | user-auth | token |
| `/api/auth/user-roles` | `taskAuth` | `task-auth` | user-auth | token |
| `/api/auth/user-roles/user_id/*` | `taskAuth` | `task-auth` | user-auth | token |
| `/api/auth/user-permissions/` | `taskAuth` | `task-auth` | user-auth | token |
| `/api/auth/user-permissions` | `taskAuth` | `task-auth` | user-auth | token |
| `/api/auth/role-users/*` | `taskAuth` | `task-auth` | user-auth | token |
| `/api/auth/roles/*` | `taskAuth` | `task-auth` | user-auth | token |
| `/api/auth/resource-groups/` | `taskAuth` | `task-auth` | user-auth | token |
| `/api/auth/resource-groups/*` | `taskAuth` | `task-auth` | user-auth | token |
| `/api/auth/verify` | `taskAuth` | `task-auth` | user-auth | token |
| `/api/auth/wechat/mp/callback/` | `taskAuth` | `task-auth` | user-auth | public |
| `/api/auth/wechat/mp/callback` | `taskAuth` | `task-auth` | user-auth | public |
| `/api/auth/wechat/mp/callback/*` | `taskAuth` | `task-auth` | user-auth | public |
| `/api/auth/wechat/mp/follow-status/` | `taskAuth` | `task-auth` | user-auth | token |
| `/api/auth/wechat/mp/follow-status` | `taskAuth` | `task-auth` | user-auth | token |
| `/api/auth/wechat/mp/follow-status/*` | `taskAuth` | `task-auth` | user-auth | token |
| `/api/auth/wechat/mp/follow-qr/` | `taskAuth` | `task-auth` | user-auth | token |
| `/api/auth/wechat/mp/follow-qr` | `taskAuth` | `task-auth` | user-auth | token |
| `/api/auth/wechat/mp/follow-qr/*` | `taskAuth` | `task-auth` | user-auth | token |
| `/api/auth/impersonation/stop/` | `taskAuth` | `task-auth` | user-auth | token |
| `/api/auth/impersonation/status/` | `taskAuth` | `task-auth` | user-auth | token |
| `/api/auth/inbox/` | `taskAuth` | `task-auth` | user-auth | token |
| `/api/auth/inbox/*` | `taskAuth` | `task-auth` | user-auth | token |
| `/api/auth/login-history/` | `taskAuth` | `task-auth` | user-auth | token |
| `/api/auth/login-history/*` | `taskAuth` | `task-auth` | user-auth | token |
| `/api/auth/` | `taskAuth` | `task-auth` | user-auth | public |
| `/api/auth/*` | `taskAuth` | `task-auth` | user-auth | public |
| `/api/accounts/users/login/` | `taskAuth` | `saas-backend` | user-auth | public |
| `/api/accounts/users/email_register/` | `taskAuth` | `saas-backend` | user-auth | public |
| `/api/accounts/users/phone_register/` | `taskAuth` | `saas-backend` | user-auth | public |
| `/api/accounts/users/resend_activation_email/` | `taskAuth` | `saas-backend` | user-auth | public |
| `/api/accounts/users/send_password_reset_link/` | `taskAuth` | `saas-backend` | user-auth | public |
| `/api/accounts/users/send_password_reset_code/` | `taskAuth` | `saas-backend` | user-auth | public |
| `/api/accounts/users/reset_password_with_code/` | `taskAuth` | `saas-backend` | user-auth | public |
| `/api/accounts/users/send_verification_code/` | `taskAuth` | `saas-backend` | user-auth | public |
| `/api/accounts/users/login-with-access-token/` | `taskAuth` | `saas-backend` | user-auth | public |
| `/api/accounts/users/activate-session/` | `taskAuth` | `saas-backend` | user-auth | public |
| `/api/accounts/users/activate-session/*` | `taskAuth` | `saas-backend` | user-auth | public |
| `/api/accounts/users/access-tokens/` | `taskAuth` | `task-auth` | user-auth | token |
| `/api/accounts/users/access-tokens/*` | `taskAuth` | `task-auth` | user-auth | token |
| `/api/accounts/users/me/account-deletion/` | `taskAuth` | `saas-backend` | user-auth | token |
| `/api/accounts/users/me/account-deletion/*` | `taskAuth` | `saas-backend` | user-auth | token |
| `/api/accounts/users/me/personal-data-export/` | `taskAuth` | `saas-backend` | user-auth | token |
| `/api/accounts/users/me/personal-data-export/*` | `taskAuth` | `saas-backend` | user-auth | token |
| `/api/accounts/users/client-ip/` | `taskAuth` | `task-auth` | user-auth | public |
| `/api/accounts/users/client-ip/*` | `taskAuth` | `task-auth` | user-auth | public |
| `/api/accounts/users/confirm_activation/*` | `taskAuth` | `saas-backend` | user-auth | public |
| `/api/accounts/users/reset-password-with-link/*` | `taskAuth` | `saas-backend` | user-auth | public |
| `/api/accounts/users/get-reset-user-info/*` | `taskAuth` | `saas-backend` | user-auth | public |
| `/api/user/*` | `taskTaskService` | `task-task-service` | referral | token |
| `/api/accounts/users/profile/*` | `taskAuth` | `saas-backend` | user-auth | token |
| `/api/accounts/users/*` | `taskAuth` | `task-auth` | user-auth | token |
| `/api/billing/*` | `taskBill` | `task-bill` | billing | token |
| `/api/tenant/*/billing/*` | `taskBill` | `task-sse` | billing | token |
| `/api/tenant/*/gitlab-oidc-sso` | `taskAuth` | `task-auth` | user-auth | token |
| `/api/tenant/*/gitlab-oidc-sso/` | `taskAuth` | `task-auth` | user-auth | token |
| `/api/tenant/*/gitlab-oidc-sso/*` | `taskAuth` | `task-auth` | user-auth | token |
| `/api/tenant/*/workspace/*/queue-schedule` | `taskTaskService` | `saas-backend` | project | token |
| `/api/tenant/*/workspace/*/queue-schedule/*` | `taskTaskService` | `saas-backend` | project | token |
| `/api/tenant/*/workspace/*/todos` | `taskTaskService` | `saas-backend` | project | token |
| `/api/tenant/*/workspace/*/todos/` | `taskTaskService` | `saas-backend` | project | token |
| `/api/tenant/*/workspace/*/todos/*` | `taskTaskService` | `saas-backend` | project | token |
| `/api/system-admin/users/*` | `taskAuth` | `saas-backend` | billing | token |
| `/api/auth/tenant-memberships` | `taskAuth` | `task-auth` | user-auth | token |
| `/api/auth/tenant-memberships/` | `taskAuth` | `task-auth` | user-auth | token |
| `/api/auth/tenant-memberships/*` | `taskAuth` | `task-auth` | user-auth | token |
| `/api/system-admin/cloud/*` | `taskCloudService` | `saas-backend` | compute | token |
| `/api/system-admin/sub-token-providers` | `taskCloudService` | `saas-backend` | compute | token |
| `/api/system-admin/sub-token-providers/` | `taskCloudService` | `saas-backend` | compute | token |
| `/api/system-admin/sub-token-providers/*` | `taskCloudService` | `saas-backend` | compute | token |
| `/api/sub-token-providers` | `taskCloudService` | `` | compute | token |
| `/api/sub-token-providers/` | `taskCloudService` | `` | compute | token |
| `/api/sub-token-providers/*` | `taskCloudService` | `` | compute | token |
| `/api/system-admin/recommended-llm-providers` | `taskCloudService` | `saas-backend` | compute | token |
| `/api/system-admin/recommended-llm-providers/` | `taskCloudService` | `saas-backend` | compute | token |
| `/api/system-admin/recommended-llm-providers/*` | `taskCloudService` | `saas-backend` | compute | token |
| `/api/system-admin/step-full-cos` | `taskCloudService` | `taskCloudService` | compute | token |
| `/api/system-admin/step-full-cos/` | `taskCloudService` | `taskCloudService` | compute | token |
| `/api/system-admin/step-full-cos/*` | `taskCloudService` | `taskCloudService` | compute | token |
| `/api/recommended-llm-providers` | `taskCloudService` | `` | compute | token |
| `/api/recommended-llm-providers/` | `taskCloudService` | `` | compute | token |
| `/api/recommended-llm-providers/*` | `taskCloudService` | `` | compute | token |
| `/api/system_admin/orders/` | `taskBill` | `saas-backend` | billing | token |
| `/api/system_admin/orders` | `taskBill` | `saas-backend` | billing | token |
| `/api/system_admin/orders/*` | `taskBill` | `saas-backend` | billing | token |
| `/api/system-admin/orders/` | `taskBill` | `task-bill` | billing | token |
| `/api/system-admin/orders` | `taskBill` | `task-bill` | billing | token |
| `/api/system-admin/orders/*` | `taskBill` | `task-bill` | billing | token |
| `/api/system_admin/profit-sharing/` | `taskBill` | `saas-backend` | billing | token |
| `/api/system_admin/profit-sharing` | `taskBill` | `saas-backend` | billing | token |
| `/api/system_admin/profit-sharing/*` | `taskBill` | `saas-backend` | billing | token |
| `/api/system-admin/profit-sharing/` | `taskBill` | `task-bill` | billing | token |
| `/api/system-admin/profit-sharing` | `taskBill` | `task-bill` | billing | token |
| `/api/system-admin/profit-sharing/*` | `taskBill` | `task-bill` | billing | token |
| `/api/system_admin/gitlab-regions/` | `taskBill` | `saas-backend` | billing | token |
| `/api/system_admin/gitlab-regions` | `taskBill` | `saas-backend` | billing | token |
| `/api/system_admin/gitlab-regions/*` | `taskBill` | `saas-backend` | billing | token |
| `/api/system-admin/gitlab-regions/` | `taskBill` | `saas-backend` | billing | token |
| `/api/system-admin/gitlab-regions` | `taskBill` | `saas-backend` | billing | token |
| `/api/system-admin/gitlab-regions/*` | `taskBill` | `saas-backend` | billing | token |
| `/api/system-admin/order-number/parse` | `taskBill` | `saas-backend` | billing | token |
| `/api/system-admin/order-number/parse/` | `taskBill` | `saas-backend` | billing | token |
| `/api/system-admin/order-number/parse/*` | `taskBill` | `saas-backend` | billing | token |
| `/api/system-admin/idempotency-records` | `taskBill` | `saas-backend` | billing | token |
| `/api/system-admin/idempotency-records/` | `taskBill` | `saas-backend` | billing | token |
| `/api/system-admin/idempotency-records/*` | `taskBill` | `saas-backend` | billing | token |
| `/api/system_admin/idempotency-records` | `taskBill` | `saas-backend` | billing | token |
| `/api/system_admin/idempotency-records/` | `taskBill` | `saas-backend` | billing | token |
| `/api/system_admin/idempotency-records/*` | `taskBill` | `saas-backend` | billing | token |
| `/api/system-admin/tenant-quotas` | `taskBill` | `task-bill` | billing | token |
| `/api/system-admin/tenant-quotas/` | `taskBill` | `task-bill` | billing | token |
| `/api/system-admin/tenant-quotas/*` | `taskBill` | `task-bill` | billing | token |
| `/api/system_admin/tenant-quotas` | `taskBill` | `saas-backend` | billing | token |
| `/api/system_admin/tenant-quotas/` | `taskBill` | `saas-backend` | billing | token |
| `/api/system_admin/tenant-quotas/*` | `taskBill` | `saas-backend` | billing | token |
| `/api/system_admin/users/*/recharges` | `taskBill` | `saas-backend` | billing | token |
| `/api/system_admin/users/*/recharges/` | `taskBill` | `saas-backend` | billing | token |
| `/api/system_admin/users/*/recharges/*` | `taskBill` | `saas-backend` | billing | token |
| `/api/system-admin/users/*/recharges` | `taskBill` | `saas-backend` | billing | token |
| `/api/system-admin/users/*/recharges/` | `taskBill` | `saas-backend` | billing | token |
| `/api/system-admin/users/*/recharges/*` | `taskBill` | `saas-backend` | billing | token |
| `/api/public/product-pricing` | `taskBill` | `saas-backend` | billing | public |
| `/api/public/product-pricing/` | `taskBill` | `saas-backend` | billing | public |
| `/api/public/product-pricing/*` | `taskBill` | `saas-backend` | billing | public |
| `/api/public/resource-pricing` | `taskBill` | `saas-backend` | billing | public |
| `/api/public/resource-pricing/` | `taskBill` | `saas-backend` | billing | public |
| `/api/public/resource-pricing/*` | `taskBill` | `saas-backend` | billing | public |
| `/api/system-admin/resource-pricing` | `taskBill` | `saas-backend` | billing | token |
| `/api/system-admin/resource-pricing/` | `taskBill` | `saas-backend` | billing | token |
| `/api/system-admin/resource-pricing/*` | `taskBill` | `saas-backend` | billing | token |
| `/api/system-admin/user-recharge-consumption` | `taskBill` | `saas-backend` | billing | token |
| `/api/system-admin/user-recharge-consumption/` | `taskBill` | `saas-backend` | billing | token |
| `/api/system-admin/user-recharge-consumption/*` | `taskBill` | `saas-backend` | billing | token |
| `/api/system-admin/refund-applications` | `taskBill` | `task-bill` | billing | token |
| `/api/system-admin/refund-applications/` | `taskBill` | `task-bill` | billing | token |
| `/api/system-admin/refund-applications/*` | `taskBill` | `task-bill` | billing | token |
| `/api/system-admin/invoice-applications` | `taskBill` | `task-bill` | billing | token |
| `/api/system-admin/invoice-applications/` | `taskBill` | `task-bill` | billing | token |
| `/api/system-admin/invoice-applications/*` | `taskBill` | `task-bill` | billing | token |
| `/api/system-admin/feedback-link-groups` | `taskBill` | `task-bill` | billing | token |
| `/api/system-admin/feedback-link-groups/` | `taskBill` | `task-bill` | billing | token |
| `/api/system-admin/feedback-link-groups/*` | `taskBill` | `task-bill` | billing | token |
| `/api/system-admin/feedback-resource-kinds` | `taskBill` | `task-bill` | billing | token |
| `/api/system-admin/feedback-resource-kinds/` | `taskBill` | `task-bill` | billing | token |
| `/api/system-admin/feedback-resource-kinds/*` | `taskBill` | `task-bill` | billing | token |
| `/api/system-admin/refund-policy` | `taskBill` | `task-bill` | billing | token |
| `/api/system-admin/refund-policy/` | `taskBill` | `task-bill` | billing | token |
| `/api/system-admin/refund-policy/*` | `taskBill` | `task-bill` | billing | token |
| `/api/accounts/users/registration-invite-codes/` | `taskAuth` | `task-auth` | user-auth | token |
| `/api/accounts/users/registration-invite-codes/*` | `taskAuth` | `task-auth` | user-auth | token |
| `/api/accounts/users/referral-codes/` | `taskReferral` | `saas-backend` | user-auth | token |
| `/api/accounts/users/referral-codes/*` | `taskReferral` | `saas-backend` | user-auth | token |
| `/api/system-admin/referral/applications/` | `taskReferral` | `task-referral` | referral | token |
| `/api/system-admin/referral/applications/*` | `taskReferral` | `task-referral` | referral | token |
| `/api/system-admin/referral/share-code/lookup/` | `taskReferral` | `task-referral` | referral | token |
| `/api/system-admin/referral/policy/` | `taskReferral` | `task-referral` | referral | token |
| `/api/system-admin/referral/policy/*` | `taskReferral` | `task-referral` | referral | token |
| `/api/system-admin/referral/config/` | `taskReferral` | `task-referral` | referral | token |
| `/api/system-admin/referral/config/*` | `taskReferral` | `task-referral` | referral | token |
| `/api/system-admin/users/*/referral-performance/` | `taskReferral` | `saas-backend` | referral | token |
| `/api/system-admin/users/*/referral-performance` | `taskReferral` | `saas-backend` | referral | token |
| `/api/system-admin/users/*/referral-performance/*` | `taskReferral` | `saas-backend` | referral | token |
| `/api/user/*/profile/referral-stats` | `taskReferral` | `saas-backend` | referral | token |
| `/api/user/*/profile/referral-stats/` | `taskReferral` | `saas-backend` | referral | token |
| `/api/user/*/profile/referral-stats/*` | `taskReferral` | `saas-backend` | referral | token |
| `/api/referral/*` | `taskReferral` | `task-referral` | referral | token |
| `/api/projects/*` | `taskProjectService` | `task-project-service` | project | token |
| `/api/kyc/admin/users/*` | `taskAuth` | `task-auth` | kyc | token |
| `/api/kyc/me/` | `taskAuth` | `task-auth` | kyc | token |
| `/api/kyc/me` | `taskAuth` | `task-auth` | kyc | token |
| `/api/kyc/me/*` | `taskAuth` | `task-auth` | kyc | token |
| `/api/tenant/*` | `taskTenantService` | `task-cloud-service` | billing | token |
| `/api/tasks/*` | `taskTaskService` | `saas-backend` | task | token |
| `/api/cloud/*` | `taskCloudService` | `` | compute | token |
| `/callback/cloudplatform/oauth2.0/aliyun` | `taskCloudService` | `` | compute | token |

## Orphaned / deny routes

| uri | id | upstream | auth |
|---|---|---|---|
| `/api/internal/*` | `deny-internal` | `None` | deny |
| `/api/*` | `api-orphaned-not-found` | `null-upstream` | public |

## Core SPA pages

| path | name | stream | source |
|---|---|---|---|
| `/onboarding/` | onboarding | user-auth | `publicRoutes.js` |
| `/pricing/` | pricing | billing | `publicRoutes.js` |
| `/login/` | login | user-auth | `publicRoutes.js` |
| `/auth/login/` | auth_login | user-auth | `publicRoutes.js` |
| `/register/` | register | user-auth | `publicRoutes.js` |
| `/projects/` | projects_root | project | `publicRoutes.js` |
| `/auth/register/` | auth_register | user-auth | `publicRoutes.js` |
| `/tenant/:tenant/projects/` | projects | project | `tenantRoutes.js` |
| `/tenant/:tenant/projects/:id/` | project_detail | project | `tenantRoutes.js` |
| `/tenant/:tenant/projects/:id/edit/` | project_edit | project | `tenantRoutes.js` |
| `/tenant/:tenant/create-project/` | create_project | project | `tenantRoutes.js` |
| `/tenant/:tenant/create-workspace/` | create_workspace | project | `tenantRoutes.js` |
| `/tenant/:tenant/profile/login-history/` | tenant_user_login_history | user-auth | `tenantRoutes.js` |
| `/tenant/:tenant/profile/referral/` | tenant_user_referral | referral | `tenantRoutes.js` |
| `/profile/referral/` | user_referral | referral | `tenantRoutes.js` |
| `/user/:id/profile/referral/` | user_referral_with_id | referral | `tenantRoutes.js` |
| `/user/:id/profile/login-history/` | user_login_history_with_id | user-auth | `tenantRoutes.js` |
| `/profile/login-history/` | user_login_history | user-auth | `tenantRoutes.js` |
| `/tenant/:tenant/billing/gitlab-resources/` |  | billing | `tenantRoutes.js` |
| `/tenant/:tenant/workspace/:workspaceId/task/:taskId/` | billing_dashboard | billing | `tenantRoutes.js` |
| `/tenant/:tenant/billing/` | billing_dashboard | billing | `tenantRoutes.js` |
| `/tenant/:tenant/billing/orders/` | billing_orders | billing | `tenantRoutes.js` |
| `/tenant/:tenant/billing/orders/create/` | order_create | billing | `tenantRoutes.js` |
| `/tenant/:tenant/billing/orders/:orderId/` | billing_order_detail | billing | `tenantRoutes.js` |
| `/tenant/:tenant/billing/transactions/` | billing_transactions | billing | `tenantRoutes.js` |
| `/tenant/:tenant/billing/usage/` | billing_usage | billing | `tenantRoutes.js` |
| `/tenant/:tenant/settings/task-panel/` | tenant_settings_task_panel | task | `tenantRoutes.js` |
| `/tenant/:tenant/settings/workspace/:workspace/feature-params/` | workspace_feature_params_settings | project | `tenantRoutes.js` |
| `/tenant/:tenant/workspace/:id/settings/` | workspace_settings | project | `tenantRoutes.js` |
| `/tenant/:tenant/workspace/:id/settings/cloud-platform/` | workspace_settings_cloud_platform | project | `tenantRoutes.js` |
| `/tenant/:tenant/workspace/:id/settings/status/` | workspace_settings_status | project | `tenantRoutes.js` |
| `/tenant/:tenant/work-panel/` | work_panel | task | `tenantRoutes.js` |
| `/tenant/:tenant/queue-schedule/` | workspace_queue_schedule | project | `tenantRoutes.js` |
| `/tenant/:tenant/workspace/:workspaceId/task-detail/:taskId/` | task_detail | project | `tenantRoutes.js` |

## Regenerate

```bash
python3 db/scripts/ci/build_service_url_api_catalog.py
python3 db/scripts/ci/build_service_url_api_catalog.py --check
```

