# 设计文档：Git 网站授权页展示全部 service_provider

**日期：** 2026-05-27  
**页面：** `http://localhost:4000/user/827923618451263488/profile/git-site-oauth/`

---

## 现象

`task2app/conf/port_config.json` 的 `gitOauth` 配置了 3 个 `service_provider`：

| provider | service_provider | website |
|----------|------------------|---------|
| github | github-official | http://github.com |
| gitlab | daydaymoney-gitlab | http://gitlab.daydaymoney.com |
| gitlab | gitlab-local | http://localhost:8012 |

授权设置页仅显示 **GitHub / GitLab** 两个切换按钮（按 `provider` 类型硬编码），无法分别管理第三个实例。

---

## 根因

`UserGitSiteOAuthSettings.vue` 中 `providerOptions` 写死为两项；未调用后端配置目录，也未在 `start` / `connection` 请求中传递 `service_provider`。

后端 `GithubAppConnectionView.get` 在无 `repo_url` 时将 `provider_key` 硬编码为 `"github"`，无法按 `service_provider` 查询绑定状态。

---

## 方案

1. **API** `GET /api/accounts/git-oauth/providers/`：从 `GIT_OAUTH_PROVIDER_CONFIGS` 返回可展示的 provider 列表（`provider`、`service_provider`、`provider_key`、`website`、展示用 `label`）。
2. **前端**：挂载时拉取列表，按 `service_provider` 渲染切换按钮；`connection` / `start` 附带 `service_provider` 查询参数。
3. **后端修补**：`GithubAppConnectionView` / `GitlabAppConnectionView` 支持 `service_provider`；`GithubAppAuthorizeStartView` 签发 JWT 时传入 `service_provider`。
4. **Playwright**：登录后访问用户页，断言切换按钮数量为 3，且文案/website 与 `port_config.json` 一致。

---

## 价值流影响

影响 `value-stream.yaml` 中用户 Git OAuth 绑定相关步骤；需更新/新增 Playwright 用例。

---

## 领域概念（轻量）

- **GitOauthProviderCatalog**：配置中的 provider 目录
- **OAuthProviderKey**：`provider:service_provider` 复合键
- **GitSiteOAuthBinding**：用户在某 provider_key 下的连接状态
