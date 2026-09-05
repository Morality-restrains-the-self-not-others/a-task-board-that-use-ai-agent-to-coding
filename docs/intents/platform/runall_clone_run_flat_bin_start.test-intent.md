# 测试意图: clone-run 无源码目录时仍能拉起 ./bin ELF

## 测试目标

证明 source-less 部署根在未设置 `DEPLOY_MODE` 时仍改写扁平 ELF 的 working_dir / conf 路径；start 不会对缺失源码目录执行 chdir；非扁平缺失目录的错误可诊断。

## 测试分层

| 层 | 覆盖 |
|----|------|
| 单元 | `sourceLessFlatBinRoot`、`resolveWorkingDirs`、`startCommandDir`、`applyCutoverEnvFile` / `LoadConfig`+`CUTOVER_ENV`、`TestStartAndCheck_ClassifiesLaunchProcessExited` |
| 脚本契约 | `runAll/run.sh` clone-run 缺 `cutover.env` 失败；有文件则 source 后带 `DEPLOY_MODE` |

## 用例矩阵

| ID | 给定 | 当 | 则 |
|----|------|----|----|
| CR-1 | 根有 `bin/runAll`、无 `taskAuth/`、`DEPLOY_MODE` 空 | `resolveWorkingDirs` | `task-auth` working_dir = 该根；value-stream start 含 `conf/value-stream.yaml` 且不含 `../conf/` |
| CR-2 | 根同时有 `bin/runAll` 与 `taskAuth/` | `sourceLessFlatBinRoot` | false（源码仓不被误判） |
| CR-3 | `StartCommand=./bin/taskAuth`，working_dir 不存在 | `startCommandDir` | dir 空、err nil |
| CR-4 | `Command=sleep 30`，working_dir 不存在 | `startAndCheck` | 失败，文案含 `working_dir`，phase=launch，code=PROCESS_EXITED |
| CR-5 | 有 `bin/runAll`、无 `taskAuth/`、无 `cutover.env` | 执行 `runAll/run.sh` | 非零退出，stderr 含 `cutover.env` 与 `up.sh` |
| CR-6 | 同时有 `bin/runAll` 与 `taskAuth/` | 同上 | `DEPLOY_MODE` 为空（源码仓不被误判） |
| CR-7 | 有 `bin/runAll`、无 `taskAuth/`、有 `cutover.env` | 执行 `runAll/run.sh`（skip-build 前 dump） | `DEPLOY_MODE=1` 等键来自该文件 |
| CR-8 | `CUTOVER_ENV` 指向含 `DEPLOY_MODE=1` 的文件 | `LoadConfig` | 进程 env 出现 `DEPLOY_MODE=1` |

## 数据与环境

- Go 测例用 `t.TempDir()`，不碰 live `/home/ljy/bin/daydaymoney-deploy`。
- 不启动真实 ELF。

## 通过标准

- `go test ./src -count=1 -timeout 60s -run 'TestResolveWorkingDirs_SourceLess|TestSourceLessFlatBinRoot|TestStartCommandDir_|TestStartAndCheck_ClassifiesLaunchProcessExited|TestApplyCutoverEnvFile_|TestLoadConfig_SourcesCutover'`
- `python3 -m pytest runAll/scripts/tests/test_runall_run_sh_sources_cutover.py`

## 业务意图 → 事件对照（测试）

运维编排器无 MQ 事件；不断言 Kafka。
