# NFR 澄清: 项目详情展示子 Git 仓库列表

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-07-15-project-nested-git-repos-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-07-15-project-nested-git-repos-value-stream.md`
> - 权限分析: `docs/superpowers/specs/2026-07-15-project-nested-git-repos-permission-analysis.md`
>
> 输出使用者: 实施计划, 构建, Review

**域分级**：项目元数据只读发现 → 默认 **L2**；安全/鉴权 → **L3**

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 安全性 / 鉴权 | L3 | 项目读权 = 详情读权；禁止跨租户；OAuth token 仅当前用户；不写入 project_repos |
| 兼容性 | L2 | 无 `.gitignore` 段 / 无 `.gitmodules` / 无关联仓时不报错崩溃；详情页其余功能不受影响 |
| 可用性 | L2 | OAuth 失败仍 200 + 可操作 error 文案；Loading/error/空三态完整 |
| 容错机制 | L2 | 远端 404 视为空文件；main/master 分支 fallback；合并去重不因单源失败中断 |
| 性能 | L2 | 详情页额外 2 次 HTTP（gitignore + gitmodules）；P95 端到端 < 5s（含 Git Provider RTT） |
| 可观测性 | L2 | handler 记录 trace_id + provider + repo_url（脱敏 token）；error 字段用户可读 |
| 可维护性 | L2 | 解析逻辑纯函数单测；与 branches 共用 OAuth 客户端 |
| 数据一致性 | L1 | 只读无事务；展示结果不持久化，每次请求重新发现 |
| 可伸缩性 | L0 | 跳过 — 详情页低频 |
| 合规与隐私 | L1 | 不记录 OAuth token；子仓 URL 非密钥 |

## 逐增量 NFR 分析

### Slice 1: 解析纯函数

#### 可维护性 — L2
- **量化目标**: gitignore / gitmodules / merge 单测覆盖率 ≥ 90% 语句（包内）
- **保障**: 表驱动测试 + ram-work fixture

#### 兼容性 — L2
- **量化目标**: 空输入 / 无标记段 / 通配符行 → 0 条 nested，不 panic
- **保障**: 单测边界用例

---

### Slice 2: 远端 + handler

#### 安全性 / 鉴权 — L3
- **量化目标**: 跨 tenant project 访问拒绝率 100%（404）；token 用户 ID = 请求用户 ID 100%
- **保障**: 对齐 `handleProjectBranches`；禁止 service account 代读私有仓

#### 容错机制 — L2
- **量化目标**: `.gitignore` 或 `.gitmodules` 单方 404 → 仍返回另一方结果；双方皆空 → `nested_repos: []`
- **保障**: 独立 HTTP 请求 + 404 → empty

#### 性能 — L2
- **量化目标**: 单 repo 两次 raw 请求并行或串行均可；handler P95 < 5s（mock 下 < 200ms）
- **降级**: 前端「刷新」可重试；不阻塞项目详情主数据加载

#### 可观测性 — L2
- **量化目标**: 每次 outbound Git 失败有 WARNING 日志（含 status，无 token）
- **保障**: 复用既有 git branch 日志模式

---

### Slice 3: 前端

#### 可用性 — L2
- **量化目标**: 三种 UI 状态（loading / error / empty）100% 覆盖 composable 分支
- **保障**: unit test + testid

#### 兼容性 — L2
- **量化目标**: API 404/500 不导致 ProjectDetail 白屏；Git 主列表区块仍可用
- **保障**: composable 内部 catch；error 局部展示

---

### Slice 4: 治理

#### 可维护性 — L2
- **量化目标**: Swagger 可见 + api_route_ownership 登记；CI check 通过
- **保障**: T4 任务

## 质量场景

### QS-01: 跨租户 IDOR

| 要素 | 内容 |
|------|------|
| 类别 | 安全性 / 鉴权 |
| 等级 | L3 |
| 刺激源 | 租户 A 成员 |
| 刺激 | GET `/api/tenant/A/projects/{B租户project}/nested-git-repos/` |
| 制品 | `handleProjectNestedGitRepos` |
| 响应 | 404 project not found |
| 响应度量 | 无 Git outbound；单测断言 |

### QS-02: 无 OAuth 不崩溃

| 要素 | 内容 |
|------|------|
| 类别 | 兼容性 |
| 等级 | L2 |
| 刺激源 | 已登录但未绑定 GitLab 的用户 |
| 刺激 | 打开含关联仓的项目详情 |
| 制品 | nested-git-repos API + Vue section |
| 响应 | 200 + empty list + 中文 error；详情页正常 |
| 响应度量 | Playwright / handler 单测 |

### QS-03: 无 nested 段不报错

| 要素 | 内容 |
|------|------|
| 类别 | 兼容性 |
| 等级 | L2 |
| 刺激源 | 普通 mono-repo（无 # Nested git repos 段） |
| 刺激 | GET nested-git-repos |
| 响应 | 200 + `nested_repos: []` + 空 error |
| 响应度量 | 单测 fixture |

### QS-04: Token 绑定当前用户

| 要素 | 内容 |
|------|------|
| 类别 | 安全性 / 鉴权 |
| 等级 | L3 |
| 刺激源 | 用户 U1 |
| 刺激 | 请求 nested-git-repos |
| 制品 | `fetchGitAccessToken` 调用链 |
| 响应 | 仅使用 U1 的 provider token |
| 响应度量 | mock 断言 userID 参数 |

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-07-15 | 初版：默认 L2；安全/鉴权 L3；兼容性 L2 |
