# NFR 澄清: GitLab OAuth Scope 合法化与启动期 Fail-Fast

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-05-26-gitoauth-gitlab-provider-config-compat-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-05-26-gitlab-oauth-scope-failfast-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L2 | OAuth start 接口服务端 P95 <= 500ms（正常负载） |
| 安全性 | L3 | provider/scope 配置错误启动即阻断，且错误信息可审计定位 |
| 数据一致性 | L2 | 配置读取与校验结果在单次启动内强一致，错误配置不进入运行态 |
| 可观测性 | L2 | 启动失败日志必须含 provider/service_provider/非法 token |
| 可维护性 | L2 | 保持 GitHub 兼容，新增校验不破坏现有配置结构（list/dict） |

## 逐增量 NFR 分析

### Increment 1: GitLab Scope Thin Slice (E2E)

#### NFR 类别: 性能
- **等级**: L2 - 标准
- **量化目标**: `GET /api/accounts/gitlab/app/start/` 服务端 P95 <= 500ms，P99 <= 1s（不含网络）
- **质量场景**: QS-01

#### NFR 类别: 数据一致性
- **等级**: L2 - 标准
- **量化目标**: GitLab provider 的 `scope` 必须使用同一配置值（`read_repository api read_user`），授权 URL 不允许出现 `repo`/`read:user`
- **质量场景**: QS-02

### Increment 2: Startup Fail-Fast Guard (Core)

#### NFR 类别: 安全性
- **等级**: L3 - 增强
- **量化目标**: `provider=gitlab` 且 scope 含 GitHub 风格 token 时，服务启动 100% 失败并给出明确错误
- **质量场景**: QS-03

#### NFR 类别: 可观测性
- **等级**: L2 - 标准
- **量化目标**: 错误信息必须包含 `provider`、`service_provider`、非法 token；排障定位时间 <= 10 分钟
- **质量场景**: QS-04

### Increment 3: Regression Guard & Verification

#### NFR 类别: 可维护性
- **等级**: L2 - 标准
- **量化目标**: 回归测试覆盖合法/非法两类 scope，且 GitHub OAuth 链路测试无回归
- **质量场景**: QS-05

## 质量场景

### QS-01: GitLab OAuth Start 响应时间基线
| 要素 | 内容 |
|------|------|
| 类别 | 性能 |
| 等级 | L2 |
| 刺激源 | 前端用户 |
| 刺激 | 点击项目详情页仓库行的 OAuth 授权按钮 |
| 制品 | `GET /api/accounts/gitlab/app/start/` |
| 环境 | 正常负载（并发 20，日均 2x） |
| 响应 | 返回 200，包含 `authorize_url` |
| 响应度量 | 服务端处理时间 P95 <= 500ms，P99 <= 1s；APM 侧测量 |

### QS-02: Scope 合法性端到端约束
| 要素 | 内容 |
|------|------|
| 类别 | 数据一致性 |
| 等级 | L2 |
| 刺激源 | 配置加载流程 |
| 刺激 | 读取 `provider=gitlab` 配置并发起 OAuth start |
| 制品 | provider 归一化 + GitLab authorize URL 构造 |
| 环境 | 正常运行 |
| 响应 | authorize URL 仅携带 GitLab 合法 scope token |
| 响应度量 | 自动化测试断言 URL scope 为 `read_repository api read_user`，不含 `repo`/`read:user` |

### QS-03: 非法 Scope 启动阻断
| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L3 |
| 刺激源 | 配置变更（误填） |
| 刺激 | `provider=gitlab` 的 `scope` 写为 `repo read:user` |
| 制品 | task2app 与 gitOauth 配置归一化入口 |
| 环境 | 服务启动阶段 |
| 响应 | 抛配置异常并终止启动 |
| 响应度量 | 启动失败率 100%，错误信息包含 provider/service_provider/非法 token |

### QS-04: 启动失败可定位性
| 要素 | 内容 |
|------|------|
| 类别 | 可观测性 |
| 等级 | L2 |
| 刺激源 | 运维/开发排障 |
| 刺激 | 服务启动失败后查看错误日志 |
| 制品 | 配置校验异常与日志 |
| 环境 | 本地与测试环境 |
| 响应 | 可直接定位到错误 provider 条目与 scope token |
| 响应度量 | 人工排障在 10 分钟内完成定位；无需额外代码埋点 |

### QS-05: 回归保护与兼容性
| 要素 | 内容 |
|------|------|
| 类别 | 可维护性 |
| 等级 | L2 |
| 刺激源 | 后续代码/配置变更 |
| 刺激 | 运行最小回归测试集 |
| 制品 | OAuth start 路由与 scope 校验逻辑 |
| 环境 | CI 或本地测试 |
| 响应 | GitLab 合法/非法 scope 场景均符合预期，GitHub 相关场景不回归 |
| 响应度量 | 目标测试文件全绿：`tests/test_github_app_start_redirect_uri.py`、`tests/test_git_oauth_scope_validation.py` |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| 启动期 fail-fast（安全 L3） | provider 配置成为显式领域约束，而非运行时容错 | 在配置相关 Value Object/Factory 中固化 `gitlab scope` 不变量 |
| 一致性 L2（单次启动内强一致） | 配置归一化与路由目标不可出现“部分合法”状态 | DDD 中将“配置有效性校验”放在聚合创建前，失败即拒绝创建 |
| 可观测性 L2（错误可定位） | 配置异常需携带上下文标识 | 领域事件/异常模型增加 `provider_key`、`service_provider`、`invalid_tokens` |
| 可维护性 L2（双结构兼容） | 同一语义输入存在 list/dict 两种外部形态 | DDD 中使用统一 Normalizer/Anti-Corruption 层，屏蔽上游结构差异 |

## 权衡与边界

### 取舍
- 选择“启动即失败”而非“运行时自动纠错”，优先保证配置安全与可预期性。
- 选择 L2 性能基线，不进行 L3/L4 级别的缓存与高并发专项优化。

### 明确不做什么
- 不在本增量实现跨 provider 的完整 scope 白名单中心化管理（仅覆盖 GitLab 与 GitHub 风格冲突项）。
- 不引入新的外部配置服务或动态热更新机制（保持现有静态配置加载模型）。
- 不为未来未落地的 provider（如 bitbucket）提前定义严格校验规则。

### 升级触发条件
- 当新增第三种及以上 Git provider 时，升级可维护性到 L3：抽象统一 scope schema 与 provider 级策略注册。
- 当出现多环境频繁配置漂移（每月 >= 2 次）时，升级可观测性到 L3：补充配置审计事件与告警。
- 当 OAuth start 流量显著增长（峰值并发 >= 200）时，升级性能到 L3：引入路由缓存与压测门禁。

## 跳过声明
- 可伸缩性: 跳过。当前变更是配置约束与启动校验，不引入新的在线计算热点或存储增长路径。
- 可用性: 跳过专项提升。当前目标是错误配置阻断而非提升 SLA。
- 容错机制: 跳过。此增量采用 fail-fast，不引入重试/断路器策略。
- 合规与隐私: 跳过。未新增个人数据处理路径与跨境数据流。

## 自检 (Hard Gate)
- [x] 每个相关 NFR 类别都有明确的支撑等级（L0-L4）
- [x] 每个 L1-L4 的 NFR 类别至少有一个量化目标
- [x] 每个 L2-L4 的 NFR 类别至少有一个质量场景（QS）
- [x] 每个质量场景的响应度量可验证（具体数字和测量方式）
- [x] 影响领域模型的 NFR 决策已标注（含具体 DDD 动作）
- [x] 权衡和边界已明确（不做什么 + 升级触发条件）
- [x] 跳过的 NFR 类别有理由说明
- [x] 文档位置正确：`docs/superpowers/plans/2026-05-26-gitlab-oauth-scope-failfast-nfr-clarification.md`

