# NFR 澄清: GitLab OAuth Application 启动期自愈创建

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-06-22-gitlab-oauth-app-bootstrap-fix-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-22-gitlab-oauth-app-bootstrap-fix-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 可用性 | L2 | OAuth 授权链路首次部署即可用，无手动介入窗口 |
| 容错机制 | L2 | Application 缺失时自愈创建，不再以 exit 1 失败 |
| 可维护性 | L2 | 脚本幂等化，重复执行无副作用 |
| 性能 | L0 | 不适用 — 启动期一次性脚本，不影响运行时性能 |
| 可伸缩性 | L0 | 不适用 — 单容器内脚本，无扩展需求 |
| 安全性 | L0 | 不适用 — 不改变认证/授权/加密机制 |
| 数据一致性 | L0 | 不适用 — 配置同步，非事务性数据 |
| 可观测性 | L1 | 基础 — 日志区分 Created / Updated |
| 合规与隐私 | L0 | 不适用 |

## 逐增量 NFR 分析

### Increment 1: OAuth App Bootstrap (唯一增量)

#### NFR 类别: 可用性
- **等级**: L2 - 标准
- **量化目标**: GitLab 容器启动后 300 秒内 OAuth Application 可用（与现有等待超时对齐）
- **质量场景**: QS-01

#### NFR 类别: 容错机制
- **等级**: L2 - 标准
- **量化目标**: Application 缺失时自动创建（而非失败退出），创建成功率 100%（给定合法 YAML 配置）
- **质量场景**: QS-02

#### NFR 类别: 可维护性
- **等级**: L2 - 标准
- **量化目标**: 脚本幂等 — 重复执行不产生重复 Application、不修改已对齐的配置
- **质量场景**: QS-03

## 质量场景

### QS-01: 首次部署 OAuth 可用
| 要素 | 内容 |
|------|------|
| 类别 | 可用性 |
| 等级 | L2 |
| 刺激源 | GitLab 容器首次启动（空数据卷） |
| 刺激 | `sync_local_oauth_app_scopes.sh` 执行 |
| 制品 | Doorkeeper::Application (GitLab 内建) |
| 环境 | 容器启动后 300s 初始化窗口内 |
| 响应 | Application 被创建，scopes 与 YAML 对齐 |
| 响应度量 | `gitlab-rails runner "puts Doorkeeper::Application.find_by(uid: '<client_id>').scopes"` 输出 `read_repository api read_user` |

### QS-02: Application 缺失时自愈
| 要素 | 内容 |
|------|------|
| 类别 | 容错机制 |
| 等级 | L2 |
| 刺激源 | 数据卷清除后的 GitLab 容器 |
| 刺激 | 脚本发现 `Doorkeeper::Application.find_by(uid:)` 返回 nil |
| 制品 | `sync_local_oauth_app_scopes.sh` Ruby runner |
| 环境 | GitLab rails console 可用 |
| 响应 | 创建新 Application（uid/secret/name/scopes/redirect_uri/confidential），exit 0 |
| 响应度量 | 脚本 exit code = 0；`docker exec gitlab gitlab-rails runner "puts Doorkeeper::Application.count"` 包含新记录 |

### QS-03: 幂等执行
| 要素 | 内容 |
|------|------|
| 类别 | 可维护性 |
| 等级 | L2 |
| 刺激源 | 运维人员或 run.sh 多次调用同步脚本 |
| 刺激 | 连续两次执行 `sync_local_oauth_app_scopes.sh` |
| 制品 | Doorkeeper::Application |
| 环境 | Application 已存在且 scopes 已对齐 |
| 响应 | 第二次执行不创建重复记录，scopes 不变 |
| 响应度量 | Application count 不变；两次执行均 exit 0 |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| 容错 L2 — 自愈创建 | Doorkeeper::Application 生命周期从「手动创建」变为「自动 bootstrap」 | 仓储接口增加 `find_or_bootstrap` 语义（在 DDD 中建模为 Domain Service） |
| 可维护性 L2 — 幂等 | Application 的状态变更是幂等的（状态转移而非增量） | 实体方法使用 `sync_scopes()` 而非 `set_scopes()` 语义 |

## 权衡与边界

### 取舍
- 选择在启动脚本中内联 Ruby 代码（`docker exec ... gitlab-rails runner`），而非独立的 Rails migration —— 保持简单，避免侵入 GitLab 内部

### 明确不做什么
- 不修改 `conf/auth/git-oauth/providers/http-localhost-8012.yaml` 的结构
- 不引入新的持久化存储或数据库表
- 不改变 OAuth 授权流程本身的性能和安全性
- 不做 Application 的健康检查探针（由现有 `gitlab-rails runner "puts 'ready'"` 覆盖）

### 升级触发条件
- 当 GitLab 镜像版本升级导致 `Doorkeeper::Application` API 变更时 → 需回归测试
- 当新增第二个 GitLab provider 配置时 → 脚本需支持多 provider YAML 遍历

## 跳过声明

- **性能**: 跳过。启动期一次性脚本（~1s），不影响运行时请求路径。
- **可伸缩性**: 跳过。单容器内操作，无扩展需求。
- **安全性**: 跳过。不改变认证/授权机制，`client_secret` 存储方式不变。
- **数据一致性**: 跳过。配置同步（YAML → Doorkeeper DB），非事务性业务数据。
- **合规与隐私**: 跳过。无用户数据处理变更。
