# DDD 建模: GitLab OAuth Scope 合法化与启动期 Fail-Fast

> 输入:
> - `docs/superpowers/specs/2026-05-26-gitoauth-gitlab-provider-config-compat-design.md`
> - `docs/superpowers/plans/2026-05-26-gitlab-oauth-scope-failfast-value-stream.md`
> - `docs/superpowers/plans/2026-05-26-gitlab-oauth-scope-failfast-nfr-clarification.md`

## 限界上下文
- OAuth Start Gateway（`accounts`）：处理 `repo_url -> provider` 路由与启动期配置校验。
- Provider Scope Policy（`accounts`）：表达 Git provider scope 的业务不变量与拒绝事件。

## 实体与值对象
- Entity / Aggregate Root
  - `GitProviderScopePolicyCatalog`：按 provider 维护禁用 token 策略（如 GitLab 禁止 `repo`/`read:user`）。
- Value Object
  - `GitProviderScope`：规范化 scope token 集合，承担输入自校验。

## 聚合与聚合根
- 聚合：`GitProviderScopePolicyCatalog`
  - 根：`GitProviderScopePolicyCatalog`
  - 成员：`GitProviderScope`（只读输入 VO）
  - 一致性规则：策略目录定义的禁用 token 与 scope token 检测必须在单次校验中原子完成。

## 领域服务
- `GitProviderScopeValidationService`
  - 用途：编排 scope 校验流程，输出拒绝事件或通过结果。
  - 依赖：`GitProviderScopePolicyRepository`（构造函数注入）。

## 仓储接口
- `GitProviderScopePolicyRepository`
  - 职责：按 provider 加载策略目录（抽象契约，基础设施层实现）。

## 领域事件
- `GitProviderScopeValidationRejected`（过去式）
  - 语义：scope 命中禁用 token，被策略拒绝。
  - 载荷：`catalog_id`、`provider`、`service_provider`、`invalid_tokens`、`occurred_at`。

## NFR 决策映射到模型
- 安全 L3（启动期 fail-fast）-> `ValidationService.ensure_valid()` 失败即抛错，拒绝进入运行态。
- 一致性 L2（单次启动内强一致）-> 聚合根单次计算 `detect_invalid_tokens()`，不产生部分通过状态。
- 可观测性 L2 -> 事件模型显式携带 `provider/service_provider/invalid_tokens`。
- 可维护性 L2（list/dict 兼容）-> 策略加载放在仓储接口层，领域层不关心配置形态差异。

## 产出文件
- `accounts/domain/value_objects/git_provider_scope.py`
- `accounts/domain/entities/git_provider_scope_policy_catalog.py`
- `accounts/domain/repositories/git_provider_scope_policy_repository.py`
- `accounts/domain/services/git_provider_scope_validation_service.py`
- `accounts/domain/events/git_provider_scope_validation_rejected.py`
- `tests/domain/accounts/test_git_provider_scope_guard_domain_model.py`

