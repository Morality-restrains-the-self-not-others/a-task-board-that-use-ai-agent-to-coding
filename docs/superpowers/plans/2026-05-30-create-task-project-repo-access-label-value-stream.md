# Value Stream: 创建任务 — 项目仓库可访问性标签（Provider 无关）

> Derived from design: `docs/superpowers/specs/2026-05-30-create-task-project-repo-access-label-design.md`

## Value Summary

用户在 work-panel 创建任务时，项目下拉前缀只显示「可访问 / 未授权 / 不可访问 / 未设置 / 无法检查」，不再出现「无GitHub」等 provider 标签；GitLab 与 GitHub 项目均走统一 OAuth + 可访问性检查。

## Related Value Streams

- **project-detail-repo-oauth-row-action**：dependency — 复用 GitProviderDetection + OAuth 换票链
- **config-aware-gitlab-repo-detection**：dependency — IP GitLab 识别已在分支预览落地，本流将其接入创建任务预检
- **task-management**：extension — 新增 create-task 预检 step

## End-to-End Flow

用户打开创建任务弹窗 → 并行请求各项目 `repo-access-check` → 后端取首个 repo_url → 换 OAuth token → `is_git_repo_accessible` → 返回 `access_status` → 前端显示 `(未授权) somanyad-tencent` 等。

## Value Increments

### Increment 1: 后端统一 repo-access-check（Thin Slice）

**Value to user:** GitLab 项目不再显示「无 GitHub 仓库」；未授权时 API 返回 `needs_auth`。  
**Scope:** `ProjectRepoAccessCheck` 服务 + 新 API + 旧 github 端点委托；单测覆盖 GitHub/GitLab IP/无 repo。  
**Depends on:** 无

### Increment 2: 前端 CreateTaskModal 状态映射

**Value to user:** 下拉前缀仅使用 provider 无关词表；删除 `无GitHub`。  
**Scope:** `CreateTaskModal.vue` 切换 API + `access_status` 映射；Vitest。  
**Depends on:** Increment 1

### Increment 3: validate-git-repo 换票 helper 收敛 + E2E

**Value to user:** 创建项目页与创建任务预检行为一致；回归护栏。  
**Scope:** 共用 token resolver；Playwright work-panel 用例。  
**Depends on:** Increment 2
