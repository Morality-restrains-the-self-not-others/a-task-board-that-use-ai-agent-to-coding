# gitService Companion

## GitLab 登录策略（禁止忽略）

平台 GitLab **禁止自行注册**，Web 登录 **仅允许 taskAuth OIDC SSO**。

- 细则：`.ai/01_project_constraints/55_gitlab_sso_only_no_self_signup.md`（约束索引第 50 条）
- ADR：`docs/adr/0016-gitlab-sso-only-no-self-signup.md`
- SSOT：`conf/infra/git-service*/config.yaml` → `signupEnabled` / `passwordAuthWeb` / `passwordAuthGit` 均为 `false`
- 启动必须跑 `scripts/apply_auth_policy.sh`（DB 级；只改 `gitlab.rb` 不够）

**禁止**为排障把上述开关改为 `true`。应修 OmniAuth / OIDC（issuer、redirect_uri、client）。

## 本机 GitLab 启动闸门（ADR-0047）

本机 `git-service`（`:8012`）**默认禁止**被 runAll 面板或 `run.sh start` 拉起。

1. 编辑 `conf/infra/git-service/config.local.yaml`，增加 `runAllStartEnabled: true`
2. 再从 http://10.2.150.68:9999/ 启动 `git-service`
3. `run.sh stop` 不需要该键；仓库 `config.yaml` 必须保持 `runAllStartEnabled: false`

其它 runAll 服务不得 `depends_on: git-service*`（ADR-0048）。GitLab 由人工在 9999 对齐；task-git-oauth 只等 MySQL。

## 上海实例启停（runAll / 9999）

- 现网 `git-service`：须先打开上一节闸门，再 `bash gitService/run.sh start|stop`（INFRA `:8012`）或 9999 面板。
- 上海 `git-service-tencent-sh-1`：9999 必须走 `scripts/runall_ssh_sh_gitlab.sh`（`ssh sh` → `/opt/daydaymoney/gitservice-tencent-sh-1` `docker compose up -d|stop`）。

## 租户流量闸门（ADR-0036）与计量（ADR-0042）

公网 `git-upload-pack` 在预购为 0 或已用尽时拒绝（**含任务贴走公网 Host 的节点**；Docker NAT 的 RFC1918 源 IP 不再当内网）。同区域 VPC（10/8、192.168）与配置的内网 Host、以及 GitLab CI job 仍放行。Initializer：`initializers/zzz_trae_gitlab_traffic_quota.rb`。taskBill `POST /api/internal/taskbill/gitlab-traffic-gate/`。闸门不可达 fail-open。

出站用量以 GitLab 日志 sidecar 为准（`gitlab-traffic-shipper` 跟随 workhorse `written_bytes`（HTTP）与 gitlab-shell `written_bytes`（SSH，幂等键 `ssh:` 与 HTTP `wh:` 不冲突），POST `charge-gitlab-traffic`）。**禁止**依赖任务容器 `received_bytes`。脚本：`scripts/ship_gitlab_git_traffic.py`。

- `scripts/deploy_tencent_sh_1.sh` 只在 **Host sh 本机**、且工作区是完整 monorepo 时使用；禁止从 INFRA 9999 直接调用。

## 多实例 OAuth App 同步

`scripts/sync_local_oauth_app_scopes.sh` 按 provider `target.website` 主机名映射到
`conf/infra/git-service*/config.yaml` 的 `containerName`（实现：`scripts/gitlab_oauth_target_container.py`）。

- 默认实例 app → 容器 `gitlab`
- `tencent-sh-1` app → 容器 `gitlab-tencent-sh-1`
- **禁止**把区域实例的 Doorkeeper Application 写入默认 `gitlab` 容器

上海实例不在 INFRA 本机时：`ssh sh` 后在该机执行同步（compose 目录见上文
`GITSERVICE_SH_COMPOSE_DIR`）。`GITLAB_CONTAINER=<name>` 可只同步映射到该容器的 YAML。

## Omnibus 版本

- SSOT：`conf/infra/git-service*/config.yaml` 的 `imageTag`（现 `19.2.4-ce.0`）。
- compose 消费 `GITLAB_IMAGE`；`run.sh` 发现运行镜像与 SSOT 不同会重建容器（卷保留）。
- 升级须走官方 required stop：`19.0.x` 最新补丁 → `19.2.x`；不可从 `19.0.0` 一次跳到 `19.5`。
- 升级前：`gitlab-backup create` + 备份 `gitlab-secrets.json`；两跳之间等 batched background migrations 完成。
- Host sh 常拉不到 Docker Hub：用 `gitService/scripts/push_gitlab_image_to_sh.sh`（OPT-20260818-041）推送
  SSOT `imageTag` 到 sh（`docker save <image> | gzip -1 | ssh sh 'gunzip | docker load'`；`--check` 可先查远端是否已加载，
  `GITLAB_IMAGE_OVERRIDE`/`--image` 供分步升级），再改 `/opt/daydaymoney/gitservice-tencent-sh-1/.env` 的 `GITLAB_IMAGE` 后 `docker compose up -d`。
- 分步升级可设 `GITLAB_IMAGE_OVERRIDE=gitlab/gitlab-ce:19.0.8-ce.0 bash gitService/run.sh start`。
