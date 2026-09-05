# 实施计划: Git 网站授权页三站点展示

- [x] 1. 后端 `GET /api/accounts/git-oauth/providers/` + `test_git_oauth_providers_catalog.py`
- [x] 2. `GithubAppConnectionView` / `GitlabAppConnectionView` 支持 `service_provider` 查询参数
- [x] 3. `GithubAppAuthorizeStartView` JWT 传入 `service_provider`
- [x] 4. `UserGitSiteOAuthSettings.vue` 动态目录 + connection/start 带参
- [x] 5. Playwright `GitSiteOAuth.provider-tabs-count.playwright.test.js`
- [x] 6. 修正相关 mock 用例 cookie 域名与路由

## 验证命令

```bash
cd task2app/Saas_project && python -m pytest tests/test_git_oauth_providers_catalog.py -q
cd task2app/playwright/front_project && npx playwright test GitSiteOAuth.provider-tabs-count.playwright.test.js
```
