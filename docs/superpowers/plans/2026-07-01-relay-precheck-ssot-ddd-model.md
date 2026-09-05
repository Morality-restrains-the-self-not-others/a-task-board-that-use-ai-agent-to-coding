# DDD 领域模型: relay 直启预检 Token SSOT 消除双写

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-07-01-relay-precheck-ssot-in-process-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-07-01-relay-precheck-ssot-value-stream.md`
> - NFR 澄清: `docs/superpowers/plans/2026-07-01-relay-precheck-ssot-nfr-clarification.md`
>
> 使用者: `/7-plans-实施计划`, `/8-build-构建`

## 1. 限界上下文 (Bounded Contexts)

本变更是**架构清理**，不引入新 BC。涉及的现有 BC：

| BC | 所属服务 | 职责 | 变更 |
|----|---------|------|------|
| **容器运行时 (Container Runtime)** | Go taskCredentialService (:8015) | Token 签发/验证/审计、凭证构建、任务快照 | 无变更 — 已是 SSOT |
| **任务协作 (Task Collaboration)** | Django Saas_project (:8001) | relayToTrae 启动编排（token-init → precheck → start） | **Token 来源切换** — 从 Django 自生成 → 调 Go |
| **云平台代理 (Cloud Platform Proxy)** | Django Saas_project | CloudServerConfig 配置持久化 | **写入缩减** — relayToTrae 路径停止写 container_access_token |

## 2. 实体 (Entities)

### 2.1 ContainerToken (Go BC — 聚合根)

已在 Go `taskCredentialService/domain/entities.go` 中定义。**本次不修改。**

```
ContainerToken (聚合根)
├── ID: snowflake
├── TaskID, CompanyID, WorkspaceID (TaskScope)
├── ContainerAccessToken, ContainerRefreshToken
├── ContainerAccessTokenExpiresAt
└── 行为: Issue(scoppe) → 复用有效或生成新
```

### 2.2 CloudServerConfig (Django BC — 遗留，写入范围缩减)

`cloud.models.CloudServerConfig` — 现有 Django Model。

**变更：** relayToTrae 路径（token_init/precheck/start/register）不再写入 `container_access_token`。mockStart 路径保留写入（独立 Phase 迁移）。

```
CloudServerConfig (遗留聚合根，写入范围缩减)
├── company_id, task_id (定位键)
├── container_access_token ← relayToTrae 路径不再写入 ⚠️
├── container_access_token_expires_at ← 同上
├── container_refresh_token ← 同上
└── 行为: 仅 mockStart 路径继续写入
```

## 3. 端口接口 (Port Interfaces) ⭐ 依赖反转核心

### 3.1 TaskCredentialServicePort — 新增

Django 需要调用 Go 签发 token，定义端口接口隔离 HTTP 实现细节。

```python
# cloud/domain/ports/task_credential_service.py
from abc import ABC, abstractmethod
from dataclasses import dataclass

@dataclass(frozen=True)
class TokenInitResult:
    """Go /v1/token/init 返回值对象"""
    access_token: str
    refresh_token: str
    expires_at: str

@dataclass(frozen=True)
class RepoCloneCredentialsResult:
    """Go repo-clone-credentials 返回值对象"""
    status_code: int
    payload: dict

class TaskCredentialServicePort(ABC):
    """taskCredentialService 端口接口

    定义 Django→Go token 操作的抽象契约。
    实现者: HTTP 适配器（生产）/ 内存 stub（测试）。
    """

    @abstractmethod
    def issue_token(self, tenant_id: str, workspace_id: str, task_id: str) -> TokenInitResult:
        """调用 Go /v1/token/init 签发或复用 token"""
        pass

    @abstractmethod
    def build_repo_clone_credentials(
        self, tenant_id: str, workspace_id: str, task_id: str, access_token: str
    ) -> RepoCloneCredentialsResult:
        """调用 Go repo-clone-credentials 构建凭证"""
        pass
```

**NFR 映射（来自 NFR 澄清文档）：**
- 性能 L2: `issue_token` 适配器实现 timeout=5s；`build_repo_clone_credentials` timeout=8s
- 容错 L2: 适配器实现 trust_env=False，异常统一转为 `requests.RequestException` → 502
- 可用性 L2: Go 不可达时适配器抛异常，应用层转为明确 502 响应

**依赖反转验证：**
```
❌ 旧（Django 直接依赖 Go 实现）:
  relay_to_trae_proxy.py → requests.post("http://127.0.0.1:8015/v1/token/init")
                          → sqlite3.connect("db/container/tokens.sqlite3")  ← 最恶劣的跨语言 DB hack

✅ 新（Django 依赖端口接口）:
  relay_to_trae_proxy.py → TaskCredentialServicePort.issue_token()
                              ↑ 实现
                          HttpTaskCredentialServiceAdapter (生产)
                          InMemoryTaskCredentialServiceStub (测试)
```

**切换验证:** 若将来 Go 服务迁移到 gRPC / 不同端口 / 不同主机，只需新增 `GrpcTaskCredentialServiceAdapter`，Django 领域层零改动。

### 3.2 已有端口（不变）

| 端口 | 位置 | 用途 | 变更 |
|------|------|------|------|
| `RelayToTraeSidecarPort` (隐式) | `_relay_http_request()` | Django → go_relayToTrae sidecar | 不变 |
| `ContainerTokenAuditPort` (隐式) | `append_container_token_audit_event()` | 审计事件写入 | 不变 — 保留 Django 侧审计调用 |

## 4. 领域服务 (Domain Services)

本次变更不引入新领域服务。现有领域服务不变：

| 领域服务 | BC | 变更 |
|---------|-----|------|
| `TokenService.IssueToken` | Go Container Runtime | 不变 |
| `CredentialService.BuildRepoCloneCredentials` | Go Container Runtime | 不变 |
| `RelayTwoStepStartupService` | Django Task Collaboration | 不变 — 仍管理 token_init → start 状态机 |

## 5. 领域事件 (Domain Events)

**无新增。** 现有审计事件保留：

- `TokenAuditEventTypes.RELAY_TOKEN_INIT_ATTEMPTED` / `SUCCEEDED` / `FAILED`
- `TokenAuditEventTypes.RELAY_START_ACCEPTED` / `DISPATCHED`
- Go 内部 `token_issued` / `token_validated` 审计事件

Django 侧 `append_container_token_audit_event` 调用保留在 token_init / precheck / start / register 函数中。

## 6. 应用服务 (Application Services)

4 个 relayToTrae 入口函数即应用服务。变更后依赖 `TaskCredentialServicePort`：

```python
# 注入方式（在 relay_to_trae_proxy.py 模块级或请求级）

# 生产环境
_credential_service: TaskCredentialServicePort = HttpTaskCredentialServiceAdapter(
    base_url=settings_manager.get_task_credential_service_url()
)

# 测试环境 → mock InMemoryTaskCredentialServiceStub

# relay_to_trae_token_init (应用服务)
def relay_to_trae_token_init(request, ...):
    """编排: 调 Go token-init → 返回 env_preview"""
    result = _credential_service.issue_token(tenant, wid, tid)
    return Response({"status": "ok", "token_initialized": True, ...})

# relay_to_trae_repo_credentials_precheck (应用服务)
def relay_to_trae_repo_credentials_precheck(request, ...):
    """编排: Go token-init → Go repo-clone-credentials → 分流响应"""
    token = _credential_service.issue_token(tenant, wid, tid)
    creds = _credential_service.build_repo_clone_credentials(tenant, wid, tid, token.access_token)
    return _build_precheck_response(creds)  # 200/409/502

# relay_to_trae_start (应用服务)
def relay_to_trae_start(request, ...):
    """编排: Go token-init → 注入 runtime_env → 发给 sidecar"""
    token = _credential_service.issue_token(tenant, wid, tid)
    runtime_env = _build_runtime_env(token.access_token)
    _relay_http_request("POST", "/v1/start", json={"env": runtime_env})
    ...

# relay_to_trae_register (应用服务)
def relay_to_trae_register(request, ...):
    """编排: Go token-init → 注册 relay state"""
    token = _credential_service.issue_token(tenant, wid, tid)
    ...
```

## 7. 基础设施适配器

### 7.1 HttpTaskCredentialServiceAdapter

```python
# cloud/infrastructure/adapters/http_task_credential_service.py
class HttpTaskCredentialServiceAdapter(TaskCredentialServicePort):
    """Go taskCredentialService 的 HTTP 适配器"""

    def __init__(self, base_url: str):
        self._base_url = base_url.rstrip("/")

    def issue_token(self, tenant_id, workspace_id, task_id):
        """POST /v1/token/init → TokenInitResult"""
        with requests.Session() as s:
            s.trust_env = False
            resp = s.post(
                f"{self._base_url}/v1/token/init",
                json={"tenant_id": tenant_id, "workspace_id": workspace_id, "task_id": task_id},
                timeout=5,
            )
            resp.raise_for_status()
            data = resp.json()
            return TokenInitResult(
                access_token=data["access_token"],
                refresh_token=data["refresh_token"],
                expires_at=data["expires_at"],
            )

    def build_repo_clone_credentials(self, tenant_id, workspace_id, task_id, access_token):
        """POST .../repo-clone-credentials/ → RepoCloneCredentialsResult"""
        url = f"{self._base_url}/api/tenant/{tenant_id}/workspace/{workspace_id}/task/{task_id}/cloud/server-container-token/repo-clone-credentials/"
        with requests.Session() as s:
            s.trust_env = False
            resp = s.post(url, json={"access_token": access_token}, timeout=8)
            return RepoCloneCredentialsResult(
                status_code=resp.status_code,
                payload=resp.json() if resp.headers.get("content-type", "").startswith("application/json") else {},
            )
```

## 8. 领域模型变更总结

| 维度 | 旧状态 | 新状态 |
|------|--------|--------|
| **Token 写入方** | Django + Go (双写) | Go only (SSOT) |
| **Token 持久化** | `cloud_cloudserverconfig` + `container_tokens` | `container_tokens` only |
| **Django→Go 通信** | 直接 `requests.post` + `sqlite3.connect` hack | `TaskCredentialServicePort` 端口接口 |
| **依赖方向** | Django → Go 实现细节（HTTP URL + SQLite 路径硬编码） | Django → Port ← HTTP Adapter |
| **可测试性** | mock `requests.post` + mock `CloudServerConfig.objects` | mock `TaskCredentialServicePort` (单接口) |

## 9. 自检

- [x] 无新 ORM 导入（删除 SQLite hack 导入）
- [x] 端口接口 `TaskCredentialServicePort` 由领域层定义
- [x] 切换 Go 通信方式（HTTP → gRPC）只需新增适配器，Django 领域层零改动
- [x] 测试可注入 `InMemoryTaskCredentialServiceStub`
- [x] 应用服务不包含业务逻辑（token 生成/验证逻辑全在 Go）
- [x] 聚合根 `ContainerToken` 归属 Go BC，Django 不持有副本
