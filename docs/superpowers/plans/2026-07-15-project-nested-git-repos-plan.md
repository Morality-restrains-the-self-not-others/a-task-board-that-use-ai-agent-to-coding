# 实施计划: 项目详情展示子 Git 仓库列表

> 输入: `docs/superpowers/specs/2026-07-15-project-nested-git-repos-design.md`  
> 价值流: `docs/superpowers/plans/2026-07-15-project-nested-git-repos-value-stream.md`  
> NFR: `docs/superpowers/plans/2026-07-15-project-nested-git-repos-nfr-clarification.md`  
> 权限: `docs/superpowers/specs/2026-07-15-project-nested-git-repos-permission-analysis.md`  
> 日期: 2026-07-15

---

## 任务清单（TDD 顺序）

### T1 — Go: `parseNestedFromGitignore` + 单测

- [ ] 新建 `taskProjectService/src/nested_git_repos_parse.go`
- [ ] 实现 `parseNestedFromGitignore(content string) []nestedRepoCandidate`
  - 匹配 `(?i)^#\s*Nested git repos` 起止段
  - 收集 `Name/` 单段目录项；排除 `*`/`?`/`[`/`!` 行
- [ ] 新建 `taskProjectService/src/nested_git_repos_parse_test.go`（表驱动 + ram-work `.gitignore` fixture）
- [ ] **验证**:

```bash
cd /tmp/ram-work/taskProjectService && go test ./src/ -count=1 -run 'TestParseNestedFromGitignore' -v
```

---

### T2 — Go: `parseGitmodules` + sibling URL + merge

- [ ] 实现 `parseGitmodules(content string) []nestedRepoCandidate`
- [ ] 实现 `deriveSiblingRepoURL(parentHTTPS, path string) (url, resolveError string)`（MVP 单段 path）
- [ ] 实现 `mergeNestedRepos(gitignore, gitmodules []nestedRepoCandidate) []NestedRepo`（path 字典序；同 path gitmodules 优先 url）
- [ ] 单测：gitmodules ini、SSH 父 URL 规范化、合并去重、多段 path → `url=""` + resolve_error
- [ ] **验证**:

```bash
cd /tmp/ram-work/taskProjectService && go test ./src/ -count=1 -run 'TestParseGitmodules|TestDeriveSibling|TestMergeNested' -v
```

---

### T3 — Go: fetch raw file + handler + OpenAPI

- [ ] 新建 `nested_git_repos_fetch.go`：`fetchRepoRawFile(userID, repoURL, filePath, gitlabSession) (string, error)`（GitLab/GitHub；main→master）
- [ ] 新建 `nested_git_handlers.go`：`handleProjectNestedGitRepos` — 对齐 `handleProjectBranches` 鉴权；缺省 `repo_url` = 首关联仓；无仓 400
- [ ] `main.go` → `handleProjectsRoute` 注册 `case "nested-git-repos":`
- [ ] OpenAPI schema：`NestedGitReposResponse`、`nested_repos` 项
- [ ] handler 单测：mock raw → 200 列表；无 token → error 文案；跨 tenant → 404
- [ ] **验证**:

```bash
cd /tmp/ram-work/taskProjectService && go test ./src/ -count=1 -run 'TestHandleProjectNestedGitRepos|TestFetchRepoRawFile' -v
cd /tmp/ram-work/taskProjectService && go build ./...
curl -sS "http://127.0.0.1:${TASK_PROJECT_SERVICE_PORT:-8089}/api/schema/" | grep -q nested-git-repos
```

---

### T4 — 登记 `api_route_ownership`

- [ ] `db/api_route_ownership.yaml`：`route_prefixes` 或等价条目注明 `nested-git-repos` → `task-project-service` / go
- [ ] 同步 `docs/architecture/api-route-to-owner.md`（若仓库惯例要求）
- [ ] **验证**:

```bash
cd /tmp/ram-work && python3 db/scripts/ci/check_django_new_api_routes.py
grep -n 'nested-git-repos' db/api_route_ownership.yaml
```

---

### T5 — 前端 composable + `ProjectDetailGitReposSection` UI

- [ ] 新建 `task2app/front_project/app/src/composables/useProjectNestedGitRepos.js`
  - 参数：`tenantId`, `projectId`, `repoUrl`（ref）
  - 状态：loading / error / nestedRepos / refresh
- [ ] 扩展 `task2app/front_project/app/src/components/ProjectDetailGitReposSection.vue`
  - 子区块「子 Git 仓库」`data-testid="project-detail-nested-git-repos"`
  - 进入页 / repo 变化自动 fetch；刷新按钮；path + 可点击 url
- [ ] composable 单测（mock fetch）
- [ ] **验证**:

```bash
cd /tmp/ram-work/task2app/front_project/app && npm run test:unit -- --run useProjectNestedGitRepos 2>/dev/null || npm run test:unit -- useProjectNestedGitRepos
```

---

### T6 — intents 文档

- [ ] `docs/intents/frontend/project_nested_git_repos.intent.md`
- [ ] `docs/intents/frontend/project_nested_git_repos.test-intent.md`
- [ ] **验证**:

```bash
test -f docs/intents/frontend/project_nested_git_repos.intent.md
test -f docs/intents/frontend/project_nested_git_repos.test-intent.md
```

---

### T7 — 单元 / 可选 Playwright

- [ ] 补全 Go 集成 smoke（若 T3 未覆盖 end-to-end mock server）
- [ ] （可选）`task2app/front_project/app/e2e/ProjectDetail.nested-git-repos.playwright.test.js`
  - 断言 `data-testid="project-detail-nested-git-repos"` 可见
- [ ] **验证**:

```bash
cd /tmp/ram-work/taskProjectService && go test ./src/ -count=1 -run 'Nested' -v
# 可选 E2E（需本地栈）:
# cd task2app/front_project/app && npx playwright test ProjectDetail.nested-git-repos
```

---

## 全量回归（Ship 前）

```bash
cd /tmp/ram-work/taskProjectService && go test ./src/ -count=1
cd /tmp/ram-work && python3 db/scripts/ci/check_django_new_api_routes.py
xvfb-run -a ./docs/architecture/Archi/Archi -application com.archimatetool.commandline.app \
  -consoleLog -nosplash \
  --loadModel docs/architecture/v28-application-integration-20260715-1135-claude.archimate
```

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-07-15 | 初版 T1–T7 可勾选清单 |
