# 实施计划：内部仓磁盘占用展示

日期：2026-07-16  
设计：`docs/superpowers/specs/2026-07-16-project-internal-repo-disk-size-design.md`

## DDD 说明（Step 6 书面例外）

- 无新聚合/实体；属项目读模型投影字段 enrich。
- 纯查询：不投递领域事件。

## 任务

### Backend（taskProjectService）

- [x] `ProviderResolver.IsInternalRepo(repoURL)`：`ProviderKey == "gitlab:gitlab-local"`
- [x] `fetchGitLabRepositorySizeBytes(repoURL, token, sessionCookie) (int64, error)`
- [x] `enrichProjectGitRepoDiskSizes(detail, userID)`：写 `is_internal` / 条件写 `disk_size_bytes`
- [x] `handleGetProject` 调用 enrich（在 status enrich 之后）
- [x] OpenAPI：`git_repo_entries` 补充字段说明
- [x] 单测：内部/外部/解析/不调用外部

### Frontend

- [x] `useProjectDetailGitRepos` 映射 `diskSizeBytes` / `isInternal`
- [x] `formatDiskSizeBytes` 工具（或内联小函数）
- [x] `ProjectDetailGitReposSection` 条件渲染徽章
- [x] Vitest：内部显示 / 外部不显示

### Docs

- [x] 意图 + 测试意图
- [x] 架构 v34 三件套 + VERSION_HISTORY
- [x] 价值流测试点（若有现成条目则追加）

## 验证命令

```bash
cd taskProjectService/src && go test ./... -count=1 -run 'Internal|DiskSize|RepositorySize'
cd task2app/front_project/app && npx vitest run src/components/ProjectDetailGitReposSection.disk-size.test.js
```
