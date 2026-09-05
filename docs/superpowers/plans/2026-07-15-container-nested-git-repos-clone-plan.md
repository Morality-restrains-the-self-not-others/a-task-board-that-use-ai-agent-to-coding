# 实施计划: 容器 task-detail 下发最新子 Git 仓库并并发克隆

> 输入: `docs/superpowers/specs/2026-07-15-container-nested-git-repos-clone-design.md`  
> 价值流: `docs/superpowers/plans/2026-07-15-container-nested-git-repos-clone-value-stream.md`  
> NFR: `docs/superpowers/plans/2026-07-15-container-nested-git-repos-clone-nfr-clarification.md`  
> 权限: `docs/superpowers/specs/2026-07-15-container-nested-git-repos-clone-permission-analysis.md`  
> 日期: 2026-07-15

---

## 任务清单（TDD 顺序）

### T1 — Go: internal nested-git-repos handler + 单测

- [ ] 在 `taskProjectService` 注册 `GET /api/internal/nested-git-repos/`
- [ ] query：`repo_url`（必填）、`user_id`（必填，>0）
- [ ] 复用 `listNestedGitRepos(userID, repoURL, …)`；返回与公网 nested 同结构 DTO
- [ ] 非法参数 → 400；非公网路由、不写入 `project_repos`
- [ ] OpenAPI 标记 internal；Swagger schema 可见
- [ ] 新建/扩展单测：mock OAuth + 远端 → 200 列表；缺参 → 400
- [ ] **验证**:

```bash
cd /tmp/ram-work/taskProjectService && go test ./src/ -count=1 -run 'TestInternalNestedGitRepos' -v
cd /tmp/ram-work/taskProjectService && go build ./...
curl -sS "http://127.0.0.1:${TASK_PROJECT_SERVICE_PORT:-8089}/api/schema/" | grep -q 'internal/nested-git-repos'
```

---

### T2 — Go: taskCredentialService enrich 共用函数 + 单测

- [ ] 新建 `application/nested_repo_enrich.go`（或等价）：`enrichProjectReposWithNested(ctx, repos, identities)`
- [ ] 流程：`FetchTaskRepos` 结果 → 取首个 `UserID>0` identity → 父仓 URL 去重 → 调 ProjectService internal API
- [ ] merge：有效 `url` 的 nested 项追加到 `git_repos` / `git_repo_entries`（`clone_alias=path`）；已存在 URL 跳过
- [ ] **nested 失败不阻断**：单父仓 API 失败 → log + skip；不向上返回 error
- [ ] 单测：mock internal 成功 → entries 含子仓；mock 500 → 仍含父仓 only
- [ ] **验证**:

```bash
cd /tmp/ram-work/taskCredentialService && go test ./application/ -count=1 -run 'TestEnrichProjectReposWithNested' -v
```

---

### T3 — Go: FetchTaskDetail + BuildRepoCloneCredentials 接入 enrich

- [ ] `TaskDetailService.FetchTaskDetail` 在返回前调用 enrich
- [ ] `CredentialService.BuildRepoCloneCredentials` 在收集 expected repos 前/后调用同一 enrich
- [ ] 子仓 URL 无独立 identity：**继承**同任务 `UserID`/`GitIdentityID`，仅 `RepoURL` 不同
- [ ] 单测：`TestFetchTaskDetailIncludesNestedRepos`；`TestBuildRepoCloneCredentialsNestedInherit`
- [ ] 单测：nested 全失败 → 父仓凭证仍完整；缺凭证仍 409
- [ ] **验证**:

```bash
cd /tmp/ram-work/taskCredentialService && go test ./application/ -count=1 -run 'TestFetchTaskDetail|TestBuildRepoCloneCredentials' -v
cd /tmp/ram-work/taskCredentialService && go test ./... -count=1
```

---

### T4 — JS: 并发信号量 + cloneReposIntoSharedLayer

- [ ] 读取 `BOOTSTRAP_CLONE_CONCURRENCY`（默认 **8**，非法值回退默认）
- [ ] 实现 pool / 信号量包装既有 clone job 列表
- [ ] 日志：`并行克隆 ${N} 仓，并发上限 ${C}`
- [ ] 单仓失败 catch，不 unhandled rejection
- [ ] 新建 `bootstrap.cloneConcurrency.test.mjs`
- [ ] **验证**:

```bash
cd /tmp/ram-work/trae-agent/onlineServiceJS && node --test src/bootstrap.cloneConcurrency.test.mjs
```

---

### T5 — JS: collectRepoCloneJobs 含 nested alias

- [ ] 确认/扩展 `collectRepoCloneJobs`（或等价）消费 `git_repo_entries` 中 nested 项
- [ ] `clone_alias` = nested path → 本地目录名（沿用 sanitize）
- [ ] 单测：entries 含父 + 子 → job 数正确、目录名正确
- [ ] **验证**:

```bash
cd /tmp/ram-work/trae-agent/onlineServiceJS && node --test src/bootstrap.collectRepoCloneJobs.test.mjs
```

---

### T6 — intents + machine_container §4.4

- [ ] `docs/intents/backend/container_nested_git_repos_clone.intent.md`
- [ ] `docs/intents/backend/container_nested_git_repos_clone.test-intent.md`
- [ ] 更新 `task2app/Saas_project/skillList/machine_container.md` §4.4：
  - task-detail 可含 nested enrich 子仓
  - `clone_alias` = nested path
  - `BOOTSTRAP_CLONE_CONCURRENCY` 说明
  - nested 发现失败不阻断父仓
- [ ] **验证**:

```bash
test -f docs/intents/backend/container_nested_git_repos_clone.intent.md
test -f docs/intents/backend/container_nested_git_repos_clone.test-intent.md
grep -n 'nested\|BOOTSTRAP_CLONE_CONCURRENCY' task2app/Saas_project/skillList/machine_container.md
```

---

### T7 — 回归与集成冒烟

- [ ] taskProjectService + taskCredentialService 全量 `go test`
- [ ] onlineServiceJS 相关 bootstrap 单测全绿
- [ ] （可选）本地 runAll 启动含 ram-work 元仓任务 → bootstrap 日志见 N/C 与子仓目录
- [ ] **验证**:

```bash
cd /tmp/ram-work/taskProjectService && go test ./... -count=1
cd /tmp/ram-work/taskCredentialService && go test ./... -count=1
cd /tmp/ram-work/trae-agent/onlineServiceJS && node --test src/bootstrap.*.test.mjs
```

---

## 全量回归（Ship 前）

```bash
cd /tmp/ram-work/taskProjectService && go test ./... -count=1
cd /tmp/ram-work/taskCredentialService && go test ./... -count=1
cd /tmp/ram-work/trae-agent/onlineServiceJS && node --test src/bootstrap.*.test.mjs src/layerFs.repoDirNameFromUrl.test.mjs
```

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-07-15 | 初版 T1–T7 可勾选清单 |
