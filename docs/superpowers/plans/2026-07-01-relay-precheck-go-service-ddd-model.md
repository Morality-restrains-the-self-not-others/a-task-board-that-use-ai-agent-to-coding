# DDD 领域模型: 容器令牌基础设施 Go 服务

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-07-01-relay-precheck-go-service-design.md`
> - 价值流: `docs/superpowers/plans/2026-07-01-relay-precheck-go-service-value-stream.md`
> - NFR 澄清: `docs/superpowers/plans/2026-07-01-relay-precheck-go-service-nfr-clarification.md`

## 限界上下文

```
┌──────────────────────────────────────────────────────────┐
│ 容器运行时上下文 (Container Runtime Context)                │
│ 职责: 容器令牌生命周期、仓库克隆凭证、任务详情查询           │
│ 进程: taskCredentialService (:8015)                       │
│ 数据库: db/container/tokens.sqlite3 (rw) + saas.sqlite3 (ro)│
└──────────────────────────────────────────────────────────┘
```

| 上下文 | 职责 | 模块 |
|--------|------|------|
| ContainerRuntime | 令牌签发/校验/交换/刷新、凭证构建、任务详情 | `taskCredentialService` |

## 实体与值对象

### 实体

| 实体 | 聚合根 | ID 类型 | 说明 |
|------|--------|---------|------|
| `ContainerToken` | ✅ | 雪花 ID (string) | 容器令牌聚合根，管理完整 token 生命周期 |
| `TokenAuditEvent` | ❌ | 雪花 ID (string) | 审计事件实体（关联 ContainerToken） |

### 值对象

| 值对象 | 不可变 | 说明 |
|--------|--------|------|
| `TaskScope` | ✅ | 租户/工作空间/任务三维标识 |
| `AccessToken` | ✅ | 容器访问令牌字符串 |
| `RefreshToken` | ✅ | 容器刷新令牌字符串 |
| `RepoCloneCredential` | ✅ | 单仓库克隆凭证（ephemeral token + provider） |
| `RepoCloneCredentialsResult` | ✅ | 凭证构建结果（成功/缺身份/token换发失败） |
| `TokenRefreshFailure` | ✅ | 单仓库 token 换发失败详情 |
| `GitIdentitySnapshot` | ✅ | 只读快照：仓库身份绑定 |
| `TaskRepoSnapshot` | ✅ | 只读快照：任务关联仓库 |
| `TaskSnapshot` | ✅ | 只读快照：任务元信息 |

## 聚合

```
ContainerToken (聚合根)
├── id: string (雪花)
├── taskID: string
├── companyID: string
├── workspaceID: string
├── containerAccessToken: AccessToken
├── containerAccessTokenExpiresAt: *time.Time
├── containerRefreshToken: RefreshToken
├── serverURL / businessAPIEndpoint / containerVscodeURL
└── authorizationID / imageID / instanceType / regionID / zoneID

TokenAuditEvent (独立实体，非聚合内)
├── id: string (雪花)
├── taskID: string
├── eventType / accessTokenSHA256 / errorCode / traceID / seq
└── 通过 taskID 关联 ContainerToken
```

**聚合边界规则:**
- `ContainerToken` 是外部唯一可操作的聚合根
- `TokenAuditEvent` 独立存储（append-only），通过 `taskID` 关联但不属于同一聚合
- 凭证构建结果 (`RepoCloneCredentialsResult`) 是查询产物，不持久化

## 端口接口（依赖反转）

```
domain/         ports/repositories.go     ports/clients.go
  ↑ 实现            ↑ 定义                 ↑ 定义
infrastructure/  sqlite_tokens.go         gitoauth_client.go
                 sqlite_business.go
```

| 端口 | 接口 | 适配器 | 说明 |
|------|------|--------|------|
| 仓储 | `ContainerTokenRepository` | `SQLiteTokenRepository` | tokens.sqlite3 读写 |
| 仓储 | `TokenAuditEventRepository` | `SQLiteAuditRepository` | 审计事件写入 |
| 仓储 | `BusinessDataRepository` | `SQLiteBusinessRepository` | saas.sqlite3 **只读** |
| 外部服务 | `GitoauthClient` | `GitoauthHTTPClient` | gitOauth HTTP 调用 |

**依赖反转验证:**
- 切换 DB: `tokens.sqlite3 → PostgreSQL` → 新增 `PostgresTokenRepository`，领域层零改动
- 切换 gitOauth 协议: `HTTP → gRPC` → 新增 `GitoauthGRPCClient`，领域层零改动
- 测试: 所有端口接口有 mock 实现，领域服务可独立测试

## 领域服务

| 服务 | 依赖端口 | 职责 |
|------|---------|------|
| `TokenService` | `ContainerTokenRepository`, `TokenAuditEventRepository` | IssueToken / ValidateToken / ExchangeToken / RefreshToken |
| `CredentialService` | `ContainerTokenRepository`, `BusinessDataRepository`, `GitoauthClient` | BuildRepoCloneCredentials |
| `TaskDetailService` | `BusinessDataRepository` | FetchTaskDetail |

## 领域事件

| 事件 | 触发条件 | 携带数据 |
|------|---------|---------|
| `TokenIssued` | `IssueToken()` 成功 | tokenID, taskID, scope |
| `TokenExchanged` | exchange-refresh 完成 | tokenID, taskID, scope |
| `TokenRefreshed` | refresh-access 完成 | tokenID, taskID, scope |
| `CredentialsFetchAttempted` | 凭证查询请求 | taskID, repoCount |
| `CredentialsFetchFailed` | 凭证不完整 | taskID, missingRepoURLs, errorCode |
| `TokenAuditRecorded` | 所有 token 生命周期事件 | TokenAuditEvent |

## 应用服务

| 服务 | 编排 | 说明 |
|------|------|------|
| `TokenService` | TokenService.IssueToken → TokenRepository.Save → AuditRepository.Save → emit TokenIssued | 注入 TokenRepository + AuditRepository |
| `CredentialService` | ValidateToken → BusinessRepository.FetchIdentities → GitoauthClient.FetchAccessToken → build result | 注入 TokenRepository + BusinessRepository + GitoauthClient |
| `TaskDetailService` | ValidateToken → BusinessRepository.FetchTaskSnapshot | 注入 BusinessRepository |

## NFR 影响

| NFR 决策 | 模型影响 |
|----------|---------|
| L2 数据一致性（快照读） | Go 只读 saas.sqlite3，不写，无需分布式事务 |
| L2 可用性（独立进程） | `ContainerRuntime` 独立限界上下文，独立 deploy |
| L2 容错（超时+回退） | `GitoauthClient` 端口封装超时，防腐层隔离 |

## 文件结构

```
taskCredentialService/
├── domain/
│   ├── entities.go      # ContainerToken, TaskScope, AccessToken...
│   └── events.go         # TokenIssued, CredentialsFetchFailed...
├── ports/
│   └── repositories.go   # ContainerTokenRepository, BusinessDataRepository, GitoauthClient
├── application/
│   └── services.go       # TokenService, CredentialService, TaskDetailService
├── infrastructure/
│   ├── composition.go    # DI 组装
│   ├── sqlite_tokens.go  # SQLiteTokenRepository + SQLiteAuditRepository
│   ├── sqlite_business.go# SQLiteBusinessRepository (saas.sqlite3 ro)
│   └── gitoauth_client.go# GitoauthHTTPClient
├── interfaces/
│   └── handlers.go       # HTTP handlers (Phase 1: token-init + 3 query endpoints)
├── cmd/
│   └── main.go           # Entry point + findMonorepoRoot
├── migrations/
│   └── 001_create_tables.sql  # DDL: container_tokens + token_audit_events
├── build.sh
└── go.mod
```

## 自检

- [x] 领域层无基础设施导入（entities.go / events.go 无 sql/HTTP 依赖）
- [x] 端口接口由领域层定义（ports/repositories.go）
- [x] 基础设施适配器实现端口接口（`var _ ports.ContainerTokenRepository = (*SQLiteTokenRepository)(nil)`）
- [x] 依赖反转成立：切换 DB 只需新增 adapter
- [x] 领域事件以过去式命名
- [x] 应用服务不包含业务逻辑（只做编排）
- [x] 分层正确：domain → ports ← infrastructure, application → domain
