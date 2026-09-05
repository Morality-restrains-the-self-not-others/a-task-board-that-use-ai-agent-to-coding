# NFR 澄清: OIDC SSL Protocol Fix + Playwright E2E

> 输入:
> - 设计文档: `docs/specs/oidc-ssl-playwright-e2e-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-24-oidc-ssl-playwright-e2e-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 安全性 | L3 | OIDC SSO 认证通道 100% 可用，无 SSL 协议错误 |
| 可维护性 | L2 | Playwright E2E 覆盖完整 SSO 登录流程，CI 可自动回归 |
| 可用性 | L1 | OIDC SSO 从不可用恢复到基础可用 |
| 性能 | L0 | 不适用 — 修复不改变请求路径，无性能目标 |
| 可伸缩性 | L0 | 不适用 — 单实例 GitLab 部署 |
| 数据一致性 | L0 | 不适用 — 无数据模型变更 |
| 容错机制 | L0 | 不适用 — 修复是配置注入，无运行时容错逻辑 |
| 可观测性 | L2 | Playwright 诊断测试提供 OIDC discovery 可达性检查 |
| 合规与隐私 | L0 | 不适用 — 无合规需求 |

## 逐增量 NFR 分析

### Increment 1: Core Fix + Diagnostic Verification

#### NFR 类别: 安全性
- **等级**: L3 - 增强
- **量化目标**: GitLab OmniAuth OIDC 认证成功率 100%（修复前为 0%），零 SSL 协议错误
- **质量场景**: QS-01

#### NFR 类别: 可维护性
- **等级**: L2 - 标准
- **量化目标**: Playwright 诊断测试覆盖 OIDC discovery 可达性验证，CI 执行 ≤ 30s
- **质量场景**: QS-03

### Increment 2: E2E SSO Login Flow

#### NFR 类别: 可维护性
- **等级**: L2 - 标准
- **量化目标**: Playwright E2E 测试覆盖完整 SSO 流程，回归测试 ≤ 60s
- **质量场景**: QS-04

### Increment 3: run.sh Integration (Persistence)

#### NFR 类别: 可用性
- **等级**: L1 - 基础
- **量化目标**: GitLab 容器重建后，fix 在 5 分钟内自动恢复（含 reconfigure 耗时）
- **质量场景**: QS-02

## 质量场景

### QS-01: OIDC SSO 认证成功
| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L3 |
| 刺激源 | 用户浏览器 |
| 刺激 | 在 GitLab 登录页点击 taskAuth SSO 按钮 |
| 制品 | GitLab OmniAuth OIDC → taskAuth OIDC Provider |
| 环境 | 正常 |
| 响应 | OIDC discovery 使用 HTTP 协议完成，用户被重定向到 taskAuth 授权页，回调成功 |
| 响应度量 | 页面不含 "record layer failure" / "Could not authenticate" 错误文本；callback URL 可达 |

### QS-02: 容器重建后修复自愈
| 要素 | 内容 |
|------|------|
| 类别 | 可用性 |
| 等级 | L1 |
| 刺激源 | 运维操作（docker compose down && up） |
| 刺激 | GitLab 容器重建，initializer 文件丢失 |
| 制品 | gitService/run.sh → fix_oidc_ssl.sh |
| 环境 | GitLab 启动后 |
| 响应 | fix_oidc_ssl.sh 自动检测并注入 initializer + reconfigure |
| 响应度量 | 容器启动后 5 分钟内 `SWD.url_builder` 恢复为 `URI::HTTP` |

### QS-03: Playwright 诊断可检测 OIDC 配置错误
| 要素 | 内容 |
|------|------|
| 类别 | 可观测性 |
| 等级 | L2 |
| 刺激源 | CI pipeline 或开发者本地 |
| 刺激 | 执行 `npx playwright test oidc-ssl-diagnostic` |
| 制品 | Playwright 诊断测试 |
| 环境 | 正常 |
| 响应 | 测试验证 OIDC discovery 端点返回 200 + 合法 JSON |
| 响应度量 | 测试通过/失败状态；失败时输出具体断言行号和期望值 |

### QS-04: E2E 回归覆盖完整 SSO 流程
| 要素 | 内容 |
|------|------|
| 类别 | 可维护性 |
| 等级 | L2 |
| 刺激源 | CI pipeline (每次部署/PR) |
| 刺激 | 执行 `npx playwright test oidc-sso-login` |
| 制品 | Playwright E2E 测试 (login → 代码仓库 → taskAuth SSO → dashboard) |
| 环境 | 正常 |
| 响应 | 完整 SSO 流程无 SSL 错误，最终页面为 GitLab dashboard |
| 响应度量 | 所有断言通过；测试耗时 ≤ 60s |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| 安全性 L3 (OIDC SSO) | 认证上下文需建模 OIDC 协议修复为领域概念 | 在 Infra Context 中引入 `OIDCProtocolFix` 值对象，封装 SWD.url_builder 状态 |
| 可维护性 L2 (E2E 测试) | 测试制品需建模为领域概念 | `PlaywrightE2ETest` 作为验证实体，关联到 `GitLabApplication` 聚合根 |
| 可用性 L1 (自愈) | fix_oidc_ssl.sh 幂等性需在模型中表达 | `OIDCFixApplication` 领域服务提供 `ensure_applied()` 幂等方法 |

> **注意**: 本修复属于基础设施配置变更，领域模型影响较轻。DDD 步骤应保持模型精简，不要为配置修复过度建模。

## 权衡与边界

### 取舍
- 选择在 run.sh 中同步执行 fix（阻塞 GitLab 启动），而非异步后台任务 —— 确保服务就绪时 OIDC 已可用
- 接受 gitlab-ctl reconfigure 耗时 1-3 分钟作为容器重建成本 —— 不实现运行时热加载

### 明确不做什么
- 不实现 GitLab Rails initializer 的热加载（无需 reconfigure）
- 不做 Playwright 测试的跨浏览器矩阵（仅 Chromium headless）
- 不处理 Gateway TLS 后的 HTTPS issuer 场景（future work）
- 不修改 taskAuth OIDC Provider 源码

### 升级触发条件
- 当 GitLab 升级到修复了此 bug 的版本时 → 移除 initializer 注入
- 当 taskGateway TLS 启用后 issuer 变为 https 时 → 撤销此修复

## 跳过声明
- **性能**: 跳过。修复不改变 OIDC 请求路径的延迟特征。
- **可伸缩性**: 跳过。单实例 GitLab 部署，无水平扩展需求。
- **数据一致性**: 跳过。纯配置变更，无数据库写入。
- **容错机制**: 跳过。修复是静态配置注入，无运行时重试/断路器需求。
- **合规与隐私**: 跳过。不涉及数据本地化或行业认证。
