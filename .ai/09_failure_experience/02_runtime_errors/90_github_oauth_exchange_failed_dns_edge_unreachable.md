# [运行时] GitHub OAuth exchange_failed：DNS 边缘 IP 不可达（api.github.com 可达）

## 基本信息

- 案例编号：OPT-20260811-049-GITHUB-DIAL-FALLBACK（2026-08-11 修复）
- 关联服务：taskGitOauth（:8002）、provider 配置（conf/auth/git-oauth/）
- 影响链路：项目页「OAuth 授权」→ GitHub 授权 → 回调换票 → 前端 Toast
  「授权失败：无法连接 GitHub 服务器，请检查网络后重试」（`github=exchange_failed`）

## 失败现象

- 页面：`/tenant/.../projects/.../` Toast `div.p-4.taskplugin-el-highlight`
- 文案：`授权失败：无法连接 GitHub 服务器，请检查网络后重试`
- 服务日志：
  - `dial tcp 20.205.243.166:443: i/o timeout`
  - 或 `http2: timeout awaiting response headers`
- 对照：`curl https://api.github.com/` 成功；`curl https://github.com/` / token 端点超时

## 根因链

1. **主根因**：本机 DNS 对 `github.com` 只返回不可达边缘 IP（`20.205.243.166`），
   TCP 443 超时。OAuth 换票 URL 为 `https://github.com/login/oauth/access_token`
   （**不是** `api.github.com`）。
2. **次因**：部分其他 github.com 前端 IP 可 TCP/TLS 连通但 HTTP 响应挂起；
   并行 Happy-Eyeballs 会「抢到」坏连接 → `ResponseHeaderTimeout`。
3. **历史**：2026-08-07 已移除死代理 `socks5://127.0.0.1:1080`（失败经验 87）；
   移除后暴露本地区 DNS 边缘不可达问题。

## 修复

- `outbound_github_dial.go`：对 `github.com` / `www.github.com` **fallback IP 优先串行拨号**，
  DNS IP 殿后；HTTP 失败调用 `ReportGithubDialHTTPFailure` 轮转优先 IP。
- 默认 fallback：`20.27.177.113` / `140.82.114.3` / `140.82.112.3`（经实测可换票）。
- `ForceAttemptHTTP2=false` + TLS `NextProtos: http/1.1`；`ResponseHeaderTimeout=8s`。
- 运维覆盖：`GITOAUTH_GITHUB_DIAL_FALLBACK_IPS=ip1,ip2`。
- 回归：单测（DNS 不可达→fallback 成功）+ 活路 `TestExchangeGitHubCodeLiveFallbackDial`。

## 防再发

- 诊断 `exchange_failed` 时先对比 `github.com` vs `api.github.com` 连通性，勿只查代理。
- 新增 fallback IP 前须用 `--resolve github.com:443:<ip>` 验证 **完整 OAuth POST**（不只 TCP）。
- 禁止再把本机 `127.0.0.1` 开发代理写进生产 provider 配置（失败经验 87）。
- 伴读：87（死代理）、40（outbound_proxy 历史缓解）、88（nested git 因 refresh 超时误标未绑定）。

## 运维段（主动告警）

- Loki 告警规则：`AiMonitor/loki/rules-files/github-oauth-edge-alerts.yml`
  - `GitHubOauthExchangeFailureRateHigh`：5m 内 `authorization_code 换票失败` > 5 次
  - `GitHubOauthDialEndpointDemoteHigh`：5m 内 `dial endpoint demoted` > 3 次
- 触发时优先看 `{job="task-git-oauth"} |~ "换票失败"` 与 `|~ "dial endpoint demoted"`，
  再按上表诊断链路排查边缘 IP / fallback 覆盖。
