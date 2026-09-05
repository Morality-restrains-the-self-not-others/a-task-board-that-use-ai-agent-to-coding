# up-from-config-repo.sh Companion

`daydaymoney-deploy` 独立拉起入口：layout → 手工密钥 overlay → 可选产物 → `cutover.env`。

- 密钥由运维**手工提供**：clone 根 `conf-local/` 或 `secrets/conf-local/`（YAML + 网关/OIDC PEM）。缺 `bin/runAll` 且 `SYNC_ARTIFACTS=1` 时硬失败。
- 默认 **不** `START=1`（避免占端口）。拉起编排器用 `./runAll/run.sh`（**依赖** `cutover.env`，勿再手 source）。`deploy-sync.sh` 与 Go `LoadConfig` 同样 source 该文件。禁止在 `run.sh` 复述 `export` 列表。
- 产物优先手拷：`./artifacts/`、`./deploy-binaries/` 或 `ARTIFACTS_DIR`（`install-local-artifacts.sh`：ELF → `bin/`，已知 `*.tar.gz` 解包）。有 `runAll` 时跳过 GitHub；`FORCE_DEPLOY_SYNC=1` 才再走 `deploy-sync`。
- 测试与一次性 shell **必须 unset `DEPLOY_ROOT`**（勿继承 `cutover.env`），否则会把 fixture rsync 进 live 树。
- 跨机 fallback：`releases.yaml` 的 `github://…@<Release tag>` + `GITHUB_TOKEN` / `gh auth`。无本地 `runAll` 时 `up.sh` 会先拉该 tag 的 `runAll`（`GODEBUG=http2client=0` 强制 HTTP/1.1，失败则 `curl --http1.1`）再 `deploy-sync`。
- `conf-local/infra-host.env` 可设 `INFRA_HOST=`；写入 `cutover.env` 默认值。
- `TASK_EVENTS_BIN` / `TASKFE_PUBLIC` / `TASKAIPROVIDER_FRONTEND` 在产物安装 **之后** overlay，仅作本机覆盖。
