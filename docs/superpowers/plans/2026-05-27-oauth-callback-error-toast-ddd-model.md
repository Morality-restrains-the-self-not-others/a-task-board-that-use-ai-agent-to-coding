# DDD 建模: OAuth 回调失败 Toast

> 输入:
> - `docs/superpowers/specs/2026-05-27-oauth-callback-error-toast-design.md`
> - `docs/superpowers/plans/2026-05-27-oauth-callback-error-toast-value-stream.md`
> - `docs/superpowers/plans/2026-05-27-oauth-callback-error-toast-nfr-clarification.md`

## 限界上下文

| 上下文 | 职责 | 与外部关系 |
|--------|------|------------|
| **OAuth Callback Presentation**（`taskFE`） | 消费 URL 中的 `gitlab`/`github` 回调码，映射用户文案，触发通知并清理 query | 上游：gitOauth 回调契约；下游：`toastService`（基础设施） |
| Git OAuth Binding（`git-oauth` / `accounts`） | 换票、落库、`bind_error` | **本增量不扩展**；仅文案与 `bind_error` 码对齐 |

本增量仅在 **OAuth Callback Presentation** 内建模。

## 实体与值对象

### 值对象

| 名称 | 文件 | 说明 |
|------|------|------|
| `OAuthGitProvider` | `oauth_git_provider_value_object.js` | `github` \| `gitlab`；自校验 |
| `OAuthCallbackCode` | `oauth_callback_code_value_object.js` | 回调 query 值（如 `profile_failed`） |
| `OAuthCallbackSeverity` | `oauth_callback_severity_value_object.js` | `success` \| `error` |
| `OAuthCallbackUserMessage` | `oauth_callback_user_message_value_object.js` | 不可变展示文案（仅来自目录） |
| `OAuthCallbackOutcome` | `oauth_callback_outcome_value_object.js` | provider + code + severity + message |

### 实体

| 名称 | 文件 | 说明 |
|------|------|------|
| `OAuthCallbackRouteSnapshot` | `oauth_callback_route_snapshot_entity.js` | 单次导航的 path + query；检测回调参数 |
| `OAuthCallbackHintCatalog` | `oauth_callback_hint_catalog_entity.js` | 静态 hints 目录（GitHub/GitLab） |

## 聚合与聚合根

**聚合：`OAuthCallbackConsumption`**

- **聚合根：** `OAuthCallbackRouteSnapshot`
- **成员：** `OAuthCallbackOutcome`（由根上的 query 解析产生）
- **一致性规则（单次导航、L0 分布式一致性）：**
  - 一次 `apply` 最多消费一个 provider query 键
  - 消费后必须从 query 中移除该键（NFR QS-03）
  - Toast 文案不得包含 query 以外的未信任字符串（NFR L2）

聚合刻意保持**小**：无跨导航状态、无持久化。

## 领域服务

| 服务 | 文件 | 职责 |
|------|------|------|
| `OAuthCallbackMessageResolutionService` | `oauth_callback_message_resolution_service.js` | 经仓储加载目录，产出 `OAuthCallbackOutcome` |
| `OAuthCallbackRouteApplicationService` | `oauth_callback_route_application_service.js` | 编排：检测 → 解析 → 通知 → 清 query → 可选发布事件 |

## 仓储接口（端口）

| 接口 | 文件 | 实现 |
|------|------|------|
| `OAuthCallbackHintCatalogRepository` | `oauth_callback_hint_catalog_repository.js` | `oauth_callback_hint_catalog_in_memory_repository.js` |
| `OAuthCallbackToastSkipPolicyRepository` | `oauth_callback_toast_skip_policy_repository.js` | `oauth_callback_toast_skip_policy_in_memory_repository.js` |

## 通知端口

| 端口 | 文件 | 基础设施实现 |
|------|------|----------------|
| `OAuthCallbackNotifierPort` | `oauth_callback_notifier_port.js` | `main.js` 中注入 `toastService.error` |

## 领域事件

| 事件 | 文件 | 载荷 |
|------|------|------|
| `OAuthCallbackConsumed` | `oauth_callback_consumed_event.js` | `provider`, `code`, `severity`, `occurredAt` |

## NFR 决策映射

| NFR | 模型动作 |
|-----|----------|
| 安全 L2 预置文案 | `OAuthCallbackHintCatalog` + `OAuthCallbackUserMessage` 禁止拼接外部输入 |
| 安全 L2 清 URL | `OAuthCallbackRouteApplicationService` 委托 snapshot 生成 cleared query |
| 可维护性 L2 单点 | hints 仅在 `OAuthCallbackHintCatalog` |
| 性能 L1 同步 | 领域服务无 async IO；`router.replace` 留在应用层 |
| skip 设置页 | `OAuthCallbackToastSkipPolicyRepository` 策略列表 |

## 应用层与领域层边界

| 层 | 路径 | 职责 |
|----|------|------|
| 领域 | `domain/oauth_callback/**` | 纯 JS，无 `vue-router` / `toastService` 导入 |
| 应用 | `utils/gitSiteOAuthCallbackUtils.js` | 组装默认仓储、注册 `setupOAuthCallbackToastGuard` |
| 基础设施 | `utils/toastService.js`、`vue-router` | 由应用层注入 |

## 产出文件

```
task2app/front_project/app/src/domain/oauth_callback/
├── value_objects/
│   ├── oauth_git_provider_value_object.js
│   ├── oauth_callback_code_value_object.js
│   ├── oauth_callback_severity_value_object.js
│   ├── oauth_callback_user_message_value_object.js
│   └── oauth_callback_outcome_value_object.js
├── entities/
│   ├── oauth_callback_route_snapshot_entity.js
│   └── oauth_callback_hint_catalog_entity.js
├── repositories/
│   ├── oauth_callback_hint_catalog_repository.js
│   ├── oauth_callback_hint_catalog_in_memory_repository.js
│   ├── oauth_callback_toast_skip_policy_repository.js
│   └── oauth_callback_toast_skip_policy_in_memory_repository.js
├── ports/
│   └── oauth_callback_notifier_port.js
├── events/
│   └── oauth_callback_consumed_event.js
└── services/
    ├── oauth_callback_message_resolution_service.js
    └── oauth_callback_route_application_service.js

task2app/front_project/app/src/tests/domain/oauth_callback/
└── oauth_callback_domain_model.test.js
```

## 自检

- [x] 领域文件位于 `domain/oauth_callback/`
- [x] 无 `vue-router`、`toastService`、ORM、HTTP 客户端导入
- [x] 通知与路由副作用经端口/应用层注入
- [x] hints 单点实体；skip 策略可扩展
- [x] 领域事件过去式命名 `OAuthCallbackConsumed`
- [x] Vitest 领域模型测试覆盖 VO 校验与解析服务

领域模型已生成到 `task2app/front_project/app/src/domain/oauth_callback/`。请审查后进入 `/6-plans-实施计划`（增量 4 回归与既有 utils 迁移验收）。
