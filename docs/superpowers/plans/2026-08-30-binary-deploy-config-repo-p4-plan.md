# 实施计划：P4 无源码部署根

- **Date:** 2026-08-30
- **Design:** ADR-0052 / `binary-deploy-config-repo-design.md` I5

验收：`$DEPLOY_ROOT` 无 `.gitmodules`、无服务 `*.go`；runAll `DEPLOY_MODE=1` 只 exec `$DEPLOY_ROOT/bin/` 与配方 `run.sh`；conf 来自 `daydaymoney-deploy`。

## 任务

- [x] T1 红：`DEPLOY_MODE=1` 时 `./bin/<elf>` 的 `working_dir` 变为 conf 父目录（不要求 `taskAuth/` 源码目录）
- [x] T2 红：valueStream `--config ../conf/...` 在部署模式改为 `conf/...`
- [x] T3 绿：`DEPLOY_MODE` 未开时源码树 `working_dir: taskAuth` 行为不变
- [x] T4 配方 `bash dockerInfra/...` / `bash run.sh` 不改成 flat bin
- [x] T5 `materialize-p4-root.sh`：拷贝 ELF + 配方（排除 gitlab-ce、`*.go`、mysql data 用数据 symlink）
- [x] T6 真实 `releases.yaml` sha + `deploy-sync` 写入 `$DEPLOY_ROOT/bin`
- [x] T7 切编排器；`check_p4_deploy_root.sh` 与健康检查通过

本机证据（2026-08-30）：`/proc/<task-auth>/exe` = `/tmp/ram-deploy/bin/taskAuth`，cwd = `/tmp/ram-deploy`，`:8003/api/health/` ok，status 80 healthy / `git-service` empty。

## 事件任务

- [x] 无 MQ — 与意图例外一致
