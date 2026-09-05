# Intent: clone-run 无源码目录时仍能拉起 ./bin ELF

## 背景与目标

用 `.daydaymoney-deploy-seed` / `daydaymoney-deploy` 在新目录部署后，http://192.168.1.10:9999/ 「全部重启」对 `task-auth`、`task-project-service`、`value-stream` 等失败：

`[LAUNCH_PROCESS_EXITED]: fork/exec /usr/bin/bash: no such file or directory`

宿主机 `/usr/bin/bash` 存在。真实原因是 `working_dir: taskAuth` 等源码目录在 source-less 树中不存在；Go `exec.Command` 的 `chdir` 失败会被包装成对 bash 的 `fork/exec` ENOENT。`applyDeployLayout` 原先只在 `DEPLOY_MODE=1` 时改写 working_dir；若运维直接 `./bin/runAll conf/runAll.yaml` 而未 `source cutover.env`，布局改写不会发生。

目标：clone-run 根只要有 `bin/runAll` 且没有 `taskAuth/`，即使未导出 `DEPLOY_MODE`，也应按扁平 `bin/` 布局启动；错误文案须点出缺失的 `working_dir`，不得伪装成「没有 bash」。

## 范围与边界

- **范围内**：`applyDeployLayout` 推断 source-less 根；start 对缺失 working_dir 的 `./bin/` 命令继承编排器 cwd；非扁平启动缺失目录须明确报错；clone-run 入口（`run.sh`、`deploy-sync.sh`、Go `LoadConfig`）**直接 source `cutover.env`**，不在脚本里复述 `export` 列表。
- **范围外**：补齐缺失 ELF、空库 DDL（仍走 9999 初始化）、业务服务自身崩溃。

## 约束与风险

- 源码仓（存在 `taskAuth/`）不得被误判为 clone-run，否则 `go test` / 本地编译 working_dir 会被改掉。
- `value-stream` 的 `../conf/` 必须改成 `conf/`，只跳过 chdir 不够。
- 运维进程生命周期，无租户域事件。

## 验收标准

1. 临时根含 `bin/runAll`、不含 `taskAuth/`、未设 `DEPLOY_MODE` 时，`./bin/taskAuth` 的 working_dir 解析为该根；value-stream start 含 `conf/value-stream.yaml`。
2. `./bin/taskAuth` + 不存在的 working_dir → `cmd.Dir` 为空（继承 cwd），不把缺失路径交给 `exec`。
3. 非 `./bin/` 启动 + 缺失 working_dir → 错误含 `working_dir`，失败码仍为 `LAUNCH_PROCESS_EXITED`。
4. `runAll/run.sh` 在 clone-run 树无 `cutover.env` 时失败并提示 `./scripts/up.sh`；有该文件时 source 后 `DEPLOY_MODE=1`。源码树（有 `taskAuth/`）不强制 deploy 模式。

## 实施计划

1. `sourceLessFlatBinRoot` + `applyDeployLayout` 在无 env 时启用。
2. `startCommandDir` 区分扁平 ELF 与其它配方。
3. `run.sh` / `deploy-sync.sh` / `LoadConfig` 在需要处 source `cutover.env`（不把主机绝对路径写进配方，也不复述 export 列表）。
4. 单测覆盖上述路径。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| clone-run 无源码目录时仍能拉起 ./bin ELF | — | — | — | — | 运维编排器启动路径，不产生业务领域事件 |

## 变更记录

- 2026-09-01：初稿。clone-run 重启误报 bash ENOENT。
- 2026-09-01：`run.sh` 自行导出 clone-run 环境，不再把启动正确性绑在 `cutover.env` 上。
- 2026-09-01：撤回「复述 export」；`cutover.env` 作为依赖在 `run.sh` / `deploy-sync.sh` / `LoadConfig` 直接 source。
