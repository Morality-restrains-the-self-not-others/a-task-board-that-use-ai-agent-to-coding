# test_runall_run_sh_sources_cutover.py Companion

clone-run（有 `bin/runAll`、无 `taskAuth/`）时 `runAll/run.sh` **必须** source `$ROOT/cutover.env`；缺文件非零退出。有文件时 dump 出的 `DEPLOY_MODE`/`RUNALL_*` 来自该文件，而不是脚本内复述的 export。源码树（存在 `taskAuth/`）不得被设成 deploy 模式。
