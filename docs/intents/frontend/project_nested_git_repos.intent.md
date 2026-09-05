# 功能意图：项目详情展示子 Git 仓库列表

## 用户故事

作为租户成员，我在项目详情页的 Git 仓库区域查看父仓自动发现的**子 Git 仓库列表**（路径与远程 URL），以便了解元仓（如 `ram-work`）下有哪些独立嵌套仓，而无需手工逐个添加到 `project_repos`。

## 验收标准

1. 有项目读权限的用户打开项目详情时，对已关联父仓可看到「子 Git 仓库」区块。
2. 列表项包含 `path`、可点击 `url`（仅来自 `.gitmodules`）、发现来源 `gitmodules`。
3. 进入页面及关联仓变化后自动请求；提供「刷新」按钮。
4. 无 OAuth 或拉取失败时展示用户可读中文提示，不阻断详情页其余内容。
5. 无子仓时展示「未发现子仓库」，不报错崩溃。
6. 项目无关联仓时 API 返回 400；前端不发起无效请求或友好提示。
7. 不自动把子仓写入 `project_repos`（只读发现）。
8. UI 锚点：`data-testid="project-detail-nested-git-repos"`。
9. 每个可推导 URL 的子仓展示与父仓一致的 OAuth 授权状态徽章（已授权 / 未授权 / 授权异常 / 无需 OAuth）。**「已授权」表示当前用户凭据对该仓库 URL 有写入能力**（GitHub `permissions.push`/`admin`/`maintain`，GitLab Developer+；或 probe 非 401/403/404）。仅有 provider 级 OAuth、仓库已切换为他人仓且仅 pull、或 App 安装范围未覆盖时，须为「授权异常」并可点「重试」。`daydaymoney-gitlab` 的 authorize `redirect_uri` 须为 `${scheme}://${subdomains.base}/redirect/gitsite/${subdomains.gitlab}/oauth/callback/`（v2 回调契约，2026-08-07 迁移；gitsite = target.website 主机名；域名只来自 `conf/base.yaml`，与 Doorkeeper / GitHub App 白名单一致）。
10. 列表按授权关注度排序：**未授权（及授权异常）置顶**，其余保持相对顺序。
11. 多仓 OAuth 状态通过 `POST …/projects/validate-git-repos/` **一次批量**拉取（失败时可回退单仓 `validate-git-repo`）。
12. 创建项目页多仓校验同样走批量接口，且 `probe_access=true` 获取 `is_accessible`；校验实现在 **Go taskProjectService**（gitOauth 换票 + 远端 probe），不新增 Django 接口。
13. 创建任务弹窗等对项目主仓的 `GET …/repo-access-check/` 同样在 **Go** 完成 probe 与 `access_status` 分类，不经 Django 内部代理。
14. 项目详情 `git_repos_status` 由 Go 原生 **换票 + 远端 probe** 填充（不再调用 Django `git-repos-status` internal）；`token_available` 表示该仓可访问，不是「任意 provider token 存在」。

## 范围

- Go `taskProjectService`：`GET …/nested-git-repos/`、`GET …/validate-git-repo/`、`POST …/validate-git-repos/`（含 `probe_access`）、`GET …/repo-access-check/`、详情 `git_repos_status` enrich
- 解析：仅 `.gitmodules`（Git 相对 URL）；无该文件或无条目 → 空列表（不读 `.gitignore`）
- 前端：`useProjectNestedGitRepos.js` + `ProjectDetailGitReposSection.vue` + `useProjectDetailGitRepos.js` + `useCreateProjectGitRepoRows.js`
- 复用：GitLab/GitHub OAuth（gitOauth）；**不再**依赖 Django `validate-git-repo` / `git-repos-status` / `repo-access-check`（internal 已删或标废弃）
- 不含：容器内目录扫描、一键关联子仓、Django 新公网路由

## 业务意图 → 事件对照

**无对应事件**：纯查询例外，只读发现，无业务状态变更，不投递领域事件。

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 项目详情展示子 Git 仓库列表 | — | — | — | 纯查询例外，无服务端业务状态变更 |
| 批量校验仓库 OAuth/可访问性 | — | — | — | 纯查询例外 |
| 项目主仓可访问性检查 | — | — | — | 纯查询例外 |

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-08-29 | 「已授权」须对应当前仓库可访问性：probe 401/403/404 时降为 token_error；详情 enrich 与 FE 批量校验走 probe_access=true |
| 2026-07-23 | 澄清：重试 GitLab OAuth 的 redirect_uri 须对齐 gitoauth_api.*（FE-20260723） |
| 2026-07-17 | 移除 `.gitignore` 回退；发现仅 `.gitmodules` |
| 2026-07-17 | SSOT 改为 `.gitmodules` + Git 相对 URL 解析 |
| 2026-07-15 | 删除 Django internal `repo-access-check`；`git-repos-status` enrich 迁 Go；另两路标废弃 |
| 2026-07-15 | Playwright 统一 `clientReachableHost`；Django internal `repo-access-check` 标废弃 |
| 2026-07-15 | `repo-access-check` 迁 Go probe；CreateProject OAuth Playwright 用 `clientReachableHost` |
| 2026-07-15 | Go 原生校验（含 probe）；创建项目页改批量；脱离 Django |
| 2026-07-15 | 增补：批量 validate-git-repos + Playwright 未授权置顶 |
| 2026-07-15 | 增补：子仓 OAuth 状态展示 + 未授权置顶 |
| 2026-07-15 | 初版 |
