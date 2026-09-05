# NFR 澄清: 项目详情页仓库行 OAuth 授权（分支预览场景）

> 输入:
> - 设计文档: `/Users/task2app/.cursor/plans/project-detail-oauth-button_92646d07.plan.md`
> - 价值流文档: `docs/superpowers/plans/2026-05-26-project-detail-oauth-button-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L2 | OAuth start 接口服务端处理时间 P95 <= 500ms（不含网络） |
| 数据一致性 | L2 | repo_url 必须稳定命中对应 provider 配置，错误路由率 = 0（在回归测试集内） |
| 安全性 | L2 | start 接口只返回 `authorize_url`，不泄露 token 明文与 client_secret |
| 可观测性 | L2 | start 失败 100% 返回结构化 `detail`（可区分配置缺失/签发失败） |
| 可维护性 | L2 | 配置模型变更后保持向后兼容：数组/对象两种 `gitOauth` 结构均可解析 |
| 可用性 | L1 | 沿用现有单实例可用性，不新增高可用专项设计 |

## 逐增量 NFR 分析

### Increment 1: 项目详情页 OAuth 薄切片

#### NFR 类别: 性能
- **等级**: L2 - 标准
- **量化目标**:
  - `GET /api/accounts/{provider}/app/start/` 服务端 P95 <= 500ms
  - 项目详情点击按钮到收到 start 响应（前端侧）P95 <= 1.2s（本地网络）

#### NFR 类别: 安全性
- **等级**: L2 - 标准
- **量化目标**:
  - start 接口响应体仅包含 `authorize_url` 或 `detail`
  - 日志与响应中不出现 `client_secret` / access token 明文

### Increment 2: repo_url 驱动 provider 路由

#### NFR 类别: 数据一致性
- **等级**: L2 - 标准
- **量化目标**:
  - 对 `http://localhost:8012/...` 仓库，start 跳转域必须命中本地 provider 服务基址
  - 不再回退 `DJANGO_GITOAUTH_BASE`（无 provider 服务基址时返回 503）

#### NFR 类别: 可观测性
- **等级**: L2 - 标准
- **量化目标**:
  - provider 配置缺失场景 100% 返回明确 `detail`
  - start JWT 签发失败场景 100% 返回统一错误文案

### Increment 3: 配置模型统一与回归护栏

#### NFR 类别: 可维护性
- **等级**: L2 - 标准
- **量化目标**:
  - `gitOauth` 配置支持数组与对象（key=allowedHost）双结构
  - 相关回归测试在 CI 中 100% 通过（`test_github_app_start_redirect_uri.py`）

#### NFR 类别: 可用性
- **等级**: L1 - 基础
- **量化目标**:
  - 配置缺失时明确失败并可人工修复，不做自动降级与重试编排

## 质量场景

### QS-01: start 接口性能基线
| 要素 | 内容 |
|------|------|
| 类别 | 性能 |
| 等级 | L2 |
| 刺激源 | 项目详情页用户 |
| 刺激 | 点击仓库行 `OAuth 授权` |
| 制品 | `GET /api/accounts/gitlab/app/start/` |
| 环境 | 正常负载（开发/测试日常并发） |
| 响应 | 返回 200 + `authorize_url` |
| 响应度量 | 服务端处理时间 P95 <= 500ms（APM 或接口测试测量，不含网络） |

### QS-02: provider 路由一致性
| 要素 | 内容 |
|------|------|
| 类别 | 数据一致性 |
| 等级 | L2 |
| 刺激源 | 项目详情页用户 |
| 刺激 | 对仓库 `http://localhost:8012/ljy/somanyad` 发起 start |
| 制品 | GitLab start 路由选择逻辑 |
| 环境 | 正常 |
| 响应 | 命中 `gitOauth["http://localhost:8012"]` 对应 provider 配置并生成本地 authorize URL |
| 响应度量 | 单测断言 authorize_url 的 host 为 `localhost:8002`，错误路由率 = 0（测试集） |

### QS-03: 配置缺失可观测失败
| 要素 | 内容 |
|------|------|
| 类别 | 可观测性 |
| 等级 | L2 |
| 刺激源 | 后端配置系统 |
| 刺激 | provider 缺少 `service_base/host/port` |
| 制品 | GitLab start 接口 |
| 环境 | 正常 |
| 响应 | 返回 503 + 明确 `detail`，不回退全局配置 |
| 响应度量 | 单测断言状态码 503 且 `detail` 包含“未配置 Git OAuth 服务根 URL” |

### QS-04: 安全响应边界
| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L2 |
| 刺激源 | 普通已登录用户 |
| 刺激 | 请求 start 接口并观察响应与错误 |
| 制品 | start 接口响应契约 |
| 环境 | 正常/异常 |
| 响应 | 不泄露敏感凭据，仅返回授权跳转 URL 或错误说明 |
| 响应度量 | API 回包字段白名单检查：不包含 `client_secret`、access token、refresh token |

### QS-05: 配置模型兼容性回归
| 要素 | 内容 |
|------|------|
| 类别 | 可维护性 |
| 等级 | L2 |
| 刺激源 | 后续配置重构 |
| 刺激 | `gitOauth` 从数组迁移到对象后启动服务并执行授权测试 |
| 制品 | settings 归一化逻辑 + start 接口 |
| 环境 | CI / 本地回归 |
| 响应 | 服务正常启动，授权链路行为一致 |
| 响应度量 | `tests/test_github_app_start_redirect_uri.py` 全量通过 |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| 数据一致性 L2（repo_url -> provider 命中必须稳定） | 路由判定应作为领域规则，而不是散落在 UI 层临时逻辑 | 在 DDD 中将 provider 解析抽象为领域服务/值对象（RepoHost -> ProviderKey） |
| 安全性 L2（禁止敏感信息泄露） | 授权流程实体/服务需区分“可返回元数据”与“敏感凭据”边界 | 在应用服务契约中仅暴露 `authorize_url/detail`，凭据留在基础设施层 |
| 可维护性 L2（配置双结构兼容） | 配置模型需要显式归一化步骤，避免聚合内分支爆炸 | DDD 里建模 `GitOauthProviderConfig` 统一结构，入域前完成 normalize |
| 可观测性 L2（结构化失败） | 错误结果应成为可建模对象，而非自由字符串 | 定义错误值对象（error_code/detail/category），供应用层和 API 复用 |

## 权衡与边界

### 取舍
- 选择 L2 一致性与可维护性：优先保证“路由正确 + 配置可演进”，不追求分布式强一致。
- 选择 L1 可用性：当前不引入高可用集群与自动故障切换，降低本轮复杂度。

### 明确不做什么
- 不在本增量实现 OAuth 授权完成后自动重拉分支（仍由用户手动触发预览）。
- 不在本增量引入多区域灾备、断路器、自动重试编排。
- 不改动 gitOauth 服务端凭据存储模型，仅调整主站路由与配置读取方式。

### 升级触发条件
- 当 start 接口 P95 持续超过 500ms（连续 7 天）时，性能从 L2 升级到 L3，需引入链路分解或缓存策略。
- 当出现多环境配置冲突导致错误路由事故时，可维护性从 L2 升级到 L3，需引入配置校验器与发布前静态检查。
- 当安全审计要求更严格（合规检查）时，安全性从 L2 升级到 L3，增加审计事件与敏感字段扫描策略。

## 跳过声明

- 可伸缩性: 跳过。当前增量不新增高吞吐入口，容量需求与现网基线一致。
- 容错机制: 跳过。本轮仅改配置命中逻辑，不新增外部调用链路层级。
- 合规与隐私: 跳过。未新增个人敏感数据采集/存储字段。

## 自检 (Hard Gate)

- [x] 每个相关 NFR 类别都有明确的支撑等级（L0-L4）
- [x] 每个 L1-L4 的 NFR 类别至少有一个量化目标
- [x] 每个 L2-L4 的 NFR 类别至少有一个质量场景（QS）
- [x] 每个质量场景的响应度量可验证（给出数字和测量方式）
- [x] 影响领域模型的 NFR 决策已标注（含具体 DDD 动作）
- [x] 权衡和边界已明确（知道不做什么 + 升级触发条件）
- [x] 跳过的 NFR 类别有理由说明
- [x] 文档位置正确：`docs/superpowers/plans/2026-05-26-project-detail-oauth-button-nfr-clarification.md`
