# DDD 领域建模: 项目详情页 OAuth 授权状态正确反映

> 输入:
> - 价值流: `docs/superpowers/plans/2026-06-28-project-detail-oauth-token-status-value-stream.md`
> - NFR 澄清: `docs/superpowers/plans/2026-06-28-project-detail-oauth-token-status-nfr-clarification.md`
>
> 输出使用者: `/6-plans-实施计划`

## 建模策略

本变更**不引入新的限界上下文、实体、聚合、仓储接口或领域事件**。变更性质为应用层增强：将已有的领域概念（`TokenStatus` VO、`resolve_repo_access_tokens` 服务）投影到项目详情 API 响应中。

### NFR 驱动的建模决策

| NFR 决策 | 模型影响 | 决策 |
|----------|---------|------|
| 容错 L2：gitOauth 不可用时静默降级 | TokenStatus 需兜底值 | `NOT_APPLICABLE` 已在 VO 中定义，无需新增 |
| 性能 L2：每 repo 调用 gitOauth | 无需 CQRS 拆分 | 读时派生，Serializer 层计算 |

## 已有领域概念（复用）

### 限界上下文：项目与工作空间 (Projects)

| 概念 | 类型 | 位置 | 说明 |
|------|------|------|------|
| `TokenStatus` | Value Object | `projects/domain/repo_access/value_objects/token_status.py` | `NOT_BOUND`, `TOKEN_ERROR`, `TOKEN_AVAILABLE`, `NOT_APPLICABLE` |
| `ProjectRepo` | Entity (应用层) | `projects/models/project_repo.py` | 项目关联的 Git 仓库 URL |
| `resolve_repo_access_tokens` | Application Service | `projects/services/repo_access_token_resolver.py` | 解析用户对 repo 的 token 可用性 |

### 限界上下文：Git OAuth 授权 (gitOauth)

| 概念 | 类型 | 位置 | 说明 |
|------|------|------|------|
| `GitOAuthAppUserCredential` | Entity | `gitOauth/api/models.py` | 用户 OAuth 凭据，`bind_status` 控制生命周期 |
| `OauthCredentialBinding` | Entity (Domain) | `gitOauth/api/domain/entities/oauth_credential_binding.py` | 凭据绑定生命周期领域实体 |

## 新增领域概念

### `GitRepoTokenStatus` — 读侧值对象

**位置**: `projects/domain/repo_access/value_objects/git_repo_token_status.py`
**类型**: Value Object (frozen dataclass)

封装单个 repo URL 的 token 状态信息，作为 `git_repos_status` API 响应数组的元素类型。

```
GitRepoTokenStatus
├── repo_url: str
├── token_status: str        # → TokenStatus value
├── oauth_provider: str       # → "github" | "gitlab" | ""
└── oauth_service_provider: str  # → "default" | "synology-gitlab" | ...
```

**为什么是 VO 而非实体？** — 无独立 ID，无生命周期，通过属性值判等，不可变。纯读侧投影。

## 领域层自检

- [x] 新增文件位于 `domain/` 目录下
- [x] 领域层无 ORM 导入
- [x] 领域层无外部服务导入
- [x] 无新增仓储接口（复用已有）
- [x] 无新增领域事件（无新写入路径）
- [x] 无数据库字段声明
- [x] 一个文件一个类

## 未建模的概念（有意识排除）

| 概念 | 排除原因 |
|------|---------|
| `ProjectRepoTokenStatusResolver` (Domain Service) | `resolve_repo_access_tokens` 已在应用层实现，无需领域层重复 |
| `RepoAuthorized` (Domain Event) | OAuth 授权事件的发布方在 gitOauth 上下文，不在 Projects 上下文 |
| `GitRepoTokenStatusRepository` | 读侧投影，无持久化需求 |
