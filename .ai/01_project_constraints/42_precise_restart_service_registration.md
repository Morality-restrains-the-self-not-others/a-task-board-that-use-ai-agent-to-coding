# 智能体编程会话：服务代码变更必须登记「精准编译重启」

## 背景

runAll 页面（clone-run 本机 `http://192.168.1.10:9999/`；历史文档亦写 `http://10.2.150.68:9999/`）提供「精准编译重启」按钮：读取登记文件
**源码仓** `$SOURCE_ROOT/.runall/precise_restart_services.txt`（gitignore 运行时状态；ADR-0056：部署树 9999 不得读 `$DEPLOY_ROOT/runAll/.runall/`），按依赖序对
登记的服务执行 **compile-then-swap：编译 → 停止 → 启动 → 健康检查**（ADR-0027；
复用 `restartService` 且仅本路径带编译标记。未配置编译命令的服务跳过编译直接重启）。
编译失败则旧进程继续、状态保持原状、登记保留。完成后清空成功项。

**全部重启 / 单服务 ↻ / start_command 不编译**（只 exec 磁盘 last-good 二进制）。
`taskEvents/run.sh start` 及其它服务 `run.sh start` 禁止隐式 `go build` / `./build.sh`。

**痛点：** 智能体编程会话修改某服务代码后，旧进程仍在运行，必须人工到 runAll 页面
点击该服务的 ↻ 重启，且容易漏掉多个被改动的服务。登记机制让「改了哪些服务」显式落盘，
一次按钮点击完成全部受影响服务的编译重启。

## 硬约束（一级，禁止忽略）

**当智能体编程会话（Claude Code / Trae / Cursor 等）修改了某个 runAll 托管服务的
代码时，必须将该服务登记到 `<仓库根>/.runall/precise_restart_services.txt`，**
方式二选一（推荐脚本，自带服务名校验）：

1. **推荐：** `scripts/register-precise-restart.sh <service-name>...`（可多个；`--list`
   查看全部可登记名；`--clear` 清空）。
2. 直接向文件追加一行服务名（`#` 开头为注释；空行忽略；自动去重）。

**服务名写法（两种等价）：**

- runAll.yaml 中的服务 `name`，如 `task-auth`、`task-bill`、`saas-backend`、`taskFE`；
- 服务工作目录名，如 `taskAuth`、`taskBill`、`taskFE`（`taskFE/app` → 别名 `taskFE`）。

**触发条件（必须登记）：** 修改了服务工作目录下的源代码、构建脚本、配置文件
（`build.sh`、`go.mod`、`package.json`、`conf/<app>/config.yaml` 等），且该服务被
`conf/runAll.yaml` 托管（含 Go 服务、Django、Vue 前端等）。

**不触发登记：** 仅修改文档（`docs/`、`*.md`）、测试用例（单元测试不在运行进程中）、
或未被 runAll 托管的服务（如 gitService 镜像仓库）。

## 验收标准

1. 修改某服务代码后，登记文件出现该服务名（`scripts/register-precise-restart.sh` 可查）。
2. 页面 `http://10.2.150.68:9999/` 头部出现紫色「精准编译重启」按钮，旁边标签显示
   「已登记 N 个服务」。
3. 点击「精准编译重启」→ 确认弹窗列出登记服务 → 按依赖序逐个编译+重启，进度面板
   实时显示；完成后登记文件被清空。
4. 编译/重启失败的服务**保留**在登记文件中（便于修复后重试），成功服务被清除。
5. 无登记时点击按钮给出提示，不触发编译/重启；`POST /api/precise-restart` 返回 200 `{status:empty}`（**禁止**当成失败横幅「精准编译重启失败」）。点击时若文件空，则 `GET ?fill_from_scan=1` 按脏工作树补登记后再确认。
6. 点击页头「全部重新编译」（`POST /api/build-all` / `Runner.BuildAll`）**正常完成**后，
   视为已消费精准编译登记：清空 `.runall/precise_restart_services.txt` 并写入
   `precise_restart_consumed_at` 水位线（与精准编译重启收尾一致）。中断取消时**不清空**。
7. 分组「全部重新编译」（`POST /api/build-group` / `Runner.BuildGroup`）成功后，仅移除
   **该组内已登记且编译成功**的服务名（`trimRegistrationsAfterGroupBuild`，OPT-20260811-036）；
   该组失败/跳过项、其它组登记与运行期间并发追加的新登记一律保留，**不清空**全局登记
   （避免只编一组却误消其它组待重启项）。
8. 「全部重启」/ 单服务 ↻ **不得**执行 `build_command`，也不得让 `start_command` 隐式编译；
   编译失败的精准编译重启必须保留旧进程（healthy）与登记。

## 强制机制（自动登记）

「修改服务代码 → 登记」由 Stop / SessionEnd hook 自动落地，无需记忆：

- `scripts/lib/register-precise-restart-scan.sh` 随 `auto-commit.sh`（Stop/SessionEnd）
  每次触发执行：扫描 meta 状态中的脏子模块，凡 runAll 托管服务（working_dir
  首段别名命中，`taskFE/app` → `taskFE`）含非文档/非测试变更即自动去重登记。
- 不触发：仅文档（`*.md`/`*.rst`/`docs/`）、仅测试（`tests/`/`__tests__`/
  `*.test.*`/`*.spec.*`/`*_test.go`/`test_*.py`）、working_dir 为 `.` 的基础设施
  服务（docker-* 等）、未被 runAll 托管的工作目录。
- 局限：仅覆盖「工作树仍脏」的变更（编辑后 Stop 即登记）。已提交且工作树干净的
  子模块不会回溯登记，需手动 `scripts/register-precise-restart.sh`。
- 扫描失败不阻断提交（fail-open），stderr 输出告警。

## 实现位置

- 后端引擎：`runAll/src/precise_restart.go`（登记文件读写 + `PreciseRestart` 编排）
- 配置热加载：`runAll/src/precise_restart_reload.go`（执行前 `LoadConfig` 磁盘 YAML，新服务 `EnsureNames` 入 StatusStore，禁止 `Init` 以免把已运行服务打回 pending）
- 全部重新编译消费登记：`runAll/src/runner.go` → `BuildAll` →
  `clearPreciseRestartRegistrationsAfterFullRebuild`（测例
  `build_all_clears_precise_restart_test.go`）
- API：`runAll/src/ui_precise_restart.go`
  - `GET  /api/precise-restart/registrations` 读取登记
  - `POST /api/precise-restart/register` 追加登记（校验服务名）
  - `POST /api/precise-restart` 触发（202 accepted）
  - `POST /api/precise-restart/cancel` 中断
  - `GET  /api/precise-restart/progress` SSE 进度
- 前端：`runAll/src/status_ui/`（按钮在 index.html 页头；`03.js` 执行与 SSE；`05.js` 绑定）
- 登记脚本：`scripts/register-precise-restart.sh`
- 源码机只编译产物（不重启）：`scripts/precise-compile.sh` → `$META_ROOT/deploy-binaries/`（ADR-0052）
- 自动登记：`scripts/lib/register-precise-restart-scan.sh`（集成于 `scripts/lib/auto-commit.sh`）

## 注意事项

- 登记文件位于 `.runall/`（已 gitignore），属运行时状态，不提交。
- 服务别名解析优先级：runAll.yaml `name` 精确匹配 > `conf_app` 匹配 > 工作目录名匹配。
- 未知服务名会被拒绝（脚本与 `/api/precise-restart/register` 均校验），防止登记错名。
- **精准编译重启会先从磁盘重载 `conf/runAll.yaml`**，因此进程启动之后才写入 YAML 的新服务（如新的 `task-events-*` 消费者）可被解析，不必为此单独重启 runAll。重载失败时保留内存配置并打 warn，已知服务仍继续重启。磁盘 YAML 中也不存在的登记名仍报 `unknown service` 并作为 failed 保留。
