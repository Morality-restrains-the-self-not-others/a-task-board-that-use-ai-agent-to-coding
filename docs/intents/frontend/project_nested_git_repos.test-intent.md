# 测试意图：项目详情展示子 Git 仓库列表

## 覆盖点

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 解析 ram-work `.gitmodules` | 返回 `task2app`、`docs`、`runAll` 等；source=gitmodules；相对 URL 已解析 |
| T2 | 解析 `.gitmodules` 单元 | path + url；source=gitmodules |
| T3 | 无 `.gitmodules`（即使有 gitignore Nested 段；父仓可见） | nested_repos=[]，不回退 |
| T3b | 父仓不可见（GitHub 对无权限私有仓 404） | nested_repos=[] + error 含「无法访问父仓库」，**不得**仅 empty |
| T4 | 相对 URL `../task2app.git` | 相对父仓 remote 解析为同 group 兄弟仓 |
| T5 | submodule url 为空 | url 空 + resolve_error |
| T6 | handler mock 远端 `.gitmodules` | 200 + 排序后的 nested_repos |
| T7 | 无 OAuth token | 200 + nested_repos=[] + error 中文 |
| T8 | 跨 tenant project_id | 404 |
| T9 | 项目无关联仓 | 400 No repository URL provided |
| T10 | composable loading → success | nestedRepos 填充；error 清空 |
| T11 | composable API 失败 | 局部 error；不 throw 至页面根 |
| T12 | UI 空列表 | 文案「未发现子仓库」 |
| T13 | （可选 Playwright）详情页 | `project-detail-nested-git-repos` 可见 |
| T14 | 子仓 OAuth 徽章 | 有 URL 的行显示 `nested-git-repo-oauth-status-*`；未授权显示「未授权」+ OAuth 按钮 |
| T15 | 未授权置顶 | 混排列表中 `not_bound` / `token_error` 排在已授权之前，同权重保持原相对序 |
| T16 | 批量 validate-git-repos | 多 URL 一次 POST；详情页 `probe_access=true`；Go 原生换票+probe；前端优先批量、失败回退单仓 |
| T17 | Playwright 未授权置顶 | 登录态 `nested-git-repo-oauth-status-0`=未授权且首行含未授权仓；batchCalls>0 |
| T18 | probe_access=true | 创建项目批量返回 is_accessible；有 token 但远端 401/403/404 时 `token_status=token_error`（不得仍为 token_available）；Go 远端 probe，不经 Django |
| T19 | 创建项目 debounce 合并 | 多行变更 500ms 内合并为一次 validate-git-repos |
| T26 | 创建项目校验请求失败 | HTTP/超时/缺结果分文案，不得一律「请检查网络」；错误节点 `data-traceId`；可点「重试」 |
| T20 | repo-access-check Go 原生 | 主仓 probe + access_status 分类；不调用 Django internal |
| T21 | CreateProject OAuth Playwright | `clientReachableHost` 处理 `0.0.0.0`；登录页可打开；OAuth 按钮场景通过 |
| T22 | Playwright host 统一 | front_project/saas 用例不再自定义 `localHost`/`clientHost`，一律 `clientReachableHost` |
| T23 | Django internal 废弃/删除 | `repo-access-check` internal **已删除**；`git-repos-status`/`validate-git-repo` internal 标 DEPRECATED；Go enrich 原生 |
| T24 | git_repos_status Go enrich | 项目详情 `git_repos_status` 由 Go 换票+probe 填充，0 次 Django `git-repos-status`；他人私有仓 404 时 token_error |
| T25 | daydaymoney-gitlab 重试授权 redirect_uri（v2 契约） | `start-from-gateway` 的 authorize_url 中 `redirect_uri` 等于 base.yaml 展开的 `${scheme}://${subdomains.base}/redirect/gitsite/${subdomains.gitlab}/oauth/callback/`（gitsite = target.website 主机名，2026-08-07 迁移；与 Doorkeeper / GitHub App 白名单一致）；provider YAML 无硬编码 FQDN；旧 gitoauth_api 子域路径兼容保留 |
| T27 | 仓库切换为他人仓 | 详情页 `https://github.com/test-ruandao/helloworld.git`：有原账号 GitHub token 但 GET /repos `permissions.push=false`（或 404）时徽章为「授权异常」而非「已授权」，并显示重试 |

## 权限 / 安全

| ID | 场景 | 期望 |
|----|------|------|
| P1 | 非租户成员 | 401/403（网关）或无法进入详情路由 |
| P2 | fetchGitAccessToken 调用 | userID = 当前登录用户 |
| P3 | 响应体 | 不含 OAuth token / 完整 cookie |

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-08-29 | T18/T24/T27：有 token 但仓库 404 不得已授权；详情 enrich 与 FE 批量走 probe |
| 2026-08-07 | T25 改 v2 契约：redirect_uri 迁 base 域 `/redirect/gitsite/<gitsite>/oauth/callback/`；SP 更名 daydaymoney-gitlab |
| 2026-07-17 | T3b：父仓不可访问不得伪装成「未发现子仓库」；优先 default_branch |
| 2026-07-17 | T1–T5 改为仅 `.gitmodules`；无文件不回退 gitignore |
| 2026-07-15 | T23 升级为删除 repo-access-check internal；增补 T24；另两路标废弃 |
| 2026-07-15 | 增补 T22/T23：Playwright host 统一；Django internal 标废弃 |
| 2026-07-15 | 增补 T20/T21：repo-access-check 迁 Go；Playwright BASE_URL 修复 |
| 2026-07-15 | 增补 T18/T19：Go probe + 创建页批量；T16 改为纯 Go |
| 2026-07-15 | 增补 T16/T17：批量接口与 Playwright 未授权置顶 |
| 2026-07-15 | 增补 T14/T15：子仓授权展示与置顶 |
| 2026-07-15 | 初版 |
