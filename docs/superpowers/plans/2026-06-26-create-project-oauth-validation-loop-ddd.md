# DDD 领域模型: Create Project 页面 GitLab OAuth 仓库校验闭环

> 输入:
> - 设计文档: `docs/specs/create-project-gitlab-oauth-validation-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-26-create-project-oauth-validation-loop-value-stream.md`
> - NFR 澄清: `docs/superpowers/plans/2026-06-26-create-project-oauth-validation-loop-nfr-clarification.md`
>
> 输出使用者: `/6-plans-实施计划`, `/7-build-构建`

## 限界上下文

本增量涉及两个已有限界上下文，不引入新上下文：

| 上下文 | 职责 | 本增量变更 |
|--------|------|-----------|
| **Git OAuth Authorization** (gitOauth) | OAuth 授权流程：start → authorize → callback → bind → token exchange | **无变更** — bind_status 生命周期、access-for-user 逻辑不变 |
| **Project Repo Validation** (task2app) | 仓库可访问性校验：URL 解析 → token 解析 → Git API 探测 | **扩展** — 值对象增加 token_status + oauth_provider；应用服务响应增加新字段 |

## 值对象

### TokenStatus (新增)

```python
# projects/domain/repo_access/value_objects/token_status.py
class TokenStatus:
    NOT_BOUND = "not_bound"         # gitOauth 404 — 用户从未授权
    TOKEN_ERROR = "token_error"     # gitOauth 网络错误/超时/500
    TOKEN_AVAILABLE = "token_available"  # 成功获取 token
    NOT_APPLICABLE = "not_applicable"    # 非 OAuth 仓库
```

**NFR 约束**: token_status 为白名单枚举 (L3 安全性)，不包含动态错误信息。

### ProjectRepoAccessResult (扩展)

```python
# projects/domain/repo_access/value_objects/project_repo_access_result.py
@dataclass(frozen=True)
class ProjectRepoAccessResult:
    is_accessible: bool
    access_status: str           # 不变
    message: str                 # 不变（内容优化）
    primary_repo_url: str | None # 不变
    token_status: str            # 新增 — TokenStatus 枚举值
    oauth_provider: str | None   # 新增 — "gitlab" | "github"
    oauth_service_provider: str | None  # 新增 — "default" | "tencent-gitlab"
```

**NFR 约束**: 新增字段向后兼容 (L2 可维护性) — `field(default)` 确保旧构造代码不崩溃。

### RepoAccessStatus (不变)

已有 5 个状态：`accessible / needs_auth / not_accessible / not_configured / uncheckable`。本次无变更。

## 领域服务

### ProjectRepoAccessCheckService (扩展)

已有 `classify_after_check()` 方法。本次变更：
- 在 factory 方法调用时传入 `token_status`, `oauth_provider`, `oauth_service_provider` 参数
- 自身逻辑不变

### RepoAccessTokenResolver (应用服务 → 领域服务映射)

对应 `projects/services/repo_access_token_resolver.py`。本次变更：
- 返回值 dict 增加 `token_status` 字段
- 区分 `not_bound` (404) vs `token_error` (网络/HTTP错误)

## 不新建的领域概念

以下概念已在设计文档中列出，但属于**应用层/基础设施层**，不进入领域模型：

| 概念 | 原因 |
|------|------|
| OAuthCallbackDetection (前端) | 纯前端路由逻辑，无后端领域概念 |
| OAuthButtonVisibility (前端) | 纯 UI 展示逻辑，由 token_status 驱动 |
| validate_git_repo_view | API 视图层，使用领域模型但不属于领域层 |

## 聚合不变

本增量不引入新聚合。已有聚合：
- **GitOauthBinding** (gitOauth 侧) — 凭据生命周期 (pending→active→failed)，本次不变
- **Project** (task2app 侧) — 项目仓库管理，本次不扩展

## 领域事件

本增量不引入新领域事件。OAuth 回调成功后的重校验由**前端路由参数**驱动（`?gitlab=ok`），而非领域事件。

NFR 澄清：不引入 WebSocket 或服务端推送，回跳检测为纯前端逻辑。

## 文件清单

| 文件 | 类型 | 动作 |
|------|------|------|
| `projects/domain/repo_access/value_objects/token_status.py` | 值对象 | **新建** |
| `projects/domain/repo_access/value_objects/project_repo_access_result.py` | 值对象 | **修改** — 新增 3 字段 + factory 方法签名 |
| `projects/domain/repo_access/value_objects/repo_access_status.py` | 值对象 | 不变 |
| `projects/domain/repo_access/services/project_repo_access_check_service.py` | 领域服务 | 不变 |
| `projects/services/repo_access_token_resolver.py` | 应用服务 | **修改** — 返回 token_status |
| `projects/services/project_repo_access_check.py` | 应用服务 | **修改** — 透传 token_status |
| `projects/views/utility_views.py` | 视图 | **修改** — validate_git_repo_view 响应增加字段 |
| `projects/git_utils.py` | 工具 | **修改** — _check_gitlab_repo 文案优化 |

## 自检

- [x] 所有实体/值对象位于 `domain/` 目录下
- [x] 领域层无 ORM 导入
- [x] 领域层无外部服务导入
- [x] 值对象 `ProjectRepoAccessResult` 为 frozen dataclass (不可变)
- [x] `TokenStatus` 为白名单枚举 (L3 安全性)
- [x] 新增字段有默认值，向后兼容 (L2 可维护性)
- [x] 领域服务接口无变更，仅参数扩展
- [x] 无新聚合，不破坏已有聚合边界
