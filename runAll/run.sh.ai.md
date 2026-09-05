# run.sh Companion

clone-run（`$ROOT/bin/runAll` 可执行且无 `taskAuth/`）**必须** `source $ROOT/cutover.env`（或 `CUTOVER_ENV`）。文件由 `./scripts/up.sh` 生成；缺失则失败，禁止在本脚本复述 `DEPLOY_MODE` / `RUNALL_*` 列表。源码仓不 source 该文件。
