# GitLab OAuth Scope Fail-Fast Implementation Plan

> For agentic workers: 推荐按 `/7-build-构建` 以 TDD（Red -> Green -> Refactor）逐任务执行。  
> Inputs:
> - Value Stream: `docs/superpowers/plans/2026-05-26-gitlab-oauth-scope-failfast-value-stream.md`
> - NFR: `docs/superpowers/plans/2026-05-26-gitlab-oauth-scope-failfast-nfr-clarification.md`
> - DDD: `docs/superpowers/plans/2026-05-26-gitlab-oauth-scope-failfast-ddd-model.md`

## Goal
让 GitLab OAuth 授权链路稳定使用合法 scope，并在配置误填 GitHub scope 时于启动阶段 fail-fast，且错误可定位。

## Architecture Notes (执行约束)
- 领域层优先：先落地并验证 `accounts/domain` 契约，再实现基础设施与配置入口接线。
- 领域层禁止基础设施依赖：`domain/` 不直接导入 ORM/HTTP/云 SDK。
- 双端一致：`task2app` 与 `gitOauth` 都必须具备同等 fail-fast 能力。

---

## File Scope

- Modify: `task2app/conf/port_config.json`
- Modify: `task2app/Saas_project/saas_project/settings.py`
- Modify: `task2app/Saas_project/accounts/infrastructure/repositories/__init__.py`
- Create: `task2app/Saas_project/accounts/infrastructure/repositories/static_git_provider_scope_policy_repository.py`
- Modify: `task2app/Saas_project/tests/test_github_app_start_redirect_uri.py`
- Create: `task2app/Saas_project/tests/test_git_oauth_scope_validation.py`
- Modify: `gitOauth/config/provider_registry.py`
- Modify: `gitOauth/api/tests.py`
- Modify: `value-stream.yaml`（如需同步新增/更新 step 引用）

---

## Task 1: 固化 Thin Slice 配置（GitLab scope 合法化）

**Depends on:** nothing  
**Value Stream:** Increment 1

- [ ] Step 1: 更新 `task2app/conf/port_config.json` 中所有 GitLab 条目的 `target.scope` 为 `read_repository api read_user`
- [ ] Step 2: 保持 GitHub 条目 `scope=repo read:user` 不变，防止跨 provider 误改
- [ ] Step 3: 自检配置未出现 GitLab + `repo/read:user` 组合

Run:
- `rg "\"scope\":|\"provider\": \"gitlab\"" task2app/conf/port_config.json`

---

## Task 2: 领域契约测试先行（accounts/domain）

**Depends on:** Task 1  
**Value Stream:** Increment 2

- [ ] Step 1: 保持/完善领域模型测试 `tests/domain/accounts/test_git_provider_scope_guard_domain_model.py`
- [ ] Step 2: 断言以下契约：
  - `GitProviderScope` 规范化 token
  - `GitProviderScopePolicyCatalog` 能识别非法 token
  - `GitProviderScopeValidationService` 在非法 scope 下返回拒绝事件或抛错
  - `GitProviderScopePolicyRepository` 为纯 ABC
- [ ] Step 3: 测试通过后再进入基础设施接线

Run:
- `cd task2app/Saas_project && pytest tests/domain/accounts/test_git_provider_scope_guard_domain_model.py -q`

---

## Task 3: 实现策略仓储（基础设施层）

**Depends on:** Task 2  
**Value Stream:** Increment 2

- [ ] Step 1: 新建 `accounts/infrastructure/repositories/static_git_provider_scope_policy_repository.py`
  - 实现 `GitProviderScopePolicyRepository`
  - 为 `gitlab` 返回策略目录（禁用 token: `repo`, `read:user`）
- [ ] Step 2: 在 `accounts/infrastructure/repositories/__init__.py` 暴露仓储实现
- [ ] Step 3: 保证仓储层不引入领域外副作用（仅映射静态策略）

Run:
- `cd task2app/Saas_project && pytest tests/domain/accounts/test_git_provider_scope_guard_domain_model.py -q`

---

## Task 4: 接入 task2app 配置加载链（Fail-Fast）

**Depends on:** Task 3  
**Value Stream:** Increment 2

- [ ] Step 1: 在 `saas_project/settings.py` 的 gitOauth 配置归一化流程中，使用领域服务执行 scope 校验
- [ ] Step 2: 发现非法 token 时抛 `ImproperlyConfigured`，错误信息包含：
  - `provider`
  - `service_provider`
  - `invalid_tokens`
- [ ] Step 3: 保持 list/dict 双结构配置兼容

Run:
- `cd task2app/Saas_project && pytest tests/test_git_oauth_scope_validation.py -q`

---

## Task 5: 接入 gitOauth 配置归一化链（Fail-Fast）

**Depends on:** Task 4  
**Value Stream:** Increment 2

- [ ] Step 1: 在 `gitOauth/config/provider_registry.py` 统一执行 GitLab scope token 校验
- [ ] Step 2: 非法 scope 直接抛错中止加载
- [ ] Step 3: 错误信息与 task2app 侧语义一致（便于排障）

Run:
- `cd gitOauth && python manage.py test api.tests.ProviderConfigCompatibilityTests -v 2`

---

## Task 6: 授权路由回归与契约测试

**Depends on:** Task 5  
**Value Stream:** Increment 3

- [ ] Step 1: 更新 `tests/test_github_app_start_redirect_uri.py` 的 GitLab 测试夹具为合法 scope
- [ ] Step 2: 更新 `gitOauth/api/tests.py` 中 GitLab 配置兼容/路由测试数据
- [ ] Step 3: 新增或完善“非法 GitLab scope 必须失败”的断言（task2app + gitOauth 两侧）

Run:
- `cd task2app/Saas_project && pytest tests/test_github_app_start_redirect_uri.py tests/test_git_oauth_scope_validation.py -q`
- `cd gitOauth && python manage.py test api.tests.ProviderConfigCompatibilityTests api.tests.GitlabOAuthStartRoutingTests -v 2`

---

## Task 7: 端到端验收与分层合规检查

**Depends on:** Task 6  
**Value Stream:** Increment 3

- [ ] Step 1: 手工校验 GitLab authorize URL 使用合法 scope，不再出现 invalid scope
- [ ] Step 2: 校验错误配置时服务启动被阻断
- [ ] Step 3: 扫描 `accounts/domain/`，确认无 ORM/HTTP/Kafka/云 SDK 依赖
- [ ] Step 4: 如有必要同步 `value-stream.yaml` 中 step/test_file 引用

Run:
- `python - <<'PY'\nfrom urllib.parse import urlencode\nimport requests\nparams={\"client_id\":\"x\",\"redirect_uri\":\"http://localhost:8001/api/accounts/gitlab-local/oauth/callback/\",\"scope\":\"read_repository api read_user\",\"state\":\"smoke\",\"response_type\":\"code\"}\nurl='http://localhost:8012/oauth/authorize?'+urlencode(params)\nr=requests.get(url,allow_redirects=False,timeout=10)\nprint(r.status_code, r.headers.get('Location',''))\nPY`
- `rg "django\\.db|requests|boto3|kafka|ForeignKey|OneToOneField|ManyToManyField" task2app/Saas_project/accounts/domain -g "*.py"`

---

## Done Criteria

- [ ] GitLab provider 配置全部为合法 scope
- [ ] task2app 与 gitOauth 均具备启动期 fail-fast
- [ ] 错误信息可直接定位 provider/service_provider/invalid tokens
- [ ] 回归测试全绿（含 GitHub 不回归）
- [ ] 领域层分层约束满足（无基础设施依赖）

