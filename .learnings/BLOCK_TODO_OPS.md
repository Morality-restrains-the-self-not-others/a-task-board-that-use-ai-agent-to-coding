# Blocked TODOs — OPS

> 本文件存放因 **运维与破坏性操作（清库、厂商门户重生模板、云安全组、重建容器、外部安装）** 阻塞而从开放清单分流的 OPT 条目。
> 阻塞解除后移回 [OPTIMIZATION_TODOS.md](./OPTIMIZATION_TODOS.md) 执行，或完成后迁入 [OPTIMIZATION_TODOS_COMPLETED.md](./OPTIMIZATION_TODOS_COMPLETED.md)。
> 分流规则见 [OPTIMIZATION_TODOS.ai.md](./OPTIMIZATION_TODOS.ai.md)「阻塞项分流」。

- **Category**: `OPS`
- **Count**: 42

---


### OPT-20260904-001 — 将 Host sh 公网 IP 加入微信服务号 IP 白名单

- **Status**: pending
- **Blocked-By**: OPS
- **Created**: 2026-09-04
- **Context**: trace `81612ca5-03e6-42a7-be52-b7b77420b2c5`：`POST /api/auth/wechat/mp/follow-qr/` 503，微信 `40164 invalid ip 120.36.185.132`。INFRA 出口为动态家宽；已部署 `wechat-mp-egress` 于 Host sh（出口 `1.117.67.121`），但该 IP **当前亦不在白名单**（SH 探针同样 40164）。代码侧 ADR-0059 已落地，二维码恢复取决于公众平台白名单。
- **Action**: (1) 登录微信公众平台 → 设置与开发 → 基本配置 / IP 白名单 → **添加 `1.117.67.121`**（可临时另加 `120.36.185.132` 应急，勿依赖）；(2) 确认 `curl -sf http://1.117.67.121:8030/healthz`；(3) http://10.2.150.68:9999/ 精准编译重启 `task-auth`；(4) 硬刷新 https://www.daydaymoney.com/profile/referral/ 断言无 `referral-mp-qr-error` 且出现动态码。
- **Why**: 无白名单时中继仍 40164；家宽 IP 不稳定不宜作长期白名单。
- **How to apply**: 微信公众平台后台；Loki `{job="task-auth"} |= "wechat_mp_follow_qr"` 不应再出现 40164。

### OPT-20260902-003 — 给上海 GitLab 容器留 swap 余量，避免 2GiB mem_limit 再次 OOM 成 502

- **Status**: pending
- **Blocked-By**: OPS
- **Created**: 2026-09-02
- **Context**: 公网 `https://gitlab-tencent-sh-1.daydaymoney.com/` 长时间 nginx 502。Host `sh` 上 `gitlab-tencent-sh-1` 为 `Exited (137)`（OOM），`127.0.0.1:8014` connection refused。`compose up -d` 后 UI 已恢复（`/` 302、`/users/sign_in` 200 + `x-gitlab-meta`），但 RSS 已到 `1.99GiB / 2GiB`（约 99%）。SH 宿主仅 ~3.6GiB RAM + 4GiB swap；`memLimit: 2g` 是精简模式 SSOT。
- **Action**: (1) 在 `conf/infra/git-service-tencent-sh-1/config.yaml` 增加可调 `memswapLimit`（建议 `4g`，即 2g RAM + 2g swap），compose `memswap_limit` 只消费该键；(2) 经现有 `deploy_tencent_sh_1_from_infra.sh` 同步到 `/opt/daydaymoney/gitservice-tencent-sh-1` 后 `docker compose up -d` 重建容器（不改 `signupEnabled`/`passwordAuth*`）；(3) 禁止把 memLimit 调到 ≥3g（宿主会被 nginx 挤死）。
- **Why**: 仅 restart 不改 cgroup swap 时，Puma/Sidekiq 再涨一点就会再次 SIGKILL，公网立刻回到 502。
- **How to apply**: `ssh sh 'docker inspect gitlab-tencent-sh-1 --format "Memory={{.HostConfig.Memory}} MemorySwap={{.HostConfig.MemorySwap}}"'` 须 `MemorySwap > Memory`；`curl -sS -m 15 -o /dev/null -w "%{http_code}" https://gitlab-tencent-sh-1.daydaymoney.com/users/sign_in` 为 `200` 或 `302`（不得 `502`）；`docker inspect ... --format "{{.State.OOMKilled}}"` 为 `false`。

### OPT-20260901-011 — 若 overlay 后 QQ SMTP 仍 535 则轮换授权码

- **Status**: pending
- **Blocked-By**: OPS
- **Created**: 2026-09-01
- **Context**: Loki 显示 `email-sent` 消费者 QQ SMTP `535 Login fail. Account is abnormal, service is not open, password is incorrect`。代码已叠 conf-local（16 位授权码骨架）。若精准重启后 AUTH 仍 535，则不是漏叠配置，而是腾讯邮箱账号/SMTP 服务未开或授权码失效。**【2026-09-01 17:45】** 同机用 `conf-local/events/domain-events/email.yaml` 做 SMTP AUTH 探针已成功；17:06–17:37 的 535 伴随 11 次重试与 systemd-resolved DNS 失败。**【2026-09-01 18:48】** clone-run 修正 `FindMonorepoRoot` 后 worker 日志 `password_configured=true`，Loki `EMAIL_SENT delivered to [author@example.com]`，无新 535。**不要轮换授权码**，除非之后再次 535。
- **Action**: (1) 确认 Loki `{job="task-auth"} |= "password_configured=true"` 且 `{job="task-events-email-sent-1-send-email"} |= "password_configured=true"`；(2) 仍 535 时在 QQ 邮箱生成新授权码，只写入 `conf-local/**/email.yaml` 的 `host_password`（三处镜像：`core/email`、`auth/task-auth`、`events/domain-events`）；(3) 重启 `task-auth` 与 `task-events-email-sent-1-send-email`；(4) 禁止把授权码写入已跟踪 `conf/`。
- **Why**: 535 文案含「账号异常 / 未开通服务」，overlay 不能修好厂商侧关停的 SMTP。
- **How to apply**: 腾讯邮箱「设置 → 账户 → POP3/IMAP/SMTP/Exchange/CardDAV/CalDAV 服务」；Loki 不得出现明文密码。

### OPT-20260831-007 — 用 GitHub Release 钉替换本机 file:// 后做一次无 ram-work 冷启动

- **Status**: pending
- **Blocked-By**: OPS
- **Created**: 2026-08-31
- **Context**: Go ELF 已在 Release `deploy-20260831`（OPT-20260830-019）。本机 live `$DEPLOY_ROOT` 为 `$HOME/bin/daydaymoney-deploy`（`/tmp/ram-deploy` 已退役），且未做过「新 clone + 手工 secrets + 不碰 ram-work」整栈启动。
- **Action**: (1) 新目录 `git clone` daydaymoney-deploy，不设 `META_ROOT`；(2) 按 `secrets.example/README.md` 把密钥放到 `conf-local/` 或 `secrets/conf-local/`；(3) `./scripts/up.sh`（`GITHUB_TOKEN`）拉 ELF 后 `check_p4`；不要改写 live `$HOME/bin/daydaymoney-deploy` 的 `releases.local.yaml`/`file://`。
- **Why**: 跨机是否真能只靠配置仓 + Release 仍未用独立树验证。
- **How to apply**: `gh release view deploy-20260831 --repo task2money/daydaymoney-deploy`；`DEPLOY_MODE=1` `runAll -command deploy-sync`；`bash scripts/check_p4_deploy_root.sh`。
- **Park-Note**: `gh release view` 已确认 tag `deploy-20260831` 19 assets。独立树 `up.sh` 会与 live `$HOME/bin/daydaymoney-deploy` 抢 :9999/:4000/:3100；禁止改写 live `file://`。需运维窗口另目录冷启动。

### OPT-20260830-017 — 轮换已进入 Git 的 GitLab root 口令并迁出 conf 明文

- **Status**: pending
- **Blocked-By**: OPS
- **Created**: 2026-08-30
- **Context**: P1 复制 conf 到 `daydaymoney-deploy` 时带上了 `infra/git-service/gitLabRootPwd.md`（该文件本已在 conf 子仓）。随后已从配置仓 HEAD 删除，但 Git 历史仍有副本。禁止 filter-repo。
- **Action**: (1) 在 GitLab 实例轮换 root 口令；(2) 只把新口令写入主机 `conf-local/` 或密码管理器（不建 *Pwd.md）；(3) ~~conf 子仓删除 `gitLabRootPwd.md`~~ **已完成**（2026-08-31，CI 拒 `*Pwd.md`）。
- **Why**: 口令一旦进 Git 即视为泄露；重写历史无效。
- **How to apply**: GitLab 管理后台或 `gitlab-rake`；daydaymoney-deploy 已删 HEAD。
- **Park-Note**: conf 子仓 HEAD 已不再跟踪 `gitLabRootPwd.md`。Git 历史仍有副本，须在 OPS 窗口轮换 live GitLab root。禁止 filter-repo。本会话不改 live GitLab 口令。

### OPT-20260825-004 — 为 ram-work Sonar 分析提供覆盖率报告以满足 Quality Gate

- **Status**: pending
- **Blocked-By**: OPS
- **Created**: 2026-08-25
- **Context**: Go/Python/JS 覆盖率已接入并复扫。`./sonarqube.sh` CE task `AaBHi_3G_VIm40notgMp` EXECUTION SUCCESS；QG `new_coverage=9.9`（门槛 80）仍 ERROR。`POST /api/qualitygates/create` HTTP 403 `Insufficient privileges`（token type PROJECT_ANALYSIS_TOKEN，`actions.create=false`，`manageConditions=false`）。Python：`pytest-cov` → `.runall/sonar-coverage/python-coverage.xml`（56 passed）。JS：`c8` lcov `.runall/sonar-coverage/onlineServiceJS/lcov.info`（vitest `@vitest/coverage-v8` npm install 失败 `edgesOut`）。
- **Action**: (1) [✅] Go `go test -coverprofile` 17 服务 (2) [✅] Python pytest-cov + JS c8 lcov 写入 `sonar-project.properties` (3) [✅] 复跑 `sonarqube.sh` new_coverage 0→11.5→9.9 仍 <80 (4) 管理员用非分析 token 调低 `new_coverage` 或补到 ≥80 后本项 completed
- **Why**: 分析 token 改不了内置 Sonar way；漏检期覆盖率无法靠本会话再堆到 80%。
- **How to apply**: 管理员登录 `http://10.2.150.68:19000/quality_gates` 调整 ram-work 门槛；或 `GET /api/qualitygates/project_status?projectKey=ram-work` 断言 `new_coverage` ≥80。证据：`./sonarqube.sh` exit 0；`curl -sS -u "$SONAR_LOGIN:" -X POST http://10.2.150.68:19000/api/qualitygates/create?name=probe` → 403。

### OPT-20260827-026 — 推送 empty-layer bootstrap_failed 镜像后重建 task_880716211791360000 容器

- **Status**: pending
- **Blocked-By**: OPS
- **Created**: 2026-08-27
- **Context**: 该任务容器 `i-m5eauxfav8t5ps9f9389` 已 BOOTSTRAP_FAILED（凭证不齐），快照仍推 empty `bootstrap_pending` 层。taskFE overlay 热更新后即可显示「Git 授权未齐」；容器侧把空层标 `bootstrap_failed` 需新 online 镜像。
- **Action**: (1) `DOCKER_PUSH=1 ./buildDocker.sh` 完成后确认 registry tag 含 `jobsRuntimeSnapshot` failed-empty 行为 (2) 对该任务 stop-vm → start-vm（或 9999 重建）(3) `GET .../layers?access_token=` 空层为 `bootstrap_failed=true` 或已有真实 clone 层
- **Why**: 存量容器不会加载新 snapshot 逻辑；只热更 taskFE 时 overlay 已可用，但层图字段仍是 pending。
- **How to apply**: `trae-agent/onlineServiceJS`；任务 `task_880716211791360000` comment `cmt_880716216098910208`；容器 UI `http://118.190.135.141:8765/ui/...`

### OPT-20260822-042 — 推送含 GitLab oauth_auth_by_repo 修复的 online 镜像后重建 task_878932440129761280 容器

- **Status**: pending
- **Blocked-By**: OPS
- **Created**: 2026-08-22
- **Context**: 自动运行第 4 步 GitLab push 因容器镜像未接线 `oauth_auth_by_repo` 失败（Loki `AUTO_RUN_DELIVERY_FAILED` 400，`still_ahead=true`）。token 接线 + `git_last_push_error.json` 已在 `trae-agent` `377a537`，镜像 `x86_64_2026-08-22_18-54`。存量容器仍跑旧镜像；**已经发生过的失败不会回写** `last_push_error`，ztree 红色「push 失败」只在新镜像再次失败时出现。`auto_run_delivery.done` 未写入，新容器启动会走 `retryPendingAutoRunDeliveries`。
- **Action**: (1) 确认 registry 镜像含 `377a537`（tag `x86_64_2026-08-22_18-54` 或更新）(2) 对该任务 stop-vm → start-vm（或 9999 重建容器）(3) Loki 查 `AUTO_RUN_DELIVERY_COMPLETE`，或 ztree `[data-testid=layer-ztree-push-error-label]` / Agent 评论出现 PR
- **Why**: 不重建则 token 接线与失败芯片都进不了运行中的容器；用户仍只看到可点的「提交并创建PR」。
- **How to apply**: `DOCKER_PUSH=1 ./buildDocker.sh`（`trae-agent/onlineServiceJS`）；任务 `task_878932440129761280` 层 `20260822_092914_76165f`；失败经验 117

### OPT-20260813-005 — 清库后在厂商/管理门户用新生成器重存 Linux UserData 模板

- **Status**: pending
- **Blocked-By**: OPS
- **Created**: 2026-08-13
- **Context**: 容器无法启动根因是 UserData 仅直连 `download.docker.com`（TLS reset）。生成器已改国内镜像优先；RunInstances 路径也有存量脚本 harden。但清库后 DB 模板需用新前端重新保存，新机器才自带镜像循环。
- **Action**: (1) 打开 ai-provider Admin/Vendor UserData 模板编辑页；(2) 对 Ubuntu Linux 模板点「重新生成」并保存；(3) 确认 content 含 `mirrors.aliyun.com/docker-ce`；(4) 再启动一台测试机验证 Docker 安装越过 GPG 步骤。
- **Why**: harden 只改写已知旧块；新模板应以生成器为准，避免依赖运行时字符串替换。
- **How to apply**: `taskAiProvider/frontend/src/utils/userDataScriptLinux.js`；`taskCloudService/src/userdata_replace.go` Harden；对照 `tmp/f2` 历史失败日志。

### OPT-20260812-047 — 厂商门户重生 UserData（含 TRACE_ID 占位符）并验收启机脚本

- **Status**: pending
- **Blocked-By**: OPS
- **Created**: 2026-08-12
- **Context**: 生成器强制 __TASK2APP_*（含 TRACE_ID）；存量已保存模板仍可能含真实 token/镜像。
- **Action**: (1) 精准编译重启 ai-provider + taskCloudService；(2) 厂商门户对该 Ubuntu 模板「生成容器脚本」并保存；(3) 新启实例 cat init_from_task2app.sh 可见 TRACE_ID=真实 trace、无模板字面量 tok_/registry 误烤；(4) boot-progress 请求带 X-Trace-Id。
- **Why**: 模板源码已改，已安装模板不会自动更新。
- **How to apply**: userDataScriptLinux.js / useUserDataScriptGenerator.js；userdata_replace.go。
- **Related**: OPT-20260812-046、OPT-20260811-001

### OPT-20260812-025 — 将 GitHub App daydaymoney 安装到 task2money 并复验子仓发现

- **Status**: pending
- **Blocked-By**: OPS
- **Created**: 2026-08-12
- **Context**: 项目详情 `proj_-2823506342987866751` 同时显示「已授权」与「无法访问父仓库」（trace `a6ef30d9-1d20-4301-a3dd-c6aa9d2a044d`）。实证：`ghu_` 可 `/user`，但 `/repos/task2money/ram-work` 404、`/user/installations`=0、`/orgs/task2money/installations` 为空；App `daydaymoney` 已有 `contents:write`，根因是**未安装到组织**而非权限为空。
- **Action**: (1) 组织管理员打开 https://github.com/apps/daydaymoney/installations/new ，安装到 **task2money**（含 `ram-work`）；(2) 项目页重新 OAuth；(3) `gh api orgs/task2money/installations --jq .total_count` ≥ 1；(4) 硬刷新后出现 `project-detail-nested-git-repo-row`。可选后续：徽标区分「已绑定」与「可访问 Contents」（产品文案）。
- **Why**: 不安装则子仓发现/自动克隆门禁会持续软失败；「已授权」仅表示换票成功，易误导。
- **How to apply**: 见 `.ai/09_failure_experience/02_runtime_errors/43_nested_git_repos_empty_on_inaccessible_parent.md` 与 `conf/auth/git-oauth/providers/github.ai.md`。

### OPT-20260811-009 — 验证 InitAllDatabases 后登录关键路径自动就绪

- **Status**: pending（daemon 已重启；清库待人工）
- **Blocked-By**: OPS
- **Created**: 2026-08-11
- **Context**: `EnsureLoginCriticalPath` 已合入；夜间 daemon 已重启，task-auth/gateway/taskFE healthy。完整「清库→初始化」属破坏性，未执行。
- **Action**: (1) 清空+初始化数据库；(2) 确认三服务 running；(3) 微信登录 302。
- **Why**: 编排不在业务精准重启列表，易漏热加载；需一次端到端证明。
- **How to apply**: `runAll/src/runner.go`；`domain/login_critical_path.go`；9999 UI。
- **Related**: OPT-20260811-003、OPT-20260811-004

### OPT-20260811-001 — 重生 UserData 并复验评论启动日志

- **Status**: pending（服务已部署，待厂商门户+实例）
- **Blocked-By**: OPS
- **Created**: 2026-08-11
- **Context**: `CONTAINER_NAME`/`report_progress` 等修复已部署；存量实例旧 `init_from_task2app.sh` 不会自动更新。
- **Action**: (1) 厂商门户对该 Ubuntu 模板「生成容器脚本」并保存；(2) 评论 @镜像启机；(3) 启动日志出现安装依赖/拉镜像等步骤，状态离开「启动中」。
- **Why**: 旧模板仍会 `docker inspect -` 卡住。
- **How to apply**: `userDataScriptLinux.js` / `userDataTemplate.js`；runbook `docs/runbooks/container-instance-stuck-in-starting-remediation.md`。
- **Related**: OPT-20260810-035

### OPT-20260810-035 — 验收评论启动日志含 UserData boot-progress

- **Status**: pending（服务已部署，待模板重生+启机）
- **Blocked-By**: OPS
- **Created**: 2026-08-10
- **Context**: 模板上报 `comment_id`、后端 CSC 路由、前端扇出已就绪并部署；旧模板仍可能无 userdata_boot 行。
- **Action**: 重生模板后评论 @镜像启动 → 启动日志出现「安装基础依赖/拉取容器镜像」等 `userdata_boot` 行。
- **Why**: 旧脚本无 `comment_id` 时公网只见容器调度三行。
- **How to apply**: `userDataTemplate.js` `report_progress`；`handleBootProgress`；任务详情启动日志面板。
- **Related**: OPT-20260811-001

### OPT-20260810-038 — 修复 SaaS→容器 :8765 HTTP 入站超时

- **Status**: pending（需云安全组）
- **Blocked-By**: OPS
- **Created**: 2026-08-10
- **Context**: 夜间 curl `47.76.218.186:8765` health/probe 仍超时（HTTP 000）；容器→SaaS push 200。根因多为平台出口 CIDR 未写入实例 SG。
- **Action**: (1) 核对 `detectPlatformEgressCIDR` / `TASK2APP_SG_EXTRA_INGRESS_CIDRS`；(2) 主机 `curl -m5 http://<public_ip>:8765/api/health`；(3) 必要时刷新 SG ingress。
- **Why**: 入站不通则层图只能靠 push，探活长期失败。
- **How to apply**: Loki `container_heartbeat_probe_fail`；`compute_event_build.go`；云控制台安全组。

### OPT-20260812-057 — 重生并发布含 Docker 国内镜像回退的 UserData 模板

- **Status**: pending
- **Blocked-By**: OPS
- **Created**: 2026-08-12
- **Context**: 诊断阿里云 ECS `init_from_task2app.sh.log` 时确认根因是 UserData 仅拉 `download.docker.com` 导致 TLS reset、Docker 未装、容器从未启动。生成器已改为阿里云/清华优先+官方兜底；2026-08-13 已 rebuild/restart ai-provider（dist 含 mirrors.aliyun）；清库后 `ai_provider_userdatatemplate` 为 **0 行**，须在厂商门户重新生成并保存模板后新启实例。
- **Action**: (1) ~~发布/部署 taskAiProvider 前端~~（已完成）；(2) 在厂商门户对受影响 OS（ubuntu/debian/centos）重新「生成脚本」并保存 UserData 模板；(3) 用新模板启一台阿里云实例，确认日志出现 `Docker CE 已从镜像安装: https://mirrors.aliyun.com/...` 且容器 running；(4) 对仍卡在旧脚本的实例重建或手工改用阿里云 docker-ce 源。
- **Why**: 只改生成器不重生模板，线上继续注入旧脚本，故障会复现。
- **How to apply**: `taskAiProvider/frontend/src/utils/userDataScriptLinux.js`；Admin UserData 模板页；对照 `.ai/09_failure_experience/02_runtime_errors/91_userdata_docker_ce_download_tls_reset.md`。

### OPT-20260814-003 — UserData docker run 挂载宿主机 runtime 目录以免 rm+run 丢失 refresh 落盘

- **Status**: pending
- **Blocked-By**: OPS
- **Created**: 2026-08-14
- **Context**: `userDataScriptLinux.js` 每次启机 `docker rm` 同名容器再 `docker run`，不挂载 `ONLINE_PROJECT_STATE_ROOT`，`container_refresh_token.json` 随容器文件系统消失。幂等 exchange-refresh 可自愈，但落盘 refresh 仍有利于离线 `refresh-access` 与 go_relay 同步。
- **Action**: (1) 在 Linux/Windows UserData 生成器为 runtime 目录加 host volume；(2) 单测断言 `docker run` 含该 `-v`；(3) 厂商门户重生并保存模板。
- **Why**: 仅靠 credential 幂等仍每次打到 SaaS 换票；volume 让重建容器本地即可 refresh-access。
- **How to apply**: `taskAiProvider/frontend/src/utils/userDataScriptLinux.js`；`userDataScriptWindows.js`；重生模板见同文件 OPT-20260813-005。

### OPT-20260815-004 — 补齐微信商户序列号、APIv3 密钥与私钥使 live Native 真正可用

- **Status**: pending
- **Blocked-By**: OPS
- **Created**: 2026-08-15
- **Context**: `conf/billing/wechatPay/conf.yaml` 已切 `mode: live`，`appid`/`notify_url` 已填；`mock-complete` 已删除。仓库仍无 `merchant_serial_no`、`api_v3_key`、`apiclient_key.pem`。live 初始化失败时支付禁用（不再回退 mock）。
- **Action**: (1) 将商户序列号、APIv3 密钥、`apiclient_key.pem` 写入 `conf-local/billing/wechatPay/` 或 `WECHAT_PAY_MERCHANT_SERIAL_NO` / `WECHAT_PAY_API_V3_KEY` / `WECHAT_PAY_PRIVATE_KEY_PATH` (2) 精准重启 `task-bill` (3) Loki 确认 `WeChat Pay live client ready`，无 `pay disabled` (4) 真实扫码入账
- **Why**: 缺这三项则 `initWechatLiveClient` 失败，下单返回「微信支付未配置」。
- **How to apply**: `taskBill/src/wechat_pay.go` `overlayWechatLocalConfig` / `applyWechatEnvOverrides` / `initWechatLiveClient`。

### OPT-20260815-014 — 回收 start-vm persist 失败留下的阿里云孤儿实例

- **Status**: pending
- **Blocked-By**: OPS
- **Created**: 2026-08-15
- **Context**: 2026-08-15 13:11 任务 `task_876332280065323008` 评论 `cmt_876332298901942272` 在评论 CSC 未创建时仍调用了 RunInstances，实例 `i-m5e9c5iq9ku9wa2dw5t4` 已创建但未写入 `cloud_server_configs`（Loki `start_vm_instance_persist_zero_rows` / start-vm 502）。代码已禁止该路径再出现，存量实例仍在云上计费。
- **Action**: (1) 在阿里云控制台/CLI 确认实例 `i-m5e9c5iq9ku9wa2dw5t4` 状态与地域；(2) 若无其它任务绑定则 Stop+Release；(3) Loki `{job="task-cloud-service"} |= "i-m5e9c5iq9ku9wa2dw5t4"` 确认无后续成功 persist。
- **Why**: 未绑定 CSC 的实例无法从任务详情停机，会持续产生云费用。
- **How to apply**: 阿里云 ECS `i-m5e9c5iq9ku9wa2dw5t4`；对照 trace `46cc14ef91909a8afcb3e9bd` / `task_876332280065323008`。

### OPT-20260818-009 — 排查实例 i-m5eiyydteuxzy5lurni0 :8080 未监听（SSH 可达）

- **Status**: pending
- **Blocked-By**: OPS
- **Created**: 2026-08-18
- **Context**: Trace `dc780ea661e49d92fb84df21`：`118.190.106.76:22` 通、`:8080` connection refused；boot-progress 曾成功但无 register-reachability/heartbeat。容器通信失败的主机侧根因。
- **Action**: (1) Cloud Assistant / SSH 查 docker/onlineServiceJS 是否运行、端口映射；(2) 必要时 `docker restart` 或重建评论容器；(3) 确认 register-reachability 成功后 binding 才 running。
- **Why**: 代码门禁修好后，若宿主机业务端口仍未起，通信仍失败。
- **How to apply**: 实例 `i-m5eiyydteuxzy5lurni0`；评论 `cmt_877417508544475136`；任务 `task_877417492908109824`

### OPT-20260817-020 — 精准编译重启 taskFE 后验收工作面板 auto_run 横幅不再用列表 stub 误报

- **Status**: pending
- **Blocked-By**: OPS
- **Created**: 2026-08-17
- **Context**: 工作面板打开任务详情时曾用列表 `comments:[]` 误报「Git / 子 Git 探测失败」。源码已改为始终 `fetchTaskDetail` 且仅在 `comments_feeds_loaded` 后展示存量横幅；`dist/` 已构建。现网 `task_877108648822730752` 评论确为空，nested-git 仍返回 OAuth 刷新超时（GitHub App 未安装到 task2money，见 BLOCK OPT-20260812-025）。
- **Action**: (1) 在 http://10.2.150.68:9999/ 点击「精准编译重启」（已登记 taskFE）(2) 硬刷新工作面板，打开**已有【自动运行】评论**的任务，确认无琥珀色误报横幅 (3) 打开 `task_877108648822730752`：feed 加载后横幅文案不再含「探测失败」；点「强制重新启动」后应落库真实 skip_reason
- **Why**: 未重启则公网仍跑旧包；本任务服务器未启动的根因是 Git 授权刷新超时，代码无法替代组织安装 GitHub App。
- **How to apply**: `.runall/precise_restart_services.txt`；`TaskDetailAutoRunSkipBanner.vue`；OPS `OPT-20260812-025`

### OPT-20260825-022 — 微信支付商户后台填写分账动账通知 URL

- **Status**: pending
- **Blocked-By**: OPS
- **Created**: 2026-08-25
- **Context**: 接口已上 `POST https://www.daydaymoney.com/api/billing/profitsharing/change-notify/`（machine、APIv3 验签解密）。商户平台「分账动账通知设置」仍须人工配置 HTTPS、无 query。
- **Action**: (1) 登录微信支付商户平台分账动账通知设置页 (2) 填入上述 URL (3) 用一笔测试分账确认 notify.id 写入 `billing_profit_sharing_change_notify`。
- **Why**: 不配置则微信不会推送动账结果，超管列表微信分账单号只能靠主动同步/出站 CreateOrder 回写。
- **How to apply**: 商户平台 cincomsplit 通知 URL；网关路由 `billing-profitsharing-notify`；taskBill `handleProfitSharingNotify`。

### OPT-20260825-003 — 轮换曾入库的阿里云短信 AccessKey

- **Status**: pending
- **Blocked-By**: OPS
- **Created**: 2026-08-25
- **Context**: SonarQube secrets:S6336 在已跟踪的 `conf/core/sms/config.yaml` 及 sync 产物中检出 Aliyun AccessKey。密钥已迁到 gitignored `conf-local/`，但旧值仍在 Git 历史中，按元规则 61 视为已泄露。
- **Action**: (1) 在阿里云 RAM 禁用并删除旧 AccessKey；(2) 签发新 Key 写入本机 `conf-local/core/sms/config.yaml` 及 sync 镜像的 conf-local 片段；(3) 跑 `python3 runAll/scripts/conf-sync.py auth/task-auth`；(4) 精准重启 taskAuth 后发一条验证码确认。
- **Why**: 历史提交里的云密钥无法靠改文件收回，只能轮换。
- **How to apply**: 阿里云控制台 RAM 用户/密钥；本地 SSOT `conf-local/core/sms/config.yaml`；taskAuth `ReadAppFragment` 合 `conf-local` 片段。

### OPT-20260825-006 — 用 Browse 权限 token 评审 ram-work 未审 Security Hotspots

- **Status**: pending
- **Blocked-By**: OPS
- **Created**: 2026-08-25
- **Context**: 夜间 `sonarqube.sh` 使用项目分析 token，Quality Gate 因 `new_security_hotspots_reviewed=0%` 失败；hotspots API 对该 token 返回 403，无法在扫描会话内标 reviewed。
- **Action**: (1) 用具备 Browse 的用户 token 打开 `http://10.2.150.68:19000/security_hotspots?id=ram-work`；(2) 评审 new-code 窗口内 hotspot 并标记 Safe/Fixed；(3) 复跑 Quality Gate 确认该条件通过。
- **Why**: 分析 token 不能改 hotspot 状态，门禁会一直红。
- **How to apply**: SonarQube 9.9 项目 `ram-work`；不要把用户 token 写进 `sonarqube.sh`。

### OPT-20260823-052 — 为 GitLab Workhorse 配置 real_ip，避免同区域 HTTP clone 被当成 127.0.0.1

- **Status**: pending
- **Blocked-By**: OPS
- **Created**: 2026-08-23
- **Context**: 流量闸门已改为：公网 Host + Docker NAT/loopback 走配额；10/8 与 192.168 源 IP 视为同区域内网。若 Workhorse 把客户端做成 `127.0.0.1`，同区域用户用公网域名 HTTP clone 会被误拦截。
- **Action**: (1) 在 gitService Omnibus 配置 `nginx['real_ip_trusted_addresses']` / `gitlab_workhorse['trusted_cidrs']` 指向 Docker 网桥 (2) 用容器探针确认 `Gitlab::RequestContext.instance.client_ip` 为 VPC 地址而非 127.0.0.1 (3) 文档写明内网域名仍可用 `TRAE_GITLAB_INTRANET_HOSTS`。
- **Why**: 不配 real_ip 时，同区域 HTTP 走公网 Host 会与任务节点一样被配额阻断，只剩 SSH 或私网 Host 能拉。
- **How to apply**: `gitService/docker-compose.yml` GITLAB_OMNIBUS_CONFIG；`initializers/zzz_trae_gitlab_traffic_quota.rb` `skip_quota?`；`conf/infra/git-service*/config.yaml`

### OPT-20260823-001 — 商户平台开通电子发票后将 fapiao.enabled 改为 true

- **Status**: pending
- **Blocked-By**: OPS
- **Created**: 2026-08-23
- **Context**: 本会话已落地申请/审批/红冲重开代码，但 `conf/billing/wechatPay/conf.yaml` 的 `fapiao.enabled` 默认为 false，避免未开通税控时审批调用微信返回 NO_AUTH。
- **Action**: (1) 在微信支付商户平台开通电子发票并配置卡券模板 (2) 核对 tax_code/tax_rate 与经营范围一致 (3) 将 `fapiao.enabled` 改为 true 并精准重启 task-bill (4) 用一笔已支付微信订单走通申请→审批→卡包
- **Why**: 开关关闭时管理员审批会失败「电子发票未开通」，功能对公网不可用。
- **How to apply**: `conf/billing/wechatPay/conf.yaml` `fapiao`；文档 https://pay.weixin.qq.com/doc/v3/merchant/4012538301

### OPT-20260823-002 — 用真实退款验证冲红后同一 transaction_id 能否重开蓝票

- **Status**: pending
- **Blocked-By**: OPS
- **Created**: 2026-08-23
- **Context**: WITH_WECHATPAY 要求 `fapiao_apply_id` 等于微信支付单号。退款后重开仍复用该单号；单测已 mock，未打真实商户。
- **Action**: (1) 开具原蓝票并等 FAPIAO.ISSUED (2) 审批一笔部分消耗后退款 (3) 确认 reverse 成功后再 issue 剩余金额 (4) 若 RESOURCE_ALREADY_EXISTS 则改查询或换 scene
- **Why**: 微信可能禁止同一 apply_id 二次开具，届时剩余成交金额无法推送新蓝票到卡包。
- **How to apply**: `taskBill/src/wechat_fapiao.go` `issueWechatFapiaoImpl` / `ReconcileInvoicesAfterRefund`

### OPT-20260823-005 — 开通税控后用真实退款验收卡包 72 小时冲红确认提醒

- **Status**: pending
- **Blocked-By**: OPS
- **Created**: 2026-08-23
- **Context**: 本会话已让退款冲红保持 `reverse_pending` 并在订单 JSON/详情页露出 72h 截止；`fapiao.enabled` 仍为 false，无法走真实卡包确认。
- **Action**: (1) 完成 OPT-20260823-001 开通电子发票 (2) 对已开蓝票订单审批退款 (3) 打开订单详情断言 `order-invoice-reverse-confirm-hint` 含 72 小时与截止时间 (4) 在微信卡包确认冲红后刷新，提醒消失且蓝票为已冲红
- **Why**: 72h 是购方确认窗口，单测无法覆盖微信卡包与回调时序。
- **How to apply**: `invoice_reverse_confirm.go`；`OrderInvoiceSection.vue`；订单 `invoice_reverse_confirm`

### OPT-20260818-016 — SystemAdmin 填写 tencent-sh-1 PAT 并验收 OIDC

- **Status**: pending
- **Blocked-By**: OPS
- **Created**: 2026-08-18
- **Context**: Host `sh` 上 `gitlab-tencent-sh-1` 已 healthy，公网 `https://gitlab-tencent-sh-1.daydaymoney.com/users/sign_in` 返回 200。seed 行 `admin_private_token` 为空，hybrid 开通无法调 Admin API。
- **Action**: (1) 在 SH GitLab 创建管理员 PAT (2) SystemAdmin 区域表写入 `tencent-sh-1` 的 token (3) 用平台账号走 OIDC 登录该实例 (4) 对测试租户点开通/重试，确认 `provisioning_status=active`
- **Why**: 无 PAT 时购买只能落到 `pending_admin`，新区域对租户不可用。
- **How to apply**: `billing_gitlab_region.admin_private_token`；OIDC client `gitlab-git-service-tencent-sh-1`（`conf/auth/task-auth/config.yaml`）；容器 `/opt/daydaymoney/gitservice-tencent-sh-1`

### OPT-20260815-016 — 精准编译重启 task-credential-service 并重建该任务容器以验证关闭自动克隆

- **Status**: pending
- **Blocked-By**: OPS
- **Created**: 2026-08-15
- **Context**: 任务详情开关已关，但启动日志仍在克隆子仓。根因是生产 `SQLiteBusinessRepository` 写死 `AutoCloneNestedRepos=true`，镜像 `collectRepoCloneJobs` 也未按 flag 跳过 nested。代码已修，已登记精准编译重启；现网容器仍是旧二进制/旧镜像。
- **Action**: (1) ~~在 http://10.2.150.68:9999/ 对 task-credential-service 执行「精准编译重启」~~ **已完成（2026-08-16 夜间）**：4 built 0 failed，taskFE/task-credential-service/task-task-service/task-container-gateway 全部 healthy；(2) ~~Docker 推送~~ 已完成：`registry.cn-qingdao.aliyuncs.com/ruandao/task2app-trae:x86_64_2026-08-15_19-44` 与 `x86_64-latest`；(3) **仍须**重建任务 `task_876416048071471104` 的容器，确认启动日志只有父仓克隆、无并行克隆子仓，且可见 `bootstrap-clone skip nested repos`。
- **Why**: 不重启则现网仍走旧路径，页面开关与容器行为继续不一致。
- **How to apply**: 服务名 `task-credential-service`；镜像目录 `trae-agent/onlineServiceJS`；验收页 `https://www.daydaymoney.com/tenant/875588283562749952/workspace/ws_-2740859684112864748/task-detail/task_876416048071471104/`。**2026-08-16**：任务详情开关已迁到评论 composer 身份行（`@镜像` 后 `comment-composer-repo-identity-row` 内 `task-nested-repos-auto-clone-toggle`），关联项目只读区不再挂该控件。

### OPT-20260816-048 — 提交后推送 trae-agent 镜像并重建任务容器以生效 LLM 404 重试

- **Status**: pending
- **Blocked-By**: OPS
- **Created**: 2026-08-16
- **Context**: 任务 `task_876757038493888512` 评论执行日志出现 `step 1: 错误: Error code: 404`。根因是容器内 LLM 客户端把网关 404 当永久失败。代码已在 `retry_utils.py` / `saasPostJson.mjs` 修好，但公网任务容器仍跑旧镜像。
- **Action**: (1) 提交 `trae-agent` 变更 (2) 在 `trae-agent/onlineServiceJS` 执行 `DOCKER_PUSH=1 ./buildDocker.sh` (3) 重建该任务评论容器，确认 step 1 不再因单次 404 立即失败
- **Why**: 不推镜像、不重建容器则现网仍会把 SaaS 重启窗口的 404 写成 step 错误并结束 job。
- **How to apply**: 镜像目录 `trae-agent/onlineServiceJS`；验收页 `https://www.daydaymoney.com/tenant/875588283562749952/workspace/ws_-2740859684112864748/task-detail/task_876757038493888512/`；trace `ab3d86a8e34788fa01393902`

### OPT-20260817-005 — 推送 onlineServiceJS 镜像并重建本任务容器

- **Status**: pending
- **Blocked-By**: OPS
- **Created**: 2026-08-17
- **Context**: 本次已改引导克隆全失败文案（1/1 不再写「其余已就绪」），逻辑在容器镜像 `trae-agent/onlineServiceJS`。现网任务 `task_876895044743753728` 的 ram-work 克隆失败摘要仍是旧镜像文案。
- **Action**: (1) 在 `trae-agent/onlineServiceJS` 执行 `DOCKER_PUSH=1 ./buildDocker.sh` (2) 重建该任务容器 (3) 确认失败摘要为「均失败」且不含「其余已就绪」
- **Why**: 只发前端无法改容器内 SSE/启动日志文案；旧镜像会继续误导「其余已就绪」。
- **How to apply**: 目录 `trae-agent/onlineServiceJS`；任务页 `https://www.daydaymoney.com/tenant/875588283562749952/workspace/ws_-2740859684112864748/task-detail/task_876895044743753728/`

### OPT-20260819-037 — 清理 hello-world 误双 fork 的重复任务与容器

- **Status**: pending
- **Blocked-By**: OPS
- **Created**: 2026-08-19
- **Context**: 上述双 fork 后两任务仍 `cloud_comment_container_bindings.status=running`（comment `cmt_877835533651308544` / `cmt_877835534007824384`），源任务 `task_877833398922539008`。
- **Action**: (1) 与用户确认保留哪一条 (2) 停止并释放另一条 CSC/binding (3) 视需要归档或删除重复任务帖
- **Why**: 两台并行跑同一「写 hello world」浪费算力；看板也出现重复卡片。
- **How to apply**: 租户 `877397588196749312` / 工作区 `ws_-2309487803472456748`；binding 表 `cloud_comment_container_bindings`

### OPT-20260821-040 — 确认登录微信 AppID 已绑定微信支付商户号

- **Status**: pending
- **Blocked-By**: OPS
- **Created**: 2026-08-21
- **Context**: 推荐资格开通后会用 `wechat_identity` 的 openid 调 `POST /v3/profitsharing/receivers/add`。支付 AppID 是 `wx31273ca77c89dffe`，Web 登录 AppID 是 `wx625802b55b33608f`；openid 按 appid 隔离。若登录 AppID 未绑定该商户号，添加接收方会 `NO_AUTH`，商户平台仍查不到人。
- **Action**: (1) 打开微信支付商户平台「产品中心 / AppID 账号管理」确认 `wx625802b55b33608f` 已关联本商户号 (2) 若未绑定则按微信流程绑定 (3) 有资格用户刷新 `/profile/referral/` 看 `wechat_receiver_status=registered`，并在「交易中心 > 管理分账接收方」核对
- **Why**: 代码已按身份所属 appid 登记；绑定缺失时本地资格仍在、商户后台仍为空。
- **How to apply**: `conf/auth/task-auth/config.yaml` `wechat.apps.web.appId`；`conf/billing/wechatPay/conf.yaml` `appid`；微信文档 https://pay.weixin.qq.com/doc/v3/merchant/4012528995

### OPT-20260822-058 — 重建运行中容器以使 job.finished_at 进入层图快照

- **Status**: pending
- **Blocked-By**: OPS
- **Created**: 2026-08-22
- **Context**: 任务关联摘要已能展示「最近指令完成 pending/时间」，但 `finished_at` 由容器内 onlineServiceJS 在 job 终态写入。已在跑的旧镜像不会给历史 completed job 补时间戳，摘要会显示 `—`。
- **Action**: (1) 推送含 `stampJobFinishedAt` 的 onlineServiceJS 镜像 (2) 对仍跑旧镜像的任务容器按既有迁移/重建流程换新镜像 (3) 新指令结束后确认摘要出现本地完成时间而非 `—`
- **Why**: 只更前端时，进行中指令已能显示 pending；完成时间依赖容器进程写入字段。
- **How to apply**: `trae-agent/onlineServiceJS` `DOCKER_PUSH=1 ./buildDocker.sh`；对照 taskCloudService 容器镜像更新路径

### OPT-20260823-057 — 误送订单 ORD-...879129787220656128 配额回收治理

- **Status**: pending
- **Blocked-By**: OPS
- **Created**: 2026-08-23
- **Context**: 根因分析确认 ORD-20260822-877397588196749312-879129787220656128 为管理员只改 VIP 时误提交默认 task_post×100 生成的零元赠送订单；该 100 帖配额尚未消耗（设计决策：不自动回滚，防复发优先）。
- **Action**: (1) 与业务确认租户 877397588196749312 是否保留该 100 帖（可能已被团队利用） (2) 若回收：提供后台冲正工具（admin_grant 反向操作：按订单扣减未消耗配额 + 标记订单 cancelled），或手工 SQL 由 DBA 执行 (3) 在管理端赠送页「最近赠送记录」展示生成订单号，便于事后核对
- **Why**: 100 帖为实打实的免费配额，长期不治理会累积账实不符；回收动作需人工确认后执行。
- **How to apply**: `taskBill/src/handlers_admin_grant.go`（冲正端点或脚本）；`SystemAdminGrantPoints.vue` 结果提示带订单号

### OPT-20260823-060 — 租户 877397588196749312 在 tencent-sh-1 区域 GitLab 组缺失

- **Status**: pending
- **Blocked-By**: OPS
- **Created**: 2026-08-23
- **Context**: `billing_tenant_gitlab_resource` 中该租户在 tencent-sh-1 存在配额行（disk_gb=1, active），但远程 gitlab-tencent-sh-1 实例上无任何 `tenant-*` 组——租户组开通流程在该区域未执行或丢失。
- **Action**: (1) 确认租户组开通入口（taskBill gitlab_region provisioning 链路）在该租户/区域的状态 (2) 按正式流程创建 `tenant-877397588196749312` 组并同步成员（本次验证用临时组已清理） (3) 部署类巡检：各区域实例租户组数量 vs 配额行数量核对
- **Why**: 无组则租户项目全部走「未映射 fail-open」，公网克隆不受配额管控——正是本漏洞在区域实例上的残余威胁面。
- **How to apply**: `taskBill/src/gitlab_region.go`（provisioning）；远程 `docker exec gitlab-tencent-sh-1 gitlab-rails runner`；`billing_tenant_gitlab_resource`

### OPT-20260824-082 — 真实 git HTTP 全链路验证（OIDC/task identity token）+ taskBill gate 日志补入参

- **Status**: pending
- **Blocked-By**: OPS
- **Created**: 2026-08-24
- **Context**: 修复已在线验证到 Rails 进程级（enforce! 抛 ForbiddenError，project_path/username 正确提取；CI 跳过保持）。剩余：真实 git-upload-pack HTTP 请求（info/refs → GitAccess → initializer → taskBill）的浏览器/客户端级验证未闭环 —— SH GitLab 实例 git HTTP basic auth 只接受 OIDC 生态凭据（本地 PAT 401，密码认证全局禁用 pw_git=false），需用 task git identity（task_repo_identities → task_git_identities，token 由 taskGitOauth refresh 端点动态签发）发起克隆。另：taskBill 的 gitlab_traffic_gate 结构化日志不含 project_path/gitlab_username/from_ci/is_intranet 入参，排查「为何 UNMAPPED」时只能靠 Loki 时间对齐推断。
- **Action**: (1) 从 task 容器/任务节点用 task git identity 对 gitlab-tencent-sh-1.daydaymoney.com 的 example-user/somanyad 发起真实 git clone，确认 403 阻断（CI/同区域 VPC 请求确认放行）；(2) taskBill handleInternalGitlabTrafficGate 的 slog 增加 project_path/gitlab_username/from_ci/is_intranet 字段。
- **Why**: 端到端证据链最后一环（Rails HTTP 入口 → GitAccess → 闸门）仍需真实客户端请求确认；入参日志缺失使线上排查效率低。
- **How to apply**: 任务节点（阿里云）配置 git 凭据后 clone；Loki 查 gitlab_traffic_gate 日志确认 allowed=false + tenant_id；taskBill src/gitlab_traffic_gate.go handleInternalGitlabTrafficGate slog.InfoContext 补字段。

### OPT-20260826-007 — 排查服务号关注回调 POST 未到达 task-auth

- **Status**: pending
- **Blocked-By**: OPS
- **Created**: 2026-08-26
- **Context**: 用户于 17:20:04 CST 扫动态 scene 码（粉丝 `qr_scene_str`=票 `880376598551883776`），当时网关/task-auth 已健康，但 `logs/task-auth.log` 无任何 POST `/api/auth/wechat/mp/callback/`。follow-status 已用 `qr_scene_str` 补偿绑定成功；后续扫码仍应走微信推送。
- **Action**: (1) 核对公众平台服务器 URL、加密模式、是否只验签 GET (2) 查 APISIX/边缘 nginx 17:20 附近微信网段入站 (3) 用带参码再扫一次确认出现 `wechat_mp_subscribed` (4) 非数字 `user_id`（如 bootstrap-admin）写入 BIGINT 票表前拒绝，避免 1366。
- **Why**: 回调丢失会让未点「我已关注」的用户一直 pending；补偿依赖粉丝列表接口配额。
- **How to apply**: 微信公众平台后台 URL；`taskGateway` callback 路由；`handleWeChatMPFollowQR` 对 `user_id` 做数字校验；对照 17:20 CST 的 access 日志。

### OPT-20260817-039 — 提交并推送含 delivery 短路修复的 onlineServiceJS 镜像

- **Status**: pending
- **Blocked-By**: OPS
- **Created**: 2026-08-17
- **Context**: TraceId `6a2a9c37101acc53f59a8487` 显示 auto_run job 已 completed 但无 `AUTO_RUN_DELIVERY_*`。根因是 close 钩子被空 `mountedAgentId` 短路；代码已修于 `jobsRuntimeCloseSideEffects.mjs`，现网容器仍跑旧逻辑。
- **Action**: (1) 提交 `trae-agent` 变更（含 CloseSideEffects 模块与回归测）(2) 在 `trae-agent/onlineServiceJS` 执行 `DOCKER_PUSH=1 ./buildDocker.sh` (3) 对该任务评论容器 stop-vm→start-vm 或重建，确认 Loki 出现 `AUTO_RUN_DELIVERY_BEGIN`
- **Why**: 不推镜像则公网任务 completed 后仍永不自动 commit/PR。
- **How to apply**: 失败经验 `101_autorun_delivery_skipped_empty_mounted_agent.md`；验收任务 `task_877171547301769216`

### OPT-20260816-040 — 现网生效：迁移 023 + 精准编译重启 + 推送容器镜像

- **Status**: pending
- **Blocked-By**: OPS
- **Created**: 2026-08-16
- **Context**: 执行步骤已改为容器 PUSH → Kafka SSE_MESSAGE → SSE 直播 + persist 落库，但现网进程/库表/容器镜像仍是旧路径。无表则 GET hydrate 失败；旧容器不会 POST `job-stream-push`。
- **Action**: (1) 在 http://10.2.150.68:9999/ 「初始化全部数据库」应用 `dataMigrate/taskCloudService/023_cloud_job_execution_event.sql` (2) 「精准编译重启」task-cloud-service、taskFE、task-container-gateway、task-agent-support、task-events-sse-message-2-persist-job-execution-event (3) 在 `trae-agent/onlineServiceJS` 执行 `DOCKER_PUSH=1 ./buildDocker.sh` 并重建任务容器
- **Why**: 代码合入后不迁表/不重启/不换镜像，页面仍只能看到容器内存日志或空白步骤。
- **How to apply**: 迁移目录 `dataMigrate/taskCloudService/`；runAll 服务名见 Action；镜像脚本 `trae-agent/onlineServiceJS/buildDocker.sh`

### OPT-20260816-062 — 存量任务级 ECS InstanceName 不会自动改成评论级

- **Status**: pending
- **Blocked-By**: OPS
- **Created**: 2026-08-16
- **Context**: 新启机已把阿里云 `InstanceName` 定为评论级 `task_{taskId}_{commentId}`；Describe/heal/orphan **只按该名**查找，不再回退任务级/`task-{taskId}`。已在跑、仍挂任务级名的实例控制台对不上，也不会被评论名 Describe 找到。
- **Action**: (1) 精准编译重启 task-cloud-service 与 task-events-cloud-server-started-1-process-server-start (2) 新评论冷启动后在 ECS 控制台核对 InstanceName 等于该评论容器名 (3) 需要对照查找的存量机：释放后按评论重启，或对实例做 `ModifyInstanceAttribute` 改名（不作为启动路径默认行为）
- **Why**: 只改 RunInstances 不影响已创建实例；控制台仍会把同任务多评论挤在同一个任务名下，验收时容易误判「没改」。
- **How to apply**: 命名 SSOT `ecsInstanceName` / `aliyun.ECSInstanceName`；查找 `ecsInstanceNames`；任务 `task_876810593758113792`
