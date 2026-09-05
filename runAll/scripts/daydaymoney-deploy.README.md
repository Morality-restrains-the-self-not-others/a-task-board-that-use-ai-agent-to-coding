# daydaymoney-deploy

Private **runtime** repo (ADR-0052): clone and run **without** the source monorepo.

Hand-copy **two trees**: `conf-local/` (YAML + PEM) and `deploy-binaries/` (ELFs + tarballs). GitHub Release download is optional fallback (`FORCE_DEPLOY_SYNC=1`).

On the **source** machine (this monorepo) compile registered services into `deploy-binaries/` (does not restart):

```
bash scripts/precise-compile.sh                 # .runall/precise_restart_services.txt
bash scripts/precise-compile.sh task-auth taskFE
bash scripts/precise-compile.sh --all
```

This host's live `$DEPLOY_ROOT` is `$HOME/bin/daydaymoney-deploy` (not `/tmp/ram-deploy`). When source and deploy live on the **same host**, `cutover.env` must set `SOURCE_ROOT` (written by `write-cutover-env.sh`; defaults to `/tmp/ram-work` if that tree exists). Then **http://\<host\>:9999/** 「精准编译重启」and 「全部重新编译」compile in `$SOURCE_ROOT`, rsync `conf-local/`, incrementally install artifacts, and (precise only) restart those processes. Do **not** use daily `update.sh` for that; keep `update.sh` as emergency bootstrap only.

`deploy-binaries/` is also refreshed **on every git commit** from already-built ELFs (meta pre-commit). Incremental: unchanged files are skipped. Disable with `SKIP_COLLECT_DEPLOY_BINARIES=1`. Copy-only (no compile):

```
bash scripts/collect-deploy-binaries.sh   # writes ./deploy-binaries/ (gitignored)
```

On the **new node**:

```
git clone git@github.com:task2money/daydaymoney-deploy.git
cd daydaymoney-deploy
rsync -a <src>/conf-local/ ./conf-local/
rsync -a <src>/deploy-binaries/ ./artifacts/
# Edit conf-local/infra-host.env if this host's IP differs
./scripts/up.sh                  # layout + overlay + install local artifacts
# Start Docker infra (MySQL/Redis/Kafka/GitLab) then:
#   ./runAll/run.sh
# (run.sh sources cutover.env from up.sh, then detaches with setsid -f nohup.
#  Same as `START=1 ./scripts/up.sh`. Do not hand-source cutover.env first.)
# Empty DB → http://<this-host>:9999/ → 初始化全部数据库 (confirm=INIT_ALL)
```

`conf-local/` and `artifacts/` are gitignored. Never commit PEMs, secrets, or ELFs. See `secrets.example/README.md`.

`./scripts/up.sh` does **not** bind ports unless `START=1`. It installs files from `./artifacts/` (or `./deploy-binaries/`, or `ARTIFACTS_DIR`): ELFs → `bin/`; `taskEvents-bin.tar.gz` → `taskEvents/bin`; `taskFE-dist.tar.gz` → `taskFE/app/public`; `taskAiProvider-frontend-dist.tar.gz` → `taskAiProvider/frontend/dist`. Installing over a running ELF uses copy-then-`mv` (Linux otherwise returns ETXTBSY /「文本文件忙」). Missing `bin/runAll` after install is a hard error. After unpack, `up.sh` fails if `taskEvents/run.sh` `INTENT_PATHS` has no matching `task-events-*` worker (re-collect on the source machine). GitHub download remains a fallback when those directories have no `runAll`.

## Layout

| Path | Role |
|------|------|
| `envs/<env>/conf/` | Non-secret YAML (git) |
| `conf-local/` | Host secrets + PEMs (hand copy, gitignore) |
| `envs/<env>/releases.yaml` | GitHub Release pins (optional fallback) |
| `scripts/up.sh` | layout + conf-local overlay + local artifact install |
| `artifacts/` | Hand-copied Release payloads (gitignore) |
| `bin/` | Installed ELFs (gitignore) |

`$DEPLOY_ROOT` may be this clone. `layout.sh` then symlinks `conf` → `envs/current/conf`, and writes `conf-local/runAll.yaml` `logging.file_root` to `$DEPLOY_ROOT/logs` (does not edit git-tracked YAML).
