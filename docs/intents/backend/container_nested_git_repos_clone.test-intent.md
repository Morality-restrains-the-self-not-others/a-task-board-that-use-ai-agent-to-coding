# 测试意图：容器 task-detail 下发最新子 Git 仓库并并发克隆

- **对应功能意图**: `container_nested_git_repos_clone.intent.md`
- **日期**: 2026-07-15
- **设计文档**: `docs/superpowers/specs/2026-07-15-container-nested-git-repos-clone-design.md`

## 测试目标

验证容器 bootstrap 链在 task-detail enrich、凭证继承、并发克隆三层行为符合设计；**nested 失败不阻断父仓**为必测回归点。

## 测试分层

| 层 | 范围 |
|----|------|
| Go 单测 | taskProjectService internal handler；Credential enrich + inherit |
| JS 单测 | 并发池；collectRepoCloneJobs alias |
| 集成/冒烟 | （可选）runAll + 元仓任务 bootstrap 日志 |

## 用例矩阵

### taskProjectService internal API

| ID | 场景 | 步骤 | 期望 |
|----|------|------|------|
| T1 | 正常发现 | GET internal nested，`repo_url`+`user_id` 有效，mock Git 文件 | 200 + nested_repos 列表 |
| T2 | 缺参 | 缺 `repo_url` 或 `user_id` | 400 |
| T3 | 无 OAuth | user 无 token | 200 + empty + error（与公网 nested 一致） |
| T4 | 非公网 | 检查网关路由表 / OpenAPI tags | 无公网 tenant 路径 |

### taskCredentialService enrich

| ID | 场景 | 步骤 | 期望 |
|----|------|------|------|
| T5 | enrich 成功 | mock internal 返回 3 个子仓 | task-detail `git_repo_entries` 含 3 项；`clone_alias`=path |
| T6 | URL 去重 | nested 与父仓 URL 重复 | 不重复 append |
| T7 | **nested 失败不阻断** | mock internal 500 | task-detail 200；父仓仍在 `git_repos`；无 error 向上抛 |
| T8 | 无 identity | 任务无 UserID>0 | 跳过 nested 调用；仅父仓 |
| T9 | 父仓 URL 去重 | 两项目同父仓 URL | internal 只调一次 |

### 凭证 inherit

| ID | 场景 | 步骤 | 期望 |
|----|------|------|------|
| T10 | 子仓 inherit | BuildRepoCloneCredentials，父仓有 identity | 子仓 URL 凭证 UserID/GitIdentityID 与父仓相同 |
| T11 | 缺凭证仍 409 | 任务无任何 Git identity | 409 REPO_CLONE_CREDENTIALS_INCOMPLETE（含子仓 URL 在 missing） |
| T12 | 不跨任务 | 两任务 mock | 任务 A 凭证不含任务 B 子仓 |

### onlineServiceJS 并发克隆

| ID | 场景 | 步骤 | 期望 |
|----|------|------|------|
| T13 | 并发上限 | 20 jobs，`BOOTSTRAP_CLONE_CONCURRENCY=8` | 峰值 active ≤ 8 |
| T14 | 默认并发 | 未设 env | C=8 |
| T15 | 日志文案 | 执行 clone pool | 含「并行克隆 N 仓，并发上限 C」 |
| T16 | 单子仓失败 | mock 1 job reject | 其余 job 仍执行；无 unhandled rejection |
| T17 | nested alias | entries 含 `clone_alias=task2app` + `parent_repo_url` | 最终目录为 `{父仓目录}/task2app`（经 staging→move） |
| T17b | staging relocate | mock 父仓+子仓 clone 成功 | 子仓先在 `.bootstrap-staging/`，完成后父仓内 path 存在且 staging 清空 |
| T17c | 克隆层锁定 | relocate+checkout 前/后读 seal | 移入完成前 `isBootstrapReposLayoutReady===false`；完成后 true；未密封时 createJob（引导上下文）拒绝 |
| T17d | nested ahead | 主仓干净、子仓 ahead≥1 | `layerGitRemoteSnapshot.ahead≥1`，ztree 可出现推送 |

### 兼容 / 回归

| ID | 场景 | 步骤 | 期望 |
|----|------|------|------|
| T18 | 无 nested mono-repo | internal 返回 [] | 与改前 task-detail 仓库列表一致 |
| T19 | 父仓 clone 优先 | nested 失败 | 父仓 clone job 仍执行成功 |

## 权限 / 安全

| ID | 场景 | 期望 |
|----|------|------|
| P1 | 无 server-container-token | task-detail 401/403 |
| P2 | enrich user_id 来源 | internal 调用 user_id = 任务 identity，非任意 query |
| P3 | 响应/日志 | 不含 OAuth token 明文 |

## 通过标准

- T5–T8、T10–T11、T13–T17 自动化单测覆盖
- T7、T19 为 merge 门禁必跑
- P1–P3 至少 P2 有 Go 单测 assert

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-07-15 | 初版 T1–T19 + P1–P3 |
