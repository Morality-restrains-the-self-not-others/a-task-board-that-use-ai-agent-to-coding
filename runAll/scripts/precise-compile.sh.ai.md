# precise-compile.sh Companion（runAll/scripts）

源码树执行 runAll `build_command`，再 `COLLECT_SKIP_SHA=1` 归集到 `$META_ROOT/deploy-binaries`。不重启进程。默认读 `.runall/precise_restart_services.txt`。
