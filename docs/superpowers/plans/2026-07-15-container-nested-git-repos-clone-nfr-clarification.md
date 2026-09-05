# NFR 澄清: 容器 task-detail 下发最新子 Git 仓库并并发克隆

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-07-15-container-nested-git-repos-clone-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-07-15-container-nested-git-repos-clone-value-stream.md`
> - 权限分析: `docs/superpowers/specs/2026-07-15-container-nested-git-repos-clone-permission-analysis.md`
>
> 输出使用者: 实施计划, 构建, Review

**域分级**：鉴权 / 安全 → **L3**；其余 → **L2**；nested 发现/克隆失败 → **不阻断**父仓（容错 L2 显式约束）

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 安全性 / 鉴权 | L3 | server-container-token 任务作用域；internal nested 仅服务间 + 任务身份 user_id；子仓凭证继承不跨任务 |
| 容错机制 | L2 | **nested 发现失败不阻断父仓** clone；单个子仓 clone 失败不 cancel 其余仓 |
| 兼容性 | L2 | 无 nested 段 / 发现为空时响应与现网父仓行为一致；`git_repos: string[]` 旧形态仍可用 |
| 性能 | L2 | 并发克隆默认上限 8；enrich 对每个父仓 URL 去重调 internal API |
| 可用性 | L2 | nested 全失败时 bootstrap 仍可完成父仓克隆；日志区分父/子仓 |
| 可观测性 | L2 | 日志含「并行克隆 N 仓，并发上限 C」；nested enrich 失败 WARNING（无 token） |
| 可维护性 | L2 | enrich 单函数供 FetchTaskDetail / BuildRepoCloneCredentials 共用；Go + JS 单测 |
| 数据一致性 | L1 | 只读 enrich，不持久化 nested；每次 task-detail 重新发现 |
| 可伸缩性 | L1 | 元仓 ~50 子仓在并发 8 下可接受；更大规模靠调 env |
| 合规与隐私 | L2 | OAuth token 不写日志；internal API 不暴露公网 |

## 逐增量 NFR 分析

### Slice 1: internal nested API

#### 安全性 / 鉴权 — L3
- **量化目标**: 公网路由表无 `/api/internal/nested-git-repos`；非法 `user_id`/`repo_url` → 400
- **保障**: 仅 taskProjectService 内网监听；网关不转发

#### 性能 — L2
- **量化目标**: 单次 internal 调用 P95 < 5s（含 Git Provider RTT，与公网 nested 一致）
- **降级**: Credential 侧 timeout 后跳过 nested，不 fail task-detail

---

### Slice 2: Credential enrich + inherit

#### 安全性 / 鉴权 — L3
- **量化目标**: 子仓凭证 `UserID`/`GitIdentityID` 100% 来自同任务 identity；inherit 单测覆盖
- **保障**: 禁止请求参数指定 user_id

#### 容错机制 — L2
- **量化目标**: internal nested 失败 / 空列表 → task-detail 200 且父仓条目完整；**阻断率 0%**（相对父仓）
- **保障**: enrich 内 try/skip；不向上抛 fatal

#### 兼容性 — L2
- **量化目标**: 无 nested 时 `git_repos` 与改前字节级一致（除顺序稳定化若已有）
- **保障**: 仅 append 新 URL；已存在 URL 跳过

---

### Slice 3: onlineServiceJS 并发

#### 性能 — L2
- **量化目标**: 同时 in-flight clone ≤ `BOOTSTRAP_CLONE_CONCURRENCY`（默认 8）；50 仓总耗时 < 串行 50 倍单仓
- **保障**: 信号量 / pool 单测

#### 容错机制 — L2
- **量化目标**: 单子仓 clone reject 不导致 pool 未捕获异常；父仓 clone 先/并行完成均可
- **保障**: per-job catch + 汇总日志

#### 可观测性 — L2
- **量化目标**: bootstrap 日志必含 N 与 C 两数字
- **保障**: 固定文案模板

---

### Slice 4: 文档

#### 可维护性 — L2
- **量化目标**: machine_container §4.4 + intent 对照表齐全
- **保障**: T6 任务

## 质量场景

### QS-01: server-container-token 作用域

| 要素 | 内容 |
|------|------|
| 类别 | 安全性 / 鉴权 |
| 等级 | L3 |
| 刺激源 | 无有效 container token 的调用方 |
| 刺激 | POST task-detail |
| 制品 | taskCredentialService |
| 响应 | 401/403（与现网一致） |
| 响应度量 | 既有单测 / 集成测 |

### QS-02: nested 发现失败不阻断父仓

| 要素 | 内容 |
|------|------|
| 类别 | 容错机制 |
| 等级 | L2 |
| 刺激源 | internal nested API 500 或 timeout |
| 刺激 | 容器 bootstrap task-detail |
| 制品 | Credential enrich |
| 响应 | 200；`git_repos` 含父仓；父仓 clone 成功 |
| 响应度量 | Go 单测 mock nested 失败 |

### QS-03: 子仓凭证继承

| 要素 | 内容 |
|------|------|
| 类别 | 安全性 / 鉴权 |
| 等级 | L3 |
| 刺激源 | 任务已绑 GitLab identity 的父仓 |
| 刺激 | repo-clone-credentials 含子仓 URL |
| 制品 | BuildRepoCloneCredentials |
| 响应 | 子仓项 OAuth 与父仓同 UserID；RepoURL 为子仓 |
| 响应度量 | Go 单测 assert 字段 |

### QS-04: 并发上限

| 要素 | 内容 |
|------|------|
| 类别 | 性能 |
| 等级 | L2 |
| 刺激源 | task-detail 返回 20 个子仓 |
| 刺激 | cloneReposIntoSharedLayer |
| 制品 | onlineServiceJS bootstrap |
| 响应 | 任意时刻 active clone ≤ 8（默认） |
| 响应度量 | JS 单测 mock + spy |

### QS-05: 无 nested 兼容

| 要素 | 内容 |
|------|------|
| 类别 | 兼容性 |
| 等级 | L2 |
| 刺激源 | 普通 mono-repo 任务 |
| 刺激 | task-detail |
| 响应 | 与改前相同仓库列表；bootstrap 无回归 |
| 响应度量 | 快照 / 单测 |

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-07-15 | 初版：鉴权 L3；其余 L2；nested 失败不阻断 |
