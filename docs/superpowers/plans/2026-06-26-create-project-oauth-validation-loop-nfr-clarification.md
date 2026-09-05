# NFR 澄清: Create Project 页面 GitLab OAuth 仓库校验闭环

> 输入:
> - 设计文档: `docs/specs/create-project-gitlab-oauth-validation-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-26-create-project-oauth-validation-loop-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L2 | validate-git-repo P95 ≤ 500ms；前端回调检测 ≤ 100ms |
| 安全性 | L3 | OAuth token 不透传到前端；token_status 不含敏感信息 |
| 可用性 | L2 | gitOauth 不可达时返回 token_status=token_error 并降级提示 |
| 数据一致性 | L2 | token_status 读取最终一致（缓存 TTL 300s 可接受） |
| 可观测性 | L2 | token_status 字段纳入 X-Trace-Id 链路 |
| 可维护性 | L2 | API 新增字段向后兼容；旧前端忽略新字段正常工作 |

跳过类别：可伸缩性、容错机制、合规与隐私（本增量不引入新数据流或外部依赖，不改变存储模型或部署拓扑）。

## 逐增量 NFR 分析

### Increment 1: 后端 token_status 区分 + 文案优化

#### NFR 类别: 性能
- **等级**: L2 - 标准
- **量化目标**: validate-git-repo 端到端 P95 ≤ 500ms（含 gitOauth 内部调用）；token_status 判定为纯内存逻辑，不增加额外 I/O
- **影响**: `resolve_repo_access_tokens` 返回 dict 增加一个字符串字段，无性能影响

#### NFR 类别: 安全性
- **等级**: L3 - 增强
- **量化目标**: token_status 字段仅暴露状态枚举值，不泄露 access_token / refresh_token / 用户身份信息
- **影响**: `validate_git_repo_view` 响应中 `token_status` 为白名单枚举，`oauth_provider` 仅暴露 provider 名

#### NFR 类别: 可用性
- **等级**: L2 - 标准
- **量化目标**: gitOauth 服务不可达时（超时/网络错误），`token_status=token_error`，API 仍返回 200 + 可操作的错误提示，不抛 500
- **影响**: `resolve_repo_access_tokens` 需区分 404 (not_bound) 与网络错误 (token_error)

#### NFR 类别: 可观测性
- **等级**: L2 - 标准
- **量化目标**: token_status 判定路径写入 request logging extra，可被 X-Trace-Id 关联到 Grafana

### Increment 2: 前端 OAuth 回调自动重校验

#### NFR 类别: 性能
- **等级**: L2 - 标准
- **量化目标**: `applyOAuthCallbackFromRoute` 检测耗时 ≤ 100ms（纯内存读取 route.query + sessionStorage）；重校验 debounce 保持现有 500ms
- **影响**: `onMounted` 同步检查 URL query params，无额外网络请求

#### NFR 类别: 可维护性
- **等级**: L2 - 标准
- **量化目标**: 新增的 OAuth 回调监听逻辑不影响 ProjectDetail、UserGitSiteOAuthSettings 等已有页面的回调处理；向后兼容
- **影响**: 复用 `gitSiteOAuthCallbackUtils.js` 现有函数，仅新增 CreateProject 专有 success 回调

### Increment 3: token_status 驱动按钮智能显隐

#### NFR 类别: 安全性
- **等级**: L3 - 增强
- **量化目标**: 前端根据 `token_status` 决定 UI 时，不将 token_error 的内部错误细节展示给用户。token_status 枚举值由后端白名单控制
- **影响**: 前端 switch(token_status) 仅接受后端文档声明的 4 个枚举值

## 质量场景

### QS-01: validate-git-repo 返回 token_status 区分度
| 要素 | 内容 |
|------|------|
| 类别 | 性能 + 可用性 |
| 等级 | L2 |
| 刺激源 | 用户在 CreateProject 页面输入私有 GitLab 仓库 URL |
| 刺激 | 前端发起 GET /api/tenant/{id}/projects/validate-git-repo/?url=... |
| 制品 | utility_views.validate_git_repo_view |
| 环境 | gitOauth 正常（未绑定）或 gitOauth 不可达 |
| 响应 | 200 OK + {"is_accessible": false, "token_status": "not_bound", "message": "..."} 或 token_status=token_error |
| 响应度量 | 服务端 P95 ≤ 500ms；token_status 值与 gitOauth 实际状态一致 |

### QS-02: OAuth 回调后 CreateProject 自动重校验
| 要素 | 内容 |
|------|------|
| 类别 | 可维护性 |
| 等级 | L2 |
| 刺激源 | 浏览器从 gitOauth callback 重定向回 CreateProject 页面 |
| 刺激 | URL 携带 `?gitlab=ok` 或 `?github=ok` 查询参数 |
| 制品 | CreateProject.vue onMounted 逻辑 |
| 环境 | 正常 |
| 响应 | 检测到 OAuth 成功回调参数 → 遍历所有已输入仓库行 → 触发 validateGitRepoRow → 清除错误状态或更新为可访问 |
| 响应度量 | onMounted 中 applyOAuthCallbackFromRoute 完成 ≤ 100ms；重校验与现有 debounce 行为一致 |

### QS-03: 新字段向后兼容
| 要素 | 内容 |
|------|------|
| 类别 | 可维护性 |
| 等级 | L2 |
| 刺激源 | 旧版前端（未升级）调用 validate-git-repo API |
| 刺激 | 收到包含 token_status / oauth_provider 新字段的响应 |
| 制品 | validate_git_repo_view 响应格式 |
| 环境 | 正常 |
| 响应 | 旧前端忽略未知字段，现有 is_accessible / message 字段语义不变 |
| 响应度量 | 旧前端 OAuth 按钮逻辑不受影响；现有 Playwright E2E 全部通过 |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| token_status 为纯展示状态 (L2 一致性) | RepoAccessResult 值对象增加 token_status 字段；无需领域事件 | DDD 步骤扩展 RepoAccessResult 值对象 |
| token_status 不暴露敏感信息 (L3 安全性) | token_status 是白名单枚举，不能包含动态错误信息 | DDD 步骤定义 TokenStatus 枚举值对象 |
| 回调检测纯前端逻辑 (L2 可维护性) | 不引入新的领域概念；OAuth 回调状态由前端路由参数承载 | DDD 步骤仅建模后端概念 |
| 向后兼容 (L2 可维护性) | API 响应仅增字段不改字段；旧字段语义不变 | DDD 步骤标记新增字段为 optional |

## 权衡与边界

### 取舍
- 选择前端 URL query 参数承载 OAuth 回调状态（简单、可观察）而非 WebSocket 推送（复杂、过度设计）
- token_status 读本地缓存（TTL 300s），接受短暂过期窗口以换取零额外 I/O

### 明确不做什么
- 不在 V1 实现 OAuth 回调的 WebSocket 实时推送
- 不修改 gitOauth 的 bind_status 生命周期（pending→active→failed 保持不变）
- 不为 token_status 增加数据库持久化（纯计算字段）
- 不做 OAuth 授权的自动刷新（token 过期后用户手动重新授权）

### 升级触发条件
- 当 gitOauth 不可达频率 > 1% 时，token_status=token_error 需附带重试建议并在告警中升级为 L3 可用性
- 当前端 OAuth 回调检测逻辑需要跨页面共享时（如多个页面都需要），升级为全局 router guard 而非 per-component

## 自检

- [x] 每个相关 NFR 类别都有明确的支撑等级（L0-L4）
- [x] 每个 L1-L4 的 NFR 类别至少有一个量化目标
- [x] 每个 L2-L4 的 NFR 类别至少有一个质量场景（QS-01/QS-02/QS-03）
- [x] 每个质量场景的响应度量可验证
- [x] 影响领域模型的 NFR 决策已标注（4 项）
- [x] 权衡和边界已明确
- [x] 跳过的 NFR 类别有理由说明
- [x] 文档位置正确
