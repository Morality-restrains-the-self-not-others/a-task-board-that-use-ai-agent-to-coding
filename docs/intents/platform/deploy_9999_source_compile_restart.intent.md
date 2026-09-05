# 功能意图：部署机 9999 编排源码编译 + 增量安装 + 精准重启

## 背景与目标

源码与部署已按 ADR-0052 分离。本机运行时 9999 在部署根（`http://192.168.1.10:9999/`，`DEPLOY_MODE=1`）。该模式下 `resolveBuildCommand` 清空，页头「精准编译重启」「全部重新编译」不会在部署树 `go build`。

当前同步靠 `.daydaymoney-deploy-seed/update.sh`：`git pull` → 整树覆盖 `artifacts/` → 整树覆盖 `conf-local/` → `up.sh` → `runAll/run.sh`。每次全量拷贝并拉起编排器，改一个服务也要整栈走一遍。

目标：在**部署侧** 9999 点现有两个按钮即可完成「源码树编译 → rsync conf-local → 只装变化产物 →（精准路径）只重启登记服务」。部署机仍禁止 `go build`。不替代 `update.sh` 的首次装配 / 配置仓 `git pull`。

## 范围与边界

- **范围内**：`DEPLOY_MODE=1` 下复用既有 `POST /api/precise-restart`、`POST /api/build-all` 及其 SSE；读 `SOURCE_ROOT/.runall/precise_restart_services.txt`；调源码树 `precise-compile.sh`；增量 `install-local-artifacts`；编译成功后 `rsync -a --delete` `conf-local/`；精准路径随后 `restartService`（无 `build_command`）。
- **范围外**：部署树 `go build`；SSH 远程编译；每次按钮 `git pull` 配置仓；自动重启 `runAll` 自身进程；GitHub Release 上传；新 Python HTTP 接口。

## 约束与风险

- ADR-0027：编译失败保留 last-good 进程；全部重启仍不编译。
- ADR-0052：部署根无源码；编译只发生在 `SOURCE_ROOT`。
- 元规则 42：登记文件仍由源码仓 Agent/hook 写入；部署 9999 **读源码仓登记**，不另建第二份。
- `SOURCE_ROOT` 仅本机路径（与 9999 同宿）；缺省或不可读则按钮失败，不得静默 skip。
- 安装运行中 ELF 必须 copy-then-mv（ETXTBSY）。

## 验收标准

1. `http://192.168.1.10:9999/` 点「精准编译重启」：只编译登记服务 → rsync conf-local → 只安装变化产物 → 只重启那些服务；不 `git pull`、不重启 runAll 编排器。
2. 点「全部重新编译」：源码树 `--all` 编译 + 增量安装；**不**停业务进程（与现网 BuildAll 语义一致）。
3. 编译失败：旧进程继续、登记保留、进度 SSE 报失败。
4. 无 `SOURCE_ROOT` 或源码树缺 `precise-compile.sh`：HTTP/进度明确失败（`data-traceId`），不假装成功。
5. 无登记时精准按钮提示、不编译不重启；`POST` 返回 200 `{status:empty}`，页面**不得**显示「精准编译重启失败」。点击时若登记空则扫描脏工作树（`fill_from_scan=1`），扫到则进入确认框。

## 实施计划

见 `docs/superpowers/specs/2026-09-02-deploy-9999-source-compile-restart-design.md`。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 部署 9999 编排源码编译并增量安装 | — | — | — | — | 运维控制面，不产生业务领域事实 |
| 精准路径重启已安装二进制 | — | — | — | — | 进程编排（ADR-0027），无 MQ |

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-09-02 | 初版。替代低效 `update.sh` 全量同步作为日常编译重启路径。 |
| 2026-09-02 | 空登记 POST 200 `{status:empty}`，禁止「精准编译重启失败」；点击扫描脏工作树。 |
