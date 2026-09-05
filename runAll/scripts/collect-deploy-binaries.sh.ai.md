# collect-deploy-binaries.sh Companion

把现网/预发布 ELF 与 tarball 归集到源码仓 `deploy-binaries/`（gitignore，禁止 commit）。

- 默认源（取 **mtime 最新**）：`$COLLECT_STAGING`（默认 `/tmp/daydaymoney-release-20260831-conf-local`）、`$RAM_DEPLOY/artifacts`（默认 `$DEPLOY_ROOT` 或 `$HOME/bin/daydaymoney-deploy`）、`META_ROOT/bin`、`META_ROOT/<elf>/bin/<elf>`。
- `COLLECT_SRC` 若设置则**只**从该目录取文件（测试用）。
- 增量：dest 已存在且 size 相同、mtime ≥ source 则跳过（提交钩子可重复跑）。
- `taskEvents-bin.tar.gz`：若 `$META_ROOT/taskEvents/bin` 或 `$RAM_DEPLOY/taskEvents/bin` 已有 `task-events-*` worker，则**优先从 live 树打包**（worker 数量变化或 live mtime 新于包时重打），避免旧 tarball 永久挡住新 intent。否则才 `copy_named` / 缺包时从 `$RAM_DEPLOY` 打包。sidecar `$DEST/taskEvents-bin.tar.gz.workers` 记 worker 数，避免每次 `tar -tzf`。
- `taskAiProvider-frontend-dist.tar.gz`：若 `$META_ROOT/taskAiProvider/frontend/dist/index.html` 或 `$RAM_DEPLOY/.../dist/index.html` 存在，则从 live 树打包；live `index.html` mtime 新于包时重打。否则 `copy_named` / 缺包时从 `$RAM_DEPLOY` 打包。
- `taskFE-dist.tar.gz`：若 `$META_ROOT/taskFE/app/public/index.html` **或** `public/html/index.html`（atomic-vite `html` → `releases/<id>`）存在，则从该 `public/` 打包；否则 `copy_named` / 缺包时从 `$RAM_DEPLOY/taskFE/app/public` 打包。不得因根目录没有 `index.html` 而跳过源码树、打成部署树旧包。
- 缺 `runAll`：默认失败；`COLLECT_SOFT=1` 时警告并 exit 0（钩子用）。
- `COLLECT_SKIP_SHA=1`：跳过与 `conf.example/releases.yaml` 的 SHA 对照（源码机精准编译产物必然与 pin 不同）。
- 写出 `MANIFEST.txt`。

