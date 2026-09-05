# NFR 澄清: 容器令牌基础设施 Go 服务迁移

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-07-01-relay-precheck-go-service-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-07-01-relay-precheck-go-service-value-stream.md`
>
> 输出使用者: `/6-ddd-领域设计驱动`, `/7-plans-实施计划`, `/8-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L2 | 预检响应从 ~8s → <1s (P95)；token-init <200ms |
| 可用性 | L2 | Go 服务 /health 探针，runAll depends_on 确保启动顺序 |
| 安全性 | L2 | 容器 access_token 鉴权（同现有契约），Go 侧只读 saas.sqlite3 |
| 数据一致性 | L2 | SQLite WAL 模式，Go 写 tokens.sqlite3、只读 saas.sqlite3 |
| 容错机制 | L2 | 超时 5s (gitOauth) / 8s (Django→Go 调用)，Django 保留回退端点 |
| 可观测性 | L2 | OTel tracing + JSON 结构化日志 + /health endpoint |
| 可伸缩性 | L1 | 单实例本地部署，无水平扩展需求 |
| 可维护性 | L2 | 遵循现有 Go 服务项目结构，`db/registry.yaml` 注册 DB |
| 合规与隐私 | L0 | 不适用 — 内部基础设施，无用户数据新增 |

## 逐增量 NFR 分析

### Increment 1: Go 服务骨架 + token init

#### NFR 类别: 性能
- **等级**: L2 - 标准
- **量化目标**: `POST /v1/token/init` 服务端 P95 < 200ms
- **背景**: Django 当前 `_issue_relay_access_token()` 是同步 DB 写入，迁移到 Go 后无额外网络跳

#### NFR 类别: 可用性
- **等级**: L2 - 标准
- **量化目标**: `/health` 在 30s 内就绪，runAll 重试 10 次
- **降级策略**: Go 服务未就绪时，runAll 阻塞后续服务启动

### Increment 2: 组 A 查询端点迁移（死锁消除）

#### NFR 类别: 性能
- **等级**: L2 - 标准
- **量化目标**: `repo-clone-credentials` P95 < 1s（含 gitOauth 调用）；`task-detail` P95 < 200ms
- **当前基线**: 死锁超时 ~8s → 修复后 <1s
- **长尾延迟**: gitOauth 调用是瓶颈，超时 5s。Go 服务可并行调多个 repo 的 gitOauth

#### NFR 类别: 数据一致性
- **等级**: L2 - 标准
- **隔离**: Go 服务只读 `saas.sqlite3`（WAL 模式，读写并发安全），不写 Django 业务表
- **一致性保证**: 凭证查询是快照读——Django 更新 TaskRepoIdentity 后 Go 立即可见（同一 SQLite 文件）

#### NFR 类别: 容错机制
- **等级**: L2 - 标准
- **超时**: gitOauth 调用 5s，Django→Go 预检调用 8s
- **重试**: 无自动重试（gitOauth 偶发失败在前端展示 token 换发失败详情）
- **回退**: Django 保留原端点代码（feature flag 可切换回 Django 实现）

### Increment 3: 组 B token 端点迁移

#### NFR 类别: 数据一致性
- **等级**: L2 - 标准
- **写冲突**: Go relay 换票是串行操作（先 exchange-refresh 再 refresh-access，中间 sleep 150ms），Go 服务单写者，无并发写冲突
- **幂等**: exchange-refresh 基于 `container_access_token` 查重，重复调用不会重复签发

### Increment 4-6

#### NFR 类别: 可观测性
- **等级**: L2 - 标准
- **日志**: JSON 结构化日志（复用 tracelog 包），含 trace_id / service / level / msg
- **追踪**: OTel gRPC export 到 `${INFRA_HOST}:4317`
- **健康检查**: `GET /health` 返回 200 + DB 连通性状态

---

## 质量场景

### QS-01: 预检响应时间（死锁消除）
| 要素 | 内容 |
|------|------|
| 类别 | 性能 |
| 等级 | L2 |
| 刺激源 | 用户点击「启动」按钮 |
| 刺激 | 触发 relay-to-trae/repo-credentials-precheck |
| 制品 | Django→Go POST .../repo-clone-credentials/ |
| 环境 | runAll 单线程模式（`--noreload`） |
| 响应 | 200 OK + repo_count（凭证完整） |
| 响应度量 | 端到端 P95 < 1.5s（含 Django→Go 网络 + Go 处理 + gitOauth 调用），服务端 P95 < 1s |

### QS-02: Token 签发延迟
| 要素 | 内容 |
|------|------|
| 类别 | 性能 |
| 等级 | L2 |
| 刺激源 | 用户点击「启动」→ 前端调 token-init |
| 刺激 | Django POST relay-to-trae/token-init → Go POST /v1/token/init |
| 制品 | Go token init handler |
| 环境 | 正常负载 |
| 响应 | 200 OK + access_token |
| 响应度量 | Go 服务端 P95 < 200ms |

### QS-03: Go 服务启动就绪
| 要素 | 内容 |
|------|------|
| 类别 | 可用性 |
| 等级 | L2 |
| 刺激源 | runAll 启动 task-credential-service |
| 刺激 | Go 进程启动 → 打开 tokens.sqlite3 + saas.sqlite3 → 监听 :8015 |
| 制品 | Go 服务 /health endpoint |
| 环境 | 冷启动（首次创建 tokens.sqlite3 + DDL migration） |
| 响应 | /health 返回 200 |
| 响应度量 | 30s 内就绪，runAll 重试 10 次（间隔 2s 起步，最大 16s） |

### QS-04: gitOauth 不可达时降级
| 要素 | 内容 |
|------|------|
| 类别 | 容错机制 |
| 等级 | L2 |
| 刺激源 | onlineServiceJS 请求 repo-clone-credentials |
| 刺激 | gitOauth 服务 (:8002) 不可达 |
| 制品 | Go repo-clone-credentials handler |
| 环境 | 降级 |
| 响应 | 502 + error_code: REPO_CLONE_TOKEN_REFRESH_FAILED + token_refresh_failures 详情 |
| 响应度量 | 5s 超时后返回错误，不阻塞其他请求 |

### QS-05: SQLite 并发读写安全
| 要素 | 内容 |
|------|------|
| 类别 | 数据一致性 |
| 等级 | L2 |
| 刺激源 | Django 写入 saas.sqlite3（TaskRepoIdentity 更新）的同时 Go 服务读取 |
| 刺激 | 并发读写同一 SQLite 文件 |
| 制品 | Go 服务只读连接 (mode=ro) + Django 读写连接 |
| 环境 | WAL 模式 |
| 响应 | Go 读取一致快照，不阻塞 Django 写入 |
| 响应度量 | 无 SQLITE_BUSY 错误（WAL 模式下读写不互斥） |

---

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| 数据一致性 L2 (最终一致，快照读) | Go 服务读取 Django 业务数据是快照读，不需要分布式事务 | Go `ContainerTokenRepository` 与 Django ORM 解耦，通过 SQLite 文件共享数据（不在同一个 aggregate） |
| 可用性 L2 (独立进程 + health check) | Go 服务是独立 deploy unit，有独立生命周期 | DDD 建模时 `ContainerRuntime` 作为独立限界上下文 |
| 容错 L2 (超时 + 回退) | 领域服务调用 gitOauth 需要防腐层 | Go `GitoauthClient` 实现超时+错误分类，基础设施层封装 |
| 可观测性 L2 (OTel + JSON 日志) | 所有 Go handler 需注入 trace context | Go `tracelog` 包复用现有模式，不引入新领域概念 |

---

## 权衡与边界

### 取舍
- 选择独立 SQLite 文件 (tokens.sqlite3) 而非共享 saas.sqlite3 — 换取清晰的表 ownership 边界，代价是 Go 服务需管理两个 DB 连接
- 选择 Phase 1 只迁查询端点 — 先解决死锁（核心痛点），写操作端点后续迁移，降低风险
- 选择 HTTP (Django→Go) 而非进程内调用 — 换取进程隔离和无死锁保障，代价是 ~1ms 网络延迟

### 明确不做什么
- 不在 V1 引入 gRPC 替代 HTTP（保持与现有 Go 服务技术栈一致）
- 不在 V1 添加断路器/重试（gitOauth 调用失败直接返回错误给调用方处理）
- 不做 Go 服务水平扩展（单实例本地部署，与 monorepo 耦合）
- 不做跨机部署（Go 服务与 Django 同机，依赖本地 SQLite 文件）

### 升级触发条件
- 当 Go 服务需要独立部署到其他机器时 → DB 从本地 SQLite 升级为网络数据库 → 需要重新评估一致性模型
- 当单实例 Go 服务成为性能瓶颈时 → 可伸缩性从 L1→L3
- 当 relay 直启成为付费功能时 → 可用性从 L2→L3（需要冗余实例）

---

## 跳过声明
- **可伸缩性**: L1 — 单实例本地部署，与 monorepo 同生命周期，无水平扩展需求
- **合规与隐私**: L0 — 内部基础设施迁移，不涉及用户数据处理变更
- **安全性**: L2 — 沿用现有容器 access_token 鉴权机制，无新增安全需求（Go 服务位于 localhost，不对外暴露）
