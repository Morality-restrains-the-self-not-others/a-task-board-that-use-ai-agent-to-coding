# deploy_layout.go Companion

`DEPLOY_MODE=1`（来自 `cutover.env`）**或** 推断为 source-less 根（存在 `bin/runAll` 且不存在 `taskAuth/`）时，`./bin/<elf>` 的 working_dir 改为配置目录的父目录（`$DEPLOY_ROOT`），不要求 `taskAuth/` 等源码目录存在。

生产入口必须加载 `cutover.env`（`run.sh` / `LoadConfig`）。source-less 推断只作布局兜底，避免裸 `./bin/runAll` 把 `chdir` 失败包装成 `fork/exec /usr/bin/bash: no such file or directory`。

- 配方启动（`bash dockerInfra/...`、`bash run.sh`）不改 working_dir。
- `../conf/` 改为 `conf/`（valueStream）。
- 禁止在此编译或 `go build`。
- `bash scripts/runall-stop.sh` 在部署模式改为按 health 端口 `lsof` 结束（源码树里的 `run.sh stop` 不存在）。
- `go test` 经 `TestMain` 清掉进程里的 `DEPLOY_MODE`，避免部署机 shell 污染 BuildGroup 单测。
