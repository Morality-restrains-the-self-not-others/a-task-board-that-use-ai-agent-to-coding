# Completed OPT Archive — 2026-09-01

> 从 OPTIMIZATION_TODOS_COMPLETED.md 按天归档，共 14 条。
> 归档执行时间：2026-09-02T03:01:17+08:00

## [OPT-20260831-013] completed

- **Status**: completed
- **Completed**: 2026-09-01
- **Summary**: 删除 load_gitservice_config.py local_path 形参与 CLI 第三参数；run.sh/部署脚本只传 config.yaml；17 loader tests pass
- **Created**: 2026-08-31
- **Context**: 两步加载落地后，`load_gitservice_config.py` 忽略第三参数，但 `gitService/run.sh` 仍构造 `PORT_CONFIG_LOCAL` 并传入；`push_gitlab_image_to_sh.sh` / `deploy_tencent_sh_1_from_infra.sh` 同样传路径。调用方会误以为旁路 `config.local.yaml` 仍生效。
- **Action**: (1) 从 `load_gitservice_config.py` 去掉 `local_path` 形参及 `_ = local_path`；(2) `run.sh` 与部署脚本只传 `config.yaml`；(3) 更新 `test_load_gitservice_config.py` 调用签名。
- **Why**: 废弃参数会让运维继续在 `config.local.yaml` 写覆盖，而加载器已不读。
- **How to apply**: `python3 gitService/scripts/test_load_gitservice_config.py`；`bash -n gitService/run.sh`；`rg -n 'config.local.yaml' gitService --glob '*.sh'` 应无传参。

---

## [OPT-20260831-020] completed

- **Status**: completed
- **Completed**: 2026-09-01
- **Summary**: LokiClient 对无 path 的 base 自动补 /loki/api/v1；compose --loki 改显式完整路径；21 tests pass
- **Created**: 2026-08-31
- **Context**: Trace Log Journey 空盘排查时 `aimonitor-log-collection-status-reporter` 每 60s 报 `http://aimonitor-loki:3100/label/service/values` HTTP 404。compose 传入 `--loki http://aimonitor-loki:3100`，脚本默认查询根是 `.../loki/api/v1`，两者不一致。
- **Action**: (1) 把 `AiMonitor/docker-compose.yaml` 的 `--loki` 改为 `http://aimonitor-loki:3100/loki/api/v1`，或让 reporter 在缺前缀时自动补；(2) 补测：给定 base=`http://loki:3100` 时请求路径含 `/loki/api/v1/label/`；(3) 复跑容器后 `docker logs` 不再 404。
- **Why**: reporter 驱动 Trace Log Explore 的 collection_status 面板；404 时 Active Log Sources 会假空，与 Loki 已有 job 矛盾。
- **How to apply**: `python3 AiMonitor/scripts/log_collection_status_reporter.py --loki http://127.0.0.1:3100 --dry-run`；`curl -sf http://127.0.0.1:3100/loki/api/v1/label/service/values` 须 200。

---

## [OPT-20260831-021] completed

- **Status**: completed
- **Completed**: 2026-09-01
- **Summary**: promtail pipeline 加 timestamp stage(source=ts,RFC3339Nano)；重生成 83 服务；runAll 生成器与 AiMonitor 配置同改；tests pass
- **Created**: 2026-08-31
- **Context**: 拉起缺失的 aimonitor-promtail 后，Seeked Offset:0 会把文件里已有行以**摄入时刻**写入 Loki（pipeline 抽了 `ts` 但没有 `timestamp` stage）。Grafana Last 30m 的 volume 图在采集恢复前几乎是平的，只在启动瞬间出现一根尖刺。
- **Action**: (1) 在 `generate-promtail-config.sh` 的 pipeline 于 json 之后加 `timestamp` stage（source=`ts`，RFC3339/RFC3339Nano）；(2) 单测生成配置含 timestamp；(3) 用一条已知过去 `ts` 的行断言 Loki 点落在该 ts 而非 ingest now。
- **Why**: 恢复采集或 truncate 后重 scrape 时，值班看 30m/1h 窗口会误判「这段时间没日志」。
- **How to apply**: `rg -n 'timestamp:' AiMonitor/promtail/promtail-local.yaml`；`python3 AiMonitor/scripts/test_promtail_local_push_config.py`。

---

## [OPT-20260831-016] completed

- **Status**: completed
- **Completed**: 2026-09-01
- **Summary**: collect 结束对照 conf.example/releases.yaml sha 校验已拷文件，不一致非零退出；12 collect tests pass
- **Created**: 2026-08-31
- **Context**: clone-run 改为手拷 `deploy-binaries/` 后，归集脚本只按文件名复制，不核对 `conf.example/releases.yaml` 的 `sha`。拷错 ELF 或混入旧 tag 时 `up.sh` 仍会装上。
- **Action**: (1) 在 `collect-deploy-binaries.sh` 结束时对每个已拷文件算 SHA-256；(2) 与 `conf.example/releases.yaml` 同名 pin 比对，不一致则非零退出；(3) 补 `test_collect_deploy_binaries.py` 用例。
- **Why**: 手拷路径没有 GitHub 钉校验，错版本会静默进新节点。
- **How to apply**: `bash scripts/collect-deploy-binaries.sh`；对照 `conf.example/releases.yaml` 的 `sha` 字段。

---

## [OPT-20260831-017] completed

- **Status**: completed
- **Completed**: 2026-09-01
- **Summary**: taskFE-dist.tar.gz 改 live taskFE/app/public 优先逻辑（文件数/newest mtime），已有包也刷新；新增重打包测试
- **Created**: 2026-08-31
- **Context**: `sync_taskevents_archive` 已按 live `taskEvents/bin` worker 数/mtime 重打包，但 `taskFE-dist.tar.gz` 仍走 `copy_named || pack_if_missing`：dest 已存在则永不从 `$META_ROOT/taskFE/app/public` 重打。
- **Action**: (1) 把 FE 包收成与 taskEvents 相同的 live-tree 优先逻辑（文件数或 newest mtime）；(2) 补 `test_collect_deploy_binaries.py`：live public 多一个文件时必须重打包。
- **Why**: 前端静态资源更新后手拷仍可能装上旧 `index.html`，clone-run 看不到新 UI。
- **How to apply**: `runAll/scripts/collect-deploy-binaries.sh` 的 `pack_if_missing`；对照 `test_collect_repacks_taskevents_when_live_has_more_workers`。

---

## [OPT-20260831-018] completed

- **Status**: completed
- **Completed**: 2026-09-01
- **Summary**: taskAiProvider-frontend-dist.tar.gz 已上传 deploy-20260831-conf-local，asset 名与 pin package 一致
- **Created**: 2026-08-31
- **Context**: clone-run 已增加 `taskAiProvider-frontend-dist` pin 与本地 collect/unpack，但 GitHub tag `deploy-20260831-conf-local` 尚无该 asset。新节点若只走 `FORCE_DEPLOY_SYNC=1` 而无手拷 `deploy-binaries/`，厂商门户仍会 404。
- **Action**: (1) 将源码仓 `deploy-binaries/taskAiProvider-frontend-dist.tar.gz`（sha `a0cdfaf721a7b0b33cb97c1730665230f6c4ddae2fed8e7bc3dbe28383f3fe4c`）上传到同一 Release；(2) 用 `gh release view` 确认 asset 名与 pin 的 `package` 一致。
- **Why**: GitHub fallback 是无本地产物时的唯一路径；缺 SPA 包会让 `provider.*` 再次 ServeFile 404。
- **How to apply**: `sha256sum deploy-binaries/taskAiProvider-frontend-dist.tar.gz`；`gh release upload deploy-20260831-conf-local … --repo task2money/daydaymoney-deploy`。

---

## [OPT-20260831-019] completed

- **Status**: completed
- **Completed**: 2026-09-01
- **Summary**: 缺 SPA dist 由 warn 改 ensureFrontendDist+tracelog.Fatal；新增 CI 门禁与 4 自测；runAll 直启不再绕过
- **Created**: 2026-08-31
- **Context**: `conf/runAll.yaml` 的 `ai-provider` 使用 `start_command: "./bin/taskAiProvider"`，绕过 `taskAiProvider/run.sh start` 对 `frontend/dist/index.html` 的硬失败。seed 部署缺 SPA 时进程仍起来，handleSPA 对 `/` 静默 404。
- **Action**: (1) 把 `start_command` 改为 `./run.sh start`，或在 `main.go` 缺 dist 时 `Fatal` 而非 warn；(2) 补单测/门禁：无 `index.html` 不得进入 listen；(3) 确认 seed layout 仍带 `taskAiProvider/run.sh`。
- **Why**: collect/install 若再漏包，启动失败比线上 404 更容易发现。
- **How to apply**: `rg -n 'start_command' conf/runAll.yaml`；`bash taskAiProvider/run.sh start` 在缺 dist 时非零退出。

---

## [OPT-20260831-015] completed

- **Status**: completed
- **Completed**: 2026-09-01
- **Summary**: taskEvents config 用 confload.MergeConfLocal 叠加 conf-local；重建 55 worker 并重打 taskEvents-bin.tar.gz 上传 Release；conf.example/seed 引脚 sha 更新
- **Created**: 2026-08-31
- **Context**: clone-run 已钉 `deploy-20260831-conf-local` 的 Go ELF，但 `taskEvents-bin.tar.gz` 仍复用旧 tag `deploy-20260831` 的 worker。若 worker 进程读 YAML，新节点只拷 conf-local 时可能仍看到空机密。
- **Action**: (1) 确认哪些 worker 调用 confload；(2) 用含 MergeConfLocal 的源码重打 `taskEvents-bin.tar.gz`；(3) 发布到同一或新 Release 并改 `conf.example/releases.yaml` 与 seed `envs/current/releases.yaml`；(4) 用 `deploy-sync` 干跑断言 worker 指纹变化。
- **Why**: 只拷 conf-local 的新节点若 worker 仍读旧二进制，会出现「主 ELF 已 overlay、定时/事件进程未 overlay」。
- **How to apply**: `rg -n 'MergeConfLocal|ReadAppConfig' taskEvents`；`sha256sum` 新旧 tarball；`python3 -c` 解析 `releases.yaml` pin。

---

## [OPT-20260901-001] completed

- **Status**: completed
- **Completed**: 2026-09-01
- **Summary**: restart-all/start-all now read MigratePendingStatus and refuse with「先点初始化全部数据库」when dataMigrate SQL is unapplied (runAll dfb10eb); guarded in RestartAllWithActor/StartAllWithActor + HTTP handlers; 3 new Go tests pass; MONOREPO_ROOT gate keeps bare tests fast
- **Created**: 2026-09-01
- **Context**: 新 clone-run 根 `~/bin/daydaymoney-deploy` 带空 MySQL 卷时，用户直接点「全部重启」。task-auth 遵守「启动不跑 DDL」，`loadUserContentTypeID` 查 `auth_django_content_type` 得到 1146 后 exit=1；task-cloud-service 仍在启动路径 `runDataMigrate`，所以只有部分库有表，失败面像单服务 bug。
- **Action**: (1) `start-all` / `restart-all` 在真正 stop/start 前读已有 `MigratePendingStatus`；(2) 若 `task_auth`（或 pendingCount>0 且含空库）则拒绝 bulk 并返回明确文案「先点初始化全部数据库」；(3) 单测：registry 有未应用 SQL 时 `RestartAllWithActor` 不调用 StartService。
- **Why**: 空库上重启会把 schema 缺口表现为 LAUNCH_PROCESS_EXITED，值班会去改 task-auth 而不是点 9999 初始化。
- **How to apply**: `runAll/src/ui_restart_all.go`、`migrate_pending_status.go`、`runAll/src/ui_restart_all_test.go`；对照 `GET /api/dev/migrate-status`。

---

## [OPT-20260901-002] completed

- **Status**: completed
- **Completed**: 2026-09-01
- **Summary**: runAll/run.sh exported into deploy payload (runAll a541dcb); up.sh START=1 uses detached setsid -f nohup launcher; README documents source cutover.env && ./runAll/run.sh; export test asserts run.sh shipped executable with setsid -f nohup
- **Created**: 2026-09-01
- **Context**: `~/bin/daydaymoney-deploy/envs/current/runAll/` 只有 `scripts/`，没有 `run.sh`。本会话用 agent shell `setsid nohup bin/runAll` 拉起后约 2s 日志出现 `orchestrator_exit_keeps_services trigger=signal`，:9999 掉线；后改 `systemd-run --user --unit daydaymoney-deploy-runall` 才稳住。
- **Action**: (1) collect/export 把 `runAll/run.sh` 打进 deploy 树；(2) `up.sh` / README 写明 `source cutover.env && ./runAll/run.sh`（`RUNALL_SKIP_BUILD=1`）；(3) 测：unpack 后 `test -x runAll/run.sh`，且脚本含 `setsid -f nohup`。
- **Why**: 无 run.sh 时人/Agent 会前台或短会话拉 ELF，编排器被 SIGTERM 后 9999 空白，托管进程还在，像「runAll 挂了栈还在」。
- **How to apply**: `runAll/scripts/export-deploy-payload.sh`、`runAll/scripts/up-from-config-repo.sh`、`runAll/scripts/daydaymoney-deploy.README.md`；`python3 runAll/scripts/tests/test_export_deploy_payload.py`。

---

## [OPT-20260901-003] completed

- **Status**: completed
- **Completed**: 2026-09-01
- **Summary**: dead _progressAutoHideTimer declaration + clearTimeout calls removed from 01/02/08/progress_done.js (runAll 927b532); new regression test asserts no file references the dead timer; JS 8/8 + Go ProgressPanelManualClear pass
- **Created**: 2026-09-01
- **Context**: 进度条已改为手动「清空」才关闭，不再 `setTimeout` 淡出；`01.js` / `08.js` / `disconnect*SSE` 仍声明并 `clearTimeout(_progressAutoHideTimer)`。
- **Action**: (1) 删掉 `_progressAutoHideTimer` 声明与所有 clearTimeout；(2) `dismissProgressPanel` 不再引用该变量；(3) `node --test src/status_ui/js/progress_done.test.js` 与 `go test ./src -run TestAssembledStatusPage_ProgressPanelManualClear` 保持绿。
- **Why**: 死变量会让后续改动误以为仍有自动隐藏。
- **How to apply**: `runAll/src/status_ui/js/01.js`、`02.js`、`08.js`、`progress_done.js`。

---

## [OPT-20260901-009] completed

- **Status**: completed
- **Completed**: 2026-09-01
- **Summary**: loadDomainEventsGlobal 与 confload.ReadAppConfig 均叠 conf-local docker-infra.yaml；UnmarshalYAMLMerged 覆盖 nested intent YAML。
- **Created**: 2026-09-01
- **Context**: EMAIL_SENT 消费者 `LoadSettings` 曾直读 tracked `email.yaml` 漏掉 conf-local 密码（QQ SMTP 535）。`loadDomainEventsGlobal` 对 `config.yaml` 已 `MergeConfLocal`，但同函数仍 `os.ReadFile` `docker-infra.yaml` 且不叠 conf-local。
- **Action**: (1) `taskEvents/config/conf_yaml.go` `loadDomainEventsGlobal` 读 `docker-infra.yaml` 后 `MergeConfLocal(root, "events/domain-events/docker-infra.yaml", …)`；(2) 测例：tracked 空/骨架 + conf-local 覆盖 redis/kafka 键；(3) `rg 'os.ReadFile' taskEvents --glob '*.go'` 确认其它 fragment 同样 overlay 或注明无机密。
- **Why**: 同类「tracked 骨架 + conf-local 真值」漏叠会再次让运行时读空机密。
- **How to apply**: `go test ./config -run TestLoadDomainEventsGlobal -count=1`（taskEvents）exit 0；新增测例断言 docker-infra conf-local 覆盖。

## [OPT-20260901-013] completed

- **Status**: completed
- **Completed**: 2026-09-01
- **Summary**: 022 identifier 改为占位符；apply_datamigrate.sh render_sql_file 与 runDataMigrateFromDir 按 overlay_conf_file/confload 渲染。pytest 13 passed；go test TestRenderBootstrapAdminSQL TestMigrateSeedsConfBootstrapAdminEmail 等 exit 0。
- **Created**: 2026-09-01
- **Context**: 管理员邮箱 SSOT 已落到 `conf/auth/task-auth/config.yaml` 的 `bootstrapAdmin.email`，init 时 `bootstrap-admin` 会覆盖 `auth_login_method.identifier`。`022_seed_bootstrap_admin.sql` 仍 INSERT IGNORE 仓库默认 `author@example.com`，仅 migrate、未跑 init.sh 时库里仍是 SQL 字面量。
- **Action**: (1) apply_datamigrate 或 022 改为按 conf 渲染 identifier；(2) 断言 SQL 产物或 migrate 后行等于 conf（含 conf-local）；(3) 去掉「SQL 默认 / conf 默认必须手工同步」的注释约定。
- **Why**: 两处默认邮箱会漂，只跑 migrate 的环境会种出与 conf 不一致的超管邮箱。
- **How to apply**: `dataMigrate/taskAuth/022_seed_bootstrap_admin.sql`；`db/scripts/apply_datamigrate.sh`；`taskAuth/src/bootstrap_admin_test.go`。

## [OPT-20260901-015] completed

- **Status**: completed
- **Completed**: 2026-09-01
- **Summary**: taskEvents smtp.IsPermanent: 535/login fail → DispatchPermanent；delivery_test 覆盖；clone-run 18:48 后 Loki EMAIL_SENT delivered to author@example.com，无新 535
- **Created**: 2026-09-01
- **Context**: 管理员重置密码 `POST /api/accounts/users/send_password_reset_link/`（trace `85ddb59f-661a-4af4-b8f1-d4030cf9687d`）HTTP 200，但 `task-events-email-sent-1-send-email` 对 QQ SMTP `535 Login fail` 返回 `DispatchRetryable`，指数退避重试 11 次后 DLT。535 文案含「login frequency limited」；同窗口另有 `lookup smtp.qq.com on 127.0.0.53:53: server misbehaving`。
- **Action**: (1) `handleEmailSent` 识别 `535` / `Login fail` 为永久失败，返回 `DispatchPermanent`；(2) 单测：mock SMTP 535 → 不重试、投 DLT；(3) 网络类错误（dial/timeout/DNS）仍 retryable。
- **Why**: 认证失败重试只会触发 QQ 登录频率限制，把偶发 535 放大成整条邮件链路死信；密码重置 API 已对用户宣称「已发送」。
- **How to apply**: `taskEvents/notifications/delivery.go` `handleEmailSent`；`taskEvents/notifications/delivery_test.go`。验收：`go test ./notifications -run 'EmailSent.*535|SMTPAuth'` exit 0。
- **Related**: OPT-20260901-011

