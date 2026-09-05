# git-service 配置 Companion

本目录是平台 GitLab **默认实例**的人工可改 SSOT。

登录/注册（元规则 50 / ADR-0016）：

- `signupEnabled: false` — 禁止 `/users/sign_up`
- `passwordAuthWeb: false` / `passwordAuthGit: false` — 禁止账密；仅 SSO
- 改这三项必须保持 `false`；排障走 OIDC，不打开注册

细则：`.ai/01_project_constraints/55_gitlab_sso_only_no_self_signup.md`
资源键（内存/并发/镜像）见元规则 47。升级 GitLab CE 只改本目录 `imageTag`（当前 `19.2.4-ce.0`）；`run.sh` 检测到镜像漂移会重建容器（数据卷保留）。官方路径：先升到当前 minor 最新补丁，再停在 `19.2` required stop。

本机启动闸门（ADR-0047）：`runAllStartEnabled` 仓库默认 `false`。需要本机 CE 时在 **`config.local.yaml`** 写 `runAllStartEnabled: true`，再从 `:9999` 启动 `git-service`。不要把 `true` 提交进 `config.yaml`。

其它 runAll 服务不得 `depends_on: git-service`（ADR-0048）。GitLab 由人工在面板对齐；OAuth 运行时 HTTP 调用。

流量闸门（ADR-0036）：`regionSlug` 与 `trafficGateTaskBillBase`（同节点 `host.docker.internal:8004`，跨节点 `${INFRA_HOST}:8004`）。
