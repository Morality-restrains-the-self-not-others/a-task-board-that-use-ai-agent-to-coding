# 接口 / 路由前缀 → 目标服务（owner）对照表

> 状态: current | 更新: 2026-08-27  
> 机器可读 SSOT：[`db/api_route_ownership.yaml`](../../db/api_route_ownership.yaml)  
> 全站网址/接口目录（网关+owner+SPA+核心路径）：[`service-url-api-catalog.md`](./service-url-api-catalog.md)（Grafana 浏览面 http://10.2.150.68:3000/d/service-url-api-catalog/service-url-api-catalog ）  
> 规范：[`.ai/01_project_constraints/20_go_service_first_apis.md`](../../.ai/01_project_constraints/20_go_service_first_apis.md)  
> 存量迁出节奏：[task2app API Go 拆分设计](../superpowers/specs/2026-07-05-task2app-api-go-split-brainstorm-design.md)

## 原则

- **新增**业务 HTTP/RPC 接口默认落 **Go**（扩展现有 Go 服务，或新建 Go 服务）。
- **存量** Django 公网 API **不要求一次性迁完**；迁出优先级、Phase 与边界以 `2026-07-05-task2app-api-go-split-brainstorm-design.md` 为准。
- CI 冻结当前 Django `urls.py` 中的 `path` / `re_path` / `url` 字面量为 baseline；**新增**字面量未登记例外则失败。

## 前缀 → 目标服务（摘要）

完整前缀表见 `db/api_route_ownership.yaml` 的 `route_prefixes`。摘要：

| 前缀 / 域 | target_owner | status |
|---|---|---|
| `/api/tenant/*/daydaymoney/` | `task-project-service` | go |
| `/api/tenant/*/projects/*/nested-git-repos/` | `task-project-service` | go |
| `/api/tenant/*/projects/validate-git-repos/` | `task-project-service` | go |
| `/api/tenant/*/projects/validate-git-repo/` | `task-project-service` | go |
| `/api/tenant/*/projects/*/repo-access-check/` | `task-project-service` | go |
| `/api/tenant/*/projects/translate-branch-title/` | `task-project-service` | go（含中文时本服务直连 fanyi_agent） |
| `/api/internal/taskproject/translate-branch-title/` | `saas-backend` | django-internal（**DEPRECATED 2026-07-20**，Go 已原生 fanyi；勿再登记） |
| `/api/internal/taskproject/git-repos-status/` | `saas-backend` | django-internal（**DEPRECATED**，Go 已原生 enrich） |
| `/api/internal/taskproject/validate-git-repo/` | `saas-backend` | django-internal（**DEPRECATED**，公网已 Go） |
| `/api/internal/nested-git-repos/` | `task-project-service` | go |
| `/api/auth/` | `task-auth` | django-legacy |
| `/api/kyc/` | `task-auth` | go（公网 me / admin） |
| `/api/internal/kyc/` | `task-auth` | go（profile / recharge-gate / aml） |
| `/api/tenant/*/gitlab-oidc-sso/` | `task-auth` | go（ADR-0043 租户自建 GitLab OIDC SSO） |
| `/api/accounts/`、`/api/user/`、`/api/tenant/` | `saas-backend` | django-legacy |
| `/api/accounts/users/client-ip/` | `task-auth` | go |
| `/api/accounts/users/access-tokens/` | `task-auth` | go |
| `/api/accounts/users/registration-invite-codes/` | `task-auth` | go |
| `/api/public/registration-invite-policy/` | `task-auth` | go |
| `/api/system-admin/registration-invite-*` | `task-auth` | go（须高于 django `/api/system-admin/`） |
| `/api/system-admin/profit-sharing` | `task-bill` | go（待分账队列只读；须高于 django 通配） |
| `/api/system-admin/tenant-quotas` | `task-bill` | go（平台员工按租户只读配额） |
| `/api/system-admin/tenant-workspaces` | `task-project-service` | go（平台员工按租户列出全部工作空间） |
| `/api/billing/profit-sharing/referrer-orders/` | `task-bill` | go（推荐人本人列表+手动分账） |
| `/api/internal/taskauth/`、`/api/internal/git-identities/`、`/api/internal/session/`、`/api/internal/task-events/`（含 intents）等 | `saas-backend` | django-internal |
| `/api/internal/tasks/`（container-snapshot） | `task-task-service` | go |
| 容器 compute `container-*` / `relay-to-trae/` | `task-container-gateway` | go |
| container-token 热路径 | `task-credential-service` | go |
| `/api/internal/budget/` | `task-cloud-service` | go |
| `/api/internal/extract-auto-run-steps/` | `task-cloud-service` | go |
| `/api/ai-provider/saas-inbound-skill-versions/` | `ai-provider` | go |
| `/api/public/image-groups/` | `ai-provider` | go（镜像组公开图标） |
| `/api/ai-provider/public-image-groups/` | `ai-provider` | go |
| `/api/tenant/*/workspace/*/work-panel-events-sse/` | `task-sse` | go |

## 维护流程

1. **在 Go 新增接口**：在对应 Go 服务实现；如有公网前缀变化，更新 `route_prefixes`；**不要**往 Django `urls.py` 加新 `path`。
2. **必须在 Django 新增接口（例外）**：设计门禁通过后，在 `approved_python_exceptions` 增加 `route`（`urls.py相对路径::pattern`）、`reason`、`tracking`；下一轮可将该 route 并入 `django_baseline_routes` 并移除例外项（或保留例外直至迁出）。
3. **从 Django 删路由（已迁 Go）**：删除代码后重新 dump baseline 或手动从 `django_baseline_routes` 剔除对应项（CI 对 stale baseline 仅 WARN）。
4. **刷新 baseline**（仅在确认当前 Django 路由即为允许存量时）：

```bash
python3 db/scripts/ci/check_django_new_api_routes.py --dump-baseline > /tmp/baseline_fragment.yaml
# 将 django_baseline_routes 段写回 db/api_route_ownership.yaml
```

## CI

```bash
python3 db/scripts/ci/check_django_new_api_routes.py
```

- 扫描 `task2app/Saas_project`、`task2app/Saas_Ai_Provider` 下 `urls.py`。
- `current - baseline` 且不在 `approved_python_exceptions` → **失败**。
- 接入：`runAll/scripts/ci/check_ddd_bdd_compliance.py` 根包装器。

## 与表所有权的关系

- 表 → 服务：[`table-to-owner.md`](./table-to-owner.md) / `db/table_ownership.yaml`
- 接口 → 服务：本文 / `db/api_route_ownership.yaml`
- 二者须一致：新接口落点服务应同时是相关表的 owner，或经 owner 转发。

## 变更记录

- 2026-08-20：登记 `/api/ai-provider/saas-inbound-skill-versions/` → ai-provider（ADR-0024 容器→SaaS 接口版本目录）。
- 2026-08-19：登记 `/api/kyc/`、`/api/internal/kyc/` → task-auth（Go）；表 `auth_kyc_*` / `auth_aml_*` 见 table_ownership。关闭陈旧 feat PR（accounts_kyc_* 命名已废弃）。
- 2026-07-16：补齐网关路由 — `POST|GET|DELETE /api/accounts/users/access-tokens/` → task-auth（此前 POST 落入 django-default 返回 405）。
- 2026-07-15：补齐网关路由 — `container-auto-run-steps*` 纳入 `container-outbound-l0`（此前漏登记导致经 task-cloud-service 通配返回 501，UI 显示「HTTP 501」）。
- 2026-07-14：`POST /api/internal/extract-auto-run-steps/` → task-cloud-service（OCI 抽取 autoRunStep.md）；L0 `container-auto-run-steps` → task-container-gateway。
- 2026-07-14：新增 Django internal `POST /api/internal/session/resolve/`（SessionStore；供 task-auth 替代直连 saas.sqlite3）。
- 2026-07-13：初版 — 落地机器可读对照表、Django 新增路由 CI、明确存量迁出节奏文档指针。
- 2026-07-13：DRF 动作 `POST /api/accounts/users/profile/plugin-screenshots/`（Chrome 插件元素截图）落 accounts 用户媒体；无新增 `urls.py` path 字面量。例外理由：与 `profile/avatar` 同属用户媒体上传，见 `docs/superpowers/specs/2026-07-13-task-chrome-plugin-element-picker-deep-enhancements-design.md`。支持 `PLUGIN_SCREENSHOT_PUBLIC_BASE_URL` / `PLUGIN_SCREENSHOT_TTL_DAYS` 与 `cleanup_plugin_screenshots`。
