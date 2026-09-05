# Intent: 二进制部署；运行时配置与源码仓隔离

## 背景与目标

当前开发机与运行时共用同一 monorepo：源码、`conf/`、`dockerInfra/` 配方、Go `bin/` 产物同树。`confload.FindMonorepoRoot()` 靠 `conf/base.yaml` / `dataMigrate/` / `.gitmodules` 定位根；runAll `working_dir` 指向各服务源码子目录。结果是：部署机必须 clone 源码，运行时配置随源码仓扩散，无法隔离。

目标：部署机**不 clone 源码仓**；只放置编译产物（Go 二进制、taskFE 静态包、taskEvents worker）与**独立配置仓**中的运行时 `conf/` + Docker 配方；源码仓只保留 schema/示例。

## 范围与边界

- **范围内**：Go 业务进程、taskEvents intent worker、taskFE nginx 静态资产、Docker 基础设施配方（MySQL/Redis/Kafka/GitLab compose + run.sh）、runAll 编排 YAML、`confload` 根定位、产物发布与版本钉。
- **范围外**：不把业务服务改成 systemd unit（仍用 runAll）；不把 GitLab-CE 源码树搬进配置仓；不把生产密钥提交进配置仓；不改变各服务数据所有权与公网 API。

## 约束与风险

- 元规则 42（conf SSOT）与 29（仅读本目录 conf）须改为「运行时 SSOT = 配置仓」；源码仓 `conf.example/` 仅开发/引导。
- ADR-0027 在部署机上的对应物是「拉取产物再 swap」，禁止在部署机 `go build`。
- `dataMigrate/*.sql` 随代码版本走产物包，不进配置仓（配置仓钉版本）。
- 密钥：`conf-local/`（gitignore），不进 Git。

## 验收标准

1. 部署根目录无 `.gitmodules`、无服务 `*.go` 源码时，runAll 仍能按配置仓启动已钉版本的二进制。
2. 运行时 `conf/` 只来自配置仓；源码仓无生产 `base.yaml` / `runAll.yaml`。
3. 升级路径：改配置仓版本钉 → 拉产物 → 热替换；失败则 last-good 不变。
4. 本地开发仍可从源码树编译；通过 `CONF_ROOT`/`DEPLOY_ROOT` 指向配置仓。

## 实施计划

见 `docs/superpowers/specs/2026-08-30-binary-deploy-config-repo-design.md`。

## 业务意图 → 事件对照

> 运维交付与配置隔离，不产生业务领域事件（与 ADR-0035 / 重启-编译分离同一类例外）。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 部署机仅用二进制 + 独立配置仓运行 | — | — | — | — | 运维交付拓扑，不产生业务领域事实 |

## 变更记录

- 2026-08-30：新增。头脑风暴：部署机无源码；整棵运行时 conf 迁出；范围含 Go / FE / Events / Docker 配方。
- 2026-08-30：P4 在本机 `/tmp/ram-deploy` 验收：无 `.gitmodules`、无服务 `*.go`；`DEPLOY_MODE=1` 下 `taskAuth` 的 exe 为 `$DEPLOY_ROOT/bin/taskAuth`、cwd 为部署根；conf 来自 `daydaymoney-deploy`。GitHub Release 上传仍可用 `publish-deploy-artifacts.sh`（本机 pins 为 `file://`）。
- 2026-08-31：配置仓可独立装配：`envs/current/` 含 Docker/GitLab 配方、`dataMigrate`、`db` 脚本与 `scripts/up.sh`。部署机 `git clone daydaymoney-deploy && ./scripts/up.sh`，不再依赖源码树 ram-work。MySQL datadir 禁止链到 ram-work。
- 2026-08-31：17 个 Go ELF 已发布为 GitHub Release 标签 `deploy-20260831`；`releases.yaml` 使用 `github://…@deploy-20260831`（`sha` 为内容哈希）。`up.sh` 在缺 `bin/runAll` 时用 `gh release download` 引导。
- 2026-08-31：`taskEvents-bin.tar.gz` 与 `taskFE-dist.tar.gz` 纳入同一 tag；pin 带 `dest` + `unpack: tar.gz`。`TASK_EVENTS_BIN` / `TASKFE_PUBLIC` 仅作本机覆盖。
- 2026-09-02：源码机 `scripts/precise-compile.sh` 按登记/`--all`/服务名编译并写入 `deploy-binaries/`（`COLLECT_SKIP_SHA=1`）；部署机仍禁止 `go build`。
- 2026-09-02：部署 9999「精准编译重启 / 全部重新编译」拟编排源码编译 + 增量安装（见 `deploy_9999_source_compile_restart`）；日常不再走 `update.sh` 全量拷贝。
- 2026-09-02：本机 live `$DEPLOY_ROOT` 改为 `$HOME/bin/daydaymoney-deploy`；`/tmp/ram-deploy` 退役，脚本默认不再指向该路径。
