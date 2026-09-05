# runAll

## 新服务注册（硬约束）

**创建新的 Go 服务时，必须同步更新 `conf/runAll.yaml` 添加服务条目**（build/start/stop/health_check/depends_on）。
同步创建 `build.sh` 和 `conf/{app}/config.yaml`。
详见 `.ai/01_project_constraints/33_new_service_runall_registration.md`。

## 约束

- `./run.sh` 默认 `setsid nohup` 独立会话写 `logs/runall-console.log`，避免 screen 前台作业被 SIGTERM 带走（2026-08-20 `detail=terminated`）。`-command doctor|takeover|build-all` 或 `RUNALL_FOREGROUND=1` 仍前台 `exec`。clone-run（`$ROOT/bin/runAll` 可执行且无 `taskAuth/`）时 `run.sh` **必须** `source $ROOT/cutover.env`（`up.sh` 写出；缺文件则失败）。Go `LoadConfig` 同样 source 该文件，因此 `./bin/runAll conf/runAll.yaml` 只要文件在配置旁也能带上 `DEPLOY_MODE`。禁止在 `run.sh` 复述一份 `export` 列表。
- **编排器 ≠ 业务栈（ADR-0035）**：UI 模式 SIGTERM/SIGINT/`Ctrl+C` **默认不停止托管服务**；新实例 adopt。整栈停止用「全部停止」。前台调试拆栈：`RUNALL_SHUTDOWN_SERVICES=1`。托管进程仅 `Setsid`（不可再叠加 Setpgid，会 EPERM）；`Kill(-pid)` 仍有效因为会话首领 pgid==pid。Setsid **不能**防 SIGPIPE：长驻服务 stdio 必须继承 `logs/<name>.log` 文件 FD，禁止 `StdoutPipe`。Linux `rt_sigtimedwait` 遇 `EINTR` 必须重试，否则 :9999 误退出。
- **GitLab 可插拔（ADR-0048）**：`conf/runAll.yaml` 中除 `git-service*` 外禁止 `depends_on: git-service*`；本机实例另受 ADR-0047 闸门约束。门禁：`TestProductionConfig_NoInboundDependsOnGitLab`。
- **9999 空窗拉起**：`:9999` 已无监听但托管端口仍在时，直接用 `./run.sh` 拉起即可——`HadPrevious=false` 时不会再对托管端口做 orphan `SIGKILL`，会走 adopt（OPT-20260820-008）；禁止先手工停托管服务再启动。
- **编排器退出指纹（OPT-20260905-005）**：优雅退出 / fatal error 会写 `.runall/last_orchestrator_exit.json`（`source`/`detail`/`pid`/`exited_at`）；watchdog `:9999` down 告警附带 `last_exit(...)`。`logs/runall-console.log` 整点 truncate 保留末尾 256KiB（另有 `logs/archive/` 短窗归档）。
- 经 `/api/shutdown-self` 热替换时，**必须**使用已含 skip-orphan 能力的二进制（稳定标记：`skip orphan port cleanup`）。
- 编译须走 `./build.sh`（会校验该标记）；禁止用缺少标记的旧 `bin/runAll` 替换正在跑的实例。
- `GracefulShutdownSelf=true` 时不得对托管端口做 orphan `SIGKILL`（见 `cleanupOrphanManagedServices`）。
- UI 启停/重启优先使用服务行 `session_id`（owning session）；start/stop/restart 后端对非 owner 会委托给注册 owner，降低热替换后摩擦。
- **全部重启**：`POST /api/restart-all`（需 `session_id`）= 按依赖序金丝雀平滑重启（ADR-0058：新进程 overlap 就绪后再排空旧进程），**不编译**（ADR-0027，只拉 last-good 二进制）；进度 SSE `/api/restart-all/progress`（单通道 `operation=restart`）；中断 `/api/restart-all/cancel`。UI 按钮「全部重启」。单服务 ↻ 同样不编译、走同一 canary swap。新二进制走页头「精准编译重启」（compile-then-swap：先编成功再 canary overlap）或「全部重新编译」。无法 overlap 的基础设施回退为排空后启动。
- Status UI 源码在 `src/status_ui/`（`index.html` + `css/*` + `js/*`，各文件 ≤500 行），由 `status_page.go` 经 `go:embed` 组装后由 `/` 下发；禁止再回退为单文件 `status.html` 巨石。
- **热替换收养**：UI 模式启动时 `AdoptRunningManagedServices` / `adoptListeningServices` 对已监听端口与 ownership.json 中仍存活的 PID 做 StatusStore 回填，避免 shutdown-self 后 UI 空白。
- 启动健康检查须区分 **LAUNCH_PROCESS_EXITED**（进程在就绪前退出，含未 wait 的僵尸进程）与 **READINESS_TIMEOUT**（进程仍在但探针失败）。UI `/api/status` 的 `error`/`hint` 应带 `failure_code` 前缀便于一眼识别。
- 启动失败（尤其是进程提前退出）须在 error 中附带 **recent stderr**（最近约 12 行），以便直接看到 TabError/SyntaxError 原文，而不是只看到 connection refused。
- 若 stderr 仅有 confload/config 噪音、而真正致命原因在 stdout（如 SQLite `database disk image is malformed`、JSON `"level":"error"`），须改附 **recent logs** 中带 fatal 特征的行，避免 UI 误导为「只有 confload」。
- Internal API live smoke：`POST /api/smoke/internal-apis/verify`（UI 一键）；`start-all` 成功后自动软跑；脚本 `scripts/smoke/internal-apis-live.sh`（`INTERNAL_API_SMOKE_MODE=required|optional`）。
- 同步全量编译 CLI：`./bin/runAll -command build-all`（复用 `Runner.BuildAll`，失败/partial → `exit 1`；无需 UI SSE）。
- **部署同步 CLI**：`./bin/runAll -command deploy-sync`（`DEPLOY_MODE=1`，`scripts/deploy-sync.sh`）。按 `releases.yaml` 下载钉版本，禁止编译。
- feature-params store SSOT 在 task-cloud：smoke 探针 `GET :8018/api/internal/feature-params-env/`；Django `:8001/api/internal/feature-params/tenant` 须固定 **410**（live smoke 哨兵 + 离线 `internal-apis-live.listening_test.sh`）。
- smoke 失败时 UI 状态栏须展示具体 `FAIL …` 原因（`summarizeSmokeOutput` + `ts-fail-detail`），不能只显示「失败」与 summary 计数。
- **清空 tee 日志**：单服务 `POST /api/logs/clear`；全量 `POST /api/logs/clear-all`（或 `scripts/truncate-ram-work-logs.sh`，其优先调 clear-all）。**禁止**对 `logging.file_root`（默认 `logs/`）下 `*.log` 使用 `rm`/`unlink`——会留下 deleted inode，Promtail→Loki probe 验证失败。`/api/dev/logs/clear` 仅清开发工具日志，不负责服务 tee 文件。
