# 实施计划 — 部署机 9999 源码编译重启

- **Date:** 2026-09-02

## Task 1 — `SOURCE_ROOT` 配置面（Red → Green）

- [x] `cutoverEnvKeys` + `cutoverSourceScript` 含 `SOURCE_ROOT`
- [x] `write-cutover-env.sh` 写出该键（本机目录存在则默认 `/tmp/ram-work`）
- [x] `EXPECTED_CUTOVER_KEYS` 同步
- [x] `preciseRestartFile`：`DEPLOY_MODE` 且无 `RUNALL_PRECISE_RESTART_FILE` 时用 `$SOURCE_ROOT/.runall/`

## Task 2 — 源码编译 + rsync + 增量安装（Red → Green）

- [x] T3：空 `SOURCE_ROOT` → PreciseRestart/BuildAll 错误含 `SOURCE_ROOT`
- [x] T8 / T8b：成功 rsync 对齐；失败不改 dest `conf-local`
- [x] T6 / T7：`install-local-artifacts.sh` 同 size+mtime 打 `unchanged` 且不 `mv`；变化 copy-then-mv
- [x] 子进程 unset `DEPLOY_MODE`；`DEPLOY_MODE` 下跳过 `SyncMonorepoConf`

## Task 3 — 按钮语义（Red → Green）

- [x] T1：登记服务 → compile 参数含该名 → install → restart；部署树 `build.sh` 不跑
- [x] T2：compile 非零 → 不 restart、登记保留
- [x] T4：空登记 → 不 compile
- [x] T5：BuildAll `--all` + install，零次进程重启
- [x] T9：失败进度/错误仍走既有 SSE / `data-traceId`

## Task 4 — 文档

- [x] `daydaymoney-deploy.README.md` 说明 9999 按钮走源码编译，不再日常 `update.sh`

无新 MQ 事件任务（意图 no-event）。
