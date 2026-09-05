# conf/auth/git-oauth — AI 伴读

## 监听 host（禁止忽略）

本目录 `providers/*.yaml` 中的服务 `host` **必须**为 **`0.0.0.0`**。

- **禁止**改为 `127.0.0.1` / `localhost`（会导致 Docker APISIX `host.docker.internal:8002` 无法达 taskGitOauth）。
- 全仓库规则：根目录 `.ai.md`；细则 `.ai/01_project_constraints/22_service_listen_host_not_localhost_only.md`。
- 踩坑案例：`.ai/09_failure_experience/02_runtime_errors/13_apisix_host_docker_internal_localhost_bind.md`。

同机客户端调用仍可用 `http://127.0.0.1:8002`；与进程监听面无关。

## provider_key 与 provider 目录对齐

`providers/*.yaml` 的 `service_provider` / 隐式 `provider_key`（如 `github:github-official-daydaymoney`）必须与 **`conf/auth/git-oauth/providers/`**（taskGitOauth 加载 / OAuth 起点 / catalog）及 **`conf/auth/task-credential/git-oauth-providers/`**（taskProjectService 换票 key）一致。

- GitLab `target.scope` **必须**含 `write_repository`（git-over-HTTP `git push`）；仅 `api`/`read_repository` 会换票成功但推送被 GitLab 以 HTTP Basic Access denied 拒绝。
- 前端 connection 查询用 `conf/auth/git-oauth/providers/` 的 `service_provider`；
- 回调落库用 taskGitOauth 的 `githubStoredProviderKey()`。
- 不一致时会出现：`?github=ok` 但页面仍显示未绑定（summary 查错 key）。

## 浏览器回调契约 v2：redirect_uri 须为 base 域 `/redirect/gitsite/<gitsite>/oauth/callback/`

（2026-08-07 迁移，取代旧 gitoauth_api 子域 `/api/accounts/<sp>/oauth/callback/`）

所有 provider YAML（含 `http-gitlab-daydaymoney-com.yaml`）的 `redirect_uri` **必须**为
`${scheme}://${subdomains.base}/redirect/gitsite/<gitsite>/oauth/callback/`，
`allowedHost` **必须**为 `${scheme}://${subdomains.base}`（域名形态只在 `conf/base.yaml` 定义）。

- **`<gitsite>` = `target.website` 的主机名**（如 `github.com`、`${subdomains.gitlab}` 展开值
  `gitlab.daydaymoney.com`），**禁止硬编码 FQDN**（含旧多层 `*.api.*` 形态与任何
  `.com/.net` 域名字面量）。
- 回调落在 base 域，经 APISISX（spa-catch-all hosts base/www）→ taskGitOauth
  `/redirect/gitsite/` 路由（gateway priority 878）→ `ResolveProviderByGitsite` 反查
  `target.website` / `match_origins` 主机名 → 按 provider 类型回调处理。
- 旧路径 `/api/accounts/<sp>/oauth/callback/`（gitoauth_api 子域，gateway priority 830）
  保留兼容，**禁止删除**；`redirect_uri` 不得回退旧形态。
- 与 GitLab Doorkeeper、GitHub App 白名单（均需登记新 base 域 redirect_uri）一致；
  改后重启 taskGitOauth 并跑 `bash gitService/scripts/sync_local_oauth_app_scopes.sh`。
- 服务提供方名：`daydaymoney-gitlab` 已更名 **`daydaymoney-gitlab`**（SP 与
  `service_provider` / task-credential 同名目录须一致；现存 DB 绑定 `gitlab:daydaymoney-gitlab`
  需一次性迁移，否则前端显示未绑定）。
- 验收：`bash gitService/scripts/test_gitlab_daydaymoney_redirect_uri_ssot.sh`（断言**全部** GitLab YAML 展开值与
  base.yaml 一致、无硬编码 FQDN，含 camelCase 子域如 `${subdomains.gitlabTencentSh1}`）。
- 踩坑（旧契约）：`.ai/09_failure_experience/02_runtime_errors/80_gitlab_oauth_redirect_uri_gitoauth_api_mismatch.md`。

## 多区域 GitLab（ADR-0014）必须独立 provider，禁止借 default token

每个 `conf/infra/git-service*` 区域实例必须在 **两棵树** 各有一条 YAML：

- `conf/auth/git-oauth/providers/http-gitlab-<slug>.yaml`（taskGitOauth 加载 / OAuth 起点 / catalog）
- `conf/auth/task-credential/git-oauth-providers/http-gitlab-<slug>.yaml`（taskProjectService 换票 key）

`service_provider` 与区域 slug 一致（如 `tencent-sh-1`），`target.website` 指向该实例
`${scheme}://${subdomains.gitlabXxx}`。占位符正则必须匹配 camelCase 子域键。

**禁止**：省略区域 YAML，导致 `matchProvider` / `gitlab-start-from-gateway` 回落到
`gitlab:default` / 默认实例；或 `access-for-user` 用 `gitlab:` 前缀把其它 CE 的 token
借给区域仓（对另一套 GitLab 调 API → 401，trace `2a45093b`）。

Doorkeeper Application 必须写入 **website 所属容器**（`containerName`），由
`gitService/scripts/gitlab_oauth_target_container.py` 映射；禁止把 tencent-sh-1 的 app
同步进默认 `gitlab` 容器。上海实例在 Host sh 上时须在该机执行
`GITLAB_CONTAINER=gitlab-tencent-sh-1 bash gitService/scripts/sync_local_oauth_app_scopes.sh`。

## GitLab provider 不可当作「未用」删除

本目录 `providers/` 是 **taskGitOauth** 加载 GitLab/GitHub OAuth App 的 SSOT。

- **禁止**删除 `*gitlab*` / `http-localhost-8012.yaml` 等 GitLab 条目（即使 Django 侧仍有同名文件）。
- 误删后健康检查 `gitlab_provider_config_count` 变为 `0`，前端会出现：
  `无法启动 GitLab 授权（bad_state） [debug: missing_allowed_host]`。
- 踩坑案例：`.ai/09_failure_experience/02_runtime_errors/48_gitlab_oauth_bad_state_missing_allowed_host.md`。
- 清理前验收：`curl -sS http://127.0.0.1:8002/api/health/ | jq .gitlab_provider_config_count`（期望 ≥1）。

## GitHub 出站代理（可选，生产默认直连 + dial fallback）

> ⚠️ **2026-08-07 事故教训**：`providers/*.yaml` 曾硬编码 `outbound_proxy: socks5://127.0.0.1:1080`
> （本机 `ssh -D` 开发代理），生产未运行该代理 → 直连抖动回退后 `dial tcp 127.0.0.1:1080:
> connection refused`，换票必失败拖 46s（失败经验 87）。**生产禁止使用本机/开发代理**。

> ⚠️ **2026-08-11 事故教训**：移除死代理后，本机 DNS 解析的 `github.com`（如
> `20.205.243.166`）TCP 不可达，而 `api.github.com` 正常；换票打
> `https://github.com/login/oauth/access_token` → `exchange_failed`
> 「无法连接 GitHub 服务器」。修复：taskGitOauth 对 `github.com` 使用 **fallback IP
> 优先串行拨号**（`outbound_github_dial.go`）+ 强制 HTTP/1.1 + HTTP 失败轮转
> （失败经验 88）。

> ⚠️ **2026-08-18**：部分 fallback IP「TCP 通但 HTTP 挂起」时，仅靠 rotate 游标
> 会反复命中坏节点（trace `6eff4b43…`）。现已 **demote last-used endpoint（2min TTL）**
> + 无 proxy 时 Exchange/Refresh 直连重试 4 轮（失败经验 90 补丁）。

- **默认直连 + dial fallback**：不配置 `outbound_proxy`；代码内置可达边缘 IP 列表。
- 覆盖拨号 IP：`GITOAUTH_GITHUB_DIAL_FALLBACK_IPS=ip1,ip2`（逗号分隔）。
- 若确需出站代理（网络拓扑要求），代理地址**必须是目标环境可达的公网地址**，
  并在此处文档化用途与运维联系方式；禁止写入 `127.0.0.1` / `localhost` 等本机回环代理。
- 支持 `socks5` / `socks5h` / `http` / `https`。
- **不得**依赖 shell 的 `HTTP(S)_PROXY`（见 `.ai/01_project_constraints/23_app_startup_no_env_proxy.md`）；业务 Client 仍默认 `Proxy: nil`，仅在直连换票/拉 profile 网络失败后回退到本字段。
- 运维覆盖：环境变量 `GITOAUTH_GITHUB_OUTBOUND_PROXY`（同样是显式配置，不是全局代理继承）。
- 未配置或代理不可达时行为：仅直连（含 dial fallback + demote）+ 有限重试（无 proxy 时最多 4 轮）。
- 配置验收：`grep -n outbound_proxy providers/*.yaml` 应为空（或仅注释说明）；如非空，
  须确认该地址在目标环境可达。
