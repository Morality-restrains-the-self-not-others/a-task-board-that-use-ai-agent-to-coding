# Architecture Version History

> 每次架构变更的完整时间线 — 谁在什么时候做了什么变更
> 由 `/1-brainstorming` 写入 target 条目，由 `/10-ship` 更新为 shipped
>
> 文件命名规范: `v<N>-<视图名>-<YYYYMMDD-HHMM>-<作者>.puml` — 版本号前置，同版本多视图自然聚合；每个版本独立文件，老版本只读保留
>
> **双 .archimate 约定**（2026-08-08 起）: 每个架构版本须同时生成 `v<N>-<视图名>-<时间戳>-<作者>.diff.archimate`（增量模型：vN-1→vN 变更元素 + Plateau/Gap/WP 链）与 `v<N>-<视图名>-<时间戳>-<作者>.full.archimate`（全量模型：变迁后完整架构拓扑），两者均须通过 Archi `--loadModel` 验证；本文件「变更文件」清单中两个文件都要列出。
>
> **`.full` Views 继承**（2026-09-04 起）: 新版 `.full` **必须**复制上一版同视图 `.full` 再 merge，保留全部既有 `ArchimateDiagramModel`，仅追加本版变迁/全量拓扑；禁止把 `.diff` 改名当 `.full`。门禁：`python3 db/scripts/ci/check_archimate_full_inherits_views.py`。设计：`docs/superpowers/specs/2026-09-04-archimate-full-view-inheritance-design.md`。**OPT-20260904-011 已回填** v126–v132（及 archive 中 v126–v127 application-integration）Views 继承链；`--strict-debt` 现与默认门禁等价全绿。
>> **自动归档**（2026-08-06 起）: `docs/.githooks/pre-commit` 在提交新视图版本时自动保留最近 5 版，其余移入 `architecture/archive/`（只移动不删除，本文件仍为完整时间线）。脚本: `architecture/scripts/archive-old-views.sh` / `install-arch-hooks.sh`；首次归档: 2026-08-06 (commit `ce6ee17`)，application-integration 保留 v62–v65、enterprise-landscape 保留 v8–v12。

---

## v132 🎯 target — 微信服务号 API 出站经 Host sh egress

- **状态**: 🎯 target（已设计，待交付 / 白名单运维门禁）
- **迭代**: wechat-mp-egress-host-sh
- **作者**: cursor
- **设计日期**: 2026-09-04 19:20
- **交付日期**: —
- **设计文档**: `docs/superpowers/specs/2026-09-04-wechat-mp-egress-host-sh-design.md`
- **ADR**: [ADR-0059](../adr/0059-wechat-mp-api-egress-host-sh.md)
- **背景**: referral follow-qr 503；INFRA 出口 `120.36.185.132` 微信 40164；整迁 taskAuth 不合适。
- **变更文件**:
  - 🆕 `v132-application-integration-20260904-1920-cursor.puml`
  - 🆕 `v132-enterprise-landscape-20260904-1920-cursor.puml`
  - 🆕 伴生格式: `.diff.archimate` + `.full.archimate` + `.mermaid.md`（每个视图）
- **变更明细**:
  - 🟢 [NEW] `wechat-mp-egress` on Host sh `:8030`
  - 🟡 [MODIFIED] taskAuth `wechat.mpEgress` 转发 cgi-bin
  - 🔴 无废弃组件
  - ⚠️ 运维硬门禁：公众平台 IP 白名单须含 `1.117.67.121`

---

## v131 ✅ current — Git PR 回复评论 SSE 推送到任务详情

- **状态**: ✅ current（已交付）
- **迭代**: git-pr-reply-sse-fanout
- **作者**: cursor
- **设计日期**: 2026-09-04 18:15
- **交付日期**: 2026-09-04 18:20
- **设计文档**: `docs/superpowers/specs/2026-09-04-git-pr-reply-sse-fanout-design.md`
- **变更摘要**:
  - 🟡 [MODIFIED] taskTaskService — 首次 git_pr 评论插入时发布 `SSE_MESSAGE(task_git_pr_reply_created)`
  - 🟡 [MODIFIED] taskFE — `establishSSEConnection` 收到事件后 `fetchTaskDetail` 重拉 Feed
  - 复用既有 `sse_message/1_send_sse_message` + taskSSE，无新 intent 进程
- **变更文件**:
  - `v131-application-integration-20260904-1815-cursor.puml`
  - `v131-application-integration-20260904-1815-cursor.diff.archimate`
  - `v131-application-integration-20260904-1815-cursor.full.archimate`
  - `v131-application-integration-20260904-1815-cursor.mermaid.md`

---

## v130 ✅ shipped — GitLab 同步来源选自已购买区域

- **状态**: ✅ current（已交付）
- **迭代**: gitlab-sync-purchased-region-select
- **作者**: cursor
- **设计日期**: 2026-09-04 15:20
- **交付日期**: 2026-09-04 15:30
- **设计文档**: `docs/superpowers/specs/2026-09-04-gitlab-sync-purchased-region-select-design.md`
- **背景**: 「从 GitLab 同步项目」模态展示平台默认 `gitlab.daydaymoney.com`，非租户已购实例。
- **变更文件**:
  - 🆕 `v130-application-integration-20260904-1520-cursor.puml`
  - 🆕 伴生格式: `.diff.archimate` + `.full.archimate` + `.mermaid.md`
  - 未改 enterprise-landscape（仅应用集成数据流）
  - Archi `--loadModel`: `.diff` / `.full` 均 `Loaded model:`
- **变更明细**:
  - 🟡 [MODIFIED] taskFE 同步模态：先拉 taskBill `gitlab-resources`，用户选区域后再带 `gitlab_host` 调 remote-repos
  - 🟢 [NEW 消费面] 同步路径只读已购 `billing_tenant_gitlab_resource` + `gitlab_web_url`
  - 🔴 无废弃组件
  - ⚠️ Python 门禁 not_applicable

---

## v129 📦 archived — runAll 金丝雀平滑重启

- **状态**: ✅ current（已交付）
- **迭代**: runall-smooth-canary-restart
- **作者**: cursor
- **设计日期**: 2026-09-03 00:40
- **交付日期**: 2026-09-03 00:50
- **设计文档**: `docs/superpowers/specs/2026-09-03-runall-smooth-canary-restart-design.md`
- **ADR**: [ADR-0058](../adr/0058-runall-smooth-canary-restart.md)
- **背景**: 全部重启 StopAll+StartAll 与精准 restart 的 kill-then-start 造成空窗；Go 服务不 Shutdown。
- **变更文件**:
  - 🆕 `v129-application-integration-20260903-0040-cursor.puml`
  - 🆕 `v129-enterprise-landscape-20260903-0040-cursor.puml`
  - 🆕 伴生格式: `.diff.archimate` + `.full.archimate` + `.mermaid.md`
  - Archi `--loadModel`: 4 文件均 `Loaded model:`；`archimate-tool.py check` 0 critical
- **变更明细**:
  - 🟢 [NEW] `tracelog.ListenAndServe`（SO_REUSEPORT + Shutdown）
  - 🟡 [MODIFIED] runAll RestartAll / precise swap → overlap+drain
  - 🔴 [DEPRECATED] 全部重启先 StopAll 再 StartAll 的空窗语义
  - ⚠️ Python 门禁 not_applicable

---

## v128 📦 archived — 阿里云 GitLab 区域目录先行

- **状态**: 📦 archived
- **迭代**: aliyun-gitlab-region-manual-node
- **作者**: cursor
- **设计日期**: 2026-09-02 14:10
- **交付日期**: 2026-09-02 14:20
- **设计文档**: `docs/superpowers/specs/2026-09-02-aliyun-gitlab-region-manual-node-design.md`
- **ADR**: [ADR-0057](../adr/0057-aliyun-gitlab-region-pending-node.md)（扩展 ADR-0014）
- **背景**: 购买页 GitLab 区域仅腾讯云已部署实例；用户要选阿里云地域，由平台人工建节点挂载后开通。
- **变更文件**:
  - 🆕 `v128-application-integration-20260902-1410-cursor.puml` (基于 v127)
  - 🆕 `v128-enterprise-landscape-20260902-1410-cursor.puml` (基于 v127)
  - 🆕 伴生格式: `.diff.archimate` (增量变迁) + `.full.archimate` (全量拓扑) + `.mermaid.md` (每个视图)
- **变更明细**:
  - 🟢 [NEW] 阿里云 GitLab 区域目录 `pending_node`；人工建节点履约过程；Kafka `GitlabManualNodeFulfillmentQueued` / `GitlabRegionInfraMarkedReady`
  - 🟡 [MODIFIED] taskFE OrderCreate 按云厂商分组；taskBill 开通门闩跳过未部署实例
  - 🔴 无废弃组件
  - ⚠️ Python 门禁 not_applicable（无新增 Python HTTP 接口）
  - Archi `--loadModel`: 4 文件均 `Loaded model:`；`archimate-tool.py check` 0 critical

---

## v127 — 部署 9999 编排源码编译

- **状态**: archived（已被 v128 取代为 current）
- **迭代**: deploy-9999-source-compile-restart
- **作者**: cursor
- **设计日期**: 2026-09-02 13:35
- **交付日期**: 2026-09-02 13:45
- **设计文档**: `docs/superpowers/specs/2026-09-02-deploy-9999-source-compile-restart-design.md`
- **ADR**: [ADR-0056](../adr/0056-deploy-9999-source-compile-restart.md)（局部取代 ADR-0052「精准编译重启仅源码树开发工具」）
- **背景**: 源码/部署分离后 `DEPLOY_MODE=1` 清空 `build_command`；日常靠 `update.sh` 全量拷 artifacts/conf-local 并重启 runAll。
- **变更文件**:
  - 🆕 `v127-application-integration-20260902-1335-cursor.puml` (基于 v126)
  - 🆕 `v127-enterprise-landscape-20260902-1335-cursor.puml` (基于 v126)
  - 🆕 伴生格式: `.diff.archimate` (增量变迁) + `.full.archimate` (全量拓扑) + `.mermaid.md` (每个视图)
- **变更明细**:
  - 🟢 [NEW] `SOURCE_ROOT` 编译面、`precise-compile.sh` 编排、登记读源码仓 `.runall/`
  - 🟡 [MODIFIED] runAll `:9999` PreciseRestart/BuildAll；编译成功后 `rsync -a --delete` conf-local + 增量 install
  - 🔴 [DEPRECATED] 日常路径依赖 `update.sh` 全量 `cp` + 重启 runAll（脚本保留应急）
  - ⚠️ Python 门禁 not_applicable（无新增 Python HTTP 接口）

---

## v126 📦 archived — 项目 L2 种下自动运行评论

- **状态**: 📦 archived
- **迭代**: 项目 L2 种下自动运行评论
- **作者**: cursor
- **设计日期**: 2026-09-01 22:44
- **交付日期**: 2026-09-01 23:10
- **设计文档**: `docs/superpowers/specs/2026-09-01-work-panel-create-task-oauth-after-project-grant-design.md`
- **ADR**: [ADR-0055](../adr/0055-project-l2-seeds-autorun-comment-grant.md)（局部取代 ADR-0049「项目 L2 永不种到自动运行评论」）
- **背景**: 项目详情 `grant_kind=project` 只写项目 L2、不签发 `grant_ticket`；工作面板创建自动运行只认 session ticket，同一用户对已授权项目仍被拦截。
- **变更文件**:
  - 🆕 `v126-application-integration-20260901-2244-cursor.puml` (基于 v125)
  - 🆕 `v126-enterprise-landscape-20260901-2244-cursor.puml` (基于 v125)
  - 🆕 伴生格式: `.diff.archimate` (增量变迁) + `.full.archimate` (全量拓扑) + `.mermaid.md` (每个视图)
- **变更明细**:
  - 🟢 [NEW] `GET /api/internal/projects/git-oauth-grant/`；创建/Fork 用项目 L2 种【自动运行】评论 L2（`via=project_l2_seed`）
  - 🟡 [MODIFIED] taskFE 创建门禁（ticket OR `validate-git-repos`+`project_id`）；`ensureAutoRunAtComment` seed；ADR-0049 自动运行条款
  - 🔴 无废弃组件
  - ⚠️ Python 门禁 not_applicable（无新增 Python HTTP 接口）

---

## v125 📦 archived — ImageMarket 恢复厂商申请入口

- **状态**: 📦 archived
- **迭代**: ImageMarket 恢复厂商申请入口
- **作者**: cursor
- **设计日期**: 2026-09-01 14:05
- **交付日期**: 2026-09-01 14:30
- **设计文档**: `docs/superpowers/specs/2026-09-01-image-market-sso-link-missing-design.md`
- **ADR**: 不新开。入口回退须 `Logic-Rollback-OK: restore ImageMarket vendor apply form; SSO still only after qualified`
- **背景**: 现网 `…/image-market` 无「厂商门户（SSO）」。Loki：账号合成邮箱、`status=none`、审核已关。用户要求申请表单回到镜像市场且从门户撤掉申请；SSO 仍仅 qualified（或关审核+真实邮箱）。
- **变更文件**:
  - 🆕 `v125-application-integration-20260901-1405-cursor.puml` (基于 v124)
  - 🆕 `v125-enterprise-landscape-20260901-1405-cursor.puml` (基于 v124)
  - 🆕 伴生格式: `.diff.archimate` (增量变迁) + `.full.archimate` (全量拓扑) + `.mermaid.md` (每个视图)
- **变更明细**:
  - 🟢 [NEW] 镜像市场申请认证业务过程；租户内 vendor-application 漏斗
  - 🟡 [MODIFIED] taskFE ImageMarket 四态；taskAiProvider 申请公网入口改为主站
  - 🔴 [DEPRECATED] provider SPA 申请认证面板（未入租户无法申请）
  - ⚠️ Python 门禁 not_applicable（无新增 Python HTTP 接口）

---

## v124 📦 archived — 新节点 clone-run（PEM 入 conf-local + 新 Release 钉）

- **状态**: 📦 archived（已被 v125 取代）
- **迭代**: daydaymoney-deploy-new-node-clone
- **作者**: cursor
- **设计日期**: 2026-08-31 18:17
- **交付日期**: 2026-08-31 18:55
- **设计文档**: `docs/superpowers/specs/2026-08-31-daydaymoney-deploy-new-node-clone-design.md`
- **ADR**: [ADR-0052](../adr/0052-binary-deploy-config-repo.md) / [ADR-0054](../adr/0054-conf-local-secrets-only.md)（不新开）
- **背景**: 只拷 conf-local 不够拉起新节点。seed `up.sh` 仍叠 `*.local.yaml`；网关/OIDC PEM 在树外；`deploy-20260831` ELF 不含 MergeConfLocal。v124：PEM 迁 conf-local，`up.sh` 自动拉新 Release，须 INFRA_HOST + Docker。
- **变更文件**:
  - 🆕 `v124-application-integration-20260831-1817-cursor.puml` (基于 v123)
  - 🆕 `v124-enterprise-landscape-20260831-1817-cursor.puml` (基于 v123)
  - 🆕 伴生格式: `.diff.archimate` (增量变迁) + `.full.archimate` (全量拓扑) + `.mermaid.md` (每个视图)
- **变更明细**:
  - 🟢 [NEW] GitHub Release 新 tag（含 MergeConfLocal）；`conf-local/gateway/task-gateway/*.pem`；`conf-local/auth/task-auth/oidc_signing_key.pem`
  - 🟡 [MODIFIED] 运维口号；`up.sh` 只 overlay conf-local + deploy-sync；taskGateway staging；taskAuth 签名钥路径
  - 🔴 [DEPRECATED] `secrets/taskGateway` / `secrets/db` overlay；`db/task-auth/oidc_signing_key.pem` 作为 SSOT
  - ⚠️ Python 门禁 not_applicable（无新增 Python HTTP 接口）

---

## v123 📦 archived — conf-local 机密收口

- **状态**: 📦 archived（已被 v124 取代）
- **迭代**: conf-local-secrets-completion
- **作者**: cursor
- **设计日期**: 2026-08-31 17:14
- **交付日期**: 2026-08-31 17:35
- **设计文档**: `docs/superpowers/specs/2026-08-31-conf-local-secrets-completion-design.md`
- **ADR**: [ADR-0054](../adr/0054-conf-local-secrets-only.md) accepted（本迭代补齐落地，不新开 ADR）
- **背景**: ADR-0054 已抽已跟踪 YAML 到 conf-local，但加载器仍在 conf-local 之后合并 `config.local.yaml`（空键抹真值）；HEAD 仍跟踪 gitLabRootPwd.md。v123 目标：只合 `conf/<app>/config.yaml` → `conf-local/<app>/config.yaml`，不读 `config.local.yaml`。基于积压的 v122，不修改 v122 文件。
- **变更文件**:
  - 🆕 `v123-application-integration-20260831-1703-cursor.puml` (基于 v122)
  - 🆕 `v123-enterprise-landscape-20260831-1703-cursor.puml` (基于 v122)
  - 🆕 伴生格式: `.diff.archimate` (增量变迁) + `.full.archimate` (全量拓扑) + `.mermaid.md` (每个视图)
- **变更明细**:
  - 🟢 [NEW] `conf-local/` 为唯一 overlay（机密 + 本机非机密）；`conf-local.example` 键名骨架
  - 🟡 [MODIFIED] confload 只合 `config.yaml` + `conf-local`（两步）
  - 🔴 [DEPRECATED] 加载路径不再读 `config.local.yaml` / `*.local.yaml`；`gitLabRootPwd.md`
  - ⚠️ Python 门禁 not_applicable（无新增 Python HTTP 接口）

---

## v122 📦 archived — 二进制部署与独立配置仓

- **状态**: 📦 archived（已被 v123 取代；puml 仅改 @status）
- **迭代**: binary-deploy-config-repo
- **作者**: cursor
- **设计日期**: 2026-08-30 15:31
- **交付日期**: —
- **设计文档**: `docs/superpowers/specs/2026-08-30-binary-deploy-config-repo-design.md`
- **ADR**: [ADR-0052](../adr/0052-binary-deploy-config-repo.md) accepted
- **背景**: 部署机不 clone 源码；运行时 conf 整棵迁入 GitHub 私有仓 `daydaymoney-deploy`；产物走 GitHub Packages。
- **变更文件**:
  - 🆕 `v122-application-integration-20260830-1531-cursor.puml` (基于 v121)
  - 🆕 `v122-enterprise-landscape-20260830-1531-cursor.puml` (基于 v121)
  - 🆕 伴生格式: `.diff.archimate` (增量变迁) + `.full.archimate` (全量拓扑) + `.mermaid.md` (每个视图)
- **变更明细**:
  - 🟢 [NEW] `daydaymoney-deploy` 配置仓；GitHub Packages；`$DEPLOY_ROOT`；`deploy-sync`；`releases.yaml`
  - 🟡 [MODIFIED] runAll 读 `CONF_ROOT`；`confload.FindConfigRoot`；Docker 配方归属配置仓
  - 🔴 [DEPRECATED] 源码仓 `conf/` 作为运行时 SSOT（P3 后仅 `conf.example/`）
  - ⚠️ Python 门禁 not_applicable（无新增 Python HTTP 接口）

---

## v121 ✅ current — 意见与建议链接（租户累计消耗可见）

- **状态**: ✅ current（已交付）
- **迭代**: tenant-feedback-links-by-consumption
- **作者**: cursor
- **设计日期**: 2026-08-30 09:22
- **交付日期**: 2026-08-30
- **设计文档**: `docs/superpowers/specs/2026-08-30-tenant-feedback-links-by-consumption-design.md`
- **背景**: 超管配置多组外链，租户侧栏「意见与建议」按组展示；可见度按租户历史累计资源消耗 **≥** 且多资源 AND。同一租户全员同一套链接。资源种类可扩展。
- **变更文件**:
  - 🆕 `v121-application-integration-20260830-0922-cursor.puml` (基于 v120)
  - 🆕 `v121-enterprise-landscape-20260830-0922-cursor.puml` (基于 v120)
  - 🆕 伴生格式: `.diff.archimate` (增量变迁) + `.full.archimate` (全量拓扑) + `.mermaid.md` (每个视图)
- **变更明细**:
  - 🟢 [NEW] `billing_feedback_link_group` / `billing_feedback_link_threshold` / `billing_feedback_link` / `billing_feedback_resource_kind`；`FEEDBACK_LINK_GROUP_CREATED|UPDATED|DELETED`
  - 🟡 [MODIFIED] taskBill 超管/租户 API 与消耗求值；taskFE 侧栏 + 超管配置页
  - 🔴 无
  - ⚠️ Python 门禁 not_applicable（Go taskBill + Vue）

---

## v120 📦 archived — 任务与项目内容历史版本

- **状态**: 📦 archived（已被 v121 取代）
- **迭代**: task-project-entity-revisions
- **作者**: cursor
- **设计日期**: 2026-08-30 00:40
- **交付日期**: 2026-08-30
- **设计文档**: `docs/superpowers/specs/2026-08-30-task-project-entity-revisions-design.md`
- **ADR**: [ADR-0051](../adr/0051-task-project-entity-revisions.md) accepted
- **背景**: 任务/项目标题与正文只留热表最新值；主要属性变更需不可变快照并可查阅。基于 v118；v119 正交未交付故未纳入本基线。
- **变更文件**:
  - 🆕 `v120-application-integration-20260830-0040-cursor.puml` (基于 v118)
  - 🆕 `v120-enterprise-landscape-20260830-0040-cursor.puml` (基于 v118)
  - 🆕 伴生格式: `.diff.archimate` (增量变迁) + `.full.archimate` (全量拓扑) + `.mermaid.md` (每个视图)
- **变更明细**:
  - 🟢 [NEW] `task_revision`；`project_revision`；`TASK_REVISION_RECORDED`；`PROJECT_REVISION_RECORDED`
  - 🟡 [MODIFIED] taskTaskService 创建/更新同事务快照 + GET revisions；taskProjectService 对称；taskFE 历史面板；taskEvents 1_observe 18072/18073
  - 🔴 无
  - ⚠️ Python 门禁 not_applicable（Go + Vue）

---

## v118 📦 archived — Git OAuth 资源使用标记

- **状态**: 📦 archived（已被 v120 取代为 current）
- **迭代**: git-oauth-resource-grant-marker
- **作者**: cursor
- **设计日期**: 2026-08-29 18:32
- **交付日期**: 2026-08-29
- **设计文档**: `docs/superpowers/specs/2026-08-29-git-oauth-resource-grant-marker-design.md`
- **ADR**: [ADR-0049](../adr/0049-git-oauth-resource-grant-marker.md) accepted
- **背景**: L1 凭据仍为 remote_userId 一份；项目页与评论/自动运行分别打本站 L2 后才能换票。项目不会自动运行。Git 主体为启用自动运行的人或发评人。
- **变更文件**:
  - 🆕 `v118-application-integration-20260829-1832-cursor.puml` (基于 v117)
  - 🆕 `v118-enterprise-landscape-20260829-1832-cursor.puml` (基于 v117)
  - 🆕 各视图 `.diff.archimate` / `.full.archimate` / `.mermaid.md`
- **变更明细**:
  - 🟢 [NEW] `project_git_oauth_grant`；`PROJECT_GIT_OAUTH_GRANTED` / `COMMENT_GIT_OAUTH_GRANTED`；grant_ticket
  - 🟡 [MODIFIED] `taskGitOauth` callback + state；`task_comments.repo_identities_json` grant 字段；换票前查 L2；排队用评论 `created_by_id`
  - 🔴 [DEPRECATED] 以 L1 `connected` 单独作为资源「已授权」
  - ⚠️ Python 门禁 not_applicable（Go + Vue）

---

## v117 📦 archived — 开放式邀请链接

- **状态**: 📦 archived（已被 v118 取代）
- **迭代**: open-invite-link
- **作者**: cursor
- **设计日期**: 2026-08-29 13:10
- **交付日期**: 2026-08-29
- **设计文档**: `docs/superpowers/specs/2026-08-29-open-invite-link-design.md`
- **背景**: 租户「复制邀请链接」仅能单次使用；需支持开放式链接供多人加入。
- **变更文件**:
  - 🆕 `v117-application-integration-20260829-1310-cursor.puml` (基于 v116)
  - 🆕 `v117-enterprise-landscape-20260829-1310-cursor.puml` (基于 v116)
  - 🆕 各视图 `.diff.archimate` / `.full.archimate` / `.mermaid.md`
- **变更明细**:
  - 🟢 [NEW] `tenant_invitation_redemption`
  - 🟡 [MODIFIED] `tenant_invitation` 增 `link_kind`/`max_uses`/`use_count`；invite/validate/join/pending；taskFE 邀请人 UI
  - ⚠️ Python 门禁 not_applicable（Go + Vue）

---

## v116 📦 archived — 邮件邀请退订

- **状态**: 📦 archived
- **迭代**: email-invite-unsubscribe
- **作者**: cursor
- **设计日期**: 2026-08-29 09:40
- **交付日期**: 2026-08-29
- **设计文档**: `docs/superpowers/specs/2026-08-29-email-invite-unsubscribe-design.md`
- **权限分析**: `docs/superpowers/specs/2026-08-29-email-invite-unsubscribe-permission-analysis.md`
- **背景**: 邀请邮件增加退订按钮；退订列表由 taskAuth 持有；再发邮件邀请时跳过 SMTP 并提示复制链接。
- **变更文件**:
  - 🆕 `v116-application-integration-20260829-0940-cursor.puml` (基于 v115)
  - 🆕 `v116-enterprise-landscape-20260829-0940-cursor.puml` (基于 v115)
  - 🆕 各视图 `.diff.archimate` / `.full.archimate` / `.mermaid.md`
- **变更明细**:
  - 🟢 [NEW] `auth_email_unsubscription`；公开退订 API；内部查询；事件 `EMAIL_UNSUBSCRIBED`
  - 🟡 [MODIFIED] 租户/超管邀请跳过 SMTP；taskEvents 模板与纵深检查；taskFE 确认页与复制提示
  - ⚠️ Python 门禁 not_applicable（Go + Vue）

---

## v115 📦 archived — 工作空间排队调度历史

- **状态**: 📦 archived（已被 v116 取代）
- **迭代**: queue-schedule-history
- **作者**: cursor
- **交付日期**: 2026-08-28 07:30
- **设计文档**: `docs/superpowers/specs/2026-08-28-queue-schedule-history-design.md`
- **权限分析**: `docs/superpowers/specs/2026-08-28-queue-schedule-history-permission-analysis.md`
- **背景**: 「自动调度安排」状态栏只展示当前时段/槽位，无法阅读调度轨迹。新增 append-only `task_queued_schedule_history`、快照 `recent_history` 与分页 GET history，前端在状态栏下展示调度历史卡。
- **变更文件**:
  - 🆕 `v115-application-integration-20260828-0720-cursor.puml` (基于 v114)
  - 🆕 `v115-enterprise-landscape-20260828-0720-cursor.puml` (基于 v114)
  - 🆕 `v115-application-integration-20260828-0720-cursor.diff.archimate`（增量变迁）
  - 🆕 `v115-application-integration-20260828-0720-cursor.full.archimate`（全量拓扑）
  - 🆕 `v115-enterprise-landscape-20260828-0720-cursor.diff.archimate`（增量变迁）
  - 🆕 `v115-enterprise-landscape-20260828-0720-cursor.full.archimate`（全量拓扑）
  - 🆕 伴生 `.mermaid.md`（每个视图）
- **变更明细**:
  - 🟢 [NEW] `task_queued_schedule_history` 月分区表；`workspace_schedule_rhythms.last_in_window`
  - 🟢 [NEW] GET `/queue-schedule/history/`；快照 `recent_history`；事件 `WorkspaceScheduleWindowEntered/Exited`
  - 🟡 [MODIFIED] taskTaskService 状态变化 fail-open append；taskFE 状态栏下 `ScheduleHistoryCard`
  - ⚠️ Python 门禁 not_applicable（Go + Vue）

---

## v114 📦 archived — 评论启动日志 COS 归档

- **状态**: 📦 archived（已被 v115 取代）
- **迭代**: ccb-startup-logs-cos-archive
- **作者**: cursor
- **交付日期**: 2026-08-27 02:15
- **设计文档**: `docs/superpowers/specs/2026-08-27-startup-logs-cos-archive-design.md`
- **权限分析**: `docs/superpowers/specs/2026-08-27-startup-logs-cos-archive-permission-analysis.md`
- **背景**: 工作台「启动日志」仅落 ADR-0023 MySQL 分片；产品要求同时写入腾讯云 COS（复用 step_full 客户端与 bucket，独立 pathRule）。
- **变更文件**:
  - 🆕 `v114-application-integration-20260827-0200-cursor.puml` (基于 v113)
  - 🆕 `v114-enterprise-landscape-20260827-0200-cursor.puml` (基于 v113)
  - 🆕 `v114-application-integration-20260827-0200-cursor.diff.archimate`（增量变迁）
  - 🆕 `v114-application-integration-20260827-0200-cursor.full.archimate`（全量拓扑）
  - 🆕 `v114-enterprise-landscape-20260827-0200-cursor.diff.archimate`（增量变迁）
  - 🆕 `v114-enterprise-landscape-20260827-0200-cursor.full.archimate`（全量拓扑）
  - 🆕 伴生 `.mermaid.md`（每个视图）
- **变更明细**:
  - 🟢 [NEW] `cloud_comment_startup_log_object` 评论级 COS 指针
  - 🟢 [NEW] COS 对象 `startup_logs.json`；事件 `CommentStartupLogArchived`
  - 🟡 [MODIFIED] `insertCCBLogRow` 后 best-effort 合并 COS；Put 成功驱逐分片；list COS 优先
  - 🟡 [MODIFIED] 管理员 COS 页增加 `startupLogsPathRule`
  - ⚠️ 全部新接口在 Go taskCloudService → 🐍 Python 门禁 not_applicable

---

## v113 📦 archived — 推荐资格服务号动态 scene 码

- **状态**: 📦 archived（已被 v114 取代）
- **迭代**: referral-mp-dynamic-qr-ticket
- **作者**: cursor
- **设计日期**: 2026-08-26 15:50
- **交付日期**: 2026-08-26 16:35
- **设计文档**: `docs/superpowers/specs/2026-08-26-referral-mp-dynamic-qr-ticket-design.md`
- **权限分析**: `docs/superpowers/specs/2026-08-26-referral-mp-dynamic-qr-ticket-permission-analysis.md`
- **背景**: 静态服务号二维码无法携带 scene，已关注用户再扫通常不推 SCAN；回调只处理 subscribe。改为 Snowflake 临时 ID 动态码，关注/SCAN 对账后取 unionId；已绑定其他账号则不抢绑，按临时 ID 提醒当前用户。
- **变更文件**:
  - 🆕 `v113-application-integration-20260826-1550-cursor.puml` (基于 v112)
  - 🆕 `v113-enterprise-landscape-20260826-1550-cursor.puml` (基于 v112)
  - 🆕 `v113-application-integration-20260826-1550-cursor.diff.archimate`（增量变迁）
  - 🆕 `v113-application-integration-20260826-1550-cursor.full.archimate`（全量拓扑）
  - 🆕 `v113-enterprise-landscape-20260826-1550-cursor.diff.archimate`（增量变迁）
  - 🆕 `v113-enterprise-landscape-20260826-1550-cursor.full.archimate`（全量拓扑）
  - 🆕 伴生 `.mermaid.md`（每个视图）
- **变更明细**:
  - 🟢 [NEW] `POST /api/auth/wechat/mp/follow-qr/`（token + Idempotency-Key）
  - 🟢 [NEW] `auth_wechat_mp_follow_ticket`（Snowflake temp_id = scene）
  - 🟢 [NEW] 业务过程「签发临时 ID 动态码」「冲突则提醒当前用户」
  - 🟡 [MODIFIED] 回调处理 SCAN + EventKey `qrscene_`；unionId 占用则 conflict 不 upsert
  - 🟡 [MODIFIED] GET follow-status 增加 `ticket_status` / `conflict_code` / `message`
  - 🟡 [MODIFIED] taskFE 闸门改用动态码，废弃静态 `jjf_qrcode.png` 作为凭证
  - 🟡 [MODIFIED] APISIX 增加 follow-qr token 路由
  - ⚠️ 全部新接口在 Go taskAuth → 🐍 Python 门禁 not_applicable

---

## v112 📦 archived — 推荐资格服务号关注闸门

- **状态**: 📦 archived（已被 v113 取代）
- **迭代**: referral-mp-follow-gate
- **作者**: cursor
- **设计日期**: 2026-08-26 09:50
- **交付日期**: 2026-08-26 10:15
- **设计文档**: `docs/superpowers/specs/2026-08-26-referral-mp-follow-gate-design.md`
- **背景**: 申请推荐资格须先关注微信支付绑定的服务号；关注事件用 unionId 锁定用户并绑定服务号 openId，再填写名称与简介。
- **变更文件**:
  - 🆕 `v112-application-integration-20260826-0950-cursor.puml`（基于 v111）
  - 🆕 `v112-enterprise-landscape-20260826-0950-cursor.puml`（基于 v111）
  - 🆕 伴生格式: `.diff.archimate` + `.full.archimate` + `.mermaid.md`（每个视图）
- **变更明细**:
  - 🟢 [NEW] 微信服务号 subscribe 回调 `GET/POST /api/auth/wechat/mp/callback/`
  - 🟢 [NEW] `auth_wechat_mp_subscribe_pending`；`WECHAT_MP_SUBSCRIBED`
  - 🟢 [NEW] `GET /api/auth/wechat/mp/follow-status/`
  - 🟡 [MODIFIED] `wechat_identity` 增加 `app_key=mp` 别名；申请须绑定
  - 🟡 [MODIFIED] taskFE 推荐资格卡先展示服务号二维码
  - ⚠️ 全部新接口在 Go → 🐍 Python 门禁 not_applicable

---

## v111 archived — 登录历史记录

- **状态**: archived（已被 v112 取代）
- **迭代**: login-history
- **作者**: cursor
- **设计日期**: 2026-08-25 22:49
- **交付日期**: 2026-08-25 23:10
- **设计文档**: `docs/superpowers/specs/2026-08-25-login-history-design.md`
- **背景**: `auth_user.last_login` 只保留最后一次时间。用户需要在账号中心查看自己的登录 IP / 入口；管理员入口登录也必须留下可区分记录。
- **变更文件**:
  - 🆕 `v111-application-integration-20260825-2249-cursor.puml`（基于 v110）
  - 🆕 `v111-enterprise-landscape-20260825-2249-cursor.puml`（基于 v110）
  - 🆕 伴生格式: `.diff.archimate` + `.full.archimate` + `.mermaid.md`（每个视图）
- **变更明细**:
  - 🟢 [NEW] `auth_login_history` 按月 RANGE 分区表；成功认证 fail-open 写入
  - 🟢 [NEW] `GET /api/auth/login-history/`（本人）与 `GET /api/system-admin/users/{id}/login-history/`（超管）
  - 🟡 [MODIFIED] `USER_LOGGED_IN` payload 增补 `client_ip` / `entry` / `user_agent`
  - 🟡 [MODIFIED] APISIX `taskauth-impersonation` 将 login-history 纳入 token 路由（高于 public `/api/auth/*`）
  - 🟢 [NEW] 账号中心侧边栏「登录历史」+ 超管用户行真实 href
  - ⚠️ 全部新接口在 Go taskAuth → 🐍 Python 门禁 not_applicable

---

## v110 📦 archived — 租户自建 GitLab 平台 OIDC SSO

- **状态**: 📦 archived（已被 v111 取代）
- **迭代**: tenant-selfhosted-gitlab-oidc-sso
- **作者**: cursor
- **设计日期**: 2026-08-25 20:26
- **交付日期**: 2026-08-25 21:10
- **设计文档**: `docs/superpowers/specs/2026-08-25-tenant-selfhosted-gitlab-oidc-sso-design.md`
- **ADR**: [ADR-0043](../adr/0043-tenant-selfhosted-gitlab-oidc-sso.md)（accepted）
- **背景**: gitlab-connection 仅能登记自建 GitLab OAuth Application（Git 网站授权）。用户需要自建 GitLab Web 也能用平台账号（taskAuth OIDC）登录。禁止复用 `gitlab-git-service*` client（区域闸门前缀冲突）；authorize 必须租户成员闸门。
- **⚠️ 积压 target**: v105 / v106 / v107 尚未 ship；本版基于已交付 current v109，不合并未交付 target。
- **变更文件**:
  - 🆕 `v110-application-integration-20260825-2026-cursor.puml`（基于 v109）
  - 🆕 `v110-enterprise-landscape-20260825-2026-cursor.puml`（基于 v109）
  - 🆕 伴生格式: `.diff.archimate` + `.full.archimate` + `.mermaid.md`（每个视图）
- **变更明细**:
  - 🟢 [NEW] 租户自建 GitLab 作为 OmniAuth OIDC RP；`gitlab-oidc-sso` API；`TenantGitLabOidcSsoEnabled/Rotated/Disabled`
  - 🟡 [MODIFIED] `auth_oidc_client`（`owner_company_id` / `purpose` / `managed_by=tenant`）；`handleOidcAuthorize` 成员闸门
  - 🟡 [MODIFIED] taskFE gitlab-connection SSO 区块 + 链路 A Application 操作说明
  - 🟡 [MODIFIED] APISIX 路由 → taskAuth
  - ⚪ 链路 A（taskGitOauth OAuth Application）行为不变
  - ⚠️ 全部新接口在 Go taskAuth → 🐍 Python 门禁 not_applicable

---

## v109 📦 archived — 系统管理租户详情（名称链接 + 配额/工作空间/订单）

- **状态**: 📦 archived（已被 v110 取代）
- **迭代**: system-admin-tenant-detail
- **作者**: cursor
- **设计日期**: 2026-08-25 19:30
- **交付日期**: 2026-08-25 19:55
- **设计文档**: `docs/superpowers/specs/2026-08-25-system-admin-tenant-detail-design.md`
- **背景**: 超管租户 Tab 公司名为纯文本；需要点进详情看剩余资源、工作空间、订单
- **变更文件**:
  - 🆕 `v109-application-integration-20260825-1930-cursor.puml`（基于 v108）
  - 🆕 `v109-enterprise-landscape-20260825-1930-cursor.puml`（基于 v108）
  - 🆕 伴生格式: `.diff.archimate` + `.full.archimate` + `.mermaid.md`（每个视图）
- **变更明细**:
  - 🟢 [NEW] GET `/api/system-admin/accounts/admin/tenants/{id}/`、`/tenant-quotas/tenant_id/{id}/`、`/tenant-workspaces/tenant_id/{id}/`
  - 🟡 [MODIFIED] taskFE 租户名链接 + 详情页；taskBill orders `tenant_id`；taskGateway URI
  - ⚠️ 纯查询，无新 MQ 事件
  - ⚠️ 全部 Go + 前端 → 🐍 Python 门禁 not_applicable

---

## v108 📦 archived — 超管微信分账单号 / 手工分账 + 动账通知

- **状态**: 📦 archived
- **迭代**: admin-profit-sharing-wechat-ids-change-notify
- **作者**: cursor
- **设计日期**: 2026-08-25 17:35
- **交付日期**: 2026-08-25 18:00
- **设计文档**: `docs/superpowers/specs/2026-08-25-admin-profit-sharing-wechat-ids-and-change-notify-design.md`
- **背景**: 超管微信分账 Tab 无法对照微信订单号/分账单号，也不能带审计缘由补发起分账；存量 profitsharing notify 未 AEAD 解密，不能填写商户后台动账通知 URL
- **变更文件**:
  - 🆕 `v108-application-integration-20260825-1735-cursor.puml`（基于 v107）
  - 🆕 `v108-enterprise-landscape-20260825-1735-cursor.puml`（基于 v107）
  - 🆕 伴生格式: `.diff.archimate` + `.full.archimate` + `.mermaid.md`（每个视图）
- **变更明细**:
  - 🟢 [NEW] `billing_profit_sharing_admin_action` 审计表；`billing_profit_sharing_change_notify` 通知收件箱
  - 🟢 [NEW] `POST /api/system-admin/profit-sharing/{id}/share/`（reason + Idempotency-Key）
  - 🟢 [NEW] `POST /api/billing/profitsharing/change-notify/`（core/notify 验签解密）
  - 🟡 [MODIFIED] 管理端列表回传 `wechat_transaction_id` / `wechat_profit_sharing_id`（仅平台员工）
  - 🟡 [MODIFIED] 存量 `/api/billing/profitsharing/notify/` 与 change-notify 共用处理函数
  - ⚠️ 无新 MQ 事件（与 executeProfitSharing 同聚合写路径）
  - ⚠️ 全部变更在 Go taskBill + taskFE + taskGateway → 🐍 Python 门禁 not_applicable

---

## v107 🎯 target — 镜像/技能 mention 存储 ID 化 + 名↔ID 映射持久化

- **状态**: 🎯 target（已设计，待交付）
- **迭代**: image-skill-id-mapping-store
- **作者**: claude
- **设计日期**: 2026-08-24 15:15
- **交付日期**: —
- **设计文档**: `docs/superpowers/specs/2026-08-24-image-skill-id-mapping-store-design.md`
- **背景**: 创建任务弹窗技能 chips 显示 `$镜像名 /技能名` mention；但任务存储仅 `installed_image_id`（镜像 ID）+ 描述内 mention **纯文本**（`$name /skill`），技能无 ID 概念、名↔ID 映射未持久化 — 镜像/技能改名或删除后历史任务无法反解。本版将存储契约升级为 ID 化
- **变更文件**:
  - 🆕 `v107-application-integration-20260824-1515-claude.puml` (基于 v106)
  - 🆕 `v107-enterprise-landscape-20260824-1515-claude.puml` (基于 v106)
  - 🆕 伴生格式: `.diff.archimate` (增量变迁) + `.full.archimate` (全量拓扑) + `.mermaid.md`（每个视图，均已通过 Archi `--loadModel`）
- **变更明细**:
  - 🟢 [NEW] 技能服务端派生 ID — taskCloudService 提取 `imageSkills.yaml` 时生成稳定 `sk_<hash>[:12]` 写入 `image_skills_json`，存量回填迁移（用户批准 D1=B）
  - 🟢 [NEW] `task_tasks.image_skill_id` 列 + `container_image_snapshot` JSON（`{image_id, image_name, skill_id, skill_name}` 名↔ID 快照，用户批准 D2=B）
  - 🟡 [MODIFIED] `cloud_tenant_installed_images.image_skills_json` — 技能项附 `id` 字段
  - 🟡 [MODIFIED] taskTaskService — create/update task 请求可选 `container_image_skill_id`，fail-closed 校验（skill_id 须属于该镜像 image_skills_json）；响应 `container_image.skill = {id, name}`（老客户端容错）
  - 🟡 [MODIFIED] taskFE — 保存时携带 `container_image_skill_id`；显示仍为 `$镜像名 /技能名` 可读 mention
  - ⚠️ 纯存储契约变更：无新 MQ 事件（意图「保存名↔ID 映射」不改变业务事实、无跨边界副作用，已记例外）
  - ⚠️ 全部变更在 Go（taskTaskService/taskCloudService）+ 前端（taskFE）→ 🐍 Python 门禁 not_applicable

---

## v106 🎯 target — 安全响应头补齐 + HTTP→HTTPS 强制跳转

- **状态**: 🎯 target（已设计，待交付）
- **迭代**: security-headers-hsts-https-redirect
- **作者**: claude
- **设计日期**: 2026-08-24 08:54
- **交付日期**: —
- **设计文档**: `docs/superpowers/specs/2026-08-24-security-headers-hsts-risk-analysis-design.md`
- **背景**: 实测公网响应无 HSTS/CSP/X-Frame-Options/X-Content-Type-Options/Referrer-Policy；无 HTTP→HTTPS 跳转配置记录。**扫描目标面 = 公网 daydaymoney.com（用户确认）；内网 10.2.150.68 各端口 out of scope（无 TLS/PKI，HSTS 对 IP 无效）**
- **变更文件**:
  - 🆕 `v106-application-integration-20260824-0854-claude.puml` (基于 v104)
  - 🆕 `v106-enterprise-landscape-20260824-0854-claude.puml` (基于 v104)
  - 🆕 伴生格式: `.diff.archimate` (增量变迁) + `.full.archimate` (全量拓扑) + `.mermaid.md`（每个视图，均已通过 Archi `--loadModel`）
- **变更明细**:
  - 🟡 [MODIFIED] 边缘 nginx (Host :443)：统一加安全响应头（HSTS 阶梯 300→86400→31536000、CSP Report-Only 灰度后 enforce、X-Frame-Options SAMEORIGIN、X-Content-Type-Options nosniff、Referrer-Policy strict-origin-when-cross-origin）+ :80→:443 308 强制跳转
  - 🟡 [MODIFIED] Let's Encrypt wildcard 证书：新增续签监控（HSTS 后证书到期=全站不可达）
  - 🟢 [NEW] 约束：HSTS 阶梯灰度、CSP Report-Only 灰度、Referrer-Policy 保留 Origin（taskGitOauth febFromRequest 回跳解析依赖）
  - ⚠️ 例外记录：主站边缘 nginx 配置位于宿主机 `/etc/nginx/conf.d/`（repo 外），repo 交付 example + runbook，Host apply 后须复扫验证
  - ⚠️ 前置工程：`dist/index.html` 内联脚本（`__TASK2APP_API_BASE_URL__`）须外置后方可启用严格 CSP

---

## 📝 docs-cleanup 2026-08-24 — 存储平台表述修正（SQLite per-service → MySQL per-service）

- **状态**: ✅ current（文档清理，无架构变更）
- **迭代**: sqlite-dual-db-doc-cleanup
- **作者**: claude
- **设计日期**: 2026-08-24
- **变更文件**: 无新版本 puml / .archimate（纯文档修正）
- **变更明细**:
  - 🟡 [MODIFIED] v12 ⚪ UNCHANGED 行「SQLite per-service」→「MySQL per-service」（v12 回填时沿袭 v11 旧口径，与运行事实不符）
  - 🟡 [MODIFIED] `README.md`、`table-to-owner.md`、`docs/runbooks/sqlite-comment-dbs-backup.md` 的 SQLite 表述 → MySQL
  - 🔴 [REMOVED] 物理空残留文件：`taskCloudService/task_cloud.db`（0B）、`taskAuth/auth.db`（空）、`taskAuth/data/auth.db`（2026-05-31 陈旧空库）
- **背景**: 存储引擎已于 2026-05~08 渐次 SQLite→MySQL（`db/registry.yaml` 全部 `driver: mysql`，`path` 字段标注 legacy SQLite 仅用于清理）；评论/任务「双库」分库模式保留（MySQL `task_task` / `task_ai_comment` 两库），设计决策见 `docs/superpowers/specs/2026-07-15-task-comments-sql-vs-nosql-scale-design.md`。MySQL 化未形成独立架构版本条目，导致文档层滞留 SQLite 表述，本次一次性修正。

---

## v105 🎯 target — 测试角色 + GitLab 区域开发/发布模式

- **状态**: 🎯 target（设计中，实现随本会话交付）
- **迭代**: tester-role-gitlab-region-access-mode
- **作者**: cursor
- **设计日期**: 2026-08-23 20:56
- **设计文档**: `docs/superpowers/specs/2026-08-23-tester-role-gitlab-region-access-mode-design.md`
- **ADR**: [ADR-0041](../adr/0041-tester-role-gitlab-region-access-mode.md)
- **变更文件**:
  - 🆕 `v105-application-integration-20260823-2056-cursor.puml` (基于 v104)
  - 🆕 `v105-enterprise-landscape-20260823-2056-cursor.puml` (基于 v104)
  - 🆕 伴生格式: `.diff.archimate` + `.full.archimate` + `.mermaid.md`（每个视图）
- **变更明细**:
  - 🟢 [NEW] `auth_user.is_tester` + `X-User-Is-Tester` + UserTesterFlagChanged
  - 🟢 [NEW] `billing_gitlab_region.access_mode` + GitlabRegionAccessModeChanged
  - 🟡 [MODIFIED] 租户 GitLab 区域目录/购买按测试角色过滤开发模式

## v104 ✅ current — ztree 层级落库 + step_full.json COS 归档

- **状态**: ✅ current（已交付）
- **迭代**: ztree-step-full-cos-archive
- **作者**: cursor
- **设计日期**: 2026-08-23 14:00
- **交付日期**: 2026-08-23 14:15
- **设计文档**: `docs/superpowers/specs/2026-08-23-ztree-step-full-cos-archive-design.md`
- **ADR**: [ADR-0039](../adr/0039-ztree-step-full-cos-archive.md)
- **变更文件**:
  - 🆕 `v104-application-integration-20260823-1400-cursor.puml` (基于 v103)
  - 🆕 `v104-enterprise-landscape-20260823-1400-cursor.puml` (基于 v103)
  - 🆕 伴生格式: `.diff.archimate` (增量变迁) + `.full.archimate` (全量拓扑) + `.mermaid.md`（每个视图）
- **变更明细**:
  - 🟢 [NEW] `cloud_job_step_full_object` + Tencent COS step_full.json
  - 🟢 [NEW] JobStepFullArchived / StepFullCOSConfigUpdated
  - 🟢 [NEW] 系统管理 COS 参数页
  - 🟡 [MODIFIED] GET container-job-execution-log 优先 COS
  - 🟡 [MODIFIED] 层图快照继续作为 ztree 层级 DB SSOT

## v103 ✅ delivered — 模拟登录审计标识与用户收信箱

- **状态**: ✅ delivered（已被 v104 取代为 current）
- **迭代**: impersonation-audit-inbox
- **作者**: cursor
- **设计日期**: 2026-08-23 07:20
- **交付日期**: 2026-08-23 07:40
- **设计文档**: `docs/superpowers/specs/2026-08-23-impersonation-audit-inbox.md`
- **ADR**: [ADR-0038](../adr/0038-impersonation-audit-label-and-inbox.md)
- **变更文件**:
  - 🆕 `v103-application-integration-20260823-0720-cursor.puml` (基于 v102)
  - 🆕 `v103-enterprise-landscape-20260823-0720-cursor.puml` (基于 v102)
  - 🆕 伴生格式: `.diff.archimate` (增量变迁) + `.full.archimate` (全量拓扑) + `.mermaid.md`（每个视图）
- **变更明细**:
  - 🟢 [NEW] `auth_user_inbox_message` + UserInboxMessageCreated
  - 🟢 [NEW] 模拟登录必填理由（8–500 字）写入会话并通知被模拟用户
  - 🟡 [MODIFIED] 访问日志/slog/forward-auth 标识 impersonator 与 session
  - 🟡 [MODIFIED] taskFE 理由弹窗与侧栏「收信箱」

## v102 📦 archived — 管理员模拟用户登录

- **状态**: 📦 archived（已被 v103 取代为 current）
- **迭代**: admin-user-impersonation
- **作者**: cursor
- **设计日期**: 2026-08-23 06:20
- **交付日期**: 2026-08-23 06:50
- **设计文档**: `docs/superpowers/specs/2026-08-23-admin-user-impersonation-design.md`
- **ADR**: [ADR-0037](../adr/0037-admin-user-impersonation.md)
- **变更文件**:
  - 🆕 `v102-application-integration-20260823-0620-cursor.puml` (基于 v101)
  - 🆕 `v102-enterprise-landscape-20260823-0620-cursor.puml` (基于 v101)
  - 🆕 伴生格式: `.diff.archimate` (增量变迁) + `.full.archimate` (全量拓扑) + `.mermaid.md`（每个视图）
- **变更明细**:
  - 🟢 [NEW] `auth_impersonation_session` + UserImpersonationStarted/Stopped
  - 🟡 [MODIFIED] taskAuth impersonate/stop/status 与 forward-auth
  - 🟡 [MODIFIED] taskFE 用户编辑页按钮与 Navbar 模拟横幅

## v101 ✅ — runAll 编排器与托管服务生命周期解耦

- **状态**: ✅ shipped（已被 v102 接续）
- **迭代**: runall-orchestrator-independence
- **作者**: cursor
- **设计日期**: 2026-08-23 02:10
- **交付日期**: 2026-08-23 02:20
- **设计文档**: `docs/superpowers/specs/2026-08-23-runall-orchestrator-independence-design.md`
- **ADR**: [ADR-0035](../adr/0035-runall-orchestrator-independence.md)
- **变更文件**:
  - 🆕 `v101-application-integration-20260823-0210-cursor.puml` (基于 v100)
  - 🆕 `v101-enterprise-landscape-20260823-0210-cursor.puml` (基于 v100)
  - 🆕 伴生格式: `.diff.archimate` (增量变迁) + `.full.archimate` (全量拓扑) + `.mermaid.md`（每个视图）
- **变更明细**:
  - 🟢 [NEW] 托管服务 Linux 独立 session（Setsid）
  - 🟡 [MODIFIED] runAll UI 退出默认保留托管进程；新实例 adopt
  - 🟡 [MODIFIED] 显式 stop-all 仍为拆栈入口

## v100 📦 archived — 订单电子发票开具与退款红冲重开

- **状态**: 📦 archived（已被 v101 取代为 current）
- **迭代**: order-wechat-fapiao
- **作者**: cursor
- **设计日期**: 2026-08-23 01:25
- **交付日期**: 2026-08-23 01:45
- **设计文档**: `docs/superpowers/specs/2026-08-23-order-invoice-wechat-fapiao-design.md`
- **ADR**: [ADR-0034](../adr/0034-order-wechat-fapiao.md)
- **变更文件**:
  - 🆕 `v100-application-integration-20260823-0125-cursor.puml` (基于 v99)
  - 🆕 `v100-enterprise-landscape-20260823-0125-cursor.puml` (基于 v99)
  - 🆕 伴生格式: `.diff.archimate` (增量变迁) + `.full.archimate` (全量拓扑) + `.mermaid.md`（每个视图）
- **变更明细**:
  - 🟢 [NEW] `billing_invoice_application` / `billing_invoice` 挂 `order_id`
  - 🟢 [NEW] 租户申请开票 + 管理员审批后微信开具插卡
  - 🟢 [NEW] 退款获批后全额红冲原蓝票并按剩余成交金额重开
  - 🟡 [MODIFIED] 订单详情支付成功区、系统管理订单页发票 Tab
  - 🟡 [MODIFIED] APISIX 发票审批与 fapiao notify

## v99 📦 archived — 管理端待分账订单列表

- **状态**: 📦 archived
- **迭代**: admin-pending-profit-sharing-list
- **作者**: cursor
- **设计日期**: 2026-08-22 21:55
- **交付日期**: 2026-08-22 22:10
- **设计文档**: `docs/superpowers/specs/2026-08-22-admin-pending-profit-sharing-list-design.md`
- **ADR**: 沿用 [ADR-0030](../adr/0030-admin-only-order-profit-sharing.md)（不新增）
- **变更文件**:
  - 🆕 `v99-application-integration-20260822-2155-cursor.puml` (基于 v98)
  - 🆕 `v99-enterprise-landscape-20260822-2155-cursor.puml` (基于 v98)
  - 🆕 伴生格式: `.diff.archimate` (增量变迁) + `.full.archimate` (全量拓扑) + `.mermaid.md`（每个视图）
- **变更明细**:
  - 🟢 [NEW] `GET /api/system-admin/profit-sharing/` 平台员工待分账队列（默认 status=open）
  - 🟢 [NEW] taskFE「待分账」Tab / `SystemAdminProfitSharingPanel`
  - 🟡 [MODIFIED] APISIX 增加 `/api/system-admin/profit-sharing/` 前缀
  - 🟡 [MODIFIED] `SystemAdminOrderRecords` Tab 栏
- **并行 target 说明**: v97 Git OAuth site 寻址仍为正交 target，勿与本版混改。基线为 v98 current。

## v98 ✅ archived — 指令闲置回收

- **状态**: ✅ archived（已被 v99 取代为 current）
- **迭代**: instruction-idle-recycle
- **作者**: cursor
- **设计日期**: 2026-08-22 15:55
- **交付日期**: 2026-08-22 17:20
- **设计文档**: `docs/superpowers/specs/2026-08-22-instruction-idle-recycle-design.md`
- **ADR**: [ADR-0031](../adr/0031-instruction-idle-recycle-no-sts-in-comments.md)（accepted）
- **变更文件**:
  - 🆕 `v98-application-integration-20260822-1555-cursor.puml` (基于 v96)
  - 🆕 `v98-enterprise-landscape-20260822-1555-cursor.puml` (基于 v96)
  - 🆕 伴生格式: `.diff.archimate` (增量变迁) + `.full.archimate` (全量拓扑) + `.mermaid.md`（每个视图）
- **变更明细**:
  - 🟢 [NEW] 容器 task-detail `idle_recycle_minutes`；CSC `instruction_idle_since`
  - 🟢 [NEW] 事件 `CONTAINER_INSTRUCTION_IDLE_MARKED` / `CONTAINER_INSTRUCTION_IDLE_CLEARED`
  - 🟡 [MODIFIED] onlineServiceJS 交付后倒计时 + 同容器抢占 + `request-machine-release`
  - 🟡 [MODIFIED] taskEvents recycle timer 识别指令闲置（`server_url` 仍可非空）
  - 🟡 [MODIFIED] 可选 STS 仅 task-detail、不进评论；交付失败禁止拆机
- **并行 target 说明**: 同日未交付的 Git OAuth site 寻址设计仍占 **v97** 号（保持 target，本交付未改）。基线由 v96 archived 切换为本版 current。

## v97 🎯 target — Git OAuth 换票路径按 site 寻址

- **状态**: 🎯 target（已设计，待交付）
- **迭代**: gitsite-access-for-user-path
- **作者**: cursor
- **设计日期**: 2026-08-22 16:32
- **交付日期**: —
- **设计文档**: `docs/superpowers/specs/2026-08-22-multi-comment-git-oauth-access-token-sharing.md`
- **变更文件**:
  - 🆕 `v97-enterprise-landscape-20260822-1632-cursor.puml` (基于 v96)
  - 🆕 `v97-application-integration-20260822-1632-cursor.puml` (基于 v96)
  - 🆕 伴生格式: `.diff.archimate` (增量变迁) + `.full.archimate` (全量拓扑) + `.mermaid.md`（每个视图）
- **变更明细**:
  - 🟢 [NEW] `POST /api/internal/gitsite/{site}/oauth/access-for-user/`（`site` = Git `host[:port]`，与审计列一致）
  - 🟡 [MODIFIED] taskGitOauth / taskProjectService / taskCredentialService / taskCloudService 换票调用改走 site 路径
  - 🟡 [MODIFIED] 旧 `/api/internal/{github|gitlab}/oauth/access-for-user/` 保留为别名，新调用方不得作为默认
  - 决策锁定：GitLab 无 AWS 式 STS；多评论仍共用用户级 OAuth；不按评论签发 GitLab token
- **并行 target 说明**: 与已交付的 v98 指令闲置回收正交；本版仍为 target，勿与 v98 current 混改。设计基线仍为 v96 拓扑（v98 增量不影响本 Git OAuth 路径）。

## v96 ✅ archived — 管理员订单分账只读展示

- **状态**: ✅ archived（已被 v98 取代为 current）
- **迭代**: admin-order-profit-sharing
- **作者**: cursor
- **设计日期**: 2026-08-22 14:05
- **交付日期**: 2026-08-22 14:20
- **设计文档**: `docs/superpowers/specs/2026-08-22-admin-order-profit-sharing-design.md`
- **ADR**: [ADR-0030](../adr/0030-admin-only-order-profit-sharing.md)
- **变更文件**:
  - 🆕 `v96-application-integration-20260822-1405-cursor.puml` (基于 v95)
  - 🆕 `v96-enterprise-landscape-20260822-1405-cursor.puml` (基于 v95)
  - 🆕 伴生格式: `.diff.archimate` + `.full.archimate` + `.mermaid.md`（每个视图）
  - Archi `--loadModel`: 4 文件均 `Loaded model:`；`archimate-tool.py check` 0 critical
- **变更明细**:
  - 🟢 [NEW] `GET /api/system-admin/orders/{order_id}/` 管理员订单详情含 `profit_sharing[]`
  - 🟢 [NEW] `056_profit_sharing_order_id_bigint.sql` — order_id/tenant_id INT→BIGINT
  - 🟡 [MODIFIED] taskFE 管理端订单展开改打管理员详情；共享展开组件按 prop 隔离
  - 🟡 [MODIFIED] 租户订单 GET 明确不含分账键（契约加固）

## v95 ✅ shipped — 推荐分账资格取消与操作审计

- **状态**: ✅ current（已交付）
- **迭代**: referral-qualification-revoke
- **作者**: cursor
- **设计日期**: 2026-08-22 13:30
- **交付日期**: 2026-08-22 13:50
- **设计文档**: `docs/superpowers/specs/2026-08-22-referral-qualification-revoke-design.md`
- **ADR**: [ADR-0029](../adr/0029-referral-qualification-revoke-audit.md)
- **变更文件**:
  - 🆕 `v95-application-integration-20260822-1330-cursor.puml` (基于 v94)
  - 🆕 `v95-enterprise-landscape-20260822-1330-cursor.puml` (基于 v94)
  - 🆕 伴生格式: `.diff.archimate` + `.full.archimate` + `.mermaid.md`（每个视图）
- **变更明细**:
  - 🟢 [NEW] `referral_qualification_audit`；事件 `REFERRAL_QUALIFICATION_REVOKED`
  - 🟢 [NEW] 超管 `revoke` / `audit` API；taskBill `disable-eligibility`
  - 🟡 [MODIFIED] `referral_code.status` 增加 `revoked`；审批强制理由

## v94 ✅ shipped — PR 回复与一键合并

- **状态**: ✅ previous（已被 v95 取代为 current）
- **迭代**: pr-reply-merge-status
- **作者**: cursor
- **设计日期**: 2026-08-22 11:20
- **交付日期**: 2026-08-22 11:40
- **设计文档**: `docs/superpowers/specs/2026-08-22-pr-reply-merge-status-design.md`
- **ADR**: [ADR-0028](../adr/0028-pr-reply-one-click-merge-audit.md)
- **变更文件**:
  - 🆕 `v94-application-integration-20260822-1120-cursor.puml` (基于 v93)
  - 🆕 `v94-enterprise-landscape-20260822-1120-cursor.puml` (基于 v93)
  - 🆕 伴生格式: `.diff.archimate` + `.full.archimate` + `.mermaid.md`（每个视图）
- **变更明细**:
  - 🟢 [NEW] 人类评论 `parent_comment_id` + `git_pr_*`；事件 `TASK_GIT_PULL_REQUEST_RECORDED`
  - 🟢 [NEW] taskGitOauth `merge-request-status` / `merge-request-merge` + `GIT_MERGE_REQUEST_MERGED`
  - 🟡 [MODIFIED] taskFE 推送后嵌套 PR 回复卡片与一键合并

## v93 ✅ shipped — 镜像容器技能列表

- **状态**: ✅ archived
- **迭代**: image-container-skill-list
- **作者**: cursor
- **设计日期**: 2026-08-21 16:35
- **交付日期**: 2026-08-21 17:35
- **设计文档**: `docs/superpowers/specs/2026-08-21-image-skills-list-design.md`
- **ADR**: [ADR-0026](../adr/0026-image-container-skill-list.md)
- **变更文件**:
  - 🆕 `v93-application-integration-20260821-1635-cursor.puml` (基于 v92)
  - 🆕 `v93-enterprise-landscape-20260821-1635-cursor.puml` (基于 v92)
  - 🆕 伴生格式: `.diff.archimate` + `.full.archimate` + `.mermaid.md`（每个视图）
- **变更明细**:
  - 🟢 [NEW] `/app/imageSkills.yaml` 约定；事件 `ContainerImageSkillsExtracted`
  - 🟡 [MODIFIED] `ai_provider_vendorcontainerimage` / `cloud_tenant_installed_images` 技能快照列
  - 🟡 [MODIFIED] taskFE 市场列表、创建任务 `/skill` 高亮、评论 `@镜像 /skill`

## v92 ✅ shipped — 多渠道推荐码与分渠道分账

- **状态**: ✅ previous（已被 v93 取代为 current）
- **迭代**: referral-channel-codes
- **作者**: cursor
- **设计日期**: 2026-08-20 21:40
- **交付日期**: 2026-08-20 21:40
- **设计文档**: `docs/superpowers/specs/2026-08-20-referral-channel-codes-design.md`
- **ADR**: ADR-0025
- **变更文件**:
  - 🆕 `v92-application-integration-20260820-2140-cursor.puml` (基于 v91)
  - 🆕 `v92-enterprise-landscape-20260820-2140-cursor.puml` (基于 v91)
  - 🆕 伴生格式: `.diff.archimate` + `.full.archimate` + `.mermaid.md`（每个视图）
- **变更明细**:
  - 🟡 [MODIFIED] taskReferral — 一用户多渠道不透明码；渠道 CRUD；stats 按渠道/时间
  - 🟡 [MODIFIED] taskBill — 边快照 channel_code + commission_eligible；无资格不计提/不分账
  - 🟡 [MODIFIED] taskFE — 渠道列表、筛选、分渠道人数与分账

---

## v91 ✅ shipped — 任务帖配额来源与订单消耗归属

- **状态**: ✅ previous（已被 v92 取代为 current）
- **迭代**: task-post-quota-source-and-order-consumption
- **作者**: cursor
- **设计日期**: 2026-08-20 19:10
- **交付日期**: 2026-08-20 19:30
- **设计文档**: `docs/superpowers/specs/2026-08-20-task-post-quota-source-and-order-consumption-design.md`
- **ADR**: 无（扩展既有 grant 批次，不引入新存储子系统）
- **变更文件**:
  - 🆕 `v91-application-integration-20260820-1910-cursor.puml` (基于 v88)
  - 🆕 `v91-enterprise-landscape-20260820-1910-cursor.puml` (基于 v88)
  - 🆕 伴生格式: `.diff.archimate` + `.full.archimate` + `.mermaid.md`（每个视图）
- **变更明细**:
  - 🟡 [MODIFIED] taskBill — 配额 GET 拆赠送/购买；消耗归属 grant + order
  - 🟡 [MODIFIED] `billing_resource_grant.source_kind` / `order_id`
  - 🟡 [MODIFIED] taskFE 账单卡与订单消耗展示

## v90 🎯 target — 容器→SaaS 接口契约版本

- **状态**: 🎯 target（待 /10-ship 升 current；based_on v88 current；与 v89 正交）
- **迭代**: saas-inbound-skill-version
- **作者**: cursor
- **设计日期**: 2026-08-20 16:50
- **设计文档**: `docs/superpowers/specs/2026-08-20-saas-inbound-skill-version-design.md`
- **ADR**: [ADR-0024](../adr/0024-saas-inbound-skill-version.md)
- **变更文件**:
  - 🆕 `v90-application-integration-20260820-1650-cursor.puml` (基于 v88)
  - 🆕 `v90-enterprise-landscape-20260820-1650-cursor.puml` (基于 v88)
  - 🆕 伴生格式: `.diff.archimate` (增量变迁 v88→v90) + `.full.archimate` (全量拓扑) + `.mermaid.md`（每个视图）
- **变更明细**:
  - 🟢 [NEW] `docs/skills/saas-container/versions.yaml` + `GET /api/ai-provider/saas-inbound-skill-versions/`
  - 🟢 [NEW] 事件 `ContainerImageSaasInboundSkillVersionAssigned`
  - 🟡 [MODIFIED] `ai_provider_vendorcontainerimage.saas_inbound_skill_version`
  - 🟡 [MODIFIED] taskAiProvider SPA — skill 页版本 badge；厂商添加/编辑镜像必选
- **决策锁定**: 契约版本 ≠ 镜像标签；不 URL 分叉 inbound API；存量回填 v1

---

## v89 🎯 target — ztree 执行日志服务端持久化

- **状态**: 🎯 target（待 /10-ship 升 current；based_on v88 current；与 v86/v87 正交）
- **迭代**: ztree-exec-log-server-persist
- **作者**: cursor
- **设计日期**: 2026-08-20 15:49
- **设计文档**: `docs/superpowers/specs/2026-08-20-ztree-exec-log-server-persist-design.md`
- **ADR**: 无（沿用既有 task_cloud JSON 列 + 023 job 表；不引入新存储子系统）
- **变更文件**:
  - 🆕 `v89-application-integration-20260820-1549-cursor.puml` (基于 v88)
  - 🆕 `v89-enterprise-landscape-20260820-1549-cursor.puml` (基于 v88)
  - 🆕 伴生格式: `.diff.archimate` (增量变迁 v88→v89) + `.full.archimate` (全量拓扑) + `.mermaid.md`（每个视图）
- **变更明细**:
  - 🟢 [NEW] `cloud_layer_graph_snapshot`（JSON 列 `graph_json`；访问键 workspace_id / task_id / comment_id）
  - 🟢 [NEW] 事件 `LayerGraphSnapshotPersisted`
  - 🟡 [MODIFIED] taskCloudService — `layer-graph-push` UPSERT 快照；`GET container-layer-graph` hydrate
  - 🟡 [MODIFIED] taskFE — 关容器仍渲染层图快照与 023 job 步骤；克隆区允许空白
  - 无标记 — `cloud_job_execution_event`（023）、`container-clone-log` 仍代理容器、CCB 启动日志 16 分表（v88）
- **决策锁定**: 不存克隆日志；不用宿主机 JSON 文件当 SSOT；一期不分片（HASH 预留）；旧容器从未 PUSH 的树不可还原

## v88 ✅ current — 评论启动日志按 workspace_id 哈希分表

- **状态**: ✅ current（已交付 /10-ship；based_on v85 archived；与 v86/v87 正交）
- **迭代**: ccb-logs-workspace-shard-v88
- **作者**: cursor
- **设计日期**: 2026-08-20 15:20
- **交付日期**: 2026-08-20 15:30
- **设计文档**: `docs/superpowers/specs/2026-08-20-ccb-logs-workspace-shard-design.md`
- **ADR**: [ADR-0023](../adr/0023-ccb-logs-workspace-hash-shards.md)
- **变更文件**:
  - 🆕 `v88-application-integration-20260820-1520-cursor.puml` (基于 v85)
  - 🆕 `v88-enterprise-landscape-20260820-1520-cursor.puml` (基于 v85)
  - 🆕 伴生格式: `.diff.archimate` (增量变迁) + `.full.archimate` (全量拓扑) + `.mermaid.md`（每个视图）
- **变更明细**:
  - 🟢 [NEW] `cloud_comment_container_binding_logs_00` … `_15`（CRC32(workspace_id)%16）
  - 🟡 [MODIFIED] taskCloudService — 读写按 workspace 选表；Snowflake PK
  - 🔴 [DEPRECATED] 应用写入遗留单表 `cloud_comment_container_binding_logs`
  - 无标记 — taskFE 仍消费 `bindings[].logs`；SSE 文案不变
- **决策锁定**: 禁止一 workspace 一表；禁止把 workspace 字符串拼进 SQL 标识符；空 workspace 不得写入

## v87 🎯 target — taskFE Docker nginx 静态常驻

- **状态**: 🎯 target（待 /10-ship 升 current；based_on v85 current；与 v86 PIPL 正交）
- **迭代**: taskfe-nginx-static-resident
- **作者**: cursor
- **设计日期**: 2026-08-20 00:58
- **设计文档**: `docs/superpowers/specs/2026-08-20-taskfe-nginx-static-resident-design.md`
- **ADR**: [ADR-0022](../adr/0022-taskfe-nginx-static-resident.md)
- **变更文件**:
  - 🆕 `v87-application-integration-20260820-0058-cursor.puml` (基于 v85 current)
  - 🆕 `v87-enterprise-landscape-20260820-0058-cursor.puml` (基于 v85 current)
  - 🆕 伴生格式: `.diff.archimate` (增量变迁) + `.full.archimate` (全量拓扑) + `.mermaid.md`（每个视图）
- **变更明细**:
  - 🟢 [NEW] `taskfe-nginx` Docker 容器 `unless-stopped`，`0.0.0.0:4000→:80`，bind-mount `taskFE/app/public/`
  - 🟢 [NEW] `public/releases/<id>/` + `html` 原子 symlink；hashed 资源缺失真 404
  - 🟡 [MODIFIED] taskFE — Vite 只 build；runAll start 确保 nginx 容器而非 `vite preview`
  - 🟡 [MODIFIED] 精准重启 — 切 symlink，仅 nginx.conf 变才 reload
  - 🔴 [DEPRECATED] runAll 路径上的 `vite preview` 作为公网 :4000 入口
  - 无标记 — APISIX `spa-catch-all` / 上游 :4000 / SH 边缘 nginx / Vue 路由
- **决策锁定**: 不装系统 nginx；mount 父目录含 symlink；构建不 recreate 容器；本机 `npm run dev` 与 :4000 互斥

---

## v86 🎯 target — 个人账号注销（PIPL）

- **状态**: 🎯 target（待 /10-ship 升 current；代码与 intent 已落地）
- **迭代**: user-account-deletion-pipl-v86
- **作者**: cursor
- **设计日期**: 2026-08-19 17:50
- **设计文档**: `docs/superpowers/specs/2026-08-19-user-account-deletion-pipl-design.md`
- **变更文件**:
  - 🆕 `v86-application-integration-20260819-1750-cursor.puml` (基于 v85 current)
  - 🆕 伴生格式: `.diff.archimate` (增量变迁) + `.full.archimate` (全量拓扑) + `.mermaid.md`
- **变更明细**:
  - 🟢 [NEW] `auth_account_deletion_request` 表 + timer intent `account-deletion-execute-due`（taskEvents timer worker 幂等触发）
  - 🟢 [NEW] Kafka 五事件编排（含 `AccountDeletionRequested`）
  - 🟡 [MODIFIED] taskAuth — 用户自助注销 API（precheck/提交/冷却撤回/execute-due 幂等）+ precheck 聚合
  - 🟡 [MODIFIED] taskFE — UserProfile 危险区（blockers 展示 + 三须知 + 冷却期可撤回）
  - 🟡 [MODIFIED] taskBill / taskTenantService / taskCloudService — internal account-deletion-blockers 只读接口
  - 🟡 [MODIFIED] taskEvents — Kafka topics + DLT
  - 无标记 — 超管 archive 路径保留（人工兜底）
- **决策锁定**: 仅个人用户账号；即时注销/法人整户/物理 DELETE/数据导出均非一期；执行失败标记 blocked

---

## v85 📦 archived — 可插拔多区域 gitService

- **状态**: 📦 archived（已被 v88 取代为 current；v86/v87 仍为正交 target）
- **迭代**: pluggable-multi-region-gitservice
- **作者**: cursor
- **设计日期**: 2026-08-18 14:39
- **交付日期**: 2026-08-18 15:40
- **设计文档**: `docs/superpowers/specs/2026-08-18-pluggable-multi-region-gitservice-design.md`
- **ADR**: [ADR-0014](../adr/0014-pluggable-multi-region-gitlab.md)
- **变更文件**:
  - 🆕 `v85-application-integration-20260818-1439-cursor.puml` (基于 v84 current)
  - 🆕 `v85-enterprise-landscape-20260818-1439-cursor.puml` (基于 v84 current)
  - 🆕 伴生格式: `.diff.archimate` (增量变迁) + `.full.archimate` (全量拓扑) + `.mermaid.md`（每个视图）
- **变更明细**:
  - 🟢 [NEW] GitLab CE SH-1（`tencent-sh-1`，HTTP :8014，SSH :2223，精简 ~2g）
  - 🟢 [NEW] 公网 `gitlab-tencent-sh-1.${baseDomain}`；部署配方 `conf/infra/git-service-tencent-sh-1`
  - 🟡 [MODIFIED] taskBill — 按 region 开通；禁止静默默认 region
  - 🟡 [MODIFIED] taskFE — 多区域选购 / SystemAdmin 登记
  - 🟡 [MODIFIED] taskAuth — 多 GitLab OIDC client
  - 🟡 [MODIFIED] SH 边缘 nginx — 新 vhost → :8014
  - 无标记 — 现网 GitLab CE / `gitlab.${baseDomain}` / `tencent-shanghai-5` 保留可售
- **决策锁定**: 每区域独立 CE；租户无平台默认须显式选购；hybrid 开通（失败 pending）；SH 不升配走精简模式
- **积压说明**: v82/v83/v78 仍为正交 target；本版 base 已升为 current

## v84 📦 archived — 评论运行态推送同步


- **状态**: 📦 archived（已被 v85 取代为 current）
- **迭代**: comment-runtime-push-sync
- **作者**: cursor
- **设计日期**: 2026-08-16 13:28
- **交付日期**: 2026-08-16
- **设计文档**: `docs/superpowers/specs/2026-08-16-comment-runtime-no-background-poll-design.md`
- **变更文件**:
  - 🆕 `v84-enterprise-landscape-20260816-1328-cursor.puml` (基于 v81 current)
  - 🆕 `v84-application-integration-20260816-1328-cursor.puml` (基于 v81 current)
  - 🆕 伴生格式: `.diff.archimate` (增量变迁) + `.full.archimate` (全量拓扑) + `.mermaid.md` (每个视图)
- **变更明细**:
  - 🟢 [NEW] Comment-scoped container heartbeat 作为直播真源
  - 🟡 [MODIFIED] taskFE — 只 apply SSE/`runtime_status`；「刷新状态」才 Describe
  - 🟡 [MODIFIED] taskCloudService — 启动成功 SSE 带 `runtime_status`；停止走 `publishTaskSSE`
  - 🟡 [MODIFIED] taskSSE — 直播通道承载 `comment_id` + `runtime_status`
  - 🔴 [DEPRECATED] FE/BE UI 轮询 runtime-status / startup-status / binding advance
- **决策锁定**: 前端和服务端都不适合轮询维持 UI；禁止 SSE 后再自动 GET Describe；OPT-018 扩轮询已取消
- **积压说明**: v82/v83 仍为正交 target，本版 base 原为 v81 current

## v83 🎯 target — 评论级仓库身份

- **状态**: 🎯 target（goal-mode 自动采纳；base = v81 current）
- **迭代**: comment-level-repo-identity-v83
- **作者**: cursor
- **设计日期**: 2026-08-16 01:35
- **交付日期**: —
- **设计文档**: `docs/superpowers/specs/2026-08-16-comment-level-repo-identity-design.md`
- **ADR**: [ADR-0009](../adr/0009-comment-level-repo-identity.md)
- **变更文件**:
  - 🆕 `v83-application-integration-20260816-0135-cursor.puml`
  - 🆕 `v83-enterprise-landscape-20260816-0135-cursor.puml`
  - 🆕 伴生格式: `.diff.archimate` + `.full.archimate` + `.mermaid.md`（每个视图）
- **变更明细**:
  - 🟢 [NEW] `task_comments.repo_identities_json` — 本次运行每仓 git 身份与可选 github_user_id
  - 🟡 [MODIFIED] taskFE — 关联项目只读展示；composer 选择运行身份
  - 🟡 [MODIFIED] taskTaskService — comment POST/列表 + snapshot?comment_id=
  - 🟡 [MODIFIED] TASK_COMMENT_IMAGE_MENTIONED — payload 增补 repo_identities
  - 🔴 [DEPRECATED] 关联项目写入 `task_repo_identities` / github-credential-approve 作为运行真源
- **决策锁定**: 运行身份随评论；禁止双写任务级身份表；旧评论 snapshot 回退任务级表

## v81 📦 archived — 工作空间任务帖人读序号

- **状态**: 📦 archived（已被 v84 取代为 current）
- **迭代**: workspace-task-display-seq-v81
- **作者**: cursor
- **设计日期**: 2026-08-14 20:57
- **交付日期**: 2026-08-14
- **设计文档**: `docs/superpowers/specs/2026-08-14-workspace-task-display-seq-design.md`
- **ADR**: [ADR-0008](../adr/0008-workspace-task-display-seq.md)
- **变更文件**:
  - 🆕 `v81-application-integration-20260814-2057-cursor.puml` (基于 v80 设计稿 / 相对 v79 current)
  - 🆕 `v81-enterprise-landscape-20260814-2057-cursor.puml`
  - 🆕 伴生格式: `.diff.archimate` (增量变迁) + `.full.archimate` (全量拓扑) + `.mermaid.md` (每个视图)
- **变更明细**:
  - 🟢 [NEW] `task_workspace_seq` 发号表；`task_tasks.workspace_seq`；Business Object「Workspace Task Seq (#N)」
  - 🟡 [MODIFIED] taskTaskService — 建帖事务内发号；JSON / TASK_CREATED 带 `workspace_seq`
  - 🟡 [MODIFIED] taskFE — 卡片/搜索/复制展示 `#N`，不再截技术 ID 后 6 位
- **决策锁定**: 技术主键不变；人读序号工作空间内从 1 递增、删除不回收；存量任务帖清空不回填；`task_git_identities` 与跨服务 CSC 不在本 DDL 级联删除

## v79 📦 archived — 厂商证照 COS 预签名直传

- **状态**: 📦 archived（已被 v81 取代为 current）
- **迭代**: vendor-docs-cos-presign-v79
- **作者**: cursor
- **设计日期**: 2026-08-14 13:33
- **交付日期**: 2026-08-14
- **设计文档**: `docs/superpowers/specs/2026-08-14-vendor-docs-cos-presign-design.md`
- **ADR**: [ADR-0006](../adr/0006-tencent-cos-vendor-documents.md)
- **变更文件**:
  - 🆕 `v79-application-integration-20260814-1333-cursor.puml` (基于 v78)
  - 🆕 `v79-enterprise-landscape-20260814-1333-cursor.puml` (基于 v13 current)
  - 🆕 伴生格式: `.diff.archimate` (增量变迁) + `.full.archimate` (全量拓扑) + `.mermaid.md` (每个视图)
- **变更明细**:
  - 🟢 [NEW] Tencent COS `ai-provider-1259712831`（ap-shanghai，私有 + SSE-COS）
  - 🟢 [NEW] `vendor-docs-path.yaml` 路径规则片段
  - 🟡 [MODIFIED] taskAiProvider — 预签名 PUT / Head / Get；VendorDocStore
  - 🟡 [MODIFIED] taskFE VendorApplicationForm — 浏览器直传
  - 🟡 [MODIFIED] `ai_provider_vendor.file_key` — COS object key（本地 key 可回退）
- **决策锁定**: 浏览器预签名 PUT（非 STS、非服务端代传）；密钥仅 `config.local.yaml`；运营 GetObject 反代；后台写回 pathRule

## v78 🎯 target — 评论级容器令牌

- **状态**: 🎯 target（goal-mode 自动采纳；base = v77）
- **迭代**: comment-scoped-container-token-v78
- **作者**: cursor
- **设计日期**: 2026-08-14 01:15
- **交付日期**: —
- **设计文档**: `docs/superpowers/specs/2026-08-14-comment-scoped-container-token-design.md`
- **ADR**: [ADR-0005](../adr/0005-comment-scoped-container-token.md)
- **变更文件**:
  - 🆕 `v78-application-integration-20260814-0115-cursor.puml`
  - 🆕 `v78-application-integration-20260814-0115-cursor.diff.archimate`
  - 🆕 `v78-application-integration-20260814-0115-cursor.full.archimate`
  - 🆕 `v78-application-integration-20260814-0115-cursor.mermaid.md`
- **变更明细 (application-integration vs v77)**:
  - 🟡 [MODIFIED] taskCredentialService — `TaskScope` + `comment_id`；UNIQUE 四元组；init/by-scope/exchange
  - 🟡 [MODIFIED] taskCloudService — bootstrap token init URL 带 `/comment/{id}`；by-scope 传 comment
  - 🟡 [MODIFIED] onlineServiceJS — exchange / register-reachability 带 `COMMENT_ID`
  - 🟡 [MODIFIED] `credential_container_tokens` — 列 `comment_id` + uk_scope
- **决策锁定**: 签发强制 comment；校验兼容无 comment 旧容器；同一评论 IssueToken 复用行且已有 refresh 时不清 refresh

## v77 🎯 target — 公司成员多 Git 身份 + MEMBER_JOINED 自动建身份

- **状态**: 🎯 target（goal-mode 自动采纳；base = v76）
- **迭代**: member-joined-auto-git-identity-v77
- **作者**: cursor
- **设计日期**: 2026-08-12 15:35
- **交付日期**: —
- **设计文档**: `docs/superpowers/specs/2026-08-12-member-joined-auto-git-identity-design.md`
- **ADR**: No-ADR: covered by existing event/consumer patterns + single-service data ownership rules
- **变更文件**:
  - 🆕 `v77-application-integration-20260812-1535-cursor.puml`
  - 🆕 `v77-application-integration-20260812-1535-cursor.diff.archimate`
  - 🆕 `v77-application-integration-20260812-1535-cursor.full.archimate`
  - 🆕 `v77-application-integration-20260812-1535-cursor.mermaid.md`
- **变更明细 (application-integration vs v76)**:
  - 🟡 [MODIFIED] taskTenantService — 成员创建路径发布 `MEMBER_JOINED`（扩展 payload）
  - 🟢 [NEW] taskEvents `1_create_default_git_identity`（:18060）→ ensure-default
  - 🟡 [MODIFIED] taskTaskService — ensure-default + 租户 Git 身份管理 API；`task_git_identities` 所有权不变
  - 🟡 [MODIFIED] taskFE PeopleManage — MemberList「Git 身份」弹窗
- **决策锁定**: 复用 MEMBER_JOINED；默认邮箱 sha256 哈希 `@daydaymoney.com`；鉴权 self / `member:manage` / `group-members:manage`；无存量回填

## v76 🎯 target — 订单评论（租户 ↔ 系统管理员）

- **状态**: 🎯 target（goal-mode 自动采纳；base = v75）
- **迭代**: billing-order-comments-v76
- **作者**: cursor
- **设计日期**: 2026-08-11 22:15
- **交付日期**: —
- **设计文档**: `docs/superpowers/specs/2026-08-11-billing-order-comments-design.md`
- **ADR**: No-ADR: trivial tech choice, no architectural impact
- **变更文件**:
  - 🆕 `v76-application-integration-20260811-2215-cursor.puml`
  - 🆕 `v76-application-integration-20260811-2215-cursor.diff.archimate`
  - 🆕 `v76-application-integration-20260811-2215-cursor.full.archimate`
  - 🆕 `v76-application-integration-20260811-2215-cursor.mermaid.md`
- **变更明细 (application-integration vs v75)**:
  - 🟢 [NEW] `billing_order_comment` + `OrderCommentThread`
  - 🟡 [MODIFIED] taskBill — tenant/admin comments API + `BILLING_ORDER_COMMENT_CREATED`
  - 🟡 [MODIFIED] taskFE BillingOrders / SystemAdminOrderRecords — 展开区评论线程
- **决策锁定**: 扁平时间线；租户写 `billing:manage`；超管 `IsPlatformStaff`；事件不含正文

## v75 ✅ shipped — 租户角色管理（可复用角色 + 多角色并集）

- **状态**: ✅ shipped（goal-mode /10-ship；application-integration current = v75；v76 仍为后续 target）
- **迭代**: tenant-role-management-v75
- **作者**: cursor
- **设计日期**: 2026-08-11 21:35
- **交付日期**: 2026-08-11 22:15
- **设计文档**: `docs/superpowers/specs/2026-08-11-tenant-role-management-v75-design.md`
- **ADR**: No-ADR: covered by existing ADR-0003/0004（粗码 RequirePerm 退役另开后续 ADR）
- **变更文件**:
  - 🆕 `v75-application-integration-20260811-2135-cursor.puml`
  - 🆕 `v75-application-integration-20260811-2135-cursor.diff.archimate`
  - 🆕 `v75-application-integration-20260811-2135-cursor.full.archimate`
  - 🆕 `v75-application-integration-20260811-2135-cursor.mermaid.md`
- **变更明细 (application-integration vs v74)**:
  - 🟢 [NEW] taskFE `PeopleRoles` + nav `people.roles` + 种子 page/region
  - 🟡 [MODIFIED] taskFE PeopleAccess — 主体多角色分配；🔴 停隐式 `访问·…`
  - 🟡 [MODIFIED] taskTenant — `role_names[]` replace-all + DELETE 单角色
  - 🟡 [MODIFIED] taskAuth — 角色 resource-groups 真源；种子 `people.roles*`
- **决策锁定**: Q1=C / Q2=A / Q3=B / Q4=A；租户粗码 RequirePerm 本版不删

## v74 🎯 target — 邀请预授页面/区域权限（invite-pending-access-grants-v74）

- **状态**: 🎯 target（goal-mode 自动采纳；base = v73）
- **迭代**: invite-pending-access-grants-v74
- **作者**: cursor
- **设计日期**: 2026-08-11 21:00
- **交付日期**: —
- **设计文档**: `docs/superpowers/specs/2026-08-11-invite-pending-access-grants-design.md`
- **ADR**: No-ADR: covered by existing ADR-0003/0004
- **变更文件**:
  - 🆕 `v74-application-integration-20260811-2100-cursor.puml`
  - 🆕 `v74-application-integration-20260811-2100-cursor.diff.archimate`
  - 🆕 `v74-application-integration-20260811-2100-cursor.full.archimate`
  - 🆕 `v74-application-integration-20260811-2100-cursor.mermaid.md`
- **变更明细 (application-integration vs v73)**:
  - 🟡 [MODIFIED] `tenant_invitation` — `pending_grants JSON`
  - 🟢 [NEW] taskAuth `POST /api/internal/authz/apply-member-grants/`
  - 🟡 [MODIFIED] taskTenant invite/join — 持久化并落权
  - 🟡 [MODIFIED] taskFE PeopleInvite — InviteAccessGrants + 快捷访问管理
- **决策锁定**: 邀请时预授；join 落自定义角色；admin 忽略 grants

## v73 🎯 target — 资源组授予效果 view/operate（rbac-resource-grant-effect-v73）

- **状态**: 🎯 target（goal-mode 自动采纳；base = v72）
- **迭代**: rbac-resource-grant-effect-v73
- **作者**: cursor
- **设计日期**: 2026-08-11 15:00
- **交付日期**: —
- **设计文档**: `docs/superpowers/specs/2026-08-11-rbac-resource-grant-effect-v73-design.md`
- **ADR**: `docs/adr/0004-resource-group-grant-effect-view-operate.md`
- **变更文件**:
  - 🆕 `v73-application-integration-20260811-1500-cursor.puml`
  - 🆕 `v73-application-integration-20260811-1500-cursor.diff.archimate`
  - 🆕 `v73-application-integration-20260811-1500-cursor.full.archimate`
  - 🆕 `v73-application-integration-20260811-1500-cursor.mermaid.md`
- **变更明细 (application-integration vs v72)**:
  - 🟡 [MODIFIED] `auth_role_resource_group` — `effect ENUM(view,operate)`
  - 🟡 [MODIFIED] `taskAuth` PDP — 注入 `:view`/`:operate` + operate 遗留 bare code
  - 🟡 [MODIFIED] `shareLib/authz` — HasRegionView/Operate、EmitLogicalRGCodes
  - 🟡 [MODIFIED] `taskFE` PeopleAccess — 可访问 / 可编辑执行双档
- **决策锁定**: 两档效果；operate⊃view；不强制合并既有 save_actions region

## v73 🎯 target — 资源组授予效果 view/operate（rbac-resource-grant-effect-v73）

- **状态**: 🎯 target（goal-mode 自动采纳并实现；待 /10-ship 切 current；base = v72）
- **迭代**: rbac-resource-grant-effect-v73
- **作者**: cursor
- **设计日期**: 2026-08-11 15:00
- **交付日期**: —
- **设计文档**: `docs/superpowers/specs/2026-08-11-rbac-resource-grant-effect-v73-design.md`
- **ADR**: `docs/adr/0004-resource-group-grant-effect-view-operate.md`
- **变更文件**:
  - 🆕 `v73-application-integration-20260811-1500-cursor.puml`
  - 🆕 `v73-application-integration-20260811-1500-cursor.diff.archimate`
  - 🆕 `v73-application-integration-20260811-1500-cursor.full.archimate`
  - 🆕 `v73-application-integration-20260811-1500-cursor.mermaid.md`
- **变更明细 (application-integration vs v72)**:
  - 🟡 [MODIFIED] `auth_role_resource_group` — `effect ENUM(view,operate)`
  - 🟡 [MODIFIED] `taskAuth` PDP — 注入 `:view` / `:operate`（operate 兼遗留 bare code）
  - 🟡 [MODIFIED] `shareLib/authz` — `HasRegionView` / `HasRegionOperate`
  - 🟡 [MODIFIED] `taskFE` PeopleAccess — 可访问 / 可编辑执行双档
- **决策锁定**: 两档效果；operate⊃view；不强制合并既有 save_actions region

## v72 🎯 target — 逻辑资源组 Page⊃Region RBAC（rbac-logical-resource-group-v72）

- **状态**: 🎯 target（设计已批准 A1/B2；待实现与 /10-ship；base = v71）
- **迭代**: rbac-logical-resource-group-v72
- **作者**: claude
- **设计日期**: 2026-08-11 11:36
- **交付日期**: —
- **设计文档**: `docs/superpowers/specs/2026-08-11-rbac-page-resource-group-v72-design.md`
- **ADR**: `docs/adr/0003-logical-resource-group-page-region-rbac.md`
- **变更文件**:
  - 🆕 `v72-application-integration-20260811-1136-claude.puml`
  - 🆕 `v72-application-integration-20260811-1136-claude.diff.archimate`
  - 🆕 `v72-application-integration-20260811-1136-claude.full.archimate`
  - 🆕 `v72-application-integration-20260811-1136-claude.mermaid.md`
- **变更明细 (application-integration vs v71)**:
  - 🟢 [NEW] `auth_resource_group` / `auth_resource_member` / `auth_role_resource_group` — page 载体 + ui_region 资源组 + 成员
  - 🟡 [MODIFIED] `taskAuth` PDP — 注入 `region:*` / `page:*`（B2：不展开旧粗码）
  - 🟡 [MODIFIED] `shareLib/authz` — `RequireRegion` / `hasRegion` / `hasPage`
  - 🟡 [MODIFIED] `taskFE` — 访问管理改为 page→region 树；侧栏用 `page:*`
  - 🟡 [MODIFIED] 业务服务 Enforce — 关键 API 挂 region（与 FE 同源 registry）
- **决策锁定**: Q1=A1（默认 ui_region；整页写 page 由 PDP 展开）；Q2=B2（仅 region/page）

## v71 🎯 target — 微信登录 APISIX 502 加固（wechat-login-apisix-502-hardening）

- **状态**: 🎯 target（已设计并实现，待 /10-ship 切 current；base = v70）
- **迭代**: wechat-login-apisix-502-hardening
- **作者**: claude
- **设计日期**: 2026-08-11 02:22
- **交付日期**: —
- **设计文档**: `docs/superpowers/specs/2026-08-11-wechat-login-apisix-502-design.md`
- **变更文件**:
  - 🆕 `v71-application-integration-20260811-0222-claude.puml`
  - 🆕 `v71-application-integration-20260811-0222-claude.diff.archimate`
  - 🆕 `v71-application-integration-20260811-0222-claude.full.archimate`
  - 🆕 `v71-application-integration-20260811-0222-claude.mermaid.md`
- **变更明细 (application-integration vs v70)**:
  - 🟡 [MODIFIED] `taskFE` — `navigateWechatOAuth` 预检；502 留页 + `data-traceId`
  - 🟡 [MODIFIED] `runAll` — `EnsureLoginCriticalPath`（task-auth → task-gateway → taskFE）
  - 🟡 [MODIFIED] `APISIX` / Promtail — access/error 进 Loki（`apisix-access` / `apisix-error`）
- **No-ADR**: trivial ops/observability hardening on existing components, no new architectural pattern

## v70 🎯 target — 厂商申请审核开关 + 镜像市场 SSO 入口内聚（vendor-review-toggle-sso-entry）

- **状态**: 🎯 target（已设计，待交付；base = v69）
- **迭代**: vendor-review-toggle-sso-entry
- **作者**: claude
- **设计日期**: 2026-08-10 19:55
- **交付日期**: —
- **设计文档**: `docs/superpowers/specs/2026-08-10-vendor-review-toggle-sso-entry-design.md`
- **变更文件**:
  - 🆕 `v70-application-integration-20260810-1955-claude.puml`
  - 🆕 `v70-application-integration-20260810-1955-claude.diff.archimate`
  - 🆕 `v70-application-integration-20260810-1955-claude.full.archimate`
  - 🆕 `v70-application-integration-20260810-1955-claude.mermaid.md`
- **变更明细 (application-integration vs v69)**:
  - 🟢 [NEW] `ai_provider_marketplace_settings` — 厂商申请审核开关（单行）
  - 🟡 [MODIFIED] `taskAiProvider` — marketplace-settings API + bridge 条件建档/激活
  - 🟡 [MODIFIED] `taskFE` — SystemAdmin 容器镜像页内聚 SSO；Sidebar 移除；ImageMarket 按开关切换
- **No-ADR**: covered by existing single-service data ownership（设置归属 taskAiProvider）

## v17/v69 🎯 target — 移除本机容器模拟启动（remove-local-mock-container-startup）

- **状态**: 🎯 target（已设计，待交付；base = v16 插件 OIDC 白名单管理 / v68 插件 OIDC 白名单管理）
- **迭代**: remove-local-mock-container-startup
- **作者**: claude
- **设计日期**: 2026-08-09 22:25
- **交付日期**: —
- **设计文档**: `docs/superpowers/specs/2026-08-09-remove-local-mock-container-startup-design.md`（现行权威）
- **变更文件**:
  - 🆕 `v17-enterprise-landscape-20260809-2225-claude.puml`（基于 v16）
  - 🆕 `v17-enterprise-landscape-20260809-2225-claude.diff.archimate`（含架构变迁 v16→v17 + 移除本机模拟启动链拓扑 + sourceConnection 连线）
  - 🆕 `v17-enterprise-landscape-20260809-2225-claude.full.archimate`（合并 v16→v17 变更后的全量拓扑 + 全量视图连线）
  - 🆕 `v17-enterprise-landscape-20260809-2225-claude.mermaid.md`
  - 🆕 `v69-application-integration-20260809-2225-claude.puml`（基于 v68）
  - 🆕 `v69-application-integration-20260809-2225-claude.diff.archimate`（含架构变迁 v68→v69 + 移除本机模拟启动链拓扑 + sourceConnection 连线）
  - 🆕 `v69-application-integration-20260809-2225-claude.full.archimate`（合并 v68→v69 变更后的全量拓扑 + 全量视图连线）
  - 🆕 `v69-application-integration-20260809-2225-claude.mermaid.md`
- **变更明细 (enterprise-landscape vs v16)**:
  - 🔴 [DEPRECATED] `mock_run_container`（Python Flask :8796）— 本机 Docker 模拟运行器删除
  - 🔴 [DEPRECATED] `go_run_container`（Go :8796）— mock_run_container Go 重写删除
  - 🔴 [DEPRECATED] `taskContainerGateway` mock-run-container 代理面（start/stop/status/env-defaults）删除
  - 🟡 [MODIFIED] `taskCloudService` — `/api/internal/mock-run/*`（env-defaults / resolve-image）删除 + `comment_csc_bootstrap` mock/空平台分支改为报错拦截
  - 🟡 [MODIFIED] `taskFE` — 「模拟启动」面板/mockRun tab/mockContainerStart flag 删除
- **变更明细 (application-integration vs v68)**:
  - 🔴 [DEPRECATED] `mockRunSidecar` — mock_run_container (Flask :8796) 本机 Docker 模拟运行器
  - 🔴 [DEPRECATED] `goRunSidecar` — go_run_container (Go :8796)
  - 🔴 [DEPRECATED] `mockRunProxy` — taskContainerGateway mock-run-container 代理面 (start/stop/status/env-defaults)
  - 🔴 [DEPRECATED] `mockRunInternalAPI` — taskCloudService `/api/internal/mock-run/*` (env-defaults / resolve-image)
  - 🟡 [MODIFIED] `gateway` — APISIX mock-run-container 代理面路由删除
  - 🟡 [MODIFIED] `taskCloud` — `/api/internal/mock-run/*` 删除 + reconcile/provision 移除 mock-run + `comment_csc_bootstrap` mock/空平台报错拦截
  - 🟡 [MODIFIED] `taskFE` — 「模拟启动」面板删除
- **设计要点**:
  - 双轨收敛：容器启动统一走云端机器（taskCloudService start-vm → cloud_server_events 状态机 → taskEvents CLOUD_SERVER_STARTED → Aliyun ECS），本机 Docker 模拟启动面全删
  - `comment_csc_bootstrap` 中 platform 为空或 mock 的评论级 CSC → 报错拦截（400），不再走 mock/空平台分支
  - `USE_IN_MEMORY_CLOUD` 保留（SDK 层 mock，与本地容器镜像模拟无关）
  - 全部变更为删除（Go 服务删接口/前端删面板）+ 报错分支 → 🐍 Python 门禁 not_applicable（无新增 Python 接口）
  - 业务意图 → 事件对照：无新事件（纯删除，不改变业务事实；删除路径无消费方依赖残留）
  - CRG unavailable：图过期（2026-08-06 构建）且无 Go 覆盖（记录于设计文档 §6）
- **视图连线验证**:
  - [x] 架构变迁视图：Plateau v16/v68 → Gap → WP → Plateau v17/v69 链路可追踪（sourceConnection 完整）
  - [x] 目标拓扑视图：移除本机模拟启动链（FE/Gateway/Cloud → mockRun → goRun）全部连线 + 废弃标注，无孤立节点、无悬空引用
  - [x] Archi CLI 验证通过：`Loaded model: 'daydaymoney Enterprise Landscape v17 Target'` / `'v69 全量 — 移除本机模拟启动 (post-change)'`
  - [x] PlantUML 渲染验证：v17/v69 均产出 PNG（既有 line 11 字体警告与 v16/v68 基线一致，非致命）

---

## v16/v68 🎯 target — 浏览器插件固定 ID 管理员后台可配置（browser-extension-oidc-extension-id-admin）

- **状态**: 🎯 target（已设计，待交付；base = v15 任务帖存续期 / v67 任务帖存续期）
- **迭代**: browser-extension-oidc-extension-id-admin
- **作者**: claude
- **设计日期**: 2026-08-08 17:32
- **交付日期**: —
- **设计文档**: `docs/superpowers/specs/2026-08-08-task-chrome-plugin-oidc-extension-id-admin-design.md`（现行权威）
- **关联**: OPT-20260808-024（taskChromePlugin OAuth2+PKCE 登录迁移 — 阶段 0a 固定 ID 一致性硬前提由本设计解决）
- **变更文件**:
  - 🆕 `v16-enterprise-landscape-20260808-1732-claude.puml`（基于 v15）
  - 🆕 `v16-enterprise-landscape-20260808-1732-claude.diff.archimate`（含架构变迁 v15→v16 + 插件 ID 白名单管理链拓扑 + sourceConnection 连线）
  - 🆕 `v16-enterprise-landscape-20260808-1732-claude.full.archimate`（合并 v15→v16 变更后的全量拓扑 + 全量视图连线）
  - 🆕 `v16-enterprise-landscape-20260808-1732-claude.mermaid.md`
  - 🆕 `v68-application-integration-20260808-1732-claude.puml`（基于 v67）
  - 🆕 `v68-application-integration-20260808-1732-claude.diff.archimate`（含架构变迁 v67→v68 + 插件 ID 白名单管理链拓扑 + sourceConnection 连线）
  - 🆕 `v68-application-integration-20260808-1732-claude.full.archimate`（合并 v67→v68 变更后的全量拓扑 + 全量视图连线）
  - 🆕 `v68-application-integration-20260808-1732-claude.mermaid.md`
- **变更明细 (enterprise-landscape vs v15)**:
  - 🟡 [MODIFIED] `taskAuth` — system-admin 新增 GET/PUT `/api/system-admin/oidc-extension/`（requireSuperuser）；bootstrap seed 对 `managed_by='admin'` 行执行 INSERT-only（不覆盖管理员改库）
  - 🟡 [MODIFIED] `taskAuth DB` — `auth_oidc_client` 新增 `managed_by` 列（bootstrap|admin）；`redirect_uris` 写入方扩展（conf 自愈 → DB 权属判定 + 管理端点写库）
  - 🟡 [MODIFIED] `taskFE` — SystemAdmin 新增「浏览器插件」管理页（/system-admin/oidc-extension/ 路由 + SystemAdminBrowserExtension.vue + 快捷卡片入口）
  - 🟡 [MODIFIED] `API Gateway` — APISIX 新增 `/api/system-admin/oidc-extension/*` 路由
- **变更明细 (application-integration vs v67)**:
  - 🟡 [MODIFIED] `taskAuth` — GET/PUT 管理端点 + seed 语义（managed_by='admin' 行 INSERT-only）+ `auth_oidc_client.managed_by`
  - 🟡 [MODIFIED] `taskFE` — SystemAdmin「浏览器插件」页 + 路由
  - 🟡 [MODIFIED] `gateway` — APISIX 新增 `/api/system-admin/oidc-extension/*`
  - 🟡 [MODIFIED] `taskAuth DB` — +managed_by 列（auth_oidc_client）
  - 🟢 [NEW] 管理链 Rel_Flow：taskFE → gateway → taskAuth → taskAuthDB（白名单管理）
- **设计要点**:
  - 核心矛盾：`ensureOidcClient` 自愈 UPDATE 会覆盖管理员改库 → 管理权属落 DB（用户审批：adminManaged 落 DB），`managed_by` 列 + seed 按行判定
  - dataMigrate 新增 `031_oidc_client_managed_by.sql`（ADD COLUMN + UPDATE chrome-extension → admin，幂等）
  - 插件 ID 输入形态：`^[a-p]{32}$` 列表（每行一个），服务端拼装 `chrome-extension://<id>/oauth-callback.html`，不开放任意 redirect_uri
  - conf chrome-extension 条目保留作 seed 默认值；全部变更在 Go（taskAuth）+ 前端（taskFE）+ 配置 → 🐍 Python 门禁 not_applicable
  - CRG unavailable：图过期（2026-08-06）且无 Go 覆盖（记录于设计文档 §6）
- **视图连线验证**:
  - [x] 架构变迁视图：Plateau v15/v67 → Gap → WP → Plateau v16/v68 链路可追踪（sourceConnection 完整）
  - [x] 目标拓扑视图：插件 ID 白名单管理链（FE→Gateway→taskAuth→authDB）全部连线，无孤立节点、无悬空引用
  - [x] Archi CLI 验证通过：`Loaded model: 'daydaymoney Enterprise Landscape v16 Target'` / `'v68 浏览器插件 OIDC 白名单管理'`
  - [x] PlantUML 渲染验证：v16/v68 均产出 PNG（既有 line 11 字体警告与 v15 基线一致，非致命）

---

## v15 🎯 target — 任务帖定价模型重构（task-post-12month-validity-renewal）

- **状态**: 🎯 target（已设计，待交付；base = v14 多会话互知 / v66 多会话互知）
- **迭代**: task-post-12month-validity-renewal
- **作者**: claude
- **设计日期**: 2026-08-07 19:20
- **交付日期**: —
- **设计文档**: `docs/superpowers/specs/2026-08-07-task-post-12month-validity-renewal-design.md`（现行权威）
- **变更文件**:
  - 🆕 `v15-enterprise-landscape-20260807-1920-claude.puml`（基于 v14）
  - 🆕 `v15-enterprise-landscape-20260807-1920-claude.diff.archimate`（含架构变迁 v14→v15 + 帖子存续期与续存入口拓扑 + sourceConnection 连线）
  - 🆕 `v15-enterprise-landscape-20260807-1920-claude.full.archimate`（合并 v14→v15 变更后的全量拓扑 + 全量视图连线）
  - 🆕 `v15-enterprise-landscape-20260807-1920-claude.mermaid.md`
  - 🆕 `v67-application-integration-20260807-1920-claude.puml`（基于 v66）
  - 🆕 `v67-application-integration-20260807-1920-claude.diff.archimate`（含架构变迁 v66→v67 + 创建/续存/到期数据流拓扑 + sourceConnection 连线）
  - 🆕 `v67-application-integration-20260807-1920-claude.full.archimate`（合并 v66→v67 变更后的全量拓扑 + 全量视图连线）
  - 🆕 `v67-application-integration-20260807-1920-claude.mermaid.md`
- **变更明细 (enterprise-landscape vs v14)**:
  - 🟡 [MODIFIED] `taskBill` — server_start 计费语义：执行费 → 创建帖费（33分/帖/12个月）；新增 `server_start_renewal` 单位（33分）；`consume-task-post-quota` 返回 `expires_at`；`charge-server-start/` 剥离为 no-op
  - 🟡 [MODIFIED] `taskTaskService` — 帖子存续期：`task_tasks.post_expires_at`（存量回填 NOW()+12M）、创建任务写存续期、到期校验（expired 帖拒绝执行/编辑/评论）、新增 `POST /tasks/{id}/renew/` 续存接口（扣 1 创建帖次数 → +12M）
  - 🟡 [MODIFIED] `taskCloudService` — bill_client `chargeServerStart` 退役（dormant 清理），服务器启动改为帖子存续期校验
  - 🟡 [MODIFIED] `taskEvents` — TASK_POST_CREATED 载荷 +expires_at；新增 TASK_POST_RENEWED / TASK_POST_EXPIRED；每日到期扫描（惰性校验兜底）
  - 🟢 [NEW] `server_start_renewal` 计费单位（taskBill 预留字段落地）
- **变更明细 (application-integration vs v66)**:
  - 🟢 [NEW] 事件 `TASK_POST_RENEWED`（taskTask 续存成功发布）/ `TASK_POST_EXPIRED`（taskEvents 到期扫描发布）
  - 🟡 [MODIFIED] `taskBill` — server_start 执行费→创建帖费 + `server_start_renewal` 续存单位 + `consume-task-post-renewal` internal API + `charge-server-start/` no-op
  - 🟡 [MODIFIED] `taskTaskService` — `task_tasks.post_expires_at` + `/tasks/{id}/renew/` + 到期校验
  - 🟡 [MODIFIED] `taskEvents` — 载荷扩展 + 到期扫描 + 新事件 handler
  - 🟡 [MODIFIED] `taskTaskDB` — +post_expires_at
  - 🔴 [DEPRECATED] `charge-server-start` 按次扣费路径（taskBill + taskCloud bill_client，迁移期后删除）
- **设计要点**:
  - 新定价模型：购买 0.33元/次 → 「创建帖次数」；创建帖消耗 1 次 → 12 个月存续期（单帖独立 `expires_at = now + 12M`）
  - 续存：消耗 1 创建帖次数 → `post_expires_at = max(now, 当前到期日) + 12M`（复用 `diskExpiresAtFromMonths` 算法，不缩短已购时长）
  - 到期判定由时间戳推导（不新增 status 枚举，避免状态漂移）；存量帖上线日统一回填 +12M
  - 全部新增接口在 Go 服务（taskTaskService/taskBill/taskEvents）→ 🐍 Python 门禁不触发（not_applicable）
  - CRG unavailable：图过期且不含 taskBill/taskTaskService（记录于设计文档 §3）
- **视图连线验证**:
  - [x] 架构变迁视图：Plateau v14→Gap→WP→Plateau v15 链路可追踪（sourceConnection 完整）
  - [x] 目标拓扑视图：续存入口/计费/存续期/到期事件链路全部连线，无孤立节点、无悬空引用
  - [x] Archi CLI 验证通过：`Loaded model: 'daydaymoney Enterprise Landscape v15 Target'` / `'v67 任务帖存续期定价 — 创建帖次数 + 单帖续存'`

---

## v14 🎯 target — 多智能体会话互知与冲突防护（agent-session-coordination）

- **状态**: 🎯 target（已设计，待交付；base = v13 Git Hooks 入库 / v65 夜间巡检）
- **迭代**: agent-session-coordination
- **作者**: claude
- **设计日期**: 2026-08-06 00:40
- **交付日期**: —
- **设计文档**: `docs/superpowers/specs/2026-08-06-agent-session-coordination-design.md`（现行权威）
- **变更文件**:
  - 🆕 `v14-enterprise-landscape-20260806-0040-claude.puml`（基于 v13）
  - 🆕 `v14-enterprise-landscape-20260806-0040-claude.diff.archimate`（含迁移视图 v13→v14 + Session Hub 拓扑视图 + sourceConnection 连线）
  - 🆕 `v14-enterprise-landscape-20260806-0040-claude.full.archimate`（合并 v13→v14 变更后的全量拓扑 + 全量视图连线）
  - 🆕 `v14-enterprise-landscape-20260806-0040-claude.mermaid.md`
  - 🆕 `v66-application-integration-20260806-0040-claude.puml`（基于 v65）
  - 🆕 `v66-application-integration-20260806-0040-claude.diff.archimate`（含迁移视图 v65→v66 + Session Hub 接入拓扑 + sourceConnection 连线）
  - 🆕 `v66-application-integration-20260806-0040-claude.full.archimate`（合并 v65→v66 变更后的全量拓扑 + 全量视图连线）
  - 🆕 `v66-application-integration-20260806-0040-claude.mermaid.md`
- **变更明细 (enterprise-landscape vs v13)**:
  - 🟢 [NEW] `Session Hub`（Technology_Artifact — logs/sessions/：注册表 registry/ + 仓库级锁 locks/ + 租约 TTL 10min + 心跳 30s + audit/）
  - 🟢 [NEW] `claude-agent session CLI`（Technology_Artifact — Go: register/list/acquire/release/heartbeat/pause/resume/shadow-* + sessionctl.sh 薄封装）
  - 🟢 [NEW] `Claude Code 会话钩子`（Technology_Artifact — SessionStart/End/UserPromptSubmit/PreToolUse/PreCompact + settings.json 分发）
  - 🟢 [NEW] `Shadow Edit 工作区`（Technology_Artifact — 快照拷贝 + SHA-256 校验和 + 原子拷回 + 冲突报告）
  - 🟢 [NEW] `开发子仓池`（Technology_Artifact — 40+ 子模块仓库，锁粒度目标）
  - 🟡 [MODIFIED] `Hooks 分发/校验器` — 分发范围扩展 settings.json 会话钩子 + pre-commit v1.1.0 锁校验
- **变更明细 (application-integration vs v65)**:
  - 🟢 [NEW] `Session Hub` / `claude-agent session CLI` 组件接入
  - 🟡 [MODIFIED] `claude-agent` — run 内建注册 + 目标仓加锁（字典序防死锁）+ 冲突升级（等待→死锁检测→暂停→shadow）
  - 🟡 [MODIFIED] `Nightly Test Sweep` — 巡检前 list 活跃会话 → 冲突仓跳过（flock=进程级 / Hub=会话级，两层叠加）
  - 🟡 [MODIFIED] `40+ 子模块仓库` — pre-commit v1.1.0 提交锁校验（他会话持锁 → 阻断）
- **设计要点**:
  - 仓库级锁 + 租约/心跳 + 僵尸窃取；多仓字典序加锁防死锁；等待超时 → 死锁检测
  - 冲突升级路径按会话类型分层：交互=警告+用户决策；无头=等待→暂停→shadow；巡检=跳过；CI=阻断提交
  - pause/resume（SIGSTOP/SIGCONT 进程树）；shadow-begin/apply/abort（校验和 + 原子 mv 拷回）
  - 纯本地开发工具链 — 零 HTTP 接口（🐍 门禁不触发）、零服务端状态变更（无 MQ 事件）、零价值流影响
- **视图连线验证**:
  - [x] 架构变迁视图：Plateau v13→Gap→WP→Plateau v14 链路可追踪（sourceConnection 完整）
  - [x] 目标拓扑视图：Session Hub 部署/接入流全部连线，无孤立节点、无悬空引用
  - [x] Archi CLI 验证通过：`Loaded model: 'daydaymoney Enterprise Landscape v14 Target'` / `'v66 多智能体会话互知与冲突防护'`

---

## v13 ✅ current — Git Hooks 入库与统一管理（git-hooks-version-control）

- **状态**: ✅ current（已交付）
- **迭代**: git-hooks-version-control
- **作者**: claude
- **设计日期**: 2026-08-06 00:08
- **交付日期**: 2026-08-06
- **设计文档**: `docs/superpowers/specs/2026-08-06-git-hooks-version-control-design.md`
- **变更文件**:
  - 🆕 `v13-enterprise-landscape-20260806-0008-claude.puml`（基于 v12）
  - 🆕 `v13-enterprise-landscape-20260806-0008-claude.diff.archimate`（含迁移视图 v12→v13 + Git Hooks 部署流拓扑视图 + sourceConnection 连线）
  - 🆕 `v13-enterprise-landscape-20260806-0008-claude.full.archimate`（合并 v12→v13 变更后的全量拓扑 + 全量视图连线）
  - 🆕 `v13-enterprise-landscape-20260806-0008-claude.mermaid.md`
- **变更明细 (vs v12)**:
  - 🟢 [NEW] `Git Hooks 模板 SSOT`（Technology_Artifact — scripts/hooks/templates/，6 语言 pre-commit + random_test_runner.sh）
  - 🟢 [NEW] `Hooks 分发/校验器`（Technology_Artifact — install-hooks-all.sh + commit_with_submodules.py --check/deploy，HOOK_VERSION 版本戳）
  - 🟢 [NEW] `.githooks/ 入库真源 ×38 仓`（Technology_Artifact — core.hooksPath 激活，零复制零漂移）
  - 🔴 [DEPRECATED] `.git/hooks/ 复制模式`（install_root_hooks.sh / install-arch-hooks.sh / db 副本退役）
- **视图连线验证**:
  - [x] 架构变迁视图：Plateau v12 → Gap → WP → Plateau v13 链路可追踪（sourceConnection 完整）
  - [x] 目标拓扑视图：SSOT → 分发器 → .githooks → GitLab + Developer 一键激活 + 旧模式迁移，全部连线
  - [x] Archi CLI 验证通过：`Loaded model: 'daydaymoney Enterprise Landscape v13 Target'`

---

## v65 🎯 target — Nightly 随机单测扫描 + 自动修复（random test sweep + auto-fix）

- **状态**: 🎯 target（已设计，待交付；base = v64 微信身份）
- **迭代**: nightly-test-sweep
- **作者**: claude
- **设计日期**: 2026-08-05 23:47
- **交付日期**: —
- **设计文档**: `architecture/v65-application-integration-20260805-2347-claude.{puml,archimate,mermaid.md}`
- **变更文件**:
  - 🆕 `v65-application-integration-20260805-2347-claude.puml`（基于 v64）
  - 🆕 `v65-application-integration-20260805-2347-claude.archimate`（含迁移视图 v64→v65 + sweep 拓扑视图 + sourceConnection 连线）
  - 🆕 `v65-application-integration-20260805-2347-claude.mermaid.md`
- **变更明细 (vs v64)**:
  - 🟢 [NEW] Nightly (00:00-08:00) 全量 ~40 子仓随机单测扫描 — `shareLib/random_test_runner.sh` SSOT，30% 随机比例
  - 🟢 [NEW] 自动修复流水线 — claude-agent 无头 Claude Code：reproduce → fix → re-verify → commit → push main
  - 🟢 [NEW] 未解决失败台账 — 追加 `UNIT_TEST_DEBT.md`（7 天去重）+ 每日报告
- **配套**: 本条目（VERSION_HISTORY.md）

---

## v12 ✅ current — enterprise-landscape 基线回填（v11 → v12 结构变迁）

- **状态**: ✅ current（enterprise-landscape 视图；application-integration current 仍为 v63）
- **迭代**: enterprise-landscape-baseline-backfill
- **作者**: claude
- **设计日期**: 2026-08-05 18:00
- **交付日期**: 2026-08-05 18:00
- **视图说明**: enterprise-landscape 自 v11 (2026-07-07) 后因「无新组件」约定停止更新，但 v29/v32/v39 等实际引入了新 Go 服务、v57 Django 完全退役 —— 本版按 application-integration v64 tip 回填真实拓扑
- **变更文件**:
  - 🆕 `v12-enterprise-landscape-20260805-1800-claude.puml` (基于 v11)
  - 🆕 `v12-enterprise-landscape-20260805-1800-claude.archimate`（含架构变迁 v11→v12 视图 + sourceConnection 连线）
  - 🆕 `v12-enterprise-landscape-20260805-1800-claude.mermaid.md`
- **变更明细 (vs v11)**:
  - 🟢 [NEW] 应用组件基线回填：taskGitOauth (:8002, v29)、taskAiProvider (:8010, v32)、taskTenantService (:8020, v39)、taskReferral (:8025)、taskSSE (:8007)、taskAgentSupport (:8011)、taskAIEndPoint (:8013)、taskContainerGateway (:8014)、taskCredentialService (:8015)、taskAIComment (:8019)、taskEvents (Kafka intents)、shareLib/authz (v63)
  - 🟡 [MODIFIED] taskAuth — PDP 集中授权 (v63) + wechat_identity (v64)；taskBill — billing_payment_pending (v64)；业务层新增 Identity & Access (RBAC) / Referral & Growth 服务
  - 🔴 [DEPRECATED] Django saas-backend — v57 完全退役 (2026-07-30)，task2app/ 已删除；ai-provider Django → taskAiProvider Go
  - ⚪ [UNCHANGED] APISIX Gateway (:18081)、GitLab CE (:8012)、Go Runtime、MySQL per-service（v12 原文误写「SQLite per-service」，已按运行事实修正，见 docs-cleanup 2026-08-24 条目）、Redis/Kafka、Aliyun ECS/STS、上游 LLM
- **端口口径**: 以 `conf/runAll.yaml` 为 SSOT（taskTaskService :8017 / taskAIComment :8019 / taskSSE :8007，纠正 v64 puml 中的端口标注）
- **视图连线验证**:
  - [x] 架构变迁视图：Plateau v11 → Gap → WP → Plateau v12 链路可追踪（sourceConnection 完整）
  - [x] Archi CLI 验证通过：`Loaded model: 'daydaymoney Enterprise Landscape v12 Current'`
- **配套更新**: `README.md` 文件索引与「当前架构版本」刷新（原停留在 v1 已过期）

---

## v63 ✅ current — RBAC 综合设计：自定义角色 + 小组资源分配（双线合并）

- **状态**: ✅ current（已交付）
- **迭代**: rbac-merged-v63
- **作者**: claude
- **设计日期**: 2026-08-05 02:38
- **交付日期**: 2026-08-05 04:00
- **设计文档**: `docs/superpowers/specs/2026-08-05-rbac-merged-v63-design.md`（现行权威）
- **合并来源**:
  - 并行线 v61 (`2026-08-04-rbac-final-design.md`) + v62 (`2026-08-04-rbac-with-groups-design.md`) — 4 固定角色 + group_admin + 资源→组分配
  - 自定义角色线 (`2026-08-05-rbac-custom-role-design.md`) — 租户自定义角色 + RequirePerm + 硬切换 + 组→角色继承 + 前端 37 处
- **变更文件**:
  - 🆕 `v63-application-integration-20260805-0238-claude.puml`
  - 🆕 `v63-application-integration-20260805-0238-claude.archimate`（含架构变迁 v62→v63 + 目标拓扑视图 + sourceConnection 完整连线）
  - 🆕 `v63-application-integration-20260805-0238-claude.mermaid.md`
  - ⚠️ 已删除冲突文件: `v61-application-integration-20260805-0233-claude.{puml,archimate,mermaid.md}`（与并行线 v61 版本号冲突，内容并入 v63）
- **变更明细 (vs v62)**:
  - 🟢 [NEW] 租户级自定义角色 — auth_role.company_id 列 + 4 CRUD API（勾选 17 个 tenant 权限码）
  - 🟢 [NEW] RequirePerm 权限码集合判定（替代 RequireRole 优先级主路径）+ X-Tenant-Perms 注入
  - 🟢 [NEW] taskFE 前端改造（usePermissions，37 处 is_admin/is_superuser → HasPerm）
  - 🟢 [NEW] RoleChanged 事件（自定义角色增改删 → PDP 缓存失效）
  - 🟡 [MODIFIED] v62 既有保留: group_admin/tenant_group_admin/tenant_resource_group_assignment + 6 组/资源事件（合并入权限码模型）
  - 🟡 [MODIFIED] shareLib/authz — RequirePerm + RequireGroupAdmin + HasGroupResourceAccess
  - 🟡 [MODIFIED] APISIX — forward-auth 注入 X-User-Roles + X-Tenant-Perms
  - 🟡 [MODIFIED] taskProject/taskTask/taskCloud/taskBill — RequirePerm + 组资源继承分支
  - 🔴 [DEPRECATED] is_admin/is_superuser 直接鉴权（硬切换）、policy.go creator 硬编码、RequireRole 优先级判定
- **新增接口**: 24 个 Go API（taskAuth 10 + taskTenantService 14），**零 Python 接口**（🐍 门禁不触发）
- **实施周期**: 10 天（D1-D2 authz+PDP+角色 CRUD；D3-D4 tenant 角色/组/资源 API；D5-D6 服务替换+网关+前端；D7 前端收尾；D8 硬切换收尾+单测；D9 E2E；D10 回归）
- **视图连线验证**:
  - [x] 架构变迁视图：Plateau v62 → Gap → WP → Plateau v63 链路可追踪
  - [x] 目标拓扑视图：Gateway→PDP→Services→DB + 8 事件 + Deprecated（sourceConnection 完整，50 关系 48 视图连线，2 无连线关系已补节点）
  - [x] Archi CLI 验证通过：`Loaded model: 'v63 RBAC Custom-Role + Group Resource Assignment'`
- **交付验证** (2026-08-05):
  - ✅ 全部迁移 SQL 在真实 MySQL 执行成功（027/028 + taskTenant 004-007；collation 已与现有表对齐 utf8mb4_0900_ai_ci）
  - ✅ 种子数据核验：5 内置角色 / 23 权限码 / 35 角色→权限关联；回填 1 super_admin + 0 遗漏
  - ✅ taskAuth + taskTenantService 新代码部署（:8003/:8020），PDP check 链路可用
  - ✅ E2E 9 用例真实环境通过（`docs/superpowers/specs/scripts/rbac-v63-e2e.sh`：角色 CRUD/权限码校验/内置锁定/列表/PDP/403/组资源/role-exists/跨租户隔离）
  - ✅ taskAuth 全量测试套件全绿（含 userId cookie 期望同步、死测试清理）
  - ✅ taskFE vue-tsc 通过（usePermissions 接入 PeopleManage/PeopleGroups）
  - 📌 v57 已标 `@status: archived`（application-integration current 切换至 v63）

---

## v64 🎯 target — 微信身份统一设计（跨应用 openid/unionid + 绑定 + 支付回调核对）

- **状态**: 🎯 target（已设计，待交付）
- **迭代**: wechat-login-openid-unionid
- **作者**: claude
- **设计日期**: 2026-08-05 17:53
- **修订日期**: 2026-08-05 18:05（移除支付身份核对）
- **交付日期**: —
- **设计文档**: `docs/superpowers/specs/2026-08-05-wechat-login-openid-unionid-design.md`（现行权威）
- **修订记录**:
  - 🔄 2026-08-05 18:05 — 移除「支付回调身份核对映射 userId」（登录/支付解耦：手机号登录+微信支付、微信登录+支付宝支付等场景正常）；删除 taskAuth internal resolve 接口与 WechatPaymentIdentityMismatch 事件；回调支付凭据（方式/appid/openid/unionid）仅记录不判定；保留 pending 持久化 + 幂等 + PAYMENT_SUCCEEDED
- **变更文件**:
  - 🆕 `v64-application-integration-20260805-1753-claude.puml` (基于 v63)
  - 🆕 `v64-application-integration-20260805-1753-claude.archimate`（含架构变迁 v63→v64 + 目标拓扑视图 + sourceConnection 完整连线）
  - 🆕 `v64-application-integration-20260805-1753-claude.mermaid.md`
- **变更明细 (vs v63)**:
  - 🟢 [NEW] `wechat_identity` 表（taskAuth DB）— (app_key+openid/unionid)→user_id 双映射；auth_login_method wechat 行为兼容层
  - 🟢 [NEW] 多应用微信 OAuth — 配置模型 apps[]（web=qrconnect 扫码 / inapp=公众号 snsapi_userinfo）；回调按 app 独立 path；state 编码 app_key+bind 标记
  - 🟢 [NEW] 微信绑定/解绑 — `/api/auth/wechat/bind/` + bind/callback（409 冲突分支）+ DELETE unbind（校验剩余登录方式）
  - 🟢 [NEW] taskBill 支付回调可靠性 — `billing_payment_pending` 表（持久化替代内存 map）+ 幂等 markOrderPaid + 凭据记录 + PAYMENT_SUCCEEDED（**登录/支付解耦，无身份核对**）
  - 🟢 [NEW] 5 事件：WechatIdentityLinked / WechatIdentityConflict / WechatBound / WechatUnbound / PAYMENT_SUCCEEDED
  - 🟡 [MODIFIED] taskAuth — 匹配顺序（unionid 优先 → (app_key,openid) → 新建）+ 绑定转移（分裂收敛）
  - 🟡 [MODIFIED] taskFE — 登录页微信入口（UA 检测 → 扫码 / 微信内一键登录）+ 绑定/解绑入口
  - 🟡 [MODIFIED] taskBill — 回调可靠性改造；`billing_resource_order` +user_id（下单会话）/pay_method/pay_app_key/pay_openid/pay_unionid（凭据记录）
  - 🟡 [MODIFIED] USER_LOGGED_IN — payload 增 provider_app（向后兼容）
- **新增接口**: 7 个 Go API（taskAuth 6 + taskBill 改造 1），**零 Python 接口**（🐍 门禁不触发）
- **场景覆盖**: 跨应用登录 G1-G5、正向/反向绑定 S1-S6、生命周期 S7-S12、支付回调可靠性（登录/支付解耦）
- **视图连线验证**:
  - [x] 架构变迁视图：Plateau v63 → Gap → WP1(identity)/WP2(payment) → Plateau v64 链路可追踪
  - [x] 目标拓扑视图：63 关系全部 sourceConnection 连线，无孤立节点
  - [x] Archi CLI 验证通过：`Loaded model: 'v64 WeChat Identity Unification + Payment Callback Identity Check'`
- **实施周期建议**: 8 天（D1-D2 wechat_identity+匹配；D3-D4 多应用 OAuth+回调；D5 绑定/解绑；D6-D7 taskBill 核对链；D8 E2E/回归）

---

## v65 🎯 target — 夜间随机单测巡检与自愈（nightly-test-sweep）

- **状态**: 🎯 target（已设计，待交付）
- **迭代**: nightly-test-sweep
- **作者**: claude
- **设计日期**: 2026-08-05 23:47
- **交付日期**: —
- **基于**: v64 target（领域独立 — 微信登录 vs DevOps 工具链，无组件冲突）
- **设计文档**: `docs/superpowers/specs/2026-08-05-nightly-test-sweep-design.md`（现行权威）
- **变更文件**:
  - 🆕 `v65-application-integration-20260805-2347-claude.puml` (基于 v64)
  - 🆕 `v65-application-integration-20260805-2347-claude.archimate`（含架构变迁 v64→v65 + 巡检拓扑视图 + v63→v64 历史参考链 + sourceConnection 完整连线，75/75 关系有连线）
  - 🆕 `v65-application-integration-20260805-2347-claude.mermaid.md`
- **变更明细 (vs v64)**:
  - 🟢 [NEW] NightlyTestSweep（TechnologyProcess）— cron 00:20 触发、窗口 00:00–08:00、flock 互斥、串行巡检 ~40 子仓、随机 30% 抽测（至少 1 单元）、每仓 40min 预算、07:30 全局硬停
  - 🟢 [NEW] `scripts/lib/random_test_runner.sh`（TechnologyArtifact）— 共享抽测库 SSOT（rt_* 函数），pre-commit 模板重构 source 复用 + deploy 脚本分发 lib + CI 校验扩展
  - 🟢 [NEW] claude-agent（ApplicationComponent）— 无头修复执行器：重跑复现 → 修复（允许改产品代码，规则 41）→ 复跑验证 → commit（禁 --no-verify，规则 6/28）→ push main
  - 🟢 [NEW] 40+ 子模块仓库巡检目标池（.gitmodules SSOT；脏工作区/无单测/无 remote 跳过）
  - 🟢 [NEW] `.learnings/UNIT_TEST_DEBT.md` 债台账联动（未解失败 7 天去重追加）
  - 🟢 [NEW] `logs/nightly-test-sweep-report-*.md` 每日报告（skip/pass/fixed/unfixed + commit hash）
- **新增接口**: 0 个（无 HTTP API；纯 cron 工具链，🐍 门禁不触发）
- **业务事件**: 无对应事件（DevOps 批处理，不改变业务事实）
- **价值流**: 无用户可见价值流影响
- **实施周期**: 4 天（D1 共享库+钩子重构+分发；D2 巡检编排+dry-run；D3 修复闭环+台账+cron；D4 E2E 验证）
- **视图连线验证**:
  - [x] 架构变迁视图：Plateau v64 → Gap → WP1(库)/WP2(巡检)/WP3(修复) → Plateau v65 链路可追踪（含 v63→v64 历史参考段）
  - [x] 巡检拓扑视图：cron→Sweep→子仓池/claude-agent/台账/报告 + sharedRunner 复用（8 连线全连接）
  - [x] Archi CLI 验证通过：`Loaded model: 'v65 Nightly Test Sweep + Auto-Fix'`

---

## v62 🎯 target — RBAC 含小组角色与资源分配

- **状态**: 🎯 target（🔄 已被 v63 合并为综合设计）
- **迭代**: rbac-with-groups-resource-assignment
- **作者**: claude
- **设计日期**: 2026-08-04 16:30
- **基于**: v61 (RBAC 基础)
- **设计文档**: `docs/superpowers/specs/2026-08-04-rbac-with-groups-design.md`
- **变更文件**:
  - 🆕 `v62-application-integration-20260804-1630-claude.puml`
  - 🆕 `v62-application-integration-20260804-1630-claude.mermaid.md`
- **变更明细 (vs v61)**:
  - 🟢 [NEW] group_admin 角色 — priority=75, 组内成员管理, scope=group_id
  - 🟢 [NEW] tenant_group_admin 表 — 小组管理员指派
  - 🟢 [NEW] tenant_resource_group_assignment 表 — 资源→组分配
  - 🟢 [NEW] RequireGroupAdmin / HasGroupResourceAccess 中间件
  - 🟢 [NEW] 6 个 Domain Events
  - 🟡 [MODIFIED] group_handlers.go — requireCompanyMember → RequireGroupAdmin
  - 🟡 [MODIFIED] 资源服务 — + 组资源继承检查
  - 🆕 10 个 API (groups + group-admin + group-members + resource-group)
- **层级**: tenant_admin 管理组 → group_admin 管理组内成员 → 成员继承组资源权限
- **实施周期**: 在 v61 基础上 +3 天

---

## v61 🎯 target — RBAC 身份角色权限体系

- **状态**: 🎯 target（已设计，待交付）
- **迭代**: rbac-identity-role-permission-system
- **作者**: claude
- **设计日期**: 2026-08-04 16:00
- **基于**: v57 current (Django 退役 + API 路径规范迁移 `4d34714` 已完成)
- **设计文档**: `docs/superpowers/specs/2026-08-04-rbac-final-design.md`
- **变更文件**:
  - 🆕 `v61-application-integration-20260804-1600-claude.puml`
  - 🆕 `v61-application-integration-20260804-1600-claude.archimate`（Archi 验证通过 ✅）
  - 🆕 `v61-application-integration-20260804-1600-claude.mermaid.md`
- **变更明细**:
  - 🟡 [MODIFIED] taskAuth — +4 RBAC 表 + 角色管理 API + PDP 端点；forward-auth 注入 X-User-Roles
  - 🟡 [MODIFIED] taskTenantService — +2 RBAC 表 + member-role/group-role API；替换 policy.go
  - 🟢 [NEW] shareLib/authz — 薄客户端 (permissions + roles + context + middleware, ~200 行)
  - 🟡 [MODIFIED] taskCloud/taskBill/taskProject/taskTask — hand-rolled 鉴权 → authz.RequireRole()
  - 🟡 [MODIFIED] taskEvents — +RoleChanged consumer (缓存失效) + CompanyCreated→creator 角色
  - 🟡 [MODIFIED] APISIX Gateway — forward-auth 注入 X-User-Roles + X-Tenant-Memberships
  - 🔴 [DEPRECATED] is_admin/is_superuser 直接鉴权 + policy.go creator 硬编码
- **新增 API**: 12 个 (taskAuth 10 + taskTenant 2)，全部 Go，零 Python
- **API 合规**: 100% 遵循 `/api/serviceName/funcName/key/value/` 规范
- **迁移 SQL**: dataMigrate/taskAuth/027+028 + dataMigrate/taskTenant/017
- **实施周期**: 7 天
- **视图连线验证**: [x] 架构变迁 Plateau→Gap→WP 链路 OK | [x] Archi load OK

---

- **状态**: 🎯 target（v4 终稿，可直接开工）
- **迭代**: rbac-identity-role-permission-system
- **作者**: claude
- **设计日期**: 2026-08-04
- **设计文档**: `docs/superpowers/specs/2026-08-04-rbac-implementation-blueprint.md` ✅ 现行唯一权威
- **旧文档** (已作废): v1 高层设计、v2 框架实施、v3 完整设计
- **变更文件**:
  - `v60-application-integration-*.puml/.archimate/.mermaid.md` — 架构 v3 迭代创建，v4 未变
- **v4 新增内容** (vs v3):
  - 完整 Go 代码骨架 (5 源文件, ~300 行, 可直接创建)
  - 14 个 API 完整请求/响应契约
  - 40+ 端点 → 权限码映射表
  - 可执行迁移 SQL + 回滚 SQL
  - APISIX 网关配置变更
  - 7 服务 × 逐 handler 改造清单
  - 10 单元测试 + 10 E2E 用例矩阵
  - 事件流拓扑
  - 7 天实施甘特
- **实施周期**: 7 天 (day1: 代码骨架+SQL; day2: PDP; day3: 租户角色; day4: 全服务迁移; day5-6: 前端+E2E; day7: 收尾)

---

## v60 🎯 target — RBAC 集中授权架构 (v3 迭代)

- **状态**: 🎯 target（已设计，待交付）
- **迭代**: rbac-identity-role-permission-system-v3
- **作者**: claude
- **设计日期**: 2026-08-04 14:00
- **交付日期**: —
- **设计文档**: `docs/superpowers/specs/2026-08-04-rbac-complete-design-v3.md`
- **替代**: v59 (shareLib 重引擎方案被跳过)
- **变更文件**:
  - 🆕 `v60-application-integration-20260804-1400-claude.puml`
  - 🆕 `v60-application-integration-20260804-1400-claude.archimate`（含架构变迁 v57→v60 + 目标拓扑视图）
  - 🆕 `v60-application-integration-20260804-1400-claude.mermaid.md`
- **变更明细 (vs v57 current)**:
  - 🔄 [REFINED] shareLib/authz — v59 重引擎 → v60 薄客户端 (Context 读取 O(1) + HTTP fallback 到 PDP)
  - 🟢 [NEW] taskAuth PDP — `/api/internal/authz/check` 集中授权判定端点 + Redis 统一缓存
  - 🟢 [NEW] tenant_group_role 表 — 组→角色继承 (Phase 1 纳入)
  - 🟢 [NEW] TenantGroupRoleChanged 事件 — 组角色变更 → 成员权限级联刷新
  - 🟢 [NEW] CompanyCreated consumer — Kafka 事件驱动创建者自动分配 tenant_admin 角色
  - 🟡 [MODIFIED] taskAuth — +4 RBAC 表 + PDP API + 角色管理 API
  - 🟡 [MODIFIED] taskTenantService — +2 RBAC 表 + 组角色 API + 替换 policy.go
  - 🟡 [MODIFIED] taskCloud/taskBill/taskProject/taskTask — hand-rolled 鉴权 → authz.RequireRole()
  - 🟡 [MODIFIED] taskEvents — 新增 RoleChanged consumer (缓存失效)
  - 🟡 [MODIFIED] APISIX Gateway — forward-auth → taskAuth PDP + 注入 X-User-Roles/X-Tenant-Memberships
  - 🔴 [DEPRECATED] is_admin 直接鉴权 → HasRole(tenant_admin)
  - 🔴 [DEPRECATED] is_superuser 直接鉴权 → HasRole(super_admin)
  - 🔴 [DEPRECATED] policy.go creator 硬编码 → CompanyCreated consumer 事件驱动
- **关键设计决策**:
  - 集中 PDP: taskAuth 为唯一授权决策点 (Policy Decision Point)
  - 组继承: tenant_company_group → tenant_group_role → 成员自动继承
  - 事件驱动: CompanyCreated → creator 自动 tenant_admin；RoleChanged → 缓存即时失效
  - 双轨兼容: 保留 is_admin/super_admin 字段，同步写入新 role 表
  - 实施周期: Phase 1 从 8-14天 精简到 5-7天
- **架构变迁视图**: Plateau v57 → Gap (缺 RBAC/集中PDP/组继承) → WP Phase1+Phase2 → Plateau v60
- **新增 API**: 14 个 Go API (taskAuth 10 + taskTenantService 4)，零 Python 接口
- **视图连线验证**:
  - [x] 架构变迁视图：Plateau→Gap→WP→Plateau 链路可追踪
  - [x] 目标拓扑视图：Gateway→PDP→Services (sourceConnection 完整)
  - [x] Archi CLI 验证通过：`Loaded model: 'v60 RBAC Centralized PDP Architecture'`

---

## v59 🎯 target — 身份角色与权限体系

- **状态**: 🎯 target（已设计，待交付）
- **迭代**: identity-role-permission-system
- **作者**: claude
- **设计日期**: 2026-08-04 14:00
- **交付日期**: —
- **设计文档**: `docs/superpowers/specs/2026-08-04-identity-role-permission-system-design.md`
- **变更文件**:
  - 🆕 `v59-application-integration-20260804-1400-claude.puml` (基于 v57)
  - 🆕 `v59-application-integration-20260804-1400-claude.archimate`（含架构变迁 v57→v59 + 目标拓扑视图 + sourceConnection 连线）
  - 🆕 `v59-application-integration-20260804-1400-claude.mermaid.md`
- **变更明细**:
  - 🟡 [MODIFIED] taskAuth — 新增 auth_role/auth_permission/auth_role_permission/auth_user_role 表；角色 CRUD API；权限查询 API；平台角色管理 /api/system-admin/roles/*
  - 🟡 [MODIFIED] taskTenantService — 新增 tenant_member_role 表；member role API；is_admin→role 迁移
  - 🟢 [NEW] shareLib/authz — 权限检查中间件：RequirePlatformRole/RequireTenantRole/HasPermission/GetUserRoles
  - 🟡 [MODIFIED] APISIX Gateway — forward-auth 增强：注入 X-User-Roles + X-User-Permissions
  - 🟡 [MODIFIED] taskAuth DB — +4 表 (auth_role/auth_permission/auth_role_permission/auth_user_role)
  - 🟡 [MODIFIED] taskTenant DB — +1 表 (tenant_member_role)
  - 🟢 [NEW] Domain Events — PlatformRoleAssigned, PlatformRoleRevoked, TenantRoleChanged
  - 🔴 [DEPRECATED] is_admin 直接鉴权 → HasTenantRole(tenant_admin)
  - 🔴 [DEPRECATED] is_superuser 直接鉴权 → HasPlatformRole(super_admin)
  - 🆕 数据迁移: `dataMigrate/taskAuth/023_role_permission.sql` + `dataMigrate/taskTenantService/016_member_role.sql`
- **架构变迁视图**: Plateau v57 → Gap (缺分层角色/细粒度权限/租户隔离) → WP Phase1+Phase2 → Plateau v59
- **新增接口**: 10 个 Go API（taskAuth 7 + taskTenantService 3），零 Python 接口
- **视图连线验证**:
  - [x] 架构变迁视图：Plateau v57 → Gap；WP1/WP2 → Gap / Plateau v59（sourceConnection 连线完整）
  - [x] 目标拓扑视图：Gateway→shareLibAuthz→Services→DB；DomainEvents→Kafka
  - [x] Archi CLI 验证通过：`Loaded model: 'v59 Identity Role Permission System — RBAC Architecture'`

---

## v58 🎯 target — 邮箱邀请系统增强

- **状态**: 🎯 target（已设计，待交付）
- **迭代**: email-invite-enhancement
- **作者**: claude
- **设计日期**: 2026-08-03 12:00
- **交付日期**: —
- **设计文档**: `docs/superpowers/specs/2026-08-03-email-invite-enhancement-design.md`
- **变更文件**:
  - 🆕 `v58-application-integration-20260803-1200-claude.puml` (基于 v57)
  - 🆕 `v58-application-integration-20260803-1200-claude.mermaid.md`
- **变更明细**:
  - 🟡 [MODIFIED] taskAuth — 新增 DB 列 `invite_reason`/`account_expires_at`/`assigned_role`；API 列表 JOIN auth_user 返回 `inviterName`；前端邀请弹窗增加角色/有效期/原因，列表表格增加4列
  - 🆕 数据库迁移: `dataMigrate/taskAuth/019_email_invite_enhancement.sql`
  - 🆕 设计文档: `docs/superpowers/specs/2026-08-03-email-invite-enhancement-design.md`
- **代码文件**:
  - ✏️ `taskAuth/src/auth_email_invite.go` — 结构体+创建+列表+重发+批量重发处理
  - ✏️ `taskFE/app/src/views/SystemAdminUsers.vue` — 邀请弹窗+列表表格增强

---

## v56 🎯 target — Django→Go 最终迁移路线图

- **状态**: 🎯 target（已设计，待交付）
- **迭代**: django-final-go-migration
- **作者**: claude
- **设计日期**: 2026-07-29 16:45
- **交付日期**: —
- **设计文档**: `docs/superpowers/specs/2026-07-29-django-to-go-final-migration-design.md`
- **变更文件**:
  - 🆕 `v56-application-integration-20260729-1645-claude.puml` (基于 v55)
  - 🆕 `v56-application-integration-20260729-1645-claude.archimate`（含架构变迁 v55→v56 + 目标拓扑视图）
  - 🆕 `v56-application-integration-20260729-1645-claude.mermaid.md`
- **变更明细**:
  - 🔴 [DEPRECATED] Django saas-backend — 8 Phase 逐步退役，最终删除 task2app/
  - 🔴 [DEPRECATED] Saas_Ai_Provider Django — 死代码清理 (Phase 1)
  - 🟢 [NEW] Kafka Consumers (Go) — 16 intent handlers 替代 Kafka→Django HTTP 回环
  - 🟡 [MODIFIED] taskAuth — 接管 users CRUD, profiles, SSO bridge, session store
  - 🟡 [MODIFIED] taskGitOauth — 接管 git identities store
  - 🟡 [MODIFIED] taskTenantService — 接管 companies, members, groups, workspace-access
  - 🟡 [MODIFIED] taskProjectService — 接管 projects CRUD, deliverable/progress, ACL
  - 🟡 [MODIFIED] taskTaskService — 接管 todos CRUD
  - 🟡 [MODIFIED] taskAIComment — 接管 comment import 直写
  - 🟡 [MODIFIED] taskCloudService — 接管 platforms, VPC, images, feature-params 全量
  - 🟡 [MODIFIED] taskBill — 接管 APISIX 直连, 订阅/支付, pricing, refund
  - 🟡 [MODIFIED] taskEvents — 接管 Kafka consumers + SSE dispatch
- **架构变迁视图**: Plateau v55 → 7 Gaps → 8 WorkPackages → Plateau v56
- **盘点结论**: 只剩 1 个活跃 Django 进程 (saas-backend, ~1,150 .py 文件)；已迁出 git-oauth (Go taskGitOauth) 和 ai-provider (Go taskAiProvider)

---

## v55 🎯 target — Gateway-Controlled Frontend Routing

- **状态**: 🎯 target（已设计，待交付）
- **迭代**: gateway-controlled-frontend-routing
- **作者**: claude
- **设计日期**: 2026-07-28 15:38
- **交付日期**: —
- **设计文档**: `docs/superpowers/specs/2026-07-28-gateway-controlled-frontend-routing-design.md`
- **变更文件**:
  - 🆕 `v55-application-integration-20260728-1538-claude.puml`
  - 🆕 `v55-application-integration-20260728-1538-claude.archimate`（含架构变迁 v54→v55 + 目标拓扑视图）
  - 🆕 `v55-application-integration-20260728-1538-claude.mermaid.md`
- **变更明细**:
  - 🟢 [NEW] `taskFE` upstream — APISIX 新增 SPA 前端上游
  - 🟢 [NEW] `taskFE-static` container — 生产环境 Nginx 容器提供 SPA 静态文件
  - 🟢 [NEW] SPA catch-all 路由 — priority 10, /* [GET,HEAD] → taskFE
  - 🟡 [MODIFIED] Nginx config — `location /` 从 Django:8001 → APISIX:18081
  - 🟡 [MODIFIED] Django auth_views.py — 移除 `_serve_spa_shell()`, spa_catch_all 回归 JSON
  - 🔴 [DEPRECATED] `auth-login-page` APISIX 路由 — 被 spa-catch-all 替代
  - 🔴 [DEPRECATED] `django-default` APISIX 路由 — 被 spa-catch-all 替代
  - 🔴 [DEPRECATED] Django SPA shell 渲染 — Django 不再承担页面渲染职责
- **架构变迁视图**: Plateau v54 → Gap → WorkPackage → Plateau v55

---
> 文件命名规范: `v<N>-<视图名>-<YYYYMMDD-HHMM>-<作者>.puml` — 版本号前置，同版本多视图自然聚合；每个版本独立文件，老版本只读保留

---

## v54 ✅ current — comment multi-CSC parallel (OPT-019)

- **状态**: ✅ current（已交付）
- **迭代**: comment-multi-csc-parallel / OPT-20260723-019
- **作者**: claude
- **设计日期**: 2026-07-23 17:05
- **交付日期**: 2026-07-23 17:10
- **设计文档**: `docs/superpowers/specs/2026-07-23-comment-multi-csc-parallel-design.md`
- **变更文件**:
  - 🆕 `v54-application-integration-20260723-1705-claude.puml` (基于 v53)
  - 🆕 伴生格式: `.archimate` + `.mermaid.md`
- **变更明细**:
  - 🟡 `cloud_server_configs` — 去掉 `UNIQUE(company,task)`；`idx(workspace,task)` + `UNIQUE(workspace,task,comment)`
  - 🟢 `ensureCommentCloudServerConfig` — 每评论独立 CSC
  - 🟡 comment_container_bindings advance — 并行挂接不同 csc_id
  - ✅ [CLOSES] Gap: independent 无法并行挂不同机器
- **视图连线验证**:
  - [x] 架构变迁视图：Plateau v53 → Gap；WP → Gap / Plateau v54
  - [x] 目标拓扑视图：FE→CCB→ensure→CSC；CCB→mock

---

## v53 ✅ archived — gitService durable GITLAB_HOME

- **状态**: ✅ archived（已被 v54 替代为 application-integration current）
- **迭代**: gitservice-durable-gitlab-home
- **作者**: claude
- **设计日期**: 2026-07-22 22:35
- **交付日期**: 2026-07-22 22:55
- **设计文档**: `docs/superpowers/specs/2026-07-22-gitservice-durable-gitlab-home-design.md`
- **变更文件**:
  - 🆕 `v53-application-integration-20260722-2235-claude.puml` (基于 v52)
  - 🆕 伴生格式: `.archimate` + `.mermaid.md`
- **变更明细**:
  - 🟡 gitService / GitLab CE — 卷从相对 `./gitlab_home`（常落 tmpfs）改为 durable `$GITLAB_HOME`
  - 🟢 GitLabHomeDurableStore
  - ✅ [CLOSES] Gap: 容器/工作区重建导致 GitLab 数据丢失
- **视图连线验证**:
  - [x] 架构变迁视图：Plateau v52 → Gap；WP → Gap / Plateau v53
  - [x] 目标拓扑视图：gitService→GitLab CE→GitLabHomeDurableStore

---

## v52 ✅ archived — 服务器运行状态评论 Tab

- **状态**: ✅ archived（已被后续版本替代）
- **迭代**: comment-runtime-server-tabs
- **作者**: claude
- **设计日期**: 2026-07-22 19:10
- **交付日期**: 2026-07-22 19:15
- **设计文档**: `docs/superpowers/specs/2026-07-22-comment-runtime-server-tabs-design.md`
- **变更文件**:
  - 🆕 `v52-application-integration-20260722-1910-claude.puml` (基于 v51)
  - 🆕 伴生格式: `.archimate` + `.mermaid.md`
- **变更明细**:
  - 🟢 ServerConfigCommentRuntimeTabs
  - 🟢 CommentRuntimeServerTab[] 前端契约
  - 🟡 ServerConfig.logic — runtime 区挂载评论 Tab
  - ✅ [CLOSES] Gap: 运行状态无法按评论关联容器切换
- **视图连线验证**:
  - [x] 架构变迁视图：Plateau v51 → Gap；WP → Gap / Plateau v52
  - [x] 目标拓扑视图：ServerConfig→Tabs→RuntimePanel→cloud API

---

## v51 ✅ archived — 评论级执行细节

- **状态**: ✅ archived（已被 v52 替代为 application-integration current）
- **迭代**: comment-execution-details
- **作者**: claude
- **设计日期**: 2026-07-22 18:50
- **交付日期**: 2026-07-22 19:00
- **设计文档**: `docs/superpowers/specs/2026-07-22-comment-execution-details-design.md`
- **变更文件**:
  - 🆕 `v51-application-integration-20260722-1850-claude.puml` (基于 v48)
  - 🆕 伴生格式: `.archimate` + `.mermaid.md`
- **变更明细**:
  - 🟢 [NEW] `TaskDetailCommentExecutionDetails` — 可折叠执行细节 + dependency badge
  - 🟢 [NEW] `TaskDetailContainerConnectionStatus` — 从 ServerStartStatus 抽出
  - 🟢 [NEW] `CommentExecutionContext` / `CommentExecutionBinding` — 前端展示契约
  - 🟡 [MODIFIED] Vue TaskDetail — layer-association 迁入 active 评论；Runtime 仅 SSE/启动
  - ⚪ [UNCHANGED] taskAIComment、taskTaskService、container API
  - 🎯 [NEW] Plateau v51
  - ✅ [CLOSES] Gap: 执行状态与评论 Feed 脱节
- **视图连线验证**:
  - [x] 架构变迁视图：Plateau v48 → Gap；WP → Gap / Plateau v51
  - [x] 目标拓扑视图：TaskDetail→ExecutionDetails→ContainerConnection/LayerAssociation

---

## v50 🎯 target — 闲置复用启动保护 + 孤儿交叉校验

- **状态**: 🎯 target（已设计，待交付切换 current）
- **迭代**: idle-reuse-boot-guard-orphan-cross-check
- **作者**: claude
- **设计日期**: 2026-07-22 18:15
- **交付日期**: —
- **设计文档**: `docs/superpowers/specs/2026-07-22-idle-reuse-boot-guard-orphan-cross-check-design.md`
- **变更文件**:
  - 🆕 `v50-application-integration-20260722-1815-claude.puml` (基于 v48)
  - 🆕 伴生格式: `.archimate` + `.mermaid.md`
- **变更明细**:
  - 🟡 [MODIFIED] taskCloudService — idle reuse 候选须 `idle_since`；排除 Starting；orphan 删除前 cross-CSC `instance_id` 持有校验
  - 🎯 [NEW] Plateau v50
  - ✅ [CLOSES] Gap: boot 误闲置 + orphan 误删已复用实例
- **视图连线验证**:
  - [x] 架构变迁视图：Plateau v48 → Gap；WP → Gap / Plateau v50
  - [x] 目标拓扑视图：TTS→TCS→CSC/policy/ECS

---

## v49 🎯 target — 引荐积分分成 15 日结算

- **状态**: ⚪ 已废弃（2026-08-06 决定不需要两级分销/引荐分成模式，设计文档与架构文件不落地）
- **迭代**: referral-points-settle-15d
- **作者**: claude
- **设计日期**: 2026-07-22 16:54
- **交付日期**: —
- **设计文档**: `docs/superpowers/specs/2026-07-22-referral-fixed-rate-one-year-design.md`
- **变更文件**:
  - 🆕 `v49-application-integration-20260722-1654-claude.puml` (基于 v48)
  - 🆕 伴生格式: `.archimate` + `.mermaid.md`
- **变更明细**:
  - 🟢 referral_edge / referral_commission_accrual（taskBill）
  - 🟢 settle job + commission-summary / sync-edge
  - 🟡 Django referral-stats（积分待结算/已结算；无二级）
  - 🟡 Vue UserReferral（积分口径）
  - 🟢 原则：一级 5% 积分 + 15 日结算 + 一年窗 + 无二级
- **视图连线验证**:
  - [x] 架构变迁视图：Plateau v48 → Gap；WP → Gap / Plateau v49
  - [x] 目标拓扑视图：Vue→Django→taskBill→accrual/edge/txn

---

## v48 ✅ current — 超管充值消费情况总览

- **状态**: ✅ current（已交付）
- **迭代**: admin-recharge-consumption-overview
- **作者**: claude
- **设计日期**: 2026-07-22 14:50
- **交付日期**: 2026-07-22 15:00
- **设计文档**: `docs/superpowers/specs/2026-07-22-admin-recharge-consumption-overview-design.md`
- **变更文件**:
  - 🆕 `v48-application-integration-20260722-1450-claude.puml` (基于 v41；v41 → archived)
  - 🆕 伴生格式: `.archimate` + `.mermaid.md`
- **变更明细**:
  - taskBill — user-recharge-consumption 聚合
  - Django system-admin — 薄代理 + 用户 enrich
  - Vue PriceManagement — tabs「充值消费情况」
  - Vue UserReferral — 仅已消费可分成说明
  - Plateau v48
  - [CLOSES] Gap: 无超管充值消费总览
- **视图连线验证**:
  - [x] 架构变迁视图：Plateau v41 → Gap；WP → Gap / Plateau v48
  - [x] 目标拓扑视图：Vue→Django→taskBill→ledger/txn

---

## v45 🎯 target — 租户退款申请审批原路退

- **状态**: 🎯 target
- **迭代**: tenant-refund-application
- **作者**: claude
- **设计日期**: 2026-07-22 12:25
- **设计文档**: `docs/superpowers/specs/2026-07-22-tenant-refund-application-design.md`
- **变更文件**:
  - 🆕 `v45-application-integration-20260722-1225-claude.puml` (基于 v41 current)
  - 🆕 伴生格式: `.archimate` + `.mermaid.md`
- **变更明细**:
  - 🟡 [MODIFIED] taskBill — freeze / approve / reject / PayPal·微信退款
  - 🟡 [MODIFIED] Vue BillingDashboard — 申请退款入口
  - 🟢 [NEW] Vue SystemAdminRefundApplications — 审批列表
  - 🟡 [MODIFIED] Django billing_bridge — 超管闸门薄代理
  - 🟢 [NEW] `billing_refund_application` / `billing_payment_ledger`；`frozen_balance`
  - 🎯 [NEW] Plateau v45 — tenant-refund-application
  - ✅ [CLOSES] Gap: 无退款申请冻结与原路退审批
- **视图连线验证**:
  - [x] 架构变迁视图：Plateau v41 → Gap；WP → Gap / Plateau v45
  - [x] 目标拓扑视图：Vue→Django→taskBill→ledger/PayPal/WeChat/Kafka

---

## v44 🎯 target — 排队调度节奏自动关闭

- **状态**: 🎯 target
- **迭代**: schedule-rhythm-auto-close
- **作者**: claude
- **设计日期**: 2026-07-22 12:17
- **设计文档**: `docs/superpowers/specs/2026-07-22-schedule-rhythm-auto-close-design.md`
- **变更文件**:
  - 🆕 `v44-application-integration-20260722-1217-claude.puml` (基于 v41 current)
  - 🆕 伴生格式: `.archimate` + `.mermaid.md`
- **变更明细**:
  - 🟡 [MODIFIED] taskTaskService — `auto_close` + AutoCloser（T-5min warn / 窗外 release）
  - 🟡 [MODIFIED] taskCloudService — lifecycle proxy 前缀 + stop-vm 回退入口
  - 🟡 [MODIFIED] taskContainerGateway — L0 `closing-soon`
  - 🟢 [NEW] onlineServiceJS `POST /task-lifecycle/closing-soon`
  - 🟡 [MODIFIED] Vue 排队调度设置 — 自动关闭勾选
  - 🎯 [NEW] Plateau v44 — schedule-rhythm-auto-close
  - ✅ [CLOSES] Gap: 无窗口自动关闭/预告释放
- **视图连线验证**:
  - [x] 架构变迁视图：Plateau v41 → Gap；WP → Gap / Plateau v44
  - [x] 目标拓扑视图：Vue→GW→TTS→Cloud→Gateway→Agent；TTS→rhythm/slots/Kafka

---

## v43 🎯 target — 平台注册邀请码每日限量

- **状态**: 🎯 target
- **迭代**: registration-invite-code-daily-quota
- **作者**: claude
- **设计日期**: 2026-07-22 11:48
- **设计文档**: `docs/superpowers/specs/2026-07-22-registration-invite-code-design.md`
- **变更文件**:
  - 🆕 `v43-application-integration-20260722-1148-claude.puml` (基于 v41 current)
  - 🆕 伴生格式: `.archimate` + `.mermaid.md`
- **变更明细**:
  - 🟡 [MODIFIED] taskAuth — 邀请码策略/码表/注册核销/Admin+用户 API
  - 🟡 [MODIFIED] task-gateway — 新路由前缀
  - 🟡 [MODIFIED] Vue SystemAdmin / UserProfile / Login / Register
  - 🟢 [NEW] `task-auth.db` 表 `accounts_registration_invite_*`
  - 🎯 [NEW] Plateau v43 — registration-invite-code-daily-quota
  - ✅ [CLOSES] Gap: 无平台注册邀请码/日配额
- **视图连线验证**:
  - [x] 架构变迁视图：Plateau v41 → Gap；WP → Gap / Plateau v43
  - [x] 目标拓扑视图：Vue→GW→taskAuth→DB/Kafka

---

## v42 🎯 target — translate-branch-title 纯 Go（fanyi native）

- **状态**: 🎯 target（PR 待合：taskProjectService#9 / docs#36 / ram-work#9）
- **迭代**: translate-branch-title-go-native
- **作者**: claude
- **设计日期**: 2026-07-20 15:55
- **设计文档**: `docs/superpowers/specs/2026-07-20-translate-branch-title-go-native-design.md`
- **变更文件**:
  - 🆕 `v42-application-integration-20260720-1555-claude.puml` (基于 v41)
  - 🆕 伴生格式: `.archimate` + `.mermaid.md`
  - 📝 `api-route-to-owner.md` — Django internal translate 标 DEPRECATED
- **变更明细**:
  - 🟡 [MODIFIED] taskProjectService — 直连 fanyi_agent，去掉 djangoPost
  - 🔴 [DEPRECATED] saas-backend `/api/internal/taskproject/translate-branch-title/`
  - 🟢 [NEW] 出站依赖 DeepSeek（conf `ai_agent_config.fanyi_agent`）
  - 🎯 [NEW] Plateau v42 — translate-go-native
  - ✅ [CLOSES] Gap: 中文翻译仍依赖缺失的 Django internal
- **视图连线验证**:
  - [x] 架构变迁视图：Plateau v41 → Gap；WP → Gap / Plateau v42
  - [x] 目标拓扑视图：Vue→GW→TPS→fanyi；REMOVED TPS→Django internal

---

## v41 ✅ current — 任务子树状态展示与终态门禁

- **状态**: ✅ current（已交付；application-integration tip 待 v42 ship 后归档）
- **迭代**: task-subtree-status-and-terminal-gate
- **作者**: claude
- **设计日期**: 2026-07-19 18:16
- **交付日期**: 2026-07-19
- **设计文档**: `docs/superpowers/specs/2026-07-19-task-subtree-status-and-terminal-gate-design.md`
- **变更文件**:
  - 🆕 `v41-application-integration-20260719-1816-claude.puml` (基于 v40)
  - 🆕 伴生格式: `.archimate` + `.mermaid.md`
- **变更明细**:
  - 🟡 [MODIFIED] taskTaskService — `GET .../subtree/`、`DescendantTerminalGate`
  - 🟡 [MODIFIED] Vue 任务详情 — 下级交付物状态面板 + 409 错误展示
  - 🎯 [NEW] Plateau v41 — task-subtree-status-and-terminal-gate
  - ✅ [CLOSES] Gap: 详情无子树状态 / 终态无子树门禁
- **说明**: 基于 v40；无新微服务；终态判定对齐 taskEvents.ResolveTerminalKind
- **视图连线验证**:
  - [x] 架构变迁视图：Plateau v40 → Gap；WP → Gap / Plateau v41
  - [x] 目标拓扑视图：Vue→GW→TTS→DB；TTS→Django columns；TTS→Kafka→TE

---

## v40 📦 archived — 顶层交付物排队自动执行调度节奏

- **状态**: 📦 archived（已被 v41 取代为 application-integration tip current）
- **迭代**: top-deliverable-queued-auto-run-schedule
- **作者**: claude
- **设计日期**: 2026-07-19 17:46
- **交付日期**: 2026-07-19
- **设计文档**: `docs/superpowers/specs/2026-07-19-top-deliverable-queued-auto-run-schedule-design.md`
- **变更文件**:
  - 🆕 `v40-application-integration-20260719-1746-claude.puml` (基于 v39)
  - 🆕 伴生格式: `.archimate` + `.mermaid.md`
- **变更明细**:
  - 🟡 [MODIFIED] taskTaskService — ScheduleRhythm、queued_auto_run、Dispatcher
  - 🟡 [MODIFIED] taskCloudService — `cloud_server_configs.started_via`
  - 🟡 [MODIFIED] Vue 任务详情 — 节奏表单 / 排队开关 / deferred 徽章
  - 🟢 [NEW] `queued_auto_run_memberships`
  - 🎯 [NEW] Plateau v40 — queued-auto-run-schedule
  - 🎯 [OPENS] Gap: 无顶层子树时段排队与隔离并发
- **说明**: 基于 v39；无新微服务；排队额度与立即 auto_run/手动隔离
- **视图连线验证**:
  - [x] 架构变迁视图：Plateau v39 → Gap；WP → Gap / Plateau v40
  - [x] 目标拓扑视图：Vue→GW→TTS→DB/QARM→Cloud；TTS→Kafka→TE

---

## v39 📦 archived — 人员组织 API 迁 taskTenantService

- **状态**: 📦 archived（已被 v40 取代为 application-integration tip current）
- **迭代**: people-member-group-go-migration
- **作者**: claude
- **设计日期**: 2026-07-19 05:10
- **交付日期**: 2026-07-19
- **设计文档**: `docs/superpowers/specs/2026-07-19-people-member-group-go-migration-design.md`
- **收口文档**: `docs/superpowers/specs/2026-07-19-people-member-group-go-migration-closeout-design.md`
- **变更文件**:
  - 🆕 `v39-application-integration-20260719-0510-claude.puml`（现为 archived）
  - 🆕 伴生格式: `.archimate` + `.mermaid.md`
  - 📦 `v38-application-integration-20260718-2340-claude.puml` → archived
- **变更明细**:
  - 🟢 [NEW] taskTenantService（:8020）公网 members/invite/groups
  - 🟢 [NEW] `data/task_tenant.db` 四表 owner
  - 🟡 [MODIFIED] task-gateway 路由切流；Django 卸公网 ViewSet
  - 🟡 [MODIFIED] Django 消费方成员门禁统一 `tenant_client.is_active_member`（string-safe）
  - 🎯 [NEW] Plateau v39 — people-member-group-go
  - ✅ [CLOSES] Gap: 人员组织公网仍 Django
- **说明**: 基于 v38；新建微服务；`accounts_company` 仍归 saas-backend
- **视图连线验证**:
  - [x] 架构变迁视图：Plateau v38 → Gap；WP → Gap / Plateau v39
  - [x] 目标拓扑视图：Vue→GW→TTS→DB；TTS→Django/TPS/Kafka

---

## v38 📦 archived — daydaymoney.yaml 元信息全链路

- **状态**: 📦 archived（已被 v39 取代为 application-integration tip current）
- **迭代**: daydaymoney-yaml-metadata
- **作者**: claude
- **设计日期**: 2026-07-18 23:40
- **交付日期**: 2026-07-19
- **设计文档**: `docs/superpowers/specs/2026-07-18-daydaymoney-yaml-metadata-design.md`
- **变更文件**:
  - 🆕 `v38-application-integration-20260718-2340-claude.puml`（现为 archived）
  - 🆕 伴生格式: `.archimate` + `.mermaid.md`
  - 📦 `v37-application-integration-20260718-1810-claude.puml` → archived
- **变更明细**:
  - 🟢 [NEW] 各仓库 `daydaymoney.yaml`（`service_id` + `tags`，禁止固化 ws/project id）
  - 🟢 [NEW] taskProjectService `GET .../daydaymoney/resolve` + `POST .../daydaymoney/parse-yaml`；`projects/?tag=`
  - 🟡 [MODIFIED] Vue 项目页同步 tags；SPA head `daydaymoney-*` meta
  - 🟡 [MODIFIED] taskChromePlugin 浮窗按 meta 反查并自动填充（一对多）
  - 🟡 [MODIFIED] tracelog / 结构化日志注入 `daydaymoney_service_id` / `daydaymoney_tags`
  - 🟡 [MODIFIED] DaydaymoneyGrafana 优先按日志元信息精确匹配项目 tags
  - 🎯 [NEW] Plateau v38 — daydaymoney-yaml-metadata
  - ✅ [CLOSES] Gap: 无稳定服务身份与多归属反查
- **说明**: 基于 v37；无新微服务；扩展 taskProjectService + 共享解析/日志
- **视图连线验证**:
  - [x] 架构变迁视图：Plateau v37 → Gap；WP → Gap / Plateau v38
  - [x] 目标拓扑视图：YAML→Head/Project/tracelog；Chrome/Grafana→GW→TPS→tags；Loki→Grafana

---

## v37 📦 archived — Work Panel 任务状态 SSE

- **状态**: 📦 archived（已被 v38 取代；application-integration tip 现为 v39）
- **迭代**: work-panel-task-status-sse
- **作者**: claude
- **设计日期**: 2026-07-18 18:10
- **交付日期**: 2026-07-18
- **设计文档**: `docs/superpowers/specs/2026-07-18-work-panel-task-status-sse-design.md`
- **变更文件**:
  - 🆕 `v37-application-integration-20260718-1810-claude.puml`（现为 current）
  - 🆕 伴生格式: `.archimate` + `.mermaid.md`
  - 📦 `v36-application-integration-20260718-1605-claude.puml` → archived
- **变更明细**:
  - 🟢 [NEW] taskSSE `GET .../work-panel-events-sse/`，hub `workspace:{ws}`
  - 🟢 [NEW] taskEvents intent `2_fanout_work_panel_sse`（TASK_STATUS_CHANGED → Redis）
  - 🟡 [MODIFIED] Vue WorkPanel：初载拉取后订阅 SSE，增量 patch 卡片
  - 🎯 [NEW] Plateau v37 — work-panel-task-status-sse
  - ✅ [CLOSES] Gap: 看板无任务状态推送
- **说明**: 基于 v36；无新微服务；复用 taskSSE/Redis/Kafka
- **视图连线验证**:
  - [x] 架构变迁视图：Plateau v36 → Gap；WP → Gap / Plateau v37
  - [x] 目标拓扑视图：Vue→GW→TTS/Kafka→TE→Redis→SSE→Vue

---

## v36 📦 archived — 租户 GitLab 设置：内建资源购买 + 自建连接

- **状态**: 📦 archived（已被 v37 取代为 application-integration tip current）
- **迭代**: tenant-gitlab-settings-resource-purchase
- **作者**: claude
- **设计日期**: 2026-07-18 16:05
- **交付日期**: 2026-07-18
- **设计文档**: `docs/superpowers/specs/2026-07-18-tenant-gitlab-settings-resource-purchase-design.md`
- **变更文件**:
  - 🆕 `v36-application-integration-20260718-1605-claude.puml`（现为 current）
  - 🆕 伴生格式: `.archimate` + `.mermaid.md`
  - 📦 `v35-application-integration-20260718-0100-claude.puml` → archived
- **变更明细**:
  - 🟢 [NEW] taskBill `GET/POST .../billing/gitlab-resources/` + 表 `billing_tenant_gitlab_resource`
  - 🟡 [MODIFIED] Vue `WorkspaceSettingsGitlabConnection` 双区块；Sidebar「GitLab」
  - 🎯 [NEW] Plateau v36 — gitlab-resource-purchase
  - ✅ [CLOSES] Gap: 无内建 GitLab 配额购买入口
- **说明**: 基于 v35；扩展既有 billing 代理，无新微服务；自建连接仍归 taskGitOauth
- **视图连线验证**:
  - [x] 架构变迁视图：Plateau v35 → Gap；WP → Gap / Plateau v36
  - [x] 目标拓扑视图：Vue→GW→billing_bridge→taskBill→resource 表；Vue→GW→taskGitOauth→conn 表

---

## v35 📦 archived — 创建任务可选字段工作区显隐

- **状态**: 📦 archived（已被 v36 取代为 application-integration tip current）
- **迭代**: create-task-field-settings
- **作者**: claude
- **设计日期**: 2026-07-18 01:00
- **交付日期**: 2026-07-18
- **设计文档**: `docs/superpowers/specs/2026-07-18-create-task-field-settings-design.md`
- **变更文件**:
  - 🆕 `v35-application-integration-20260718-0100-claude.puml`（现为 current）
  - 🆕 伴生格式: `.archimate` + `.mermaid.md`
  - 📦 `v34-application-integration-20260716-1430-claude.puml` → archived
- **变更明细**:
  - 🟢 [NEW] taskProjectService `GET/PUT .../create-task-field-settings/` + 表 `workspace_create_task_field_settings`
  - 🟡 [MODIFIED] Vue settings/task-panel「创建字段」配置；WorkPanel CreateTaskModal 按配置显隐
  - 🎯 [NEW] Plateau v35 — create-task-field-settings
  - ✅ [CLOSES] Gap: 创建任务可选字段无法按工作区显隐
- **说明**: 基于 v34；扩展既有 workspaces 通配 path，无新微服务
- **视图连线验证**:
  - [x] 架构变迁视图：Plateau v34 → Gap；WP → Gap / Plateau v35
  - [x] 目标拓扑视图：Settings/Work→GW→Project→settings 表

---

## v34 📦 archived — 项目详情内部仓磁盘占用

- **状态**: 📦 archived（已被 v35 取代为 application-integration tip current）
- **迭代**: project-internal-repo-disk-size
- **作者**: claude
- **设计日期**: 2026-07-16 14:30
- **交付日期**: 2026-07-16
- **设计文档**: `docs/superpowers/specs/2026-07-16-project-internal-repo-disk-size-design.md`
- **变更文件**:
  - 🆕 `v34-application-integration-20260716-1430-claude.puml`（现为 current）
  - 🆕 伴生格式: `.archimate` + `.mermaid.md`
  - 📦 `v33-application-integration-20260716-1145-claude.puml` → archived
- **变更明细**:
  - 🟡 [MODIFIED] taskProjectService GET project：enrich `is_internal` + `disk_size_bytes`（60s 短缓存）
  - 🟡 [MODIFIED] Vue ProjectDetailGitReposSection：内部仓磁盘徽章
  - 🟢 [NEW] GitLab `?statistics=true` 只读调用（仅 gitlab-local）
  - 🎯 [NEW] Plateau v34 — project-internal-repo-disk-size
  - ✅ [CLOSES] Gap: 详情页不展示内部仓磁盘占用
- **说明**: 基于 v33；无新微服务/新表/新公网 path
- **视图连线验证**:
  - [x] 架构变迁视图：Plateau v33 → Gap；WP → Gap / Plateau v34
  - [x] 目标拓扑视图：Vue→GW→Project→GitLab statistics；仅内部仓

---

## v33 📦 archived — 终态优雅通知容器收尾与回调释放

- **状态**: 📦 archived（已被 v34 取代为 application-integration tip current）
- **迭代**: terminal-graceful-container-shutdown
- **作者**: claude
- **设计日期**: 2026-07-16 11:45
- **交付日期**: 2026-07-16
- **设计文档**: `docs/superpowers/specs/2026-07-16-terminal-graceful-container-shutdown-design.md`
- **变更文件**:
  - 🆕 `v33-application-integration-20260716-1145-claude.puml`（已 archived）
  - 🆕 伴生格式: `.archimate` + `.mermaid.md`
  - 📦 `v32-application-integration-20260716-0013-claude.puml` → archived
- **变更明细**:
  - 🟢 [NEW] onlineServiceJS `POST /api/task-lifecycle/shutdown`
  - 🟢 [NEW] Cloud inbound `request-machine-release` + SoleContainerGate
  - 🟢 [NEW] `TASK_GRACEFUL_SHUTDOWN_AWAIT` 补偿硬释放（:18047）
  - 🟡 [MODIFIED] taskEvents 终态：先 notify，成功则延迟硬释放
  - 🟡 [MODIFIED] CGW L0 `container-task-lifecycle-shutdown`
  - 🎯 [NEW] Plateau v33 — terminal-graceful-container-shutdown
  - ✅ [CLOSES] Gap: 终态硬停无容器收尾窗口；释放非容器主动
- **说明**: 基于 v32；扩展 v27 硬释放语义为优雅主路径 + 超时补偿
- **视图连线验证**:
  - [x] 架构变迁视图：Plateau v32 → Gap；WP → Gap / Plateau v33
  - [x] 目标拓扑视图：Events→OSJS shutdown；OSJS→Cloud release；sole gate

---

## v32 📦 archived — ai-provider Python→Go（taskAiProvider）

- **状态**: 📦 archived（已被 v33 取代为 application-integration tip current）
- **迭代**: ai-provider-go-migration
- **作者**: claude
- **设计日期**: 2026-07-16 00:13
- **交付日期**: 2026-07-16
- **设计文档**: `docs/superpowers/specs/2026-07-16-ai-provider-python-to-go-migration-design.md`
- **变更文件**:
  - 📦 `v32-application-integration-20260716-0013-claude.puml` → archived
  - 📦 伴生格式: `.archimate` + `.mermaid.md`
  - 📦 `v31-application-integration-20260715-2040-claude.puml` → archived
- **变更明细**:
  - 🟢 [NEW] `taskAiProvider` Go 服务 :8010（契约兼容原 Django Saas_Ai_Provider）
  - 🟡 [MODIFIED] runAll `ai-provider` working_dir → taskAiProvider
  - 🔴 [DEPRECATED] Django `task2app/Saas_Ai_Provider` Python 实现
  - 🎯 [NEW] Plateau v32 — ai-provider-go-migration
  - ✅ [CLOSES] Gap: ai-provider 仍为 Django，无法按 Go-first 清理 Python
- **说明**: 基于 v31 current；同库 `db/ai-provider`；废止 2026-07-05「ai-provider 不迁 Go」Non-Goal
- **视图连线验证**:
  - [x] 架构变迁视图：Plateau v31 → Gap；WP → Gap / Plateau v32
  - [x] 目标拓扑视图：Vue/GW→taskAiProvider→taskAuth/Cloud/DB/OCI

---

## v31 📦 archived — OTP/SMS 全切 Go（原生云短信）

- **状态**: 📦 archived（已被 v32 取代为 application-integration tip current）
- **迭代**: otp-go-full-native-sms
- **作者**: claude
- **设计日期**: 2026-07-15 20:40
- **交付日期**: 2026-07-15
- **设计文档**: `docs/superpowers/specs/2026-07-15-otp-go-full-native-sms-design.md`
- **变更文件**:
  - 🆕 `v31-application-integration-20260715-2040-claude.puml`
  - 🆕 伴生格式: `.archimate` + `.mermaid.md`
- **变更明细**:
  - 🟢 [NEW] taskAuth → Aliyun/Tencent SendSms（HTTP 签名，无 Django 桥）
  - 🟢 [NEW] 密码重置手机码存发验本地
  - 🟡 [MODIFIED] phone+code 登录 → enrich-login（废弃 OTP forward-login）
  - 🔴 [DEPRECATED] Go 发码路径对 `dispatch-sms` 依赖
  - 🎯 [NEW] Plateau v31 — otp-go-full-native-sms
  - ✅ [CLOSES] Gap: 云短信桥接 + OTP forward-login + 重置码分裂
- **说明**: 充值公网路由仍 Django，OTP 真源与短信通道均为 Go
- **视图连线验证**:
  - [x] 架构变迁视图：Plateau v29 → Gap；WP → Gap / Plateau v31
  - [x] 目标拓扑视图：Vue→GW→taskAuth→SMS/enrich/DB

---

## v30 ✅ shipped — 租户级自建 GitLab OAuth 连接

- **状态**: ✅ shipped（已交付；application-integration 视图 tip current 仍为 **v31**）
- **迭代**: tenant-gitlab-oauth-connection
- **作者**: claude
- **设计日期**: 2026-07-15 20:14
- **交付日期**: 2026-07-15 22:44
- **设计文档**: `docs/superpowers/specs/2026-07-15-tenant-gitlab-oauth-connection-design.md`
- **变更文件**:
  - 📦 `v30-application-integration-20260715-2014-claude.puml` → archived（已交付并行切片）
  - 📦 伴生格式: `.archimate` + `.mermaid.md`
  - 📦 `v29-application-integration-20260715-1736-claude.puml` → archived（纠正与 v31 双 current）
- **变更明细**:
  - 🟢 [NEW] `tenant_gitlab_oauth_connections`（taskGitOauth / git-oauth DB）
  - 🟢 [NEW] 租户 CRUD `/api/tenant/{tid}/gitlab-oauth-connection/`
  - 🟡 [MODIFIED] taskGitOauth Resolve + Vue 设置页 / providers 合并
  - 🎯 [NEW] Plateau v30 — tenant-gitlab-oauth-connection
  - ✅ [CLOSES] Gap: 无租户自建 GitLab OAuth App 产品配置
- **说明**: 基于 v29；与 v31（OTP）并行交付。因 v31 已是同视图 tip current，v30 固化为 archived（shipped），不覆盖 v31。
- **视图连线验证**:
  - [x] 架构变迁视图：Plateau v29 → Gap；WP → Gap / Plateau v30
  - [x] 目标拓扑视图：Vue→GW→taskGitOauth→DB/GitLab/Kafka；Django providers merge
  - [x] 交付固化：清理 Change Legend / `[NEW|MODIFIED v30]`；Archi load OK

---


## v29 📦 archived — gitOauth Python→Go（taskGitOauth）

- **状态**: 📦 archived（已被 v31 取代为 application-integration tip current；v30 并行切片亦已 shipped）
- **迭代**: gitoauth-go-migration
- **作者**: claude
- **设计日期**: 2026-07-15 17:36
- **交付日期**: 2026-07-15 18:31
- **设计文档**: `docs/superpowers/specs/2026-07-15-gitoauth-python-to-go-migration-design.md`
- **变更文件**:
  - 🆕 `v29-application-integration-20260715-1736-claude.puml`
  - 🆕 伴生格式: `.archimate` + `.mermaid.md`
- **变更明细**:
  - 🟢 [NEW] `taskGitOauth` Go 服务 :8002（契约兼容原 Django gitOauth）
  - 🟡 [MODIFIED] runAll `git-oauth` working_dir → taskGitOauth
  - 🔴 [DEPRECATED] Django `gitOauth/` Python 实现
  - 🎯 [NEW] Plateau v29 — gitoauth-go-migration
  - ✅ [CLOSES] Gap: gitOauth 仍为 Django，无法按 Go-first 清理 Python
- **说明**: 基于 v27 current；同库 `db/git-oauth`；废止 2026-07-05「git-oauth 不迁 Go」Non-Goal
- **视图连线验证**:
  - [x] 架构变迁视图：Plateau v27 → Gap；WP → Gap / Plateau v29
  - [x] 目标拓扑视图：Vue→GW→taskGitOauth→Provider/bind/DB

---

## v28 🎯 target — 项目详情子 Git 仓库发现

- **状态**: 🎯 target（已设计，PR 交付中）
- **迭代**: project-nested-git-repos
- **作者**: claude
- **设计日期**: 2026-07-15 11:35
- **交付日期**: —
- **设计文档**: `docs/superpowers/specs/2026-07-15-project-nested-git-repos-design.md`
- **变更文件**:
  - 🆕 `v28-application-integration-20260715-1135-claude.puml`
  - 🆕 伴生格式: `.archimate` + `.mermaid.md`
- **变更明细**:
  - 🟢 [NEW] `GET .../projects/{id}/nested-git-repos/`（taskProjectService 只读）
  - 🟡 [MODIFIED] Vue ProjectDetail Git 区块展示子仓列表
  - 🟡 [MODIFIED] Rel_Flow：ProjectService → GitLab/GitHub raw `.gitignore`/`.gitmodules`
  - 🎯 [NEW] Plateau v28 — project-nested-git-repos
  - ✅ [CLOSES] Gap: 元仓项目详情无法直接看到嵌套子 Git 仓库
- **说明**: 基于 v27 current；无新微服务/无新表；零 Python 公网接口；纯查询无领域事件
- **视图连线验证**:
  - [x] 架构变迁视图：Plateau v27 → Gap；WP → Gap / Plateau v28
  - [x] 目标拓扑视图：Vue→GW→taskProjectService→GitLab/GitHub

---

## v27 ✅ current — 任务终态硬释放 + 镜像容器迁移

- **状态**: ✅ current（已交付）
- **迭代**: terminal-hard-release-container-migrate
- **作者**: claude
- **设计日期**: 2026-07-15 10:10
- **交付日期**: 2026-07-15 10:30
- **设计文档**: `docs/superpowers/specs/2026-07-15-terminal-hard-release-container-migrate-design.md`
- **变更文件**:
  - 🆕 `v27-application-integration-20260715-1010-claude.puml`（现为 current）
  - 🆕 伴生格式: `.archimate` + `.mermaid.md`
  - 📦 `v25-application-integration-20260714-2055-claude.puml` → archived
- **变更明细**:
  - 🟢 [NEW] Constraint 终态硬释放禁 reuse；CSC.`terminal_released`；internal bindings/migrate/mark APIs
  - 🟡 [MODIFIED] taskEvents 终态 intent：先 migrate 兄弟 busy 容器再 CLOUD_SERVER_STOPPED
  - 🟡 [MODIFIED] idle reuse 解绑 source；排除 terminal_released
  - 🟡 [MODIFIED] 真实 ECS migrate → start-vm-auto；mock/reuse 后 gateway 按原镜像拉起容器
  - 🎯 [NEW] Plateau v27 — terminal-hard-release-container-migrate
  - ✅ [CLOSES] Gap: 终态节点仍可 reuse；共享实例误杀外任务容器；无迁回所属任务
- **说明**: 基于 v25（现 archived）；与 v26 并行 target 仍待交付；零 Python 新接口
- **视图连线验证**:
  - [x] 架构变迁视图：Plateau v25 → Gap；WP → Gap / Plateau v27；均有 `sourceConnection`
  - [x] 目标拓扑视图：Vue→GW→Task→Kafka→Events→Cloud/CGW→ECS/Docker；Constraint 影响 Cloud/Events；均有连线

---

## v26 🎯 target — 容器镜像 @ 模式 + 统一评论区

- **状态**: 🎯 target（已设计，待交付）
- **迭代**: container-image-at-mention
- **作者**: claude
- **设计日期**: 2026-07-15 00:10
- **交付日期**: —
- **设计文档**: `docs/superpowers/specs/2026-07-14-container-image-at-mention-design.md`
- **变更文件**:
  - 🆕 `v26-application-integration-20260715-0010-claude.puml`
  - 🆕 伴生格式: `.archimate` + `.mermaid.md`
- **变更明细**:
  - 🟢 [NEW] `workspaces.container_image_at_mode_enabled`；`container_agent_comments`；Constraint `@模式默认关闭`
  - 🟡 [MODIFIED] Vue 统一评论区 + TaskPanel 开关；taskProjectService；taskTaskService mentions；taskAIComment 编排/SSE；onlineServiceJS ContextPack
  - 🟡 [MODIFIED] 复用 taskCloudService start-vm + prefer_idle_reuse（无新 path）
  - 🎯 [NEW] Plateau v26 — container-image-at-mention
  - ✅ [CLOSES] Gap: 无@镜像开关/触发/Agent评论SSE/统一评论区
- **说明**: 基于 v25 current；与 v22/v23/v24 并行 target；无新微服务；零 Python 新接口
- **视图连线验证**:
  - [x] 架构变迁视图：Plateau v25 → Gap；WP → Gap / Plateau v26；均有 `sourceConnection`
  - [x] 目标拓扑视图：Vue→GW→Project/Task→Kafka→AIComment→Cloud/OSJS→SSE；均有连线

---

## v25 📦 archived — 自动安全组入网白名单

- **状态**: 📦 archived（已被 v27 取代为 application-integration current）
- **迭代**: auto-sg-ingress-whitelist
- **作者**: claude
- **设计日期**: 2026-07-14 20:55
- **交付日期**: 2026-07-14 21:15
- **设计文档**: `docs/superpowers/specs/2026-07-14-auto-sg-ingress-whitelist-design.md`
- **变更文件**:
  - 📦 `v25-application-integration-20260714-2055-claude.puml` → archived（v27 交付时）

---

## v24 🎯 target — 导航栏多账号切换

- **状态**: 🎯 target（goal-mode 已批准设计，实现中）
- **迭代**: navbar-multi-account-switcher
- **作者**: claude
- **设计日期**: 2026-07-14 15:26
- **交付日期**: —
- **设计文档**: `docs/superpowers/specs/2026-07-14-navbar-multi-account-switcher-design.md`
- **变更文件**:
  - 🆕 `v24-application-integration-20260714-1526-claude.puml`
  - 🆕 伴生格式: `.archimate` + `.mermaid.md`
- **变更明细**:
  - 🟢 [NEW] taskAuth `POST /api/accounts/users/activate-session/`
  - 🟢 [NEW] Vue `savedAccounts` 槽 + AccountSwitcher 下拉
  - 🟡 [MODIFIED] Navbar / Login
  - 🎯 [NEW] Plateau v24 — multi-account switcher
  - ✅ [CLOSES] Gap: 无多账号切换；昵称不可下拉
- **说明**: 与 v22/v23 并行 target；基于 v21 current
- **视图连线验证**:
  - [x] 架构变迁视图：Plateau v21 → Gap；WP → Gap / Plateau v24；均有 `sourceConnection`
  - [x] 目标拓扑视图：Vue→GW→taskAuth→Django；Vue↔slots；taskAuth→tokens；均有连线
  - [x] Archi CLI：`--loadModel …v24-….archimate`（**必须带 `.archimate` 后缀**）→ `Loaded model: '…v24 Target'`；校验脚本 `architecture/scripts/verify-archimate-load.sh`

---

## v23 🎯 target — autoRunStep.md 镜像说明抽取与展示

- **状态**: 🎯 target（goal-mode 已批准设计，实现中）
- **迭代**: auto-run-steps-md
- **作者**: claude
- **设计日期**: 2026-07-14 14:26
- **交付日期**: —
- **设计文档**: `docs/superpowers/specs/2026-07-14-auto-run-steps-md-design.md`
- **变更文件**:
  - 🆕 `v23-application-integration-20260714-1426-claude.puml`
  - 🆕 伴生格式: `.archimate` + `.mermaid.md`
- **变更明细**:
  - 🟢 [NEW] `/app/autoRunStep.md` + onlineServiceJS `GET /api/auto-run-steps`
  - 🟢 [NEW] taskCloudService `POST /api/internal/extract-auto-run-steps/`
  - 🟡 [MODIFIED] ai-provider catalog / vendor-development-catalog 字段（含开发中）
  - 🟡 [MODIFIED] tenant_installed_images 快照字段
  - 🟡 [MODIFIED] Vue ImageMarket / CreateTask / TaskDetail
  - 🎯 [NEW] Plateau v23 — autoRunStep.md
  - ✅ [CLOSES] Gap: 无 autoRun 说明；容器未起不可展示
- **说明**: 与 v22（微信支付 target）并行；基于 v21 current
- **视图连线验证**:
  - [x] 架构变迁视图：Plateau v21 → Gap；WP → Gap / Plateau v23；均有 `sourceConnection`
  - [x] 目标拓扑视图：Vue→GW→Cloud↔AIP；Vue→CGW→OSJS→md；均有连线

---

## v22 🎯 target — 租户充值页接入微信支付 Native 扫码

- **状态**: 🎯 target（已设计，待交付）
- **迭代**: wechat-pay-recharge
- **作者**: claude
- **设计日期**: 2026-07-14 02:30
- **交付日期**: —
- **设计文档**: `docs/superpowers/specs/2026-07-14-wechat-pay-recharge-design.md`
- **变更文件**:
  - 🆕 `v22-application-integration-20260714-0230-claude.puml`
  - 🆕 伴生格式: `.archimate` + `.mermaid.md`
- **变更明细**:
  - 🟢 [NEW] WeChatPay 外部系统 — Native 预下单 + 支付回调
  - 🟡 [MODIFIED] taskBill — Native prepay / notify / mock-complete + wechatpay-go 公钥验签
  - 🟡 [MODIFIED] Django billing_bridge — SMS 门禁薄编排 `recharge_wechat_*` + notify 代理
  - 🟡 [MODIFIED] Vue BillingRecharge — 支付方式选择 + 扫码弹层 + 状态轮询
  - 🎯 [NEW] Plateau v22 — WeChat Pay Native 充值
  - ✅ [CLOSES] Gap: 充值页仅 PayPal；无微信 Native 扫码通道
- **视图连线验证**:
  - [ ] 架构变迁视图：Plateau v21 → Gap；WP → Gap / Plateau v22；Plateau→taskBill Composition 均有 `sourceConnection`
  - [ ] 目标拓扑视图：Vue→Django→taskBill→WeChatPay；WeChatPay→taskBill/Django notify；taskBill→pending/txn 均有连线
  - [ ] 架构变迁链路可追踪（v21→Gap→WP→v22）
  - [ ] Archi CLI `--loadModel` 验证

---

## v21 ✅ shipped — cloud_server_events / IAM 迁入 taskCloudService

- **状态**: ✅ shipped（本会话交付）
- **迭代**: cloud-events-task-cloud-migration
- **作者**: claude
- **设计日期**: 2026-07-14 01:30
- **交付日期**: 2026-07-14
- **设计文档**: `docs/superpowers/specs/2026-07-14-cloud-events-task-cloud-migration-design.md`
- **部署清单**: `docs/intents/backend/cloud_domain_tables_task_cloud_DEPLOY.md`
- **变更文件**:
  - 🆕 `v21-application-integration-20260714-0130-claude.puml`（现为 current）
  - 🆕 伴生格式: `.archimate` + `.mermaid.md`
  - 📦 `v20-...puml` `@status` → archived
- **变更明细**:
  - 🟢 [NEW] `cloud_server_events` / `access_key_iam_associations`（task_cloud.db）
  - 🟡 [MODIFIED] taskCloudService — finalize 本地持久化、native status poll、internal APIs
  - 🟡 [MODIFIED] taskEvents — 事件/IAM 改打 Cloud internal HTTP
  - 🔴 [DEPRECATED] Django persist-start-vm-auto / delete-start-vm-event；start-server / server-startup-status → 410；cloud.0052 DROP
  - 🎯 [NEW] Plateau v21 — cloud events/IAM on task_cloud
  - ✅ [CLOSES] Gap: events/IAM 仍在 saas；finalize 经 Django persist
- **视图连线验证**:
  - [x] 架构变迁视图：Plateau v20 → Gap；WP → Gap / Plateau v21；Plateau→Cloud Composition 均有 `sourceConnection`
  - [x] 目标拓扑视图：Vue→GW→Cloud→Cred/Bill/EV/CFG/TE/IAM；TE↔Cloud；Django DEPRECATED→Cloud 均有连线
  - [x] 架构变迁链路可追踪（v20→Gap→WP→v21）
  - [x] Archi CLI `--loadModel`：`Loaded model: 'AI Dev Platform — Application Integration v21 Target'`（exit 0，2026-07-14）
  - [x] 本机 DEPLOY 清单全项勾选：`docs/intents/backend/cloud_domain_tables_task_cloud_DEPLOY.md` + `_deploy_evidence_20260714/`

---

## v20 ✅ shipped — 工作面板过滤选项持久化（Go）

- **状态**: ✅ shipped（本会话交付）
- **迭代**: work-panel-filter-persistence
- **作者**: claude
- **设计日期**: 2026-07-13 20:05
- **交付日期**: 2026-07-13
- **设计文档**: `docs/superpowers/specs/2026-07-13-work-panel-filter-persistence-design.md`
- **变更文件**:
  - 🆕 `v20-application-integration-20260713-2005-claude.puml`（现为 current）
  - 🆕 伴生格式: `.archimate` + `.mermaid.md`
  - ⚠️ 早期 Django 草稿 `...1956...puml` 已归档为 `.archived-django-draft`（用户要求落 Go）
- **变更明细**:
  - 🟢 [NEW] `user_workspace_work_panel_filters` DataObject（task_project.db）
  - 🟡 [MODIFIED] taskProjectService — GET/PUT work-panel-filters + OpenAPI
  - 🟡 [MODIFIED] Vue WorkPanel — init GET / debounce PUT / 切空间加载
  - 🎯 [NEW] Plateau v20 — work-panel filter persistence
  - ✅ [CLOSES] Gap: 过滤栏仅内存，刷新/切空间丢失

---

## v19 ✅ shipped — 工作空间机器节点闲置策略

- **状态**: ✅ shipped（本会话交付）
- **迭代**: workspace-machine-idle-policy
- **作者**: claude
- **设计日期**: 2026-07-13 15:37
- **交付日期**: 2026-07-13
- **设计文档**: `docs/superpowers/specs/2026-07-13-workspace-machine-idle-policy-design.md`
- **变更文件**:
  - 🆕 `v19-application-integration-20260713-1537-claude.puml`（现为 current）
  - 🆕 伴生格式: `.archimate` + `.mermaid.md`
- **变更明细**:
  - 🟢 [NEW] `workspace_machine_policies` DataObject
  - 🟢 [NEW] taskCloudService idle recycle ticker + internal POST
  - 🟡 [MODIFIED] taskCloudService — policy/summary/start-vm 门禁与闲置复用
  - 🟡 [MODIFIED] Vue WorkPanel / WorkspaceSettingsTaskPanel
  - 🟡 [MODIFIED] `cloud_server_configs.idle_since`
  - 🎯 [NEW] Plateau v19 — workspace machine idle policy
  - ✅ [CLOSES] Gap: 无 workspace 级启用节点/闲置回收/优先复用/面板汇总

---

## v17 ✅ shipped — 任务状态变更事件与终态释放服务器

- **状态**: ✅ shipped（本会话交付）
- **迭代**: task-status-changed-release-servers
- **作者**: claude
- **设计日期**: 2026-07-13 14:28
- **交付日期**: 2026-07-13
- **设计文档**: `docs/superpowers/specs/2026-07-13-task-status-changed-release-servers-design.md`
- **变更文件**:
  - 🆕 `v17-application-integration-20260713-1428-claude.puml`
  - 🆕 伴生格式: `.archimate` + `.mermaid.md`
- **变更明细**:
  - 🟢 [NEW] Kafka `TASK_STATUS_CHANGED` / topic `task-status-changed`
  - 🟢 [NEW] taskEvents intent `task_status_changed/1_release_servers_on_terminal` (:18043)
  - 🟡 [MODIFIED] taskTaskService — 状态变化后发事件
  - 🟡 [MODIFIED] 复用 CLOUD_SERVER_STOPPED / relay·mock stop 编排释放
  - 🎯 [NEW] Plateau v17 — task terminal release servers
  - ✅ [CLOSES] Gap: 任务终态不发事件、不释放服务器

---

## v18 🎯 target — auto_run 首指令与自动交付

- **状态**: 🎯 target（已设计，待交付）
- **迭代**: auto_run 首指令与自动交付
- **作者**: claude
- **设计日期**: 2026-07-13 14:27
- **交付日期**: —
- **设计文档**: `docs/superpowers/specs/2026-07-13-auto-run-first-instruction-and-delivery-design.md`
- **变更文件**:
  - 🆕 `v18-application-integration-20260713-1427-claude.puml`
  - 🆕 伴生格式: `.archimate` + `.mermaid.md`
- **变更明细**:
  - 🟡 [MODIFIED] taskCredentialService — task-detail 增 `auto_run` + `repo_git_identities`
  - 🟡 [MODIFIED] onlineServiceJS — bootstrap 后自动首指令 job；job 完成后 identities/commit/push/PR 交付
  - 🟡 [MODIFIED] machine_container.md §4.4
  - 🎯 [NEW] Plateau v18 — auto_run 容器闭环交付
  - ✅ [CLOSES] Gap: bootstrap 后无首指令；Agent 完成后无自动 commit/push/PR

---

## v16 🎯 target — 多入口同源 API（域名 + IP）

- **状态**: 🎯 target（已设计，待交付）
- **迭代**: multi-entry-same-origin-api
- **作者**: claude
- **设计日期**: 2026-07-11 17:26
- **交付日期**: —
- **设计文档**: `docs/superpowers/specs/2026-07-11-multi-entry-same-origin-api-design.md`
- **变更文件**:
  - 🆕 `v16-application-integration-20260711-1726-claude.puml`
  - 🆕 伴生格式: `.archimate` + `.mermaid.md`
- **变更明细**:
  - 🟢 [NEW] Edge nginx — 多入口 `/`→Vue、`/api/`→gateway
  - 🟡 [MODIFIED] Vue — `API_BASE_URL` 默认同源
  - 🟡 [MODIFIED] Vite — `/api` 代理恢复
  - 🟡 [MODIFIED] conf — `publicEntryOrigins` 白名单
  - 🔴 [DEPRECATED] 浏览器默认绝对 `http://IP:18081` API 基址
  - 🎯 [NEW] Plateau v16 — multi-entry same-origin API
  - ✅ [CLOSES] Gap: HTTPS 入口仍用 HTTP 绝对 gateway → Mixed Content

---

## v15 🎯 target — relayToTrae 启动拉取并运行所选镜像

- **状态**: 🎯 target（已设计并实现于本会话；待正式 ship 切换 current）
- **迭代**: relay 直接启动消费 TenantInstalledImage（docker pull/run）
- **作者**: claude
- **设计日期**: 2026-07-10 20:09
- **交付日期**: —
- **设计文档**: `docs/superpowers/specs/2026-07-10-relay-start-selected-image-design.md`
- **变更文件**:
  - 🆕 `v15-application-integration-20260710-2009-claude.puml`
  - 🆕 伴生格式: `.archimate` + `.mermaid.md`
- **变更明细**:
  - 🟡 [MODIFIED] Vue ServerConfig — start body 传 `installed_image_id`
  - 🟡 [MODIFIED] taskContainerGateway — resolve-image 后转发 `image` 至 go_relayToTrae
  - 🟡 [MODIFIED] go_relayToTrae — 有 image 时 docker pull/run；stop 停容器
  - 🟡 [MODIFIED] Django relay_to_trae_start — 兜底路径同样解析镜像
  - 🎯 [NEW] Plateau v15 — relay selected image runtime
  - ✅ [CLOSES] Gap: relay 启动忽略用户所选镜像

---

## v14 🎯 target — relayToTrae Token 生命周期云主机对齐

- **状态**: 🎯 target（已设计并实现于本会话；待正式 ship 切换 current）
- **迭代**: relayToTrae Token 生命周期与云主机 UserData 启动对齐
- **作者**: claude
- **设计日期**: 2026-07-10 19:10
- **交付日期**: —
- **设计文档**: `docs/superpowers/specs/2026-07-10-relay-token-cloud-parity-design.md`
- **变更文件**:
  - 🆕 `v14-application-integration-20260710-1910-claude.puml`
  - 🆕 伴生格式: `.archimate` + `.mermaid.md`
- **变更明细**:
  - 🟡 [MODIFIED] go_relayToTrae — 启动不再预换票 / 不再 TRAE_SKIP；等 child 落盘后 sync
  - 🟡 [MODIFIED] onlineServiceJS — `container_refresh_token.json` 含 access_token；模拟路径默认换票
  - 🔴 [DEPRECATED] 父进程首次 exchange + TRAE_SKIP 作为模拟默认
  - 🎯 [NEW] Plateau v14 — relay Token cloud parity
  - ✅ [CLOSES] Gap: mock start token lifecycle diverges from cloud VM

## v13 ✅ shipped — Container Exec Hot-Path Zero-Django

- **状态**: ✅ shipped（Phase A–F：热路径全清 + GitLab readiness + AI env agent + djangoInternalApiBase 删除）
- **迭代**: 容器执行热路径绕过 Django（validate/resolve 迁 taskAuth + taskCloudService）
- **作者**: claude
- **设计日期**: 2026-07-10 15:05
- **交付日期**: 2026-07-10
- **设计文档**: `docs/superpowers/specs/2026-07-10-container-exec-bypass-django-design.md`
- **变更文件**:
  - 🆕 `v13-application-integration-20260710-1505-claude.puml`
  - 🆕 伴生格式: `.archimate` + `.mermaid.md`
- **变更明细**:
  - 🟡 [MODIFIED] taskContainerGateway — `django_validate`/`django_resolve` → `auth_validate`/`cloud_resolve`；移除 djangoInternalApiBase
  - 🟡 [MODIFIED] taskAuth — 新增 container-gateway validate-session（身份+租户成员）；sessionid 解析
  - 🟡 [MODIFIED] taskCloudService — container-target；layer-git-push；runtime-session/open；mock-run env（agent+heuristic）；auth-context GitLab
  - 🟡 [MODIFIED] taskContainerGateway — git-push / mock-run / open-runtime-session 全走 Cloud
  - 🟡 [MODIFIED] taskGateway — mock-run-container-proxy（priority 887）
  - 🟡 [MODIFIED] taskEvents — open-runtime-session → Cloud；workflow transition NO-OP
  - 🔴 [DEPRECATED] Django tcg internal 全部路由（urlpatterns 空）
  - 🎯 [NEW] Plateau v13 — Container Exec Hot-Path Zero-Django
  - ✅ [CLOSES] Gap: tcg 每请求 Django internal 导致 saas-backend 日志噪声与 502 单点
  - ⏳ 残留：无（Python 同步 clear_container_reachability 薄包装可后续直调 Cloud）

## v12 🎯 target — relay clear-logs server-side (path scope)

- **状态**: 🎯 target（已设计，实现中/本会话落地）
- **迭代**: 清理启动日志同步清空 go-relay 缓冲；TaskScope 全部在 path
- **作者**: claude
- **设计日期**: 2026-07-09 20:15
- **交付日期**: —
- **变更文件**:
  - 🆕 `v12-application-integration-20260709-2015-claude.puml` (基于 v11)
  - 🆕 伴生格式: `.archimate` + `.mermaid.md`
- **变更明细**:
  - 🟡 [MODIFIED] go-relay — `POST /v1/tenant/{t}/workspace/{w}/task/{task}/clear-logs`
  - 🟡 [MODIFIED] taskContainerGateway / APISIX — 代理 clear-logs
  - 🟡 [MODIFIED] Vue ServerConfig — 清理时调用 clear-logs
  - 🟡 [MODIFIED] Django thin forward 兜底 `relay-to-trae/clear-logs`
  - 🎯 [NEW] Plateau v12 — path-scoped clear-logs

---

## v11 🎯 target — Vendor Cloud Test Credentials SSOT

- **状态**: ✅ shipped（厂商门户云平台测试密钥 Phase 1）
- **迭代**: 厂商自管测试 AccessKey — taskCloudService CRUD/verify + ai-provider 同源反代与 internal lookup
- **作者**: claude
- **设计日期**: 2026-07-07 22:41
- **交付日期**: 2026-07-07
- **变更文件**:
  - 🆕 `v11-enterprise-landscape-20260707-2241-claude.puml` (基于 v10)
  - 🆕 `v11-application-integration-20260707-2241-claude.puml` (基于 v10)
  - 🆕 伴生格式: `.archimate` + `.mermaid.md`
- **变更明细**:
  - 🟢 [NEW] taskCloudService `vendor_cloud_platform_credentials` 表 + Vendor JWT API + internal lookup
  - 🟢 [NEW] ai-provider VendorPortal「云平台测试密钥」Tab + HTTP 反代
  - 🟡 [MODIFIED] ai-provider CloudSDK 凭证链 — Go lookup 替代 env AK
  - 🔴 [DEPRECATED] 厂商云镜像登记仅依赖平台 env `ALIYUN_ACCESS_KEY`（dev fallback 可选）
  - ✅ [CLOSED] Gap: 厂商无法自配云测试密钥 — WP-v11-vendor-cloud-credentials
  - 🎯 [NEW] Plateau v11 — Vendor Cloud Test Credentials

---

## v10 🎯 target — Cloud Platform CPA + Aliyun Query API Go SSOT

- **状态**: ✅ shipped（Phase 3g+3h 交付）
- **迭代**: Phase 3g CPA/ActiveMethod + Phase 3h Aliyun SDK 查询全量迁入 taskCloudService
- **作者**: claude
- **设计日期**: 2026-07-07 10:16
- **交付日期**: 2026-07-07
- **变更文件**:
  - 🆕 `v10-enterprise-landscape-20260707-1016-claude.puml` (基于 v9)
  - 🆕 `v10-application-integration-20260707-1016-claude.puml` (基于 v9)
  - 🆕 伴生格式: `.archimate` + `.mermaid.md`
- **变更明细**:
  - 🟡 [MODIFIED] taskCloudService — CPA + ActiveMethod + Aliyun regions/zones/price/instances/bandwidth 完整响应契约
  - 🟡 [MODIFIED] APISIX — cloud-platform/*/cloud/* 与授权 CRUD 统一 :8018
  - 🔴 [DEPRECATED] Django `tenant_cloud_platform_views_part2` 与 `urls_tenant_cloud_platform`（已删除）
  - ✅ [CLOSED] Gap: Cloud split-brain — WP Phase 3g+3h
  - 🎯 [NEW] Plateau v10 — Cloud Platform Query Go SSOT
  - 🟡 [MODIFIED] GET `/api/cloud/regions/` — 由 Go `handleCloudRegions` 提供（Django `cloud_regions_global` 已删除）

---

## v9 🎯 target — Cloud 域 Go 真源 + Instruct Worker 下沉（Phase 3）

- **状态**: 🎯 target（Phase 3a–3f 交付）
- **迭代**: CloudServerConfig / container-target / instruct_worker 迁 Go；APISIX cloud 切 :8018
- **作者**: claude
- **设计日期**: 2026-07-06 20:45
- **变更文件**:
  - 🆕 `v9-enterprise-landscape-20260706-2045-claude.puml` (基于 v8)
  - 🆕 `v9-application-integration-20260706-2045-claude.puml` (基于 v8)
  - 🆕 伴生格式: `.archimate` + `.mermaid.md`
- **变更明细**:
  - 🟢 [NEW] taskCloudService CloudServerConfig CRUD + histories + container-target internal
  - 🟢 [NEW] taskAIComment instruct_worker（Trae/legacy 拉流 → taskSSE + Kafka）
  - 🟡 [MODIFIED] APISIX workspace/task cloud 路由 → :8018
  - 🟡 [MODIFIED] `cutover_cloud_configs` 切流命令 + migration 0049 DROP Django cloud 表
  - ✅ [CLOSED] Gap: Phase 3 taskCloudService — WP Phase 3a–3f
  - 🎯 [NEW] Plateau v9 — Cloud Domain Go SSOT
  - 🔴 [DEPRECATED] Django `cloud/views/` 公网（逐步退役；Aliyun SDK pending 完整移植）

---

## v8 🎯 target — 任务域 Go 真源交付（Phase 2）

- **状态**: 🎯 target（Phase 2 已交付，Phase 3 云域待续）
- **迭代**: 任务域 Go 真源 — Django Todo 零残留
- **作者**: claude
- **设计日期**: 2026-07-06 18:34
- **交付日期**: 2026-07-06（Phase 2）
- **变更文件**:
  - 🆕 `v8-enterprise-landscape-20260706-1834-claude.puml` (基于 v7)
  - 🆕 `v8-application-integration-20260706-1834-claude.puml` (基于 v7)
  - 🆕 伴生格式: `.archimate` + `.mermaid.md`
- **变更明细**:
  - 🟡 [MODIFIED] taskTaskService — SQLite `task_task.db` 唯一真源，CRUD/评论/功能参数
  - 🟡 [MODIFIED] saas-backend — 删除 Todo/Comment ORM 与公网路由 (migration 0052)
  - 🟢 [NEW] `projects/task_client.py` + 测试 helper 直连 Go
  - ✅ [CLOSED] Gap: Task/Comment in Django — WP Phase 2 交付
  - 🎯 [NEW] Plateau v8 — Phase 2 Task Domain Delivered
  - ⏳ [OPEN] Gap: Phase 3 taskCloudService
  - 🟢 [NEW] taskAIComment stub (:8019) — health + 503；ai-comments 仍走 saas-backend
  - 🟡 [MODIFIED] runAll — task-project-service → saas-backend；task-task-service → task-bill
  - 🟡 [MODIFIED] v8 `.archimate` — TTS→BILL、GW→BE/TAI sourceConnection、Plateau v8

---

## v7 🎯 target — 项目域 Go 真源交付（Phase 1）

- **状态**: 🎯 target（Phase 1 + Phase 1b 已交付，Phase 2/3 待续）
- **迭代**: 项目域 Go 真源 — 移除 Django fallback 与 projects_* 表
- **作者**: claude
- **设计日期**: 2026-07-06 17:10
- **交付日期**: 2026-07-06（Phase 1 部分）
- **变更文件**:
  - 🆕 `v7-enterprise-landscape-20260706-1710-claude.puml` (基于 v6)
  - 🆕 `v7-application-integration-20260706-1710-claude.puml` (基于 v6)
  - 🆕 伴生格式: `.archimate` + `.mermaid.md` (每个视图)
- **变更明细**:
  - 🟡 [MODIFIED] taskProjectService — SQLite `task_project.db` 唯一真源，无 Django Strangler fallback
  - 🟡 [MODIFIED] saas-backend — 删除 projects_* 表 (migration 0051)、摘除 project/workspace 公网路由；Todo 保留 loose workspace_id
  - 🟢 [NEW] SQLite 项目域存储 — 替代 PostgreSQL projects_project/workspace
  - ✅ [CLOSED] Gap: Project/Workspace in Django — WP Phase 1 交付
  - ✅ [CLOSED] Gap: Phase 1b endpoints — switch-workspace、batch-delete、gitlab-sync、workspace-access
  - ⏳ [OPEN] Gap: Phase 2 taskTaskService / Phase 3 taskCloudService
  - 🎯 [NEW] Plateau v7 — Phase 1 Delivered（taskProjectService live）
  - 🔴 [REMOVED] Django projects/views 公网 API、BE_PROJ 拓扑节点

---

## v6 🎯 target — Django 项目/任务/阿里云接口 → 3 个 Go 服务拆分

- **状态**: 🎯 target（已设计，待交付）
- **迭代**: Django 项目(Project)/任务(Task)/阿里云(Cloud)接口 → Go 微服务拆分
- **作者**: claude
- **设计日期**: 2026-07-06 15:43
- **交付日期**: —
- **变更文件**:
  - 🆕 `v6-enterprise-landscape-20260706-1543-claude.puml` (基于 v4)
  - 🆕 `v6-application-integration-20260706-1543-claude.puml` (基于 v5)
  - 🆕 伴生格式: `.archimate` + `.mermaid.md` (每个视图)
- **变更明细**:
  - 🟢 [NEW] taskProjectService (Go :8016) — 项目/工作空间/访问控制/交付物体系
  - 🟢 [NEW] taskTaskService (Go :8017) — 任务 CRUD/评论/AI 评论流/功能参数
  - 🟢 [NEW] taskCloudService (Go :8018) — 阿里云授权/OAuth/实例/网络/镜像/地域
  - 🟢 [NEW] 3 个 PostgreSQL database (task_project_db, task_task_db, task_cloud_db)
  - 🟢 [NEW] APISIX 路由规则 (3 组 upstream + route)
  - 🟡 [MODIFIED] saas-backend (Django :8001) — 移除 projects/views (15 文件) + cloud/views (25 文件)，退化为 accounts + company internal 真源
  - 🟡 [MODIFIED] APISIX Gateway — 新增 3 组 Go 服务 upstream
  - 🔴 [DEPRECATED] Django projects/views/ — 交付后删除
  - 🔴 [DEPRECATED] Django cloud/views/ — 交付后删除
  - 🔴 [DEPRECATED] Django cloud/providers/aliyun/ (13 文件) — 交付后删除
  - ℹ️ LLM Budget (llm_budget_views.py 498行) 归属 taskAIEndPoint，不迁移

---

## v5 ✅ shipped — relay lifecycle 事件消费者

- **状态**: ✅ shipped（Inc 1-4 已交付）
- **迭代**: relay lifecycle 公网绕 Django + 副作用事件化
- **作者**: claude
- **设计日期**: 2026-07-05 17:00
- **交付日期**: 2026-07-05
- **变更文件**:
  - 🆕 `v5-application-integration-20260705-1700-claude.puml` (基于 v4)
  - 🆕 伴生格式: `.archimate` + `.mermaid.md`
- **变更明细**:
  - 🟡 [MODIFIED] taskContainerGateway — relay lifecycle 编排 + Kafka 事件
  - 🟢 [NEW] taskEvents relay-lifecycle 4 intents (audit/session/reachability/workflow)
  - 🟢 [NEW] Django internal relay lifecycle APIs
  - 🔴 [DEPRECATED] Django relay 公网代理（410 stub）

---

## v4 🎯 target — task2app 接口 Go 拆分

- **状态**: 🎯 target（已设计，待交付）
- **迭代**: task2app 接口 Go 拆分 — 容器交互出站/relay 公网入口
- **作者**: claude
- **设计日期**: 2026-07-05 15:31
- **交付日期**: —
- **变更文件**:
  - 🆕 `v4-enterprise-landscape-20260705-1531-claude.puml` (基于 v3)
  - 🆕 `v4-application-integration-20260705-1531-claude.puml` (基于 v3)
  - 🆕 伴生格式: `.archimate` + `.mermaid.md` (每个视图)
- **变更明细**:
  - 🟢 [NEW] Application Layer: Container Proxy API、Relay Lifecycle API；onlineServiceJS 显式外部组件
  - 🟢 [NEW] Implementation Layer: Plateau v1/v4、Gap（outbound/relay/SSE）、WorkPackage Phase 0-2
  - 🟡 [MODIFIED] taskContainerGateway — 完整 compute outbound + job-stream goroutine
  - 🟡 [MODIFIED] go-relay — 浏览器 relay 公网入口（APISIX 直路由，绕过 Django）
  - 🟡 [MODIFIED] taskAgentSupport — inbound 业务逻辑 Phase 2 下沉 Go
  - 🟡 [MODIFIED] saas-backend — 瘦身为 CRUD + Django internal 真源
  - 🔴 [DEPRECATED] Django inline SSE（urls.py threading）；relay_to_trae_proxy 公网路径

---

## v3 🎯 target — claude-agent Go 重写

- **状态**: 🎯 target（已设计，待交付）
- **迭代**: claude-agent Go 重写 — sidecar CLI
- **作者**: claude
- **设计日期**: 2026-07-02 15:00
- **交付日期**: —
- **变更文件**:
  - 🆕 `v3-enterprise-landscape-20260702-1500-claude.puml` (基于 v2)
  - 🆕 `v3-application-integration-20260702-1500-claude.puml` (基于 v2)
- **变更明细**:
  - 🟢 [NEW] Application Layer: claude-agent (Go CLI Sidecar) — Claude Code CLI subprocess 封装
  - 🟢 [NEW] External: Claude Code CLI (Node.js) — npm @anthropic-ai/claude-code
  - 🟡 [MODIFIED] Technology Layer: Go Runtime — 新增 claude-agent 服务
  - 🔴 [DEPRECATED] claude_agent/ Python 实现 — 交付后由 Go 版本替代

---

## v2 🎯 target — taskAIEndPoint 架构纳入 + token 验证迁移

- **状态**: 🎯 target（已设计，待交付）
- **迭代**: taskAIEndPoint 使用说明撰写 & token 验证迁移至 taskCredentialService
- **作者**: claude
- **设计日期**: 2026-07-01 17:45
- **交付日期**: —
- **变更文件**:
  - 🆕 `v2-enterprise-landscape-20260701-1745-claude.puml` (基于 v1)
  - 🆕 `v2-application-integration-20260701-1745-claude.puml` (基于 v1)
- **变更明细**:
  - 🟢 [NEW] Application Layer: taskAIEndPoint (Go LLM Gateway :8013), taskCredentialService (Go Token SSOT :8015)
  - 🟢 [NEW] Business Layer: LLM AI Service (预算管控 + 代理网关)
  - 🟢 [NEW] Technology Layer: Go Runtime, Upstream LLM Provider (DeepSeek/OpenAI)
  - 🟢 [NEW] Application Interface: LLM Proxy API (OpenAI 兼容 Chat Completions)
  - 🟢 [NEW] 数据流: Agent 容器 → taskAIEndPoint → upstream LLM; taskAIEndPoint → Django internal API; taskAIEndPoint → taskCredentialService token 验证
  - 🟡 [MODIFIED] validate_proxy_token: CloudServerConfig.container_access_token 直查 → GoTokenValidator (taskCredentialService SSOT)

---

## v1 ✅ current — 初始架构基线

- **状态**: ✅ current（已交付）
- **迭代**: 初始架构基线建立
- **作者**: claude
- **设计日期**: 2026-07-01 16:30
- **交付日期**: 2026-07-01 16:30
- **变更文件**:
  - 🆕 `v1-enterprise-landscape-20260701-1630-claude.puml`: 企业架构全景图 — Business/Application/Technology 三层
  - 🆕 `v1-application-integration-20260701-1630-claude.puml`: 应用组件架构 — 39 服务组件 + 18 领域事件消费者
- **变更明细**:
  - 🟢 [NEW] Business Layer: Developer, Platform Admin, Enterprise Customer; 4 Business Services, 2 Business Processes
  - 🟢 [NEW] Application Layer: APISIX Gateway, Django SaaS, Auth/Provider/Git/CI-CD/SSE Services, 4 APIs
  - 🟢 [NEW] Technology Layer: Docker, PostgreSQL, Redis, Kafka, GitLab; LB + 3 Host nodes
  - 🟢 [NEW] Domain Events: 18 Go consumers across 5 event domains
  - 🟢 [NEW] Cross-cutting: APISIX forward-auth → task-auth, OIDC SSO, DEPLOY_MODE switch
