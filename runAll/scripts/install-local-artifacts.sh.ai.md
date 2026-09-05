# install-local-artifacts.sh Companion

把手工拷贝的 Release 产物装进 `$DEPLOY_ROOT`。不访问 GitHub。

- ELF（非 `.tar.gz`）→ `$DEPLOY_ROOT/bin/` 并 `chmod +x`；若源目录不是 `artifacts/` 则同时拷到 `artifacts/`。覆盖已在跑的 ELF 时 **cp 到旁路再 `mv`**（避免 ETXTBSY /「文本文件忙」）。
- `taskEvents-bin.tar.gz` → `taskEvents/bin`（先解到临时目录再 `mv` 每个文件，同样避开正在跑的 worker）；`taskFE-dist.tar.gz` → `taskFE/app/public`；`taskAiProvider-frontend-dist.tar.gz` → `taskAiProvider/frontend/dist`。其它 `*.tar.gz` 若 `releases.yaml` 有 `dest` + `unpack: tar.gz` 则按 pin 解包，否则跳过。
- 解包后若存在 `taskEvents/run.sh` 的 `INTENT_PATHS`，须每个 path 下有 `task-events-*`，否则失败（避免 start-all 空等 60s READINESS_TIMEOUT）。
- 归档内含 `..` 或绝对路径则失败。
- 跳过 `*.sha`、`README*`、`MANIFEST*`、`*.workers`。
