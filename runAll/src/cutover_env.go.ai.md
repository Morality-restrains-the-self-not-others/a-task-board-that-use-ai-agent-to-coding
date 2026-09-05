# cutover_env.go Companion

`cutover.env`（`up.sh` / `prepare-ram-deploy.sh` 写出）是 clone-run 部署环境的 SSOT。`LoadConfig` 在解析 YAML 前经 bash `source` 加载（保留 `${INFRA_HOST:-…}` 展开），再 `os.Setenv` 白名单键。

- 查找顺序：`CUTOVER_ENV` → `$DEPLOY_ROOT/cutover.env` → `dirname(config)/../cutover.env`（`configPath` 为空则不从 cwd 推断）。
- `go test` 默认不加载，避免本机 `DEPLOY_ROOT` 把 `DEPLOY_MODE=1` 漏进源码仓单测；测例须显式 `CUTOVER_ENV`。
- 白名单含 `SOURCE_ROOT`（ADR-0056：部署 9999 在源码树编译）。
- `preciseRestartFile` / `requireSourceRoot` 在进程 env 缺 `SOURCE_ROOT` 时从 cutover.env 再解析一次（编排器启动后补写 cutover 不必先重启 9999 才能看到源码仓登记）。
- 禁止在 `run.sh` 再手写一份相同的 `export` 列表。
