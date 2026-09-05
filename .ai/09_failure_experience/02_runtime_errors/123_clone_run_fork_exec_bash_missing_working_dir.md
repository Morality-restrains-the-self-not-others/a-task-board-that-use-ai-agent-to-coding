# [运行时] clone-run 全部重启误报 fork/exec /usr/bin/bash

## 基本信息

- 版本：1.0.0
- 案例编号：RT-20260901-0123
- 录入日期：2026-09-01
- 最后更新：2026-09-01
- 录入人：Cursor Agent

## 现象

- 新目录 `.daydaymoney-deploy-seed` / `~/bin/daydaymoney-deploy` 部署后，http://192.168.1.10:9999/ 「全部重启」失败：
  `LAUNCH_PROCESS_EXITED: fork/exec /usr/bin/bash: no such file or directory`
- 受影响服务：`task-auth`、`task-bill`、`task-referral`、`task-project-service`、`task-agent-support`、`task-ai-endpoint`、`task-container-gateway`、`value-stream`、`go-relay`
- 宿主机 `ls /usr/bin/bash` 正常（ELF、可执行）

## 环境与上下文

- 部署根：`/home/ljy/bin/daydaymoney-deploy`（source-less，无 `taskAuth/` 等源码目录，ELF 在 `bin/`）
- 编排器：`./bin/runAll conf/runAll.yaml`（cwd=部署根，**未** `source cutover.env`，environ 无 `DEPLOY_MODE`）
- `cutover.env` 里有 `DEPLOY_MODE=1`，但直接 exec 二进制不会读该文件
- Go `exec.Command(bash, "-c", start)` + `cmd.Dir = <missing taskAuth>` → `chdir` 失败；旧包装文案为 `fork/exec /usr/bin/bash: no such file or directory`

## 根因

1. `conf/runAll.yaml` 源码仓路径：`working_dir: taskAuth` + `start_command: ./bin/taskAuth`。
2. `applyDeployLayout` 只在 `DEPLOY_MODE=1` 时把 working_dir 改成部署根。
3. 未带该 env 时 resolved working_dir 指向不存在的绝对路径。
4. start 路径未走已有的 `lifecycleWorkDir`（stop 才跳过缺失目录）。
5. 表象像「系统没有 bash」。

## 修复

1. 根上存在 `bin/runAll` 且不存在 `taskAuth/` 时，即使未设 `DEPLOY_MODE` 也启用 deploy layout（含 `../conf/` → `conf/`）。
2. `startCommandDir`：`./bin/` 启动遇缺失 working_dir 则继承编排器 cwd；其它配方明确报 `working_dir %q does not exist`。
3. `runAll/run.sh` 在 clone-run 树 **source `$ROOT/cutover.env`**（缺文件失败）；Go `LoadConfig` 同样 source 配置旁的该文件。不要在 `run.sh` 复述 `export` 列表。

## 验收

```bash
go test ./src -count=1 -timeout 60s -run 'TestResolveWorkingDirs_SourceLess|TestSourceLessFlatBinRoot|TestStartCommandDir_|TestStartAndCheck_ClassifiesLaunchProcessExited'
# 热替换部署根 bin/runAll 后用 ./runAll/run.sh 拉起编排器，9999 全部重启不再出现 bash ENOENT
```

## 预防

- 禁止只看 `fork/exec bash` 就去装 bash；先 `stat` 该服务 `working_dir`。
- clone-run 须先 `./scripts/up.sh` 写出 `cutover.env`，再 `./runAll/run.sh`（脚本会 source 该文件）。不要复述一份 env，也不要先手 source 再启动。
