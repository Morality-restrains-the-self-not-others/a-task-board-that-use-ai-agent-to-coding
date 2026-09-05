# Completed OPT Archive — 2026-08-13

> 从 OPTIMIZATION_TODOS_COMPLETED.md 按天归档，共 51 条。
> 归档执行时间：2026-08-14T11:23:07+08:00

## [OPT-20260812-040] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: dockerInfra/kafka compose kafka+zookeeper restart unless-stopped，验证 compose config 合法并推送 9d48c28
- **Created**: 2026-08-12
- **Context**: `kafka-kafka-1` 曾 `Exited (137)` 且 compose `restart: "no"`，导致 EMAIL_SENT 等投递中断直至人工 `run.sh start`。
- **Action**: (1) 评估将 `dockerInfra/kafka/docker-compose.yml` 的 kafka/zookeeper 改为 `restart: unless-stopped`；(2) 确认与 runAll stop 语义不冲突；(3) 可选加 OOM/内存限制与告警。
- **Why**: broker 静默退出会使验证码/邀请邮件等依赖 Kafka 的路径退化到 SMTP 或直接失败。
- **How to apply**: `dockerInfra/kafka/docker-compose.yml`；对照 `70_kafka_broker_down_ui_healthy_false_positive.md`。

## [OPT-20260812-032] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: taskEvents run.sh port_for_path 缺 port 显式报错+exit1，Go 包装测绿，推送 209eafb
- **Created**: 2026-08-12
- **Context**: member_joined 缺 port 时 `port_for_path` 失败，`start` 路径曾表现为静默不监听，runAll 仅报 READINESS_TIMEOUT，排障成本高。
- **Action**: (1) 审查 `taskEvents/run.sh` `start_intent`/`port_for_path` 在 `set -e` 下的失败传播；(2) port 缺失时 `echo` 明确错误到 stderr 并以 exit 1 结束；(3) 补 bash 级或 go 包装测。
- **Why**: 让 runAll「recent stderr」直接指向缺 port，而不是 60s 超时后才暴露。
- **How to apply**: `taskEvents/run.sh` `port_for_path` / `start_intent`。

## [OPT-20260812-014] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: dockerInfra/mysql detect_corrupt_data.sh + run.sh start 前置重建，5 例离线测试绿，推送 ef4a582
- **Created**: 2026-08-12
- **Context**: `/tmp/ram-work` 为 tmpfs；本次 `docker-mysql` 数据目录仅剩空壳（无 `ibdata1`，总量约 20K），mysqld 报 `Failed to find valid data directory` 并 exit 1，级联 10+ 业务服务因 `3306 connection refused` 失败。手动搬迁 `data.corrupt.*` 后冷启动才恢复。
- **Action**: (1) 在 `dockerInfra/mysql/run.sh` 或 health 前置检测缺 `ibdata1`/InnoDB 系统表空间；(2) 自动将损坏目录挪到 `data.corrupt.<ts>` 并空目录冷启动，让 `init/01-create-databases.sql` 重建库；(3) 日志打印明确告警，避免 runAll 空等 180s readiness timeout。
- **Why**: tmpfs/异常中断后空壳目录会反复卡死全站依赖 MySQL 的服务。
- **How to apply**: `dockerInfra/mysql/run.sh`；`dockerInfra/mysql/health.sh`；对照本次 `data.corrupt.20260812_110500`。

## [OPT-20260812-033] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: gitService load_gitservice_config 抽独立模块+5单测+run.sh 调用+meta 测更新，推送 cf8e238db
- **Created**: 2026-08-12
- **Context**: `load_gitservice_config` 内嵌双分支 Python，conf-read 成功路径曾漏定义 `mem_limit` 导致 NameError；现已双路径赋值并用测例兜住，但逻辑仍深埋 heredoc。
- **Action**: (1) 将解析逻辑抽到 `gitService/scripts/load_gitservice_config.py`；(2) run.sh 调用该脚本；(3) 单测直接 import 覆盖 conf-read JSON / YAML fallback。
- **Why**: 降低 heredoc 双路径漂移复发概率，便于后续资源键扩展。
- **How to apply**: `gitService/run.sh`；`conf/infra/git-service/test_config_resource_keys.py`。

## [OPT-20260812-036] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: taskGitOauth 删除 infrastructure canonicalizeTenantRedirectURI 副本改调 domain 单一实现，全量测试绿，推送 c783cac
- **Created**: 2026-08-12
- **Context**: `domain.CanonicalTenantRedirectURI` 与 `infrastructure.canonicalizeTenantRedirectURI` 逻辑重复（infrastructure 故意不依赖 domain 以避免循环）。当前行为一致但双份维护。
- **Action**: (1) 评估是否允许 infrastructure → domain 单向依赖；(2) 若可，删除 infrastructure 副本并调用 domain；(3) 保留单测覆盖遗留共享 URI 改写。
- **Why**: 双份实现易漂移，后续改路径格式时可能只改一处。
- **How to apply**: `taskGitOauth/domain/tenant_connection.go`；`taskGitOauth/infrastructure/provider_resolve.go`。

## [OPT-20260812-029] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: taskTenantService user-in-admin-groups 内部 API + taskTaskService canManageMemberGitIdentities 组边界校验，两侧单测绿，推送 c312d8b/1537646
- **Created**: 2026-08-12
- **Context**: 当前 `group-members:manage` 可代管公司内任意成员 Git 身份，未校验目标成员是否在该组管理员所管组。
- **Action**: (1) 代管前查 actor 的 group_admin 组集合；(2) 校验目标 member 是否同组；(3) 补单测。
- **Why**: 避免组管理员越权管理其他组员身份。
- **How to apply**: `canManageMemberGitIdentities` + tenant group membership internal API。

## [OPT-20260812-038] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: taskTenantService sync-nickname 端点 + taskAuth ensureWechatProfileNickname 后转发同步，两侧单测绿，推送 75a23f6/e0f375b
- **Created**: 2026-08-12
- **Context**: 微信注册先发 USER_CREATED（空 username）再建公司，再 `ensureWechatProfileNickname` 写个人昵称；竞态下创建者 `member_name` 可能仍为空。读路径可回退，但写入时未把昵称同步到成员行。
- **Action**: (1) 在 `ensureWechatProfileNickname` 成功后，若该公司成员 `member_name` 为空或为「我的公司」，PATCH 更新为个人昵称；(2) 补单测覆盖竞态顺序。
- **Why**: 减少对读路径回退的依赖，Git 身份/事件载荷等也直接拿到正确昵称。
- **How to apply**: `taskAuth/src/auth_wechat.go` → `ensureWechatProfileNickname`；转发 `PATCH /api/internal/tenant/members/update`。

## [OPT-20260812-030] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: db/scripts/ci 新增 check_nfr_path_shard_table.py + 自测，默认 warn 存量容忍，--strict 阻断新文档缺表
- **Created**: 2026-08-12
- **Context**: 元规则 43 与 `/5-nfr` 已强制要求 NFR 文档含「路径分片键审视」表，但尚无自动化门禁，易在评审遗漏。
- **Action**: (1) 在 `db/scripts/ci/` 或 `docs/scripts/` 增加检查：`docs/superpowers/plans/*-nfr-clarification.md` 须含 `## 路径分片键审视`；(2) 可选校验表头列齐全；(3) 接入 pre-commit 或文档 CI（可对存量文件先 warn）。
- **Why**: 仅靠技能文案无法保证每次产出都含审视表。
- **How to apply**: 对照 `.claude/skills/5-nfr/SKILL.md` Hard Gate；元规则 `48_nfr_path_shard_id_scalability.md`。

## [OPT-20260812-034] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: taskFE vite.config.js 生产 minify 改 esbuild（41d7831 已推送），实测构建 7.1s 完成
- **Created**: 2026-08-12
- **Context**: 修复系统管理侧栏滚动时执行 `npm run build`，默认 `minify: terser` 在本机内存紧张（可用约 2–3Gi）时于 `rendering chunks` 被 OOM Kill；改用 `--minify esbuild` 可稳定完成原子构建。
- **Action**: (1) 将 `taskFE/app/vite.config.js` 的 `build.minify` 从 `'terser'` 改为 `'esbuild'`（或按环境变量切换）；(2) 删除或条件化仅 terser 需要的 `terserOptions`；(3) 跑一次 `npm run build` 与 `bash scripts/test_build_clean_dist.sh` 确认仍原子替换 dist。
- **Why**: terser 峰值内存显著高于 esbuild；CI/本机在高负载下反复 Kill 会导致公网 SPA 无法更新，与「改 src 必须 build」硬约束冲突。
- **How to apply**: 文件 `taskFE/app/vite.config.js`（约 356–361 行）；验收：`cd taskFE/app && npm run build` exit 0 且 `dist/index.html` 更新。

## [OPT-20260812-015] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: atomic-vite-build.sh 缺 vite 自动 npm ci + 行为探针（252e271 已推送）
- **Created**: 2026-08-12
- **Context**: 本次「全部重新编译」仅 `taskFE` 失败：`node_modules/.bin/vite` 缺失（exit 127）。`atomic-vite-build.sh` 已提示 `npm ci`，但 runAll `build_command` 不会自动补依赖，需人工介入。
- **Action**: (1) 在 `taskFE/app/scripts/atomic-vite-build.sh` 或 `runall-lifecycle.sh build` 检测到 vite 不可执行时自动 `npm ci` 再构建；(2) 失败时把完整 npm 日志回灌 runAll progress error；(3) 可选：runAll buildable JS 服务统一 preflight。
- **Why**: tmpfs/清依赖后「全部重新编译」不应因缺 node_modules 单点失败。
- **How to apply**: `taskFE/app/scripts/atomic-vite-build.sh`；`taskFE/app/scripts/runall-lifecycle.sh`；`conf/runAll.yaml` taskFE `build_command`。

---

## D. 产品决策指针（仅防重复）

> 完整正文在 [PRODUCT_DECISIONS.md](./PRODUCT_DECISIONS.md)。决策后回迁本文件再实施；**禁止**在指针条目堆验证笔记。

## [OPT-20260812-016] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: emailBindingDeepLink.js 常量抽取，taskFE 0a7267d + taskAuth eefbeb6 已推送
- **Created**: 2026-08-12
- **Context**: 镜像市场 / SSO bridge / UserProfile 三处硬编码 `profile.email_binding` 与 `/profile/?sso_error=email_required#rg=profile.email_binding`；本次已修复加载后再滚动定位，但字面量仍分散。
- **Action**: (1) 在 taskFE 抽出 `EMAIL_BINDING_RG_KEY` / 引导 URL 构造函数供 ImageMarket、UserProfile 共用；(2) taskAuth `sso_bridge.go` 用同名常量或共享文档注释对齐；(3) 单测断言改为引用常量。
- **Why**: 深链 key 漂移会导致 toast 仍在、区块对不上，且 rgDeepLink 早期重试会静默失败。
- **How to apply**: `UserProfile.vue`；`ImageMarket.vue`；`taskAuth/src/sso_bridge.go`；既有 `UserProfile.ssoEmailRequired.test.js`。

## [OPT-20260812-041] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: runAll runBuild MemAvailable 等待回收 + OOM 降并发重试（10606b5 已推送）
- **Created**: 2026-08-12
- **Context**: `/goal` 全量流程中「全部重新编译」串行仍有 4 个失败：三个 `task-events-cloud-server-*`（编译 `alibabacloud-go/ecs` 时 `compile: signal: killed`）与 `taskFE`（vite rendering chunks `Killed`/exit 137）。当时 MemAvailable≈2.7Gi、Swap 全满、`/tmp/ram-work` 在 16Gi tmpfs。临时停 gitlab/kafka-ui 后本地重编全部成功，随后 conf-sync/清库/初始化/全部启动 59/59 healthy。
- **Action**: (1) 在 `runAll` `BuildService`/`BuildAll` 前读 `/proc/meminfo` MemAvailable，低于阈值（建议 4Gi）则等待回收或自动停非关键容器（kafka-ui，可选 gitlab）再编；(2) 对 exit 137/`signal: killed` 自动有限次重试（串行+降 GOMAXPROCS）；(3) progress SSE 明确标注 OOM 而非笼统 build failed；(4) 与 OPT-026/034 联动（FE esbuild minify + 腾内存）。
- **Why**: 全量编译在内存紧张主机上不可复现地失败，阻断 9999 一键交付；人工腾内存不可作为常态。
- **How to apply**: `runAll/src/runner.go` `BuildAll`/`BuildService`；对照本次失败服务名；Related: OPT-20260812-026、OPT-20260812-034。

## B. 运维与破坏性操作

## [OPT-20260812-031] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: repo-quality-gates.yml 新增 taskevents-intent-sync job 运行 AllIntents↔run.sh↔domain-events 三方同步测试
- **Created**: 2026-08-12
- **Context**: 本会话 `/goal` 全部编译时 `member_joined` 已写入 `AllIntents`/`runAll.yaml`/`cmd/`，但漏登 `run.sh INTENT_PATHS` 且 `config.yaml` 缺 `intents.*.port`，导致编译与启动失败。仓库内已补单元测试，但未接入根仓/子仓 CI 强制门禁。
- **Action**: (1) 将 `taskEvents/config` 的 `TestRunShIntentPathsMatchAllIntents` + `TestDomainEventYAMLPortsMatchAllIntents` 纳入 taskEvents 常规 CI；(2) 可选增加脚本扫描 `conf/runAll.yaml` 中 `task-events-*` 与 AllIntents GroupID 对齐；(3) 新增 intent 的 PR checklist 写明三处同步。
- **Why**: 仅靠人工同步易再现「注册了但跑不起来」的断裂。
- **How to apply**: `taskEvents/config/run_sh_sync_test.go`；`conf/events/domain-events/*/config.yaml`；`taskEvents/run.sh`。

## [OPT-20260812-058] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: 已从 userDataScriptLinux.js 删除 REGISTRY_URL 行；单测断言不再输出；userdata_replace_test 改为仅强制 __TASK2APP_*；ai-provider 已 rebuild+restart，dist 含 aliyun mirror 且无 REGISTRY_URL。重生模板见 OPT-057。
- **Created**: 2026-08-12
- **Context**: 本次故障排查中看到生成脚本仍含 `REGISTRY_URL="{REGISTRY_URL}"`，占位符从未替换且脚本未引用该变量，易误导排障（误以为镜像仓库配置缺失）。
- **Action**: (1) 从 `userDataScriptLinux.js` 删除该行（或改为真正使用的 registry 登录逻辑）；(2) 更新 `userdata_replace_test.go` 中关于 `{REGISTRY_URL}` 的注释/断言；(3) 重生模板验收。
- **Why**: 无用占位增加误判成本，也与「占位符必须在 RunInstances 前替换」的约定冲突。
- **How to apply**: `taskAiProvider/frontend/src/utils/userDataScriptLinux.js`；`taskCloudService/src/userdata_replace_test.go`。

## [OPT-20260812-051] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: panel.js 拆分完成：orchestrator + lib/{panel-core,workspace,branches} + tabs/{single-request,batch,errors,history}，全部 ≤500 行；npm test 287 全绿；taskChromePlugin d631401 已推送。e2e 需真实 Chrome 未在本会话运行。
- **Created**: 2026-08-12
- **Context**: 为「清空列表」仅追加约 15 行时，`panel/panel.js` 已约 1815 行（远超源文件 500 行门禁）。本会话未做大爆炸拆分以免扩大功能范围。
- **Action**: (1) 按 Tab 边界抽出 `panel/tabs/single-request.js`、`batch.js`、`errors.js`、`history.js`；(2) 共享 DOM/`sendMessage`/workspace 加载进 `panel/lib/`；(3) 保持 `panel.html` 脚本顺序与现有单测/e2e 全绿。
- **Why**: 超标文件继续堆功能会放大回归面，也违反行数门禁元规则。
- **How to apply**: `taskChromePlugin/panel/panel.js`；对照 `lib/panel-request-bootstrap.js` 的纯逻辑拆分模式；验收 `wc -l panel/**/*.js` 各 ≤500 且 `npm test` + clear/refresh e2e 通过。

## [OPT-20260812-044] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: 审计 16 个业务 Go 服务：仅 taskGitOauth 读 X-Auth-User-Id 但缺 gatewayUserMiddleware（taskBill/taskReferral/taskAiProvider 等直接读 X-User-Id，无缺口）。修复：App.gatewayUserMiddleware + main.go 链路上挂载，新增 2 回归测试全绿；taskGitOauth 88af438 已推送。
- **Created**: 2026-08-12
- **Context**: 修复 taskCloudService 任务详情 GitHub 账号下拉为空时确认根因是 APISIX 只注入 `X-User-Id`，而 `getAuthUser` 只读 `X-Auth-User-Id`；cloud 原先未挂 `gatewayUserMiddleware`。`taskProjectService`/`taskTenantService`/`taskTaskService`/`taskAIComment` 已有等价中间件，但 `taskBill`/`taskReferral`/`taskAiProvider` 等 ListenAndServe 链路上未见。
- **Action**: (1) 枚举所有业务 Go 服务的 ListenAndServe 包装链；(2) 凡依赖 `getAuthUser`/`X-Auth-User-Id` 且经网关对外的服务补上 `gatewayUserMiddleware`（或 authMiddleware 内 `ApplyGatewayUser`）；(3) 为每服务补一条「仅 X-User-Id + verified」回归测试。
- **Why**: 同类缺口会导致「库里有用户态数据、API 200 但业务字段为空」的隐性故障，排障成本高。
- **How to apply**: 对照 `taskProjectService/src/auth.go` 的 `gatewayUserMiddleware`；`shareLib/gatewayauth.ApplyGatewayUser`；各服务 `main.go` ListenAndServe。

## [OPT-20260812-037] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: 新增 taskTenantService backfill-member-name 子命令（--dry-run/--limit），复用 isMisSeededMemberName + fetchPersonalNicknamesFromAuth + healMisSeededMemberName。生产 task_tenant 实测 0 行需回填（数据已干净），dry-run 与 exec 均通过；taskTenantService e1cd5ce 已推送。
- **Created**: 2026-08-12
- **Context**: 人员管理页「公司成员名称」曾把 onboarding 公司名「我的公司」写入 `tenant_company_member.member_name`。读路径已按需自愈，但未访问人员页的租户行仍可能残留误种子。
- **Action**: (1) 写运维脚本扫描 `member_name IN ('我的公司') OR member_name = company.name`；(2) 批量取 `auth_user_profile.username` 回填；(3) DRY_RUN 后执行并记录影响行数。
- **Why**: 读路径自愈依赖访问 company_members；角色列表等其它展示面仍可能读到原始误种子。
- **How to apply**: `tenant_company_member` + `tenant_company` + taskAuth `auth_user_profile`；可参考 `taskTenantService/src/member_display_name.go` 的判定函数。

## [OPT-20260812-028] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: 新增 db/_infra/backfill_system_auto_git_identities.py：扫描 task_tenant.tenant_company_member，对缺 system-auto 身份的成员调用 taskTaskService 幂等 ensure-default（对齐 taskEvents memberjoined 模式）。实测生产 members=1 with_system_auto=1 missing=0（数据已就绪），dry-run 与 --confirm RUN 均通过；db 5509bbe 已推送。
- **Created**: 2026-08-12
- **Context**: 自动建身份仅对新 MEMBER_JOINED 生效；历史成员无 system-auto 行。
- **Action**: (1) 写一次性脚本/管理命令扫描 `tenant_company_member`；(2) 对缺失对应邮箱身份的成员调用 ensure-default；(3) 记录幂等与失败。
- **Why**: 存量成员在任务克隆时可能缺少默认公司身份。
- **How to apply**: 复用 `POST /api/internal/git-identities/ensure-default/`。

## [OPT-20260812-024] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: 审计扫描 dockerInfra/gitService/db 全部 docker-compose：仅 git-service 有 mem_limit（已在 OPT-20260812-023 迁入 conf/infra/git-service/config.yaml → ${GITLAB_MEM_LIMIT:-6g}），kafka/mysql/redis/docker-infra 等无 mem_limit:/cpus:/deploy: 可调键；规则 47 资源键 SSOT 已满足，无剩余可迁移项。
- **Created**: 2026-08-12
- **Context**: 元规则 47 要求人工可改配置 SSOT 在 conf；本会话仅落地 git-service 资源键样例。其它 infra（kafka/mysql/redis/docker-infra 等）compose 内默认值需按触及迁移。
- **Action**: (1) `rg -n 'mem_limit:|cpus:|deploy:' --glob '**/docker-compose*.yml'`（排除第三方）；(2) 逐服务把可调键写入对应 `conf/infra/<app>/config.yaml`；(3) run.sh/compose 改为消费；(4) 补短测。
- **Why**: 避免元规则落地后仍多处改参。
- **How to apply**: `.ai/01_project_constraints/47_conf_app_human_editable_config_ssot.md`。

## [OPT-20260813-001] cancelled

- **Status**: cancelled
- **Completed**: 2026-08-13
- **Summary**: 已被任务详情「自动克隆子仓库」开关取代：不再仅展示 disabled-hint，改为可切换 toggle（data-testid=task-nested-repos-auto-clone-toggle）
- **Created**: 2026-08-13
- **Context**: 项目详情取消「自动克隆子仓库」后，任务详情仍展示「子仓库克隆状态」刷新区，与 bootstrap 不再 enrich 子仓的行为矛盾。FE 已按 `auto_clone_nested_repos` 改为展示禁用提示；需精准编译重启 taskFE 后公网硬刷新验收。
- **Action**: (1) http://10.2.150.68:9999/ 「精准编译重启」含 taskFE；(2) 硬刷新 https://www.daydaymoney.com/tenant/875362439758114816/workspace/ws_-2794705041295509749/task-detail/task_15661754461291451866/；(3) 确认无 `task-nested-repos-clone-status`，有 `task-nested-repos-auto-clone-disabled-hint` 文案「已关闭自动克隆子仓库」。
- **Why**: SPA 产物未更新前公网仍见旧 UI；硬刷新验收闭环本次修复。
- **How to apply**: `TaskDetailNestedReposCloneStatus.vue` + `TaskDetailLinkedProjectsViewMode.vue`；项目 `proj_-2794396911392257746` 已关开关。

## [OPT-20260812-026] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: atomic-vite-build.sh 增加构建前 MemAvailable 预检（<6GiB 告警回收 cache、<3.5GiB 自动停 kafka-ui，EXIT trap 恢复）+ OOM(rc=137/heap out of memory) 释放内存重试一次；修复 if ! 取反导致 $? 为 0 拿不到真实退出码。test_build_clean_dist.sh 新增 3 组回归（告警不阻断构建/OOM 重试成功/静态断言）。taskFE 提交 f6542ce 已推送。
- **Created**: 2026-08-12
- **Context**: 本机 30Gi、`/tmp/ram-work` 在 tmpfs（shared≈18Gi）时，全栈跑满后 `vite build` 常在 rendering chunks 被 kill（exit 137）；runAll `/api/build` taskFE 同失败。本会话靠临时停 gitlab/kafka 才编过。
- **Action**: (1) `atomic-vite-build.sh` / runAll build 前检测 `MemAvailable`；(2) 不足则提示或自动停非关键容器（kafka-ui/gitlab）再建；(3) 或把 workspace 迁出 tmpfs；(4) 补文档到 runbook。
- **Why**: 每次发版依赖人工腾内存，易阻塞 UI 热修发布。
- **How to apply**: `taskFE/app/scripts/atomic-vite-build.sh`；`runAll` build 路径；与 OPT-20260812-023 memLimit 相关。

## [OPT-20260812-049] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: clear/init 异步化：/api/dev/clear-databases 与 init-databases 改为 202 accepted + run_id 后台执行，进度/结果经 /api/progress?run_id= SSE 送达（StartAllProgressEvent 新增 Detail 承载完整结果）；UI _execClear/InitAllDatabases 改连 SSE 用 ev.detail 汇总。runAll 提交 ee921b6 已推送，全量测试 104s 通过。
- **Created**: 2026-08-12
- **Context**: 已用 `context.WithoutCancel(r.Context())` 修复「客户端超时 → mysql-reset context canceled → status=partial」。但 clear/init 仍是同步 HTTP，代理/浏览器长连接仍脆弱；UI 只能靠轮询 `/api/dev/logs`。
- **Action**: (1) 将 `/api/dev/clear-databases` 与 `/api/dev/init-databases` 改为 accepted + 后台执行（同 `runCancellableLifecycleActionAsync`）；(2) 增加 progress SSE 或复用 generic `/api/progress`；(3) UI `_execClearAllDatabases`/`_execInitAllDatabases` 改为连 SSE；(4) 保留 confirm token 与互斥锁。
- **Why**: 长同步请求在网关/代理层仍可能被掐断；异步化与 build-all/start-all 一致，也便于取消与进度展示。
- **How to apply**: `runAll/src/ui.go` `registerDevDatabaseHandler`；`status_ui/js/04.js`/`10.js`；对照 `runCancellableLifecycleActionAsync`。

## [OPT-20260813-002] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: init-databases 加固：背景 goroutine 加 panic recover（init 路径 panic 不再击穿 runAll 主进程，OPT-20260813-002 #3）；状态落盘 .runall/db-init|db-clear.last_status（start=running、终态 ok/partial/blocked/panic，进程 OOM 时停留 running 作诊断线索，#4）；同步随 049 异步化避免长阻塞 handler（#2）。runAll 提交 ee921b6 已推送。
- **Created**: 2026-08-13
- **Context**: 2026-08-13 goal 流程中，首次 POST `/api/dev/init-databases` 在 migrate task-auth 阶段返回 curl Empty reply，9999 进程消失；重启 runAll 后二次清库+初始化成功。当时内存压力较高（约 23Gi/30Gi used、free≈369Mi），且有并行 `random_test_runner`。
- **Action**: (1) 复现：高内存负载下 clear→init，确认是否 OOM/panic；(2) 为 runAll 增加进程监督或 init 异步化（HTTP 立即 accepted + 进度 SSE，避免长阻塞 handler）；(3) init 路径禁止 `os.Exit`/致命 panic 未恢复；(4) 记录退出码与 last log 到 `.runall/`。
- **Why**: 9999 中途挂掉会中断清库初始化流水线，运维只能手工重启。
- **How to apply**: `runAll/src/runner.go` InitAllDatabases；`runAll/src/ui.go` `/api/dev/init-databases`；对照 `logs/db-init.log`。

## [OPT-20260812-053] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: Composer 内 ServerConfigImageSectionHints 现可收到非空 task/viewerTaskId：TaskDetail 经 commentsSectionProps→TaskDetailCommentsSection→TaskDetailCommentsPanel 透传 localTask 与 taskId，imageHintsTask/imageHintsTaskId 改读 props。taskFE 提交 953a523 已推送，task-detail 36 文件/150 用例全绿。
- **Created**: 2026-08-12
- **Context**: Composer 内 `ServerConfigImageSectionHints` 目前 `task=null` / `viewer-task-id=''`，机器归属等提示在评论区可能不展示。
- **Action**: (1) 从 TaskDetail/CommentsPanel 向 Composer 传入 taskId/task；(2) 接上 hints 所需 props；(3) 补单测断言 hints 收到非空 taskId。
- **Why**: 镜像迁入评论区后，原卡内 hints 能力不应静默丢失。
- **How to apply**: `TaskDetailCommentComposer.vue` `imageHintsTask*`；`TaskDetailCommentsPanel.vue`。

## [OPT-20260812-043] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: 评论级复用任务级 inflight 机器时同步 instance/public_ip 到评论 CSC 行（幂等，行不存在先 ensure）；防双行漂移与反复附着。taskCloudService 提交 cc81470 已推送，相关用例 5.5s 全绿。
- **Created**: 2026-08-12
- **Context**: `task_15643848305890470866` 出现两行 CSC：任务级有 `instance_id`+公网 IP，评论级 `csc_-2802882173382949747` 的 `instance_id/public_ip` 仍空；`inflight_machine_attach` 反复附着。`loadCloudServerConfigForRuntime` 能选到有实例的行，但评论级行长期不一致。
- **Action**: (1) inflight attach / start-vm 成功后把 instance_id+public_ip 同步到评论级 CSC；(2) 或明确评论级仅存绑定、读路径统一走 runtime loader 并文档化；(3) 补单测防双行漂移。
- **Why**: 双行漂移会导致部分读路径（只用任务级/只用评论级）行为不一致，排障成本高。
- **How to apply**: `workspace_machine_inflight_attach.go`；`comment_csc_ensure.go`；`server_config_store.go`。

## [OPT-20260811-084] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: taskEvents 新增 intent billing_order_comment_created/1_notify_other_side（port 18061）：校验 payload/author_side，按 author_side 向订单级 Redis 通道 sse:billing:order:<order_id> 发布 SSE 通知对侧（一期 SSE，payload 含 author_side 供过滤），SSE 失败可重试；tracelog 契约。注册三处同步（AllIntents/run.sh/conf yaml）。taskEvents 提交 d52ff61、conf 提交 0931ef9 已推送；handler 7 单测全绿，taskEvents 全量 go test 通过。
- **Created**: 2026-08-11
- **Context**: 已发布 `BILLING_ORDER_COMMENT_CREATED`（payload 不含正文），一期无消费者。
- **Action**: (1) taskEvents 注册消费者；(2) 按 author_side 通知对侧（邮件或 SSE）；(3) 补意图测与 DLT。
- **Why**: 无通知则双方需手动刷新才能发现新留言，降低客服效率。
- **How to apply**: topic `billing-order-comment-created`；参考退款事件消费者模式。

## [OPT-20260812-056] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: 厂商门户关联 region 本已经 prop 只读传入 RegionEnvModal（无独立 region 下拉），后端 UpsertAssociation/批量写入以 CSI.region 为准；新增回归测试锁定：模态框无 region 绑定下拉、保存载荷 region 取 props.region?.id、region 声明为 Object prop。taskAiProvider 提交 81e16e7 已推送，frontend 60 单测全绿。
- **Created**: 2026-08-12
- **Context**: 后端已在读路径与 UpsertAssociation 写入路径以 CSI.region 为准；厂商门户若仍展示/提交独立 region 下拉，可能让运营误以为可自由选地域。
- **Action**: (1) 检查 taskAiProvider 厂商门户关联表单；(2) 将 region 改为只读展示 CSI.region，或提交前禁用与 CSI 不一致的选项；(3) 补一条前端单测。
- **Why**: 防止运营再次写入漂移关联，降低「未找到匹配地域」复发率。
- **How to apply**: `taskAiProvider/frontend` 关联编辑组件；对照 `infrastructure/store_associations.go` UpsertAssociation。

## [OPT-20260812-042] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: server-content 对 connection refused 返回 HTTP 200 + status=starting（serverContentHTTPClient 可注入）；前端 fetchServerContentApi 对 status=starting 降级展示「容器业务端口尚未就绪，正在启动中」而非成功/硬错误。taskCloudService 提交 8dd21ac、taskFE 提交 540891e 已推送，各带回归单测。
- **Created**: 2026-08-12
- **Context**: 修复 CSC.public_ip 未落库后，`server-content` 已能回填到阿里云公网 IP（例 `47.105.105.136`），但对 `:8080` 仍 `connection refused`——UserData/容器尚未监听。前端任务详情在启动中途会打该接口并展示硬错误。
- **Action**: (1) 前端在 `runtime_status=Running` 且 boot-progress/容器心跳就绪前降级提示「启动中」而非错误；(2) 或 server-content 对 connection refused 返回可区分的 `status=starting`；(3) E2E 已有 server-content 轮询助手可复用。
- **Why**: 公网 IP 有了不等于业务端口就绪；否则用户仍看到「拉取失败」误判为云故障。
- **How to apply**: `taskFE/.../useServerConfigRuntimeFetch.js`；`taskCloudService/src/compute_server_content.go`；`taskFE/tests/helpers/startVmAutoE2e.js`。

## [OPT-20260812-023] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: GitLab 容器重建：HostConfig.Memory 3GiB→6GiB，docker inspect 确认，GitLab web 200，runAll git-service healthy
- **Created**: 2026-08-12
- **Context**: 本会话将 `memLimit` SSOT 迁入 `conf/infra/git-service/config.yaml`（默认 6g）；当前运行中容器仍可能是旧 hard limit（曾见 3GiB）。需 recreate 后 `docker inspect` 确认。
- **Action**: (1) 确认 conf `memLimit`；(2) `bash gitService/run.sh stop --clean` 后 `bash gitService/run.sh`（或 runAll 重启 git-service）；(3) `docker inspect gitlab --format '{{.HostConfig.Memory}}'` 应为 6GiB；(4) 硬刷新 GitLab 可用性。
- **Why**: 仅改 conf/compose 不重建容器则 Docker hard limit 不变。
- **How to apply**: `conf/infra/git-service/config.yaml`；`gitService/run.sh`；元规则 47。

## [OPT-20260813-006] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: watchdog 新增 runAll UI(:9999) 失联告警 + 冷却内自动重启 + start-all（ensure_services_healthy.py，runAll a20e222，61 tests passed）；夜间 OPT 约束记录于条目
- **Created**: 2026-08-13
- **Context**: 2026-08-13 01:42:57 全业务日志同时写入 `[runAll] default state: not started`（`NewRunner`→`seedPendingLifecycleLogs`），随后 :9999 无进程、APISIX(:18081) 仍存活但对上游返回 openresty **502**。同期夜间 OPT agent 正在改/编/测 runAll（`bin/runAll` 01:45 重建、`go test ./src/...`、async clear/init commit）。无 OOM。
- **Action**: (1) 夜间 OPT 与 `go test`/重建路径禁止对生产 `:9999` 调用 takeover/shutdown-self/二次启动，除非随后自动 `start-all`；(2) 增加 runAll UI 进程守护（systemd/cron）：`:9999` 失联则拉起并可选触发 start-all；(3) 在 `ensure_services_healthy.py` 对「runAll UI 宕机」单独告警，避免只修后端、控制台长期不可用。
- **Why**: runAll 被替换且未 start-all 时，Docker 网关仍在 → 全站 502；控制台也挂掉，无法从 UI 恢复。
- **How to apply**: `runAll/src/main.go`（previous instance kill / orphan cleanup）；`runAll/scripts/ensure_services_healthy.py`；夜间 OPT runner 约束；可选 `runAll/run.sh` 守护包装。

## [OPT-20260813-007] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: runAll 退出原因日志：main.go 手动 signal.Notify 记录触发源（signal vs shutdown-self）+ 退出时 lifecycle 状态快照（summarizeServiceLifecycle）。提交 e03b503 已推送。单测 6 例 + 全量 go test ./src/ 107s 通过。
- **Created**: 2026-08-13
- **Context**: 2026-08-13 03:05–03:30 夜间窗口内 runAll 至少 3 次以「优雅关闭」方式退出（控制台日志：对全部托管服务依次 SIGTERM + stopped，即 `runner.Run` 的 `<-ctx.Done()` → `Shutdown()`），非 OOM/SIGKILL，亦非 shutdown-self（无 `[api] shutdown-self requested` 行，服务被停止说明 skipShutdownServices=false）。:9999 随后无进程，服务进程存活为孤儿，APISIX 对部分上游 502。03:22:01 那次发生在 meta 提交 61282c0（03:19:35）之后 ~2.5min，与 pre-commit/pre-push 无直接时间重叠；03:29 再次复现。周期性（≈5-8min）疑似与 cron `*/5` 任务或某后台守护向 runAll 进程组发 SIGTERM 相关；已用 `ensure_services_healthy.py` watchdog（OPT-20260813-006）验证可自动恢复（检测失联→拉起→start-all→59/59），但根因未明。
- **Action**: (1) 在 runAll 常驻运行期间记录进程 PID/会话/进程组（`ps -o pid,ppid,pgid,sid,cmd`），对齐每次死亡时间戳；(2) 排查 cron（`*/5` ensure_services_healthy / ensure-edge-tunnels）、systemd、claude-agent session hub、以及 Bash 工具超时清理对 setsid 进程组的影响；(3) 在 runAll main 增加退出原因日志（signal name / last lifecycle log）便于定位；(4) 确认后修复并补回归。
- **Why**: runAll 是控制平面；周期性掉线会反复造成运维入口不可用 + 服务孤儿，虽已被 watchdog 兜底但每次恢复都要全量 start-all，拉长部署窗口。
- **How to apply**: `runAll/src/main.go`（signal.NotifyContext 处理段加退出原因日志）；`runAll/scripts/ensure_services_healthy.py`（已具备自动恢复）；对照 `logs/runall-console.log`、`logs/service-watchdog.log`。

## [OPT-20260812-048] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: 热替换补偿实现：shutdown-self 后对 adopt-skip 且配置探针+有 ownership 的 down 服务按 DAG 依赖序自动补 start；新增 4 回归单测全绿，runAll 全量 go test 101s 通过；提交 runAll 3a4e8d4（已推 github+gitlab）；bin/runAll 已重建并经 04:41 watchdog 重启上线（/proc/2414416/exe 含 compensation 字符串，正常重启未触发补偿——gating 正确），下次 shutdown-self 热替换即生效
- **Created**: 2026-08-12
- **Context**: 本会话在修复 clear/init 的 request-context 取消后，对 runAll 执行 `./run.sh` 热替换（shutdown-self 200、skip orphan）。新实例日志出现大量 `adopt skip … connection refused`，`/api/status` 一度仅 43/59 healthy，需再跑 `POST /api/start-all` 才恢复 59/59。**2026-08-13 代码分析**：托管服务以 `Setpgid:true` 独立进程组启动（`runner.go:664`），shutdown-self 设 `skipShutdownServices=true`，不会被 SIGTERM 连带；`adopt skip connection refused` 路径在 `runner_adopt.go`——无存活 ownership PID + 无端口监听 + 探针 3 次失败才跳过。即**热替换只回填「探针仍通过」的存活服务，替换前已 down 的服务不会自动重启**（只有 start-all 会补拉起）。复现/根因确认需一次真实热替换（低峰可做）；修复方向：热替换后对 adopt-skip 且配置了探针的服务自动补 start，而非要求人工 start-all。回归断言「热替换后 healthy 数不下降」。
- **Action**: (1) 复现：全部启动后对 runAll 做 shutdown-self 热替换，记录哪些 PID/端口在替换前后存活；(2) 确认托管进程是否在 runAll 进程组内、是否被 SIGTERM 连带；(3) 若确认误杀，改为 setsid/独立进程组启动，或热替换后自动补偿 start 未收养服务；(4) 补回归测试/脚本断言「热替换后 healthy 数不下降」。
- **Why**: 热替换本应零中断；若每次部署 runAll 都要手工 start-all，会拉长目标流程并误伤线上会话。
- **How to apply**: `runAll/src/main.go`（shutdown-self / skip orphan）；`runner_adopt.go`；`run.sh`/`build.sh`；对照本会话 `/tmp/runall-hotreplace.log`。

## [OPT-20260812-020] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: 默认服务器配置表单补充竞价策略与 IoOptimized：新增 DDL 017（io_optimized/spot_strategy 列）、taskCloudService POST/GET 持久化与 read-back、taskFE 表单 IoOptimized 勾选+竞价策略下拉、useSetDefaultConfigForm 采集/恢复、defaultConfigToRunTemplate 映射 filter_options；Go+FE 单测全绿（taskCloudService 全包 152s 通过）。
- **Created**: 2026-08-12
- **Context**: 实例筛选区含 IoOptimized/竞价策略，但「设置默认服务器启动配置」未采集这两项；应用模版后仍用面板默认值。
- **Action**: (1) 评估是否纳入默认配置持久化；(2) 若纳入：扩展表单、DDL、`defaultConfigToRunTemplate` 映射；(3) 单测覆盖。
- **Why**: 用户期望「模版参数」完整对齐实例区，不仅限于 CPU/内存/盘型。
- **How to apply**: `SetDefaultConfigFormFields.vue`；`016_cloud_server_config_defaults_hardware.sql` 后续迁移。

## [OPT-20260812-022] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: 重建 E2E 测试账号 + 修复 gatewayLoginE2e 明文密码：auth_user/auth_login_method/tenant_company/member 已幂等重建（seed_e2e_account.py），登录验证 200；gatewayLoginE2e.js 直发明文（taskAuth bcrypt 比对明文，pbkdf2 产物>72字节恒失败）；CreateProject.git-repo-top.playwright.test.js 通过
- **Created**: 2026-08-12
- **Context**: 全量编译/重启后，Playwright `loginViaGatewayApi` 用约定账号 `contact@daydaymoney.com` / `rgNodkdq8677!ci`（经 PBKDF2）对 `/api/auth/` 返回 400「用户名或密码错误」（例 trace_id=`dbc85eb4c165a6b7a35b1b9d4452ac14`）。Loki 仅见 http_request status=400。布局 vitest 已绿，E2E 被登录阻断。**2026-08-13 排查结论**：task_auth / task_tenant 已清库重建（auth_user 仅 bootstrap-admin + 1 个 wechat 用户；tenant_company 仅 875432463248158720），测试账号 `contact@daydaymoney.com` 与租户 `850256677331562496` 均不存在 → `findLoginMethodByIdentifier` 返回 nil → 400。修复需重建测试账号（`auth_user` + `auth_login_method`，password_hash=bcrypt(pbkdf2_sha256(密码, identifier))，见 `PasswordHasher.js` / `taskAuth/src/auth_login.go`）及租户生态（company/member/workspace/feature-params，正常由 COMPANY_CREATED 事件链初始化，手工 INSERT 易缺件）。建议固化一个「重建 E2E 测试账号」脚本（多库 seed）再执行。
- **Action**: (1) 在 MySQL task-auth 查该 identifier 的 login_method/password_hash；(2) 对照 `PasswordHasher.js` 与 `checkPasswordHash`；(3) 修复测号密码或更新 E2E 凭据；(4) 重跑 `CreateProject.git-repo-top.playwright.test.js`。
- **Why**: 无可用登录则无法在浏览器 E2E 断言公网页元素顺序与 OAuth 交互。
- **How to apply**: `gatewayLoginE2e.js`；`taskAuth/src/auth_login.go`；测例 `CreateProject.git-repo-top.playwright.test.js`。

## [OPT-20260812-019] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: 新增 CreateProject.run-template-filter-sync.playwright.test.js：mock server-config-default（cpu/memory/instance_type/disk），选择工作空间+默认模版后断言 filter cores/memory/disk 与选中 ecs.g6.large「已选」恢复，本地 headless 全绿（420ms），已推送 taskFE a9ed5e2
- **Created**: 2026-08-12
- **Context**: 本次仅有 utils/Go 单测；缺少 create-project 选择「快速应用默认模版」后 filter UI 对齐的 E2E。
- **Action**: (1) 新增 Playwright：mock `server-config-default` 列表含 cpu/memory/instance_type/disk；(2) 选择工作空间与模版；(3) 断言 filter cores/memory/disk 与 selected_instance 恢复。
- **Why**: 防止回归再次把 `hardware_config`/`filter_options` 映射成空对象。
- **How to apply**: 对照 `ProjectDetail.hardware-config-sync.playwright.test.js`；目标页 create-project。

## [OPT-20260811-082] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: 邀请页改为选择可复用角色（v75 角色优先）。taskTenantService handleInvite 接受 role_names（非 admin，校验存在）存 pending_role_names，handleJoin 按角色 ensureMemberRole 绑定；pending_grants 保留兼容。dataMigrate 010 已应用到生产 task_tenant。taskFE PeopleInvite 多选可复用角色。5 Go 单测 + 3 vitest 全绿；commit: taskTenantService a5df7c3 / dataMigrate 3adb3da / taskFE 6efc3fb（均已推送）。
- **Created**: 2026-08-11
- **Context**: v75 访问管理已改为主体多角色 `role_names[]`，停隐式「访问·…」；邀请流仍偏 pending_grants / 隐式访问角色路径，与角色优先产品面不一致。
- **Action**: (1) PeopleInvite 支持多选已有自定义/系统角色；(2) join 落权走角色绑定而非仅隐式访问角色；(3) 补意图测与 vitest；(4) 兼容存量 pending_grants。
- **Why**: 邀请是角色分配入口之一；不改则新成员仍走旧一对一访问角色，削弱角色管理页价值。
- **How to apply**: `PeopleInvite.vue`；taskTenant invite/join；设计 `docs/superpowers/specs/2026-08-11-tenant-role-management-v75-design.md`；关联 v74 pending_grants。

## [OPT-20260812-021] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: 公网浏览器验收通过（2026-08-13 夜间）：创建项目页 #gitRepo0 位于 #projectName 之上（DOM compareDocumentPosition 确认 + 视觉 top 236 < 420），Git 仓库区块含 data-testid=create-project-git-repos-section。新 dist 已随精准编译重启部署。
- **Created**: 2026-08-12
- **Context**: 本会话已将 CreateProject 表单中 Git 仓库区块（含 `#gitRepo0`）移到项目名称之前，便于用户先完成仓库授权；本地 dist/`/static/assets` 与 vitest 已确认 DOM 顺序。
- **Action**: (1) 硬刷新 https://www.daydaymoney.com/tenant/875216530801979392/create-project/；(2) 确认 `#gitRepo0` 视觉与 DOM 均在 `#projectName` 之上；(3) 可选：填私有仓 URL blur 后仍可见 OAuth 授权按钮。
- **Why**: 公网 CDN/浏览器缓存可能导致仍见旧 chunk，需硬刷新验收。
- **How to apply**: `CreateProject.vue`；`data-testid=create-project-git-repos-section`；测例 `CreateProject.gitRepoTop.test.js`。

## [OPT-20260812-005] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: 部署+验收通过（2026-08-13 夜间）：taskFE/task-task-service 随精准编译重启重启；task-project-service/task-credential-service 运行含 auto_clone_nested_repos 的新二进制（06:12 启动）；create-project 填 Git URL 后出现 create-project-auto-clone-nested-repos 勾选且默认勾选。012 migration 已应用。注：项目详情开关需含 git 仓库的项目才能验证，本会话测试项目无仓库未验。
- **Created**: 2026-08-12
- **Context**: 本会话已应用 `012_auto_clone_nested_repos.sql` 到 `task_project`，功能代码已推送到各服务仓；公网仍需编译重启后可见勾选框。
- **Action**: (1) http://10.2.150.68:9999/ 「精准编译重启」含 task-project-service、task-task-service、task-credential-service、taskFE；(2) 硬刷新 create-project，填 Git URL 后确认 `create-project-auto-clone-nested-repos`；(3) 项目详情确认开关。
- **Why**: 未重启则公网仍无创建页选项。
- **How to apply**: `.runall/precise_restart_services.txt`；验收 URL `https://www.daydaymoney.com/tenant/.../create-project/`。

## [OPT-20260813-008] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: seed_e2e_account.py 补齐公司默认 progress/deliverable 体系与工作空间；回归测试 6 例全绿；DB 三 API + validate-progress-column 已验证
- **Created**: 2026-08-13
- **Context**: 2026-08-13 夜间用 `contact@daydaymoney.com`（seed_e2e_account.py 直连 SQL 种子）公网验收时发现：`tenant_company_member` 无 company-created 事件链（直接 SQL 插入），公司 `850256677331562496` 无 `project_progress_systems_default_tenant`/`project_deliverable_systems_default_tenant` 默认行。据此新建 workspace（如「夜间验收工作空间」）后 `task-events-workspace-created-1-process-workspace-creation` 的 `ensureWorkspaceProgress` 因 `findTenantDefaultProgress` 为空而跳过绑定 → 任务创建报「进度列不属于当前工作空间」、面板显示「未加载交付物类别」。
- **Action**: 在 seed_e2e_account.py（或后续 E2E 环境初始化）中补齐 company-created 事件链产物：默认 progress 系统（含列）+ 默认 deliverable 系统 + 默认 workspace，使 E2E 账号可完成建任务/看板验收；或让测试公司复用 `ps_default_system` 并注入列。
- **Why**: 任何依赖 E2E 账号做浏览器验收（创建任务、work-panel 看板、任务详情）的会话都会在任务创建处被拦，无法闭环。
- **How to apply**: `taskFE/tests/helpers/seed_e2e_account.py`；对照 `taskProjectService` company-created 事件处理（`internal/repository/saas/` company 相关）。

## [OPT-20260812-002] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: 生产浏览器验收通过：数据盘下拉含「不要数据盘」选项（项目详情/创建项目同组件），公网 SPA 已发布
- **Created**: 2026-08-12
- **Context**: 本会话已在数据盘下拉增加「不要数据盘」（空值），并打通查询参数/创机跳过挂盘；需精准重启后硬刷新公网页验收。
- **Action**: (1) http://10.2.150.68:9999/ 精准编译重启含 taskFE（已 build）+ task-cloud-service；(2) 打开 create-project 硬件配置，数据盘类型选「不要数据盘」；(3) 确认仍能拉可用实例且查询不含 DataDiskCategory；(4) 有条件时创机验证无数据盘。
- **Why**: 未发布则公网仍只有各盘型选项，无法选「不要数据盘」。
- **How to apply**: `HardwareConfigFilterFields.vue`；`data-testid=data-disk-category-select`；URL `https://www.daydaymoney.com/tenant/.../create-project/`。

## [OPT-20260812-027] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: 生产浏览器验收通过：工作面板直进无静默回弹（URL 保持 /work-panel/），Navbar「工作面板/个人资料」为真实 a[href] 链接
- **Created**: 2026-08-12
- **Context**: Profile 点「工作面板」曾因 `/me/` 403/404 静默 `replace(profile)` 回弹；已改为弹窗确认，Navbar 改为真实 `<a href>`。相关单测 40 通过；`taskFE/app` 已 `npm run build`（WorkPanel chunk 含「前往个人资料/留在本页」、无静默 `replace(.../profile)`）；`taskFE` 已在精准重启登记。仍待运维点击重启后公网硬刷新验收。
- **Action**: (1) http://10.2.150.68:9999/ 「精准编译重启」含 taskFE；(2) 硬刷新打开 profile，确认「工作面板」为 `a[href*="work-panel"]`；(3) 人为制造 `/me/` 403（或 mock）进入 work-panel，应出现「前往个人资料 / 留在本页」弹窗，点留在本页后 URL 仍含 `/work-panel/`；(4) 可选跑 `WorkPanel.access-denied-confirm-no-bounce.playwright.test.js`（CDP 9222）。
- **Why**: 线上进程未换新 dist 前用户仍会看到旧静默回弹。
- **How to apply**: 设计文档 `docs/superpowers/specs/2026-08-12-work-panel-no-silent-profile-bounce-design.md`；元规则 `49_no_link_click_interception.md`。

## [OPT-20260811-048] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: 生产浏览器验收通过：SystemAdmin 侧栏 aside overflow-y-auto、scrollHeight 1144>clientHeight 513 可滚到底部安全策略
- **Created**: 2026-08-11
- **Context**: `SystemAdminSidebar`/`Sidebar` 已加 `min-h-0 overflow-y-auto`；本地 vitest 与 `npm run build` 已通过。公网仍需精准编译重启发布后人工确认。
- **Action**: (1) 在 http://10.2.150.68:9999/ 对已登记 `taskFE` 执行「精准编译重启」；(2) 打开 https://www.daydaymoney.com/system-admin/ ，各组保持展开；(3) 在左侧 `aside`/`nav` 上滚轮：可滚到底部「安全策略」并滚回顶部「系统总览」；(4) 硬刷新排除旧 chunk 缓存。
- **Why**: 单元测只断言 class 契约，不能替代生产静态托管与浏览器滚动行为。
- **How to apply**: `SystemAdminSidebar.vue`；`Sidebar.vue`；chunk `SystemAdminSidebar-*.js`。

## [OPT-20260811-053] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: 生产浏览器验收通过：项目详情页最后内容贴滚动容器底 gap=0，无大块灰空白（底部空白消除）
- **Created**: 2026-08-11
- **Context**: 本会话已修 taskFE：html/body/#app overflow hidden、去掉全局 main/footer 叠高 padding、壳内页 min-h-screen→min-h-full、ProjectDetail 底部 padding 收敛；本地 preview 登录页 scrollOverflow 自 462→376（去掉约 navbar 高度的假空白）。公网仍挂旧 chunk，需精准编译重启后验收项目详情 URL。
- **Action**: (1) 在 http://10.2.150.68:9999/ 执行「精准编译重启」含 taskFE；(2) 硬刷新打开 https://www.daydaymoney.com/tenant/874941752761413632/projects/proj_-2894567986635221745/；(3) 滚至页面最底部：最后卡片与滚动容器底边间距应约为 pb-4（~16px），无大块灰空白；(4) 顺带打开 /auth/login/ 确认 html/body overflow=hidden 且 login 根为 min-h-full。
- **Why**: 未发布则用户仍看到旧静态资源，底部空白问题在公网不可验证。
- **How to apply**: `.runall/precise_restart_services.txt` 已含 taskFE；产物 `taskFE/app/dist/style-*.css` 已含 `html,body,#app{height:100%;overflow:hidden}`。

## [OPT-20260812-035] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: 生产浏览器验收通过：gitlab-connection 只读框显示租户级 redirect URI /api/accounts/tenant-{tid}/oauth/callback/；存量租户 GitLab Application 白名单更新为运维跟进项
- **Created**: 2026-08-12
- **Context**: 本会话已将 redirect URI 改为 `/api/accounts/tenant-{company_id}/oauth/callback/`；已登记 `task-git-oauth`/`taskFE` 精准重启。既有租户若 GitLab Application 仍登记共享 `tenant-gitlab` 回调，授权会在白名单校验失败。
- **Action**: (1) 在 http://10.2.150.68:9999/ 执行「精准编译重启」；(2) 打开 `/tenant/{tid}/settings/gitlab-connection/` 确认只读框为 `…/tenant-{tid}/oauth/callback/`；(3) 对已配置自建 GitLab 的租户，指导管理员把 Application Redirect URI 改成新值并重新保存连接。
- **Why**: 代码已切租户级 URI，未同步 GitLab 白名单的存量连接会 OAuth 失败。
- **How to apply**: 页面 `https://www.daydaymoney.com/tenant/<tid>/settings/gitlab-connection/`；GitLab Admin → Applications → Redirect URI。

## [OPT-20260813-010] cancelled

- **Status**: cancelled
- **Completed**: 2026-08-13
- **Summary**: ForRuntime 已删除；container-task-ui-context 无 comment_id 现为 400「缺少评论ID」，不再读任务级 CSC。前端页面级 fetch 传 comment_id 见后续 OPT。
- **Created**: 2026-08-13
- **Context**: Workbench / runtime-status / stop-vm 已按 `comment_id` 走 `resolveScopedCloudServerConfig`。`buildContainerTaskUIContext` 在无 `comment_id` 时仍只读任务级 CSC；handler 已能读 query `comment_id`，但前端 `fetchContainerTaskUiContext` 尚未传。评论级 VS Code / 容器页 URL 在任务级无 server_url 时可能仍为空。
- **Action**: (1) 为 `handleContainerTaskUIContext` 写回归测：任务级无 server_url、评论级有可达 URL 且请求带 `comment_id` 时应返回 registered；(2) 无 `comment_id` 时评估是否改 `loadCloudServerConfigForRuntime`；(3) 前端 `fetchContainerTaskUiContext` 在评论卡语境传 `comment_id`；(4) 跑 `go test ./src -run TestContainerTaskUIContext`。
- **Why**: 与 workbench-link / runtime-status / stop-vm 同一类双行漂移；不改则「打开服务器 VS Code」在评论级机器上仍可能缺失。
- **How to apply**: `taskCloudService/src/compute_handlers.go` `buildContainerTaskUIContext`；对照 `compute_workbench_link.go`。

## [OPT-20260813-014] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: 从 comments bindings/Section/Panel/TaskDetail 删除 showHardwareConfigBar 三项透传与 adjust/restore 转发；硬件条仅留 CommentComposerHardwareCard。
- **Created**: 2026-08-13
- **Context**: 硬件摘要条组件 `HardwareConfigCommentBar.vue` 已删除（与「环境与硬件」卡重叠）。`TaskDetailCommentsSection` / `CommentsPanel` / `taskDetailSectionBindings` 仍透传 `showHardwareConfigBar`、`isUsingProjectHardwareTemplate`、`tempHardwareConfigLabel`，且 TaskDetail 仍转发已无 UI 触发的 `adjust-hardware-config` / `restore-hardware-defaults`。
- **Action**: (1) 从 bindings、Section、Panel 删除这三项 props；(2) 去掉 TaskDetail 上已无 UI 触发的 `adjust-hardware-config` / `restore-hardware-defaults` 转发（若确认无其它调用方）。
- **Why**: 死透传会让后续改显隐时误以为条还在评论区顶部独立渲染。
- **How to apply**: `taskFE/app/src/views/taskDetailSectionBindings.js`、`TaskDetailCommentsSection.vue`、`TaskDetailCommentsPanel.vue`、`TaskDetail.vue`。

## [OPT-20260813-012] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: fetchContainerTaskUiContext 解析当前执行评论 comment_id；无 id 不请求并保持 unregistered；vitest taskDetailContainerFns.commentId.test.js。
- **Created**: 2026-08-13
- **Context**: 运行态 CSC 已强制评论级；`handleContainerTaskUIContext` 无 `comment_id` 返回 400。任务详情页级 `fetchContainerTaskUiContext` 仍只带 `task_id`，打开容器页 / VS Code 链接会一直空。
- **Action**: (1) 从当前执行评论 / 有绑定的评论取 `comment_id` 追加到 `container-task-ui-context` query；(2) 无评论 id 时不发请求并保持 unregistered；(3) 补 vitest：带 comment_id 才请求。
- **Why**: 否则评论级容器已登记时，页面级「打开容器」仍拿不到 URL。
- **How to apply**: `taskFE/app/src/composables/taskDetail/taskDetailContainerFns.js` `fetchContainerTaskUiContext`；`useTaskDetail.js` 传入 active comment id。

## [OPT-20260813-015] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: 拆掉 ServerConfig 功能卡 Teleport；composer 就地挂载 FeatureParamsBlock + feature-params bridge；存量 Teleport 补 Teleport-OK。
- **Created**: 2026-08-13
- **Context**: 已落地约束第 45 条（默认禁止 Teleport）。`ServerConfig.logic.vue` 仍把功能卡 Teleport 到评论区 `[data-testid=…]` 槽，属新规则禁止的跨树搬迁；下拉/搜索类 Teleport 到 body 可能仍符合 overflow-clip 例外但缺 `Teleport-OK` 注释。
- **Action**: (1) `rg '<Teleport' --glob '*.vue' taskFE` 列出全部用法；(2) 功能卡改为在视觉父（如 composer）直接挂载，拆掉 data-testid 槽 Teleport；(3) 保留的 overflow/层叠例外补 `Teleport-OK` 注释。
- **Why**: 规则只约束新增不够，存量跨树 Teleport 仍会造成 defer 时序空洞与双实例分叉。
- **How to apply**: `taskFE/app/src/components/ServerConfig.logic.vue`；`TaskDetailCommentComposer.vue` 槽位；对照 `.ai/01_project_constraints/50_no_unnecessary_vue_teleport.md`。

## [OPT-20260811-079] cancelled

- **Status**: cancelled
- **Completed**: 2026-08-13
- **Summary**: 意图 028（2026-08-13）已将用户可见文案统一为「智能体资源配置」；src 中无「智能体资源设置」，无需再改。
- **Created**: 2026-08-11
- **Context**: 任务详情/创建任务不可用提示已按产品要求改为「智能体资源设置」；侧栏、工作面板、个人配置页等仍广泛使用「智能体资源配置」。同功能两套称谓并存。
- **Action**: (1) 与产品确认最终用词（设置 vs 配置）；(2) 全局 grep 用户可见文案并统一；(3) 同步 wording 单测与 Playwright 断言。
- **Why**: 术语不一致会降低可发现性，用户从提示跳转到设置页后看到不同称谓易困惑。
- **How to apply**: `rg '智能体资源设置|智能体资源配置' taskFE/app/src`；优先改展示文案，保留代码/路由 `feature-params` 命名不动。

## [OPT-20260811-045] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: 写能力改为 subject_list/region_matrix operate；存量 save_actions 双读；目录不再勾选独立 save_actions。Go/vitest 已绿。
- **Created:** 2026-08-11
- **Context:** v73 允许同一 region 上 view/operate；people.access 仍保留独立 save_actions region。
- **Action:** 评估将 save 能力并入 region_matrix/subject_list 的 operate，减少种子分裂。
- **Why:** 降低访问管理认知负担。
- **How to apply:** 设计迁移 + 双写过渡 + FE/BE 切 Operate。

## [OPT-20260813-017] completed

- **Status**: completed
- **Completed**: 2026-08-13
- **Summary**: 删除无消费者的 useServerConfigHardwareContext provide；保留 expand/restore。源码断言 + vitest 已绿。
- **Created**: 2026-08-13
- **Context**: 删除 `HardwareConfigCommentBar` 后，全仓已无 `inject('serverConfigHardwareContext')`，但 `useServerConfigHardwareContext` 仍 `provide` 一整包运行态/镜像 props。`expandHardwareForComment` / `restoreHardwareToProjectDefaults` 仍被 ServerConfig.logic 使用，与 provide 无关。
- **Action**: (1) 从 `useServerConfigHardwareContext` 去掉 `provide` 及仅供 inject 的参数；(2) 确认 logic 仍暴露 expand/restore；(3) 补/改 `useServerConfigHardwareContext.expandNest.test.js`。
- **Why**: 无消费者的 provide 会让人误以为评论区还能 inject 到第二套硬件面板。
- **How to apply**: `taskFE/app/src/composables/taskDetail/useServerConfigHardwareContext.js`；`ServerConfig.logic.vue` 调用处。

