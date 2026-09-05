# NFR 澄清: gitOauth HTTP Client Proxy Bypass Fix

> 输入:
> - 设计文档: `docs/specs/oauth-gitlab-token-exchange-fix-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-27-gitoauth-http-client-proxy-bypass-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 可用性 | L2 | OAuth token 交换不再受宿主机代理状态影响 |
| 容错机制 | L2 | HTTP 出站调用绕过代理，故障隔离 |
| 可观测性 | L1 | exchange 失败时区分代理/网络/GitLab 业务错误 |
| 性能 | L0 | 不适用 — 无性能变化 |
| 可伸缩性 | L0 | 不适用 — 无架构变化 |
| 安全性 | L0 | 不适用 — 无认证/授权变更 |
| 数据一致性 | L0 | 不适用 — 无状态变化 |
| 合规与隐私 | L0 | 不适用 |
| 可维护性 | L0 | 不适用 — 无 API 变更 |

## 逐增量 NFR 分析

### Increment 1: Proxy-Free gitOauth HTTP Client (唯一增量)

#### NFR 类别: 可用性
- **等级**: L2 - 标准
- **量化目标**: GitLab OAuth token 交换成功率不受 `ALL_PROXY`/`HTTP_PROXY`/`HTTPS_PROXY` 环境变量影响；代理不可用时交换仍可完成
- **质量场景**: QS-01

#### NFR 类别: 容错机制
- **等级**: L2 - 标准
- **量化目标**: gitOauth → GitLab token 端点的 HTTP 调用不经过任何外部代理，直连内网地址
- **质量场景**: QS-02

#### NFR 类别: 可观测性
- **等级**: L1 - 基础
- **量化目标**: exchange 失败日志中可区分 `ProxyError`、`ConnectionError`、GitLab HTTP 4xx/5xx 等不同失败模式
- **质量场景**: QS-03

## 质量场景

### QS-01: SOCKS 代理不可用时 OAuth 交换仍成功
| 要素 | 内容 |
|------|------|
| 类别 | 可用性 |
| 等级 | L2 |
| 刺激源 | GitLab OAuth 回调（用户授权后 GitLab 的重定向） |
| 刺激 | gitOauth 接收 callback code，发起 `POST {gitlab}/oauth/token` 交换 token |
| 制品 | `gitOauth/api/gitlab_tokens.py:exchange_authorization_code_for_tokens()` |
| 环境 | 宿主机 `ALL_PROXY=socks5://127.0.0.1:7890` 已设置但代理进程未运行 |
| 响应 | HTTP 请求直连 GitLab token 端点（127.0.0.1:8012 或 183.250.1.132:8012），不经过 SOCKS 代理 |
| 响应度量 | `requests.Session.trust_env == False`；代理不可用时 POST 不抛出 `ProxyError` 或 `SOCKSHTTPConnectionPool` 异常 |

### QS-02: HTTP_PROXY 设置时 GitHub OAuth 交换不受影响
| 要素 | 内容 |
|------|------|
| 类别 | 容错机制 |
| 等级 | L2 |
| 刺激源 | GitHub OAuth 回调 |
| 刺激 | gitOauth 接收 callback code，发起 `POST https://github.com/login/oauth/access_token` |
| 制品 | `gitOauth/api/github_tokens.py:exchange_code_for_token()` |
| 环境 | 宿主机 `HTTP_PROXY=http://proxy:8080` 已设置但不可达 |
| 响应 | HTTP 请求绕过代理直接访问 GitHub API |
| 响应度量 | 使用 `trust_env=False` 的 Session，不抛出 `ProxyError` |

### QS-03: 代理类错误可区分
| 要素 | 内容 |
|------|------|
| 类别 | 可观测性 |
| 等级 | L1 |
| 刺激源 | Token 交换失败 |
| 刺激 | `requests.RequestException` 被捕获 |
| 制品 | `exchange_authorization_code_for_tokens()` except 块 |
| 环境 | 任意 |
| 响应 | 日志中记录异常类型（如 `ProxyError`、`ConnectionError`、`HTTPError`）和关键上下文（status_code、response body 摘要） |
| 响应度量 | 从日志中可直接判断失败原因是代理、网络还是 GitLab 业务错误，无需查代码或复现 |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| trust_env=False（可用性 L2 + 容错 L2） | 无领域模型变化。这是基础设施层（HTTP transport）的配置修复 | 无需修改领域层；只需在基础设施层创建 `http_client.py` 模块 |
| exchange 失败日志增强（可观测性 L1） | 无领域模型变化 | 无需修改领域层；在 `gitlab_tokens.py` 的 except 块增加分类日志 |

**本次修复不涉及领域模型变更。** 纯基础设施层修复 + 可观测性增强，DDD 步骤可跳过（或标注"不适用"）。

## 权衡与边界

### 取舍
- 选择 `trust_env=False` 全局绕过代理，而非白名单模式（信任特定代理）。gitOauth 作为内部服务，所有出站 HTTP 调用（GitHub API / GitLab / 内部 bind API）均应直连，不存在需要通过代理访问的外部资源。

### 明确不做什么
- 不实现 per-request 的代理开关（不需要部分请求走代理的场景）
- 不修改 task2app 主站的 HTTPClient（已有 `trust_env=False`）
- 不添加代理健康检查或自动恢复逻辑
- 不做代理配置的 UI 化

### 升级触发条件
- 如果未来 gitOauth 需要同时访问内网 GitLab（直连）和外网 GitHub（需代理）→ 需引入 per-destination 的代理策略，而非全局 `trust_env=False`

## 跳过声明

- **性能**: 跳过。`trust_env=False` 不会改变 HTTP 请求的性能特征（仅跳过了代理的 `getproxies()` 调用）
- **可伸缩性**: 跳过。单实例 serve，无扩展场景
- **安全性**: 跳过。不改变认证/授权/加密逻辑
- **数据一致性**: 跳过。不改变数据读写路径
- **合规与隐私**: 跳过。不涉及用户数据处理
- **可维护性**: 跳过。无 API 变更或配置格式变更
