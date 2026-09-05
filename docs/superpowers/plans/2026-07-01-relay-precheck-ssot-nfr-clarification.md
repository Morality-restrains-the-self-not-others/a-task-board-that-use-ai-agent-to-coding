# NFR 澄清: relay 直启预检 Token SSOT 消除双写

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-07-01-relay-precheck-ssot-in-process-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-07-01-relay-precheck-ssot-value-stream.md`
>
> 输出使用者: `/6-ddd-领域设计驱动`, `/7-plans-实施计划`, `/8-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L2 | Django→Go HTTP 调用 P95 ≤ 200ms，预检全链路 P95 ≤ 500ms |
| 可用性 | L2 | Go 不可达时 Django 返回明确 502，不静默失败；Go 是 runAll 一部分，启动顺序保证 |
| 容错机制 | L2 | HTTP 超时 5s (token-init) / 8s (repo-clone-credentials)，连接失败明确报错 |
| 数据一致性 | L2 | Go `container_tokens` 为 SSOT，无跨服务数据同步窗口 |
| 可观测性 | L2 | Go 调用失败记录 ERROR 日志（含 status_code + response body 摘要） |
| 安全性 | L2 | 无新增端点，权限模型不变；Go `/v1/token/init` 仅 localhost 绑定（已有风险，非本次引入） |

**跳过声明：**
- 可伸缩性: 不适用。Django→Go 均为 localhost 进程间通信，无水平扩展需求。
- 合规与隐私: 不适用。无新增数据存储或传输路径。
- 可维护性: 已隐含在代码清理（删除 SQLite hack）中，不单独定级。

## 逐增量 NFR 分析

### Increment 1: token-init 改调 Go

#### NFR 类别: 性能
- **等级**: L2 - 标准
- **量化目标**: Django→Go `/v1/token/init` 往返 P95 ≤ 200ms（localhost HTTP，Go 内 IssueToken 含一次 SQLite 查询）
- **对比基线**: 旧路径（Django ORM 查 CloudServerConfig + SQLite hack 写 Go DB）约 50-100ms；新路径增加一次 HTTP 往返，但消除跨进程 DB 直写，整体延迟仍在可接受范围

#### NFR 类别: 可用性
- **等级**: L2 - 标准
- **量化目标**: Go 不可达时 100% 返回明确错误响应（502 + message），不静默失败
- **依赖**: Go taskCredentialService 由 runAll 管理，Django 启动前 Go 已就绪

#### NFR 类别: 容错机制
- **等级**: L2 - 标准
- **量化目标**: HTTP 超时 5s，超时后返回 502；不重试（token-init 可幂等重试由前端控制）

### Increment 2: precheck + start + register 改调 Go

#### NFR 类别: 性能
- **等级**: L2 - 标准
- **量化目标**: precheck 全链路（token-init + repo-clone-credentials）P95 ≤ 500ms；start 全链路 P95 ≤ 300ms（token-init + 异步 dispatch）
- **尾部延迟考量**: precheck 需两次串行 Go 调用（token-init → repo-clone-credentials），P99 可能因两次调用的尾部延迟叠加而显著放大。缓解：token-init 内 Go IssueToken 复用已有有效 token（FindByTaskID 命中时仅一次 SELECT），第二次调用延迟稳定。

#### NFR 类别: 可用性
- **等级**: L2 - 标准
- **量化目标**: 同 Increment 1

#### NFR 类别: 容错机制
- **等级**: L2 - 标准
- **量化目标**: repo-clone-credentials 超时 8s（Go 内需调 gitOauth 换 ephemeral token，可能较慢）

### Increment 3: 删除双写 hack 代码

无独立 NFR 要求（纯代码清理）。

### Increment 4: 测试更新

无独立 NFR 要求（测试覆盖属于质量保证，不在此 NFR 框架内）。

## 质量场景

### QS-01: token-init Go 调用正常响应
| 要素 | 内容 |
|------|------|
| 类别 | 性能 |
| 等级 | L2 |
| 刺激源 | 前端 relayToTrae 面板 |
| 刺激 | 用户点击「启动」→ 前端 POST token-init |
| 制品 | Django `relay_to_trae_token_init()` → Go `/v1/token/init` |
| 环境 | 正常负载（单用户操作，Go 服务空闲） |
| 响应 | 200 + `{status:"ok", token_initialized:true, env_preview:{...}}` |
| 响应度量 | 服务端处理时间 P95 ≤ 200ms（含 Go HTTP 往返），由 Django 日志记录 |

### QS-02: precheck Go 两阶段调用正常
| 要素 | 内容 |
|------|------|
| 类别 | 性能 |
| 等级 | L2 |
| 刺激源 | 前端 relayToTrae 面板 |
| 刺激 | token-init 成功后前端 POST precheck |
| 制品 | Django `relay_to_trae_repo_credentials_precheck()` → Go `/v1/token/init` + `repo-clone-credentials` |
| 环境 | 正常负载；OAuth 已授权 + 账号已保存 |
| 响应 | 200 + `{status:"ok", repo_count: N}` |
| 响应度量 | 服务端处理时间 P95 ≤ 500ms（含两次 Go HTTP 往返 + gitOauth 调用） |

### QS-03: Go 不可达时优雅降级
| 要素 | 内容 |
|------|------|
| 类别 | 可用性 + 容错机制 |
| 等级 | L2 |
| 刺激源 | Django relayToTrae 请求 |
| 刺激 | Go taskCredentialService 端口不可达（进程未启动/已崩溃） |
| 制品 | Django `_credential_service_post()` |
| 环境 | 故障状态 |
| 响应 | 返回 502 + `{status:"error", message:"仓库凭证预检请求失败: ..."}` |
| 响应度量 | 100% 请求在超时（5s/8s）内返回错误响应；Django 日志记录 ERROR 级别 + 异常详情 |

### QS-04: Token SSOT 一致性
| 要素 | 内容 |
|------|------|
| 类别 | 数据一致性 |
| 等级 | L2 |
| 刺激源 | Django token-init → precheck 两次请求 |
| 刺激 | 先后调用 Go `/v1/token/init`（复用 token）和 `repo-clone-credentials`（验证 token） |
| 制品 | Go `TokenService.IssueToken` + `CredentialService.BuildRepoCloneCredentials` |
| 环境 | 正常 |
| 响应 | 同一 scope 第二次 IssueToken 返回同一有效 token；ValidateToken 通过 |
| 响应度量 | Go `FindByTaskID` 复用逻辑：未过期 token 100% 复用，无重复签发 |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| Go 为 Token SSOT (L2 一致性) | Django 域不再持有 `ContainerToken` 实体，仅作为 consumer | DDD 中 `ContainerToken` 聚合根归属 Go `taskCredentialService` BC；Django 侧 `CloudServerConfig.container_access_token` 标记为 legacy |
| HTTP 超时 + 无重试 (L2 容错) | Django→Go 调用需防腐层封装 | `_credential_service_post()` 统一处理 timeout / trust_env / error 转换 |
| Go 不可达需明确报错 (L2 可用性) | 错误分类需要区分"Go 不可达"vs"Go 返回业务错误" | `_relay_error_response()` 处理 `requests.RequestException` → 502；Go HTTP 4xx/5xx → 透传 |

## 权衡与边界

### 取舍
- 选择 HTTP 调用 Go（增加 ~5-10ms 网络延迟）以换取架构清洁（消除跨语言 SQLite hack）
- token-init 不重试——由前端控制重试节奏，避免服务端重复签发

### 明确不做什么
- 不在 Django 内缓存 Go 返回的 token（无状态，每次调 Go）
- 不引入 internal secret 认证（Go `/v1/token/init` 加固 → 独立 Phase）
- 不在 Django→Go 间使用连接池（localhost 短连接开销可忽略）

### 升级触发条件
- 当 Go 调用 P95 > 500ms → 需 profiling Go IssueToken / BuildRepoCloneCredentials 性能
- 当 Go 不可达频率 > 1次/天 → 需检查 Go 进程稳定性或添加健康检查 + 自动重启
- 当需要跨网络部署 Django 和 Go → 需引入连接池 + internal mTLS
