# DDD 领域建模: gitOauth Provider 配置加载路径修复

> 输入:
> - 设计文档: `.claude/skills/1-brainstorming-设计文档/design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-26-gitoauth-provider-config-path-fix-value-stream.md`
> - NFR 澄清: `docs/superpowers/plans/2026-06-26-gitoauth-provider-config-path-fix-nfr-clarification.md`

## 跳过声明

**本次变更不涉及新的领域概念，跳过 DDD 建模。**

理由（符合 `/5-ddd-领域设计驱动` 跳过条件）：

| 条件 | 判定 |
|------|------|
| 纯前端/UI 改动 | 否 |
| **配置变更** | ✅ **是 — 仅修改 `port_config.py:53` 一行路径字符串** |
| 文档修改 | 否 |
| 价值流增量不涉及新的业务概念 | ✅ **是 — 无新 entity/VO/aggregate/event** |

### 变更内容

```diff
-    prov_dir = root / "conf" / "git-oauth" / "providers"
+    prov_dir = root / "conf" / "auth" / "git-oauth" / "providers"
```

此为基础设施配置路径修正，不触及任何领域逻辑：

- 无新增限界上下文
- 无新增实体、值对象或聚合
- 无新增仓储接口
- 无新增领域事件
- 无新增领域服务

### 已有领域模型（不受影响）

本修复不改变 gitOauth 中已有的领域模型：

| 已有概念 | 位置 | 影响 |
|----------|------|------|
| `OauthProviderRouteRule` | `gitOauth/api/domain/entities/` | 无 — 路由规则逻辑不变 |
| `OauthProviderRoutingCatalog` | `gitOauth/api/domain/entities/` | 无 — 匹配逻辑不变 |
| `OauthAuthorizeRouteDomainService` | `gitOauth/api/domain/services/` | 无 — 路由解析逻辑不变 |
| `RepoOrigin` | `gitOauth/api/domain/value_objects/` | 无 |
| `OauthAuthorizeTarget` | `gitOauth/api/domain/value_objects/` | 无 |
| `SettingsOauthProviderRouteRuleRepository` | `gitOauth/api/infrastructure/repositories/` | 无 — 仅消费 `get_provider_configs()` |

修复仅影响上游数据源（`load_merged()` 返回空 → 返回实际配置），下游消费者无需变更。

## 自检

- [x] 跳过理由明确（配置变更 + 无新业务概念）
- [x] 已有领域模型不受影响已确认
