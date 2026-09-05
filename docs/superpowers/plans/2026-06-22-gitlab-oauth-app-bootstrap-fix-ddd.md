# DDD 领域建模: GitLab OAuth Application 启动期自愈创建

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-06-22-gitlab-oauth-app-bootstrap-fix-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-22-gitlab-oauth-app-bootstrap-fix-value-stream.md`
> - NFR 澄清: `docs/superpowers/plans/2026-06-22-gitlab-oauth-app-bootstrap-fix-nfr-clarification.md`
>
> 输出使用者: `/6-plans-实施计划`, `/7-build-构建`
>
> **模块类型：基础设施（bash + Docker + 内联 Ruby）。领域模型为概念层面——git-service 不包含 Python 服务代码。核心实体（`Doorkeeper::Application`）归 GitLab 内部所有；我们的领域服务通过 `docker exec ... gitlab-rails runner` 操作它。**

## 1. 限界上下文

| 上下文 | 职责 | 实现位置 |
|--------|------|---------|
| **git-service** | 自托管 GitLab 基础设施生命周期管理——容器编排、启动健康检查、OAuth Application 配置同步 | `gitService/` (bash + docker compose) |
| **git-oauth** (上游消费者) | 消费 Doorkeeper Application 凭证（client_id/secret）执行 OAuth authorize + token 换票 | `gitOauth/` (Django) |

**上下文关系：**
```
git-service (provisioner) ──bootstraps──> Doorkeeper::Application (inside GitLab container)
                                                  │
git-oauth (consumer) ──reads client_id/secret from──> conf/auth/git-oauth/providers/*.yaml
```

git-service **写入** Application 到 GitLab 数据库；git-oauth **读取** 对应凭证并用于 OAuth 流程。两者通过 YAML 配置文件（`client_id` 作为关联键）解耦。

## 2. 实体

### Doorkeeper::Application（外部实体，GitLab 内部）

GitLab 的 Doorkeeper OAuth 引擎所有。我们通过 `gitlab-rails runner` 操作它。

| 属性 | 类型 | 描述 |
|------|------|------|
| `uid` | string (token) | OAuth client_id — 唯一标识，与 `conf/.../client_id` 关联 |
| `secret` | string (token) | OAuth client_secret |
| `name` | string | 人类可读标签（`"gitOauth Local GitLab"`） |
| `scopes` | string | 空格分隔的 OAuth scope 列表 |
| `redirect_uri` | string | OAuth 回调 URL |
| `confidential` | boolean | 机密客户端标志（`true` = 可安全保存 secret） |

**生命周期：**
```
[不存在] ──find_or_initialize_by──> [new_record?] ──save!──> [已持久化]
                                                                    │
                                              sync_local_oauth_app_scopes.sh (后续调用)
                                                                    │
                                                              [scopes 已更新]
```

**标识：** `uid`（`has_secure_token` 生成的令牌）——由我们的 YAML 配置**显式设置**以保持跨容器重建的稳定性。

## 3. 值对象

### OAuthScope

```python
# 概念性 — 无 Python 实现（此逻辑存在于内联 Ruby 中）
# 在 gitOauth/provider_registry.py 的 Django 侧验证

@dataclass(frozen=True)
class OAuthScope:
    """GitLab OAuth scope 集合（空格分隔）"""
    tokens: tuple[str, ...]  # 例如 ('read_repository', 'api', 'read_user')

    def __post_init__(self):
        invalid = [t for t in self.tokens if t in ('repo', 'read:user')]
        if invalid:
            raise ValueError(f"GitHub 风格的 scope token 对 GitLab 无效: {invalid}")
```

**归属：** 值对象逻辑存在于 `gitOauth/config/provider_registry.py`（Django 启动时 scope 校验）。同步脚本信任 YAML 值并按原样传递。

### ProviderConfig

```python
# 概念性 — 无 Python 代码（此逻辑存在于 sync_local_oauth_app_scopes.sh 内联 Python 中）

@dataclass(frozen=True)
class ProviderConfig:
    """从 conf/auth/git-oauth/providers/<name>.yaml 解析的不可变快照"""
    client_id: str
    client_secret: str
    scope: str
    redirect_uri: str
    provider: str          # "gitlab"
    service_provider: str  # "gitlab-local" | "synology-gitlab" | ...
```

## 4. 聚合

### OAuth Application 聚合

```
┌─────────────────────────────────────────┐
│ Doorkeeper::Application (聚合根)          │
│  - uid                                  │
│  - secret                               │
│  - name                                 │
│  - scopes (OAuthScope VO)               │
│  - redirect_uri                         │
│  - confidential                         │
└─────────────────────────────────────────┘
```

**一致性边界：** 单实体聚合。无子实体。`scopes`、`redirect_uri` 和 `confidential` 必须在每次同步时保持一致。

**聚合规则（来自 NFR）：**
- **幂等：** `sync_scopes()` 必须在重复调用时产生相同结果（NFR 可维护性 L2）
- **自愈：** 聚合根缺失时自动创建（NFR 容错 L2）
- **创建稳定：** `uid` + `secret` 从 YAML 显式赋值——不是自动生成（保持跨容器重建的关联）

## 5. 仓储接口

### DoorkeeperApplicationRepository（概念性）

```python
# 概念性接口 — 实现在内联 Ruby 中（gitlab-rails runner）
# 无 Python 代码；接口在此作为语义契约

class DoorkeeperApplicationRepository(ABC):
    """GitLab Doorkeeper Application 的仓储抽象。
    实现在 GitLab 容器内部通过 ActiveRecord 完成。
    """

    @abstractmethod
    def find_or_bootstrap(self, uid: str, config: BootstrapConfig) -> Application:
        """幂等查找或创建。若不存在，以完整配置创建 Application。
        NFR: 容错 L2 — 缺失时自愈；可维护性 L2 — 幂等。
        """
        ...

    @abstractmethod
    def sync_scopes(self, app: Application, scopes: str, redirect_uri: str) -> None:
        """更新已存在 Application 的 scopes 与 redirect_uri。若 redirect_uri 为空则跳过。"""
        ...
```

**基础设施实现（Ruby / ActiveRecord）：**
```ruby
# 实现在 sync_local_oauth_app_scopes.sh 内联 Ruby runner 中
app = Doorkeeper::Application.find_or_initialize_by(uid: uid)
if app.new_record?
  app.secret = secret
  app.name = "gitOauth Local GitLab"
  app.redirect_uri = redirect
  app.scopes = scopes
  app.confidential = true
else
  app.redirect_uri = redirect if app.redirect_uri.to_s.strip.empty?
  app.scopes = scopes
end
app.save!
```

## 6. 领域服务

### OAuthApplicationBootstrapService

```python
# 概念性领域服务 — 实现在 sync_local_oauth_app_scopes.sh 中

class OAuthApplicationBootstrapService:
    """编排 GitLab 容器内的 OAuth Application 引导流程。

    职责:
    1. 等待 GitLab 就绪（gitlab-rails runner "puts 'ready'"）
    2. 从 provider YAML 解析 ProviderConfig
    3. 委托给仓储执行 find_or_bootstrap
    4. 发送同步完成事件
    """

    def __init__(self, repo: DoorkeeperApplicationRepository):
        self._repo = repo

    def bootstrap(self, config: ProviderConfig) -> Application:
        """确保 Application 存在并将 scopes 与 YAML 配置对齐。"""
        app = self._repo.find_or_bootstrap(config.client_id, config)
        self._repo.sync_scopes(app, config.scope, config.redirect_uri)
        return app
```

## 7. 领域事件

### OAuthApplicationBootstrapped

```python
# 概念性 — 当前通过 stdout 日志隐式发出（无事件总线）

@dataclass(frozen=True)
class OAuthApplicationBootstrapped:
    """Application 首次创建（之前不存在）时触发。"""
    uid: str
    name: str
    scopes: str
    provider: str          # YAML 中的 service_provider
    occurred_at: datetime
```

### OAuthApplicationScopesSynced

```python
# 概念性 — 当前通过 stdout 日志隐式发出

@dataclass(frozen=True)
class OAuthApplicationScopesSynced:
    """Application 已存在，scopes 已更新（或确认已对齐）时触发。"""
    uid: str
    previous_scopes: str | None   # 若为创建则为 None
    new_scopes: str
    occurred_at: datetime
```

**事件发出：** 当前通过脚本 stdout 隐式发出（`echo "OK: #{app.name} scopes=#{app.scopes}"`）。可演进为结构化日志或 runAll 事件。对于 L2 可观测性（NFR），当前基于日志的方式已足够。

## 8. NFR 驱动的设计决策

| NFR 决策 | 领域影响 | 应用方式 |
|----------|---------|---------|
| 容错 L2 — 自愈创建 | 仓储必须支持 `find_or_bootstrap`，而非仅 `update` | `find_or_initialize_by` + `new_record?` 分支 |
| 可维护性 L2 — 幂等 | 聚合操作必须基于「状态转移」（`sync_scopes`），而非「增量变更」 | 已存在时保留 uid/secret；重复执行不产生重复记录 |
| 可用性 L2 — 300s 超时 | 引导由 readiness 探针门控（`gitlab-rails runner "puts 'ready'"`） | 在尝试仓储操作前等待就绪，最多等待 300s |

## 9. 自检

- [x] 已识别限界上下文（git-service、git-oauth）
- [x] 实体已识别（Doorkeeper::Application — 外部，GitLab 内部）
- [x] 值对象已建模（OAuthScope、ProviderConfig — 概念性）
- [x] 聚合边界已定义（单实体，以 Doorkeeper::Application 为根）
- [x] 仓储接口已指定（find_or_bootstrap + sync_scopes）
- [x] 领域服务已识别（OAuthApplicationBootstrapService）
- [x] 领域事件已记录（Bootstrapped、ScopesSynced）
- [x] 无基础设施导入（概念性 Python 伪代码无 ORM/HTTP/Kafka 导入）
- [x] NFR 决策已关联到建模选择

**模块说明：** git-service 是基础设施模块（bash + Docker + 内联 Ruby）。本 DDD 文档为概念层面——记录了领域语义、建模选择及 NFR 约束。领域「实现」为 `sync_local_oauth_app_scopes.sh` 中的内联 Ruby runner 块（第 67-83 行）。不生成 Python 骨架代码。
