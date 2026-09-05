# NFR 澄清: 项目详情页 OAuth 授权状态正确反映

> 输入:
> - 设计文档: `docs/specs/oauth-auth-state-not-reflected-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-28-project-detail-oauth-token-status-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L2 | 项目详情 API P95 增加 ≤ 300ms（含 gitOauth 调用） |
| 容错机制 | L2 | gitOauth 调用 2s 超时 + 静默降级（token_status=not_applicable） |
| 可观测性 | L2 | token 解析失败时记录 WARNING 日志（含 repo_url + user_id） |
| 可伸缩性 | L1 | 无需专项 — 多数项目 1-2 个 repo |
| 安全性 | L1 | 复用已有 token auth + forward-auth，无新增安全面 |
| 数据一致性 | L1 | 读时计算，无新写入路径 |
| 可用性 | L0 | 无专项保证 — gitOauth 不可用时静默降级 |
| 合规与隐私 | L0 | 不适用 |
| 可维护性 | L0 | 不适用 — 小变更，无版本化需求 |

## 逐增量 NFR 分析

### Increment 1: 后端项目详情 API 返回 git_repos_status

#### NFR 类别: 性能
- **等级**: L2 - 标准
- **量化目标**: 项目详情 API 响应时间 P95 增加 ≤ 300ms（含 gitOauth `summary-for-user/` 调用）。单个 repo 调用 gitOauth 耗时 P95 ≤ 200ms。
- **说明**: `resolve_repo_access_tokens` 对每个 repo URL 调用 gitOauth 内部 API。大多数项目仅有 1-2 个 repo，额外延迟可控。若项目有 5+ repo，总耗时可能超标——此为已知边界，后续可按需优化为一次 `summary-for-user/` 批量查询。

#### NFR 类别: 容错机制
- **等级**: L2 - 标准
- **量化目标**: gitOauth 调用超时 2s，超时或异常时 token_status 回退为 `not_applicable`（不阻塞项目详情返回），记录 WARNING 日志。
- **说明**: 项目详情 API 的核心价值是返回项目信息；token_status 是辅助字段。gitOauth 不可用时不应阻塞主流程。

#### NFR 类别: 可观测性
- **等级**: L2 - 标准
- **量化目标**: token 解析失败时记录 `logger.warning("resolve_repo_access_tokens failed: repo_url=%s user_id=%s error=%s")`，含完整上下文。
- **说明**: 方便排查"为什么按钮不显示/一直显示"类问题。

### Increment 2: 前端 ProjectDetail 按钮状态感知

#### NFR 类别: 性能
- **等级**: L1 - 基础
- **量化目标**: 按钮状态切换 ≤ 16ms（单帧渲染预算），纯客户端计算。
- **说明**: 无额外网络请求——token_status 已在项目详情响应中。

### Increment 3: OAuth 回调后自动刷新状态

#### NFR 类别: 性能
- **等级**: L1 - 基础
- **量化目标**: 回调后重新拉取项目详情，UI 在 1s 内反映新状态。
- **说明**: 复用已有 `fetchProjectDetail()`，无新增延迟。

## 质量场景

### QS-01: 项目详情 API 在 gitOauth 正常时返回完整 token_status
| 要素 | 内容 |
|------|------|
| 类别 | 性能 |
| 等级 | L2 |
| 刺激源 | 前端 SPA 用户导航至项目详情页 |
| 刺激 | GET /api/tenant/{id}/projects/{id}/ |
| 制品 | 项目详情 API (Django `ProjectViewSet.retrieve`) |
| 环境 | 正常负载，gitOauth 健康 |
| 响应 | 200 OK，包含 `git_repos_status` 数组每项含 `repo_url`, `token_status`, `oauth_provider` |
| 响应度量 | P95 总响应时间 ≤ 800ms（原基线 ~500ms + 新增 ≤ 300ms）；`git_repos_status` 数组长度 = `git_repos` 数组长度 |

### QS-02: 项目详情 API 在 gitOauth 不可用时静默降级
| 要素 | 内容 |
|------|------|
| 类别 | 容错机制 |
| 等级 | L2 |
| 刺激源 | gitOauth 服务宕机或网络不通 |
| 刺激 | 同上 GET 请求 |
| 制品 | 项目详情 API |
| 环境 | gitOauth 不可用（超时/连接拒绝） |
| 响应 | 200 OK，`git_repos_status` 每项 token_status = `not_applicable`，日志记录 WARNING |
| 响应度量 | 不抛出 500/503；`git_repos` 字段正常返回；日志包含 repo_url + error |

### QS-03: 已授权 repo 不显示 OAuth 按钮
| 要素 | 内容 |
|------|------|
| 类别 | 功能正确性（由性能支撑） |
| 等级 | L2 |
| 刺激源 | 已完成 OAuth 授权的用户访问项目详情页 |
| 刺激 | 页面加载，`git_repos_status` 中该 repo 的 `token_status = "token_available"` |
| 制品 | ProjectDetail.vue `shouldShowRepoOAuthButton` |
| 环境 | 正常 |
| 响应 | OAuth 按钮不渲染 |
| 响应度量 | `shouldShowRepoOAuthButton(repoUrl)` 返回 `false`；DOM 中无 OAuth 按钮元素 |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| 容错 L2：gitOauth 不可用时静默降级 | `TokenStatus` 值对象需支持 `not_applicable` 作为兜底值 | 已在 `token_status.py` 定义 `NOT_APPLICABLE`，无需新增 |
| 性能 L2：每 repo 调用 gitOauth | 暂无需 CQRS 拆分，`git_repos_status` 作为读时派生字段 | Serializer 层计算，不进入领域模型 |
| 可观测性 L2：失败日志含上下文 | 无需新领域事件，基础设施日志即可 | 在 `resolve_repo_access_tokens` 调用处加 try/except + logger |

## 权衡与边界

### 取舍
- 选择在项目详情 API 中内联计算 `git_repos_status`（而非独立 endpoint），接受额外 300ms 延迟，换取前端单次请求即可渲染完整页面
- 选择 gitOauth 不可用时静默降级（`token_status=not_applicable`），接受按钮可能误显示，换取项目详情页不因 gitOauth 故障而 500

### 明确不做什么
- 不做批量 `summary-for-user/` 查询优化（V1 保持逐 repo 调用）
- 不做 `git_repos_status` 缓存（读时计算，保证实时性）
- 不新增独立 endpoint（`git_repos_status` 内聚在项目详情 API 中）
- 不做授权状态图标（绿勾/警告）——留待后续 enhancement

### 升级触发条件
- 当项目平均 repo 数 > 5 时，性能从 L2 升级到 L3 → 改为一次 `summary-for-user/` 批量查询 + 本地匹配
- 当 gitOauth 故障率 > 1% 时，容错从 L2 升级到 L3 → 增加短期缓存兜底（TTL 60s）

## 跳过声明
- **可伸缩性**: 跳过。当前日活预期 < 100 用户，单实例即可满足，无扩展需求。
- **可用性**: 跳过。gitOauth 不可用时已设计静默降级，无额外可用性保证需求。
- **安全性**: 跳过。复用已有 token auth + forward-auth + X-User-Id 注入，无新增安全面。
- **数据一致性**: 跳过。`git_repos_status` 为读时派生字段，无新写入路径，无一致性风险。
- **合规与隐私**: 跳过。无新增数据存储或跨境传输。
- **可维护性**: 跳过。小变更（< 50 行后端 + < 30 行前端），无 API 版本化需求。
