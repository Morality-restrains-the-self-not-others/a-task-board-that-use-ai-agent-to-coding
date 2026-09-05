# source_compile_deploy.go Companion

`DEPLOY_MODE=1` 时 9999 的精准编译重启 / 全部重新编译不在部署树 `go build`（ADR-0052 / ADR-0056）。

编排：`prepareDeploySourceArtifacts` → `$SOURCE_ROOT/scripts/precise-compile.sh`（子进程 **unset `DEPLOY_MODE`**）→ 成功才 `rsync -a --delete` `conf-local/` → `install-local-artifacts.sh`。

- 缺 `SOURCE_ROOT`：fail-fast，禁止静默 skip。进程 env 为空时从 `cutover.env` 再解析（与 `preciseRestartFile` 同源）。
- 日志只打路径名，不打印 `conf-local` 文件内容。
- 测试通过替换 `prepareDeploySourceArtifactsFn` 注入假 compile。
