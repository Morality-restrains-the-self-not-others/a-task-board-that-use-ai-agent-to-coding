# test_collect_deploy_binaries.py Companion

`COLLECT_SRC` + 空 `RAM_DEPLOY` / `COLLECT_STAGING` 隔离 live 树。断言归集 ELF/tarball、写 `MANIFEST.txt`、增量跳过 unchanged、多源取较新 mtime；无 `runAll` 失败，`COLLECT_SOFT=1` 则放行。live `taskEvents/bin` worker 数多于已有包时必须重打包（含 `project_deleted/1_detach_task_projects`）；sidecar 计数一致且 dest 更新则跳过。live `taskAiProvider/frontend/dist/index.html` 存在时必须打 `taskAiProvider-frontend-dist.tar.gz`；live mtime 新于已有包时必须重打。
