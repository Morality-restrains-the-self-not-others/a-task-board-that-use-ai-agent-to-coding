# 实施计划: 项目详情页 OAuth 授权状态正确反映

> 输入:
> - 设计文档: `docs/specs/oauth-auth-state-not-reflected-design.md`
> - 价值流: `docs/superpowers/plans/2026-06-28-project-detail-oauth-token-status-value-stream.md`
> - NFR: `docs/superpowers/plans/2026-06-28-project-detail-oauth-token-status-nfr-clarification.md`
> - DDD: `docs/superpowers/plans/2026-06-28-project-detail-oauth-token-status-ddd.md`

## 价值流增量 → 任务映射

### Increment 1: 后端项目详情 API 返回 git_repos_status (Thin Slice)

#### 1.1 领域层：GitRepoTokenStatus 值对象 ✅ (已完成)
- [x] 创建 `projects/domain/repo_access/value_objects/git_repo_token_status.py`
- [x] `from_resolver_result()` 工厂方法
- [x] `to_dict()` 序列化方法

#### 1.2 应用层：Serializer 增强
- [ ] **文件**: `task2app/Saas_project/projects/serializers/project_serializer.py`
- [ ] 在 `to_representation()` 中，`git_repos` 赋值后增加 `git_repos_status` 计算
- [ ] 导入 `resolve_repo_access_tokens` 和 `GitRepoTokenStatus`
- [ ] 对每个 `project_repo` 调用 `resolve_repo_access_tokens(request.user, repo_url)`
- [ ] 用 `GitRepoTokenStatus.from_resolver_result()` 包装结果
- [ ] try/except 包裹：gitOauth 不可用时 token_status 回退 `not_applicable` + logger.warning
- [ ] **验证**: `git_repos_status` 数组长度 === `git_repos` 数组长度

#### 1.3 后端单元测试
- [ ] **文件**: `task2app/Saas_project/tests/test_project_detail_git_repos_status.py`
- [ ] 测试 1: 已授权 repo 返回 `token_status = "token_available"`
- [ ] 测试 2: 未授权 repo 返回 `token_status = "not_bound"`
- [ ] 测试 3: 公开 repo 返回 `token_status = "not_applicable"`
- [ ] 测试 4: gitOauth 不可用时静默降级 — 不抛异常，`token_status = "not_applicable"`
- [ ] 测试 5: `git_repos_status` 数组长度与 `git_repos` 一致
- [ ] **验证**: `DJANGO_SETTINGS_MODULE=saas_project.settings_test pytest tests/test_project_detail_git_repos_status.py -v`

---

### Increment 2: 前端 ProjectDetail 按钮状态感知

#### 2.1 ProjectDetail.vue — token_status 初始化
- [ ] **文件**: `task2app/front_project/app/src/views/ProjectDetail.vue`
- [ ] 新增 `const gitRepoTokenStatus = ref({})`
- [ ] 在 `fetchProjectDetail()` 中：解析 `data.git_repos_status`，填充 `gitRepoTokenStatus`
- [ ] Key 使用 `normalizeRepoUrlKey(s.repo_url)` 对齐已有 button key 逻辑

#### 2.2 ProjectDetail.vue — shouldShowRepoOAuthButton 升级
- [ ] **修改函数** `shouldShowRepoOAuthButton(repoUrl)`
- [ ] 增加 `token_status` 判断：`token_available` → return false
- [ ] 增加 `token_status` 判断：`not_applicable` → return false
- [ ] 保持原有 provider catalog 检查逻辑

#### 2.3 ProjectDetail.vue — repoOAuthButtonLabel 升级
- [ ] **修改函数** `repoOAuthButtonLabel(repoUrl)`
- [ ] `token_error` → 返回 `'重试'`
- [ ] 保持原有 loading/默认逻辑

#### 2.4 前端单元测试
- [ ] **文件**: `task2app/front_project/app/src/views/__tests__/ProjectDetail-oauth-button.spec.js` (或 Vitest)
- [ ] 测试 1: `token_available` → 按钮不渲染
- [ ] 测试 2: `not_bound` → 按钮渲染，文案"OAuth 授权"
- [ ] 测试 3: `token_error` → 按钮渲染，文案"重试"
- [ ] 测试 4: `not_applicable` → 按钮不渲染
- [ ] **验证**: `npm test -- ProjectDetail-oauth-button`

---

### Increment 3: OAuth 回调后自动刷新 + E2E

#### 3.1 ProjectDetail.vue — 回调刷新
- [ ] **文件**: `task2app/front_project/app/src/views/ProjectDetail.vue`
- [ ] 在 `onMounted` 中调用 `applyOAuthCallbackFromRoute(route, router, { onSuccess: ... })`
- [ ] `onSuccess` 回调中调用 `fetchProjectDetail()` 刷新状态
- [ ] 消除 `?gitlab=ok` / `?github=ok` 查询参数（由 `applyOAuthCallbackFromRoute` 内部清除）

#### 3.2 Playwright E2E 测试
- [ ] **文件**: `e2e-tests/playwright/front_project/tests/project-detail-oauth-token-status.spec.js`
- [ ] 场景 1: 已授权用户进入项目详情 → 不显示"OAuth 授权"按钮
- [ ] 场景 2: 未授权用户 → 显示"OAuth 授权"按钮 → 点击 → 授权完成回跳 → 按钮消失
- [ ] **验证**: `npx playwright test project-detail-oauth-token-status`

---

## 依赖关系

```
1.1 (VO) ──→ 1.2 (Serializer) ──→ 1.3 (Backend Test)
                                        │
                                        ▼
                              2.1 (init token_status) ──→ 2.2 (button logic) ──→ 2.3 (label) ──→ 2.4 (Frontend Test)
                                                                                                      │
                                                                                                      ▼
                                                                                              3.1 (callback refresh) ──→ 3.2 (E2E)
```

## 文件变更清单

| # | 文件 | 操作 | 增量 |
|----|------|------|------|
| 1 | `projects/domain/repo_access/value_objects/git_repo_token_status.py` | 新建 | 1 |
| 2 | `projects/serializers/project_serializer.py` | 修改 | 1 |
| 3 | `tests/test_project_detail_git_repos_status.py` | 新建 | 1 |
| 4 | `front_project/app/src/views/ProjectDetail.vue` | 修改 | 2+3 |
| 5 | `front_project/app/src/views/__tests__/ProjectDetail-oauth-button.spec.js` | 新建 | 2 |
| 6 | `e2e-tests/playwright/front_project/tests/project-detail-oauth-token-status.spec.js` | 新建 | 3 |

## 风险与回滚

| 风险 | 缓解 |
|------|------|
| `resolve_repo_access_tokens` 对多 repo 项目增加显著延迟 | NFR 已设定 P95 增量 ≤ 300ms；若超标则改为批量查询 |
| gitOauth 不可用导致项目详情 500 | try/except + 静默降级（L2 容错） |
| 前端 token_status mapping key 与已有 button key 不一致 | 统一使用 `normalizeRepoUrlKey()` |

## 完成标准

- [ ] `test_project_detail_git_repos_status.py` 全部通过
- [ ] `ProjectDetail-oauth-button.spec.js` 全部通过
- [ ] `project-detail-oauth-token-status.spec.js` (E2E) 全部通过
- [ ] 手动验证：已授权仓库进入项目详情页，"OAuth 授权"按钮不显示
