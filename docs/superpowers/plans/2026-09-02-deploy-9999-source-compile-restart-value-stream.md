# 价值流 — 部署机 9999 源码编译重启

- **Date:** 2026-09-02
- **Classification:** `skipped_non_product`（ops 控制面，非租户产品价值流）
- **YAML:** 不写入 `valueStream/*.yaml`（无 pytest 业务步骤）

## 增量（唯一交付切片）

运维在 `http://192.168.1.10:9999/` 点击 **精准编译重启** 或 **全部重新编译**：

1. 编排器读 `$SOURCE_ROOT/.runall/precise_restart_services.txt`（或 `--all`）
2. 子进程 unset `DEPLOY_MODE`，跑 `$SOURCE_ROOT/scripts/precise-compile.sh`
3. 成功则 `rsync -a --delete` `conf-local/`，再增量安装 `deploy-binaries/` → `$DEPLOY_ROOT`
4. 精准：`restartService`（空 `build_command`，stop→start→health）；全部重新编译：不杀进程
5. 失败：不 rsync、不 install、保留 last-good 与登记

`update.sh` 仅应急 bootstrap，不在本增量日常路径。
