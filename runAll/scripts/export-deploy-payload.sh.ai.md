# export-deploy-payload.sh Companion

将源码仓中的**配方**（无 `*.go`、无 `gitlab-ce`、无 MySQL datadir、无 PEM）同步进 `daydaymoney-deploy` 的 `envs/<env>/`。

- `trae-agent/onlineServiceJS/` 必须整树同步（`run.sh`、`Dockerfile`、`buildDocker.sh`、`src/`），并带上 `pyproject.toml` / `trae_agent`（go-relay 标记 + 镜像构建上下文，OPT-20260830-021）。**排除** `docker/code-server/*.tar.gz`（GitHub 单文件 100MB 限制；镜像构建时由 `fetch-code-server-bundles.sh` 拉取）。
- 不把二进制 commit 进配置仓；ELF 默认手拷到 clone 根 `artifacts/`（源码仓 `deploy-binaries/` 归集）。GitHub Release + `releases.yaml` `github://owner/repo/asset@tag` 仅为无本地产物时的 fallback。
- 不覆盖配置仓已有 `envs/<env>/conf/`（conf 由配置仓自己维护）。
- 同步后配置仓 `scripts/up.sh` 可在无 ram-work 的机器上 layout。`write-cutover-env.sh` **必须**与 `up.sh` 同目录导出（`up.sh` 经 `$SCRIPT_DIR` 调用）；缺文件则 `update.sh` 无法写出 `SOURCE_ROOT`（ADR-0056）。
- `secrets.example/README.md` 来自 `daydaymoney-secrets.example.md`（`conf-local/` 放置方法）。禁止再导出 HOST_SECRETS 登记册。
