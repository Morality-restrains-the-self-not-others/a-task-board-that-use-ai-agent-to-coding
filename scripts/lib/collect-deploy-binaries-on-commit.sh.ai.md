# collect-deploy-binaries-on-commit.sh Companion

pre-commit / auto-commit 调用的 fail-open 归集入口。刷新 gitignored `$META/deploy-binaries/`。

- `SKIP_COLLECT_DEPLOY_BINARIES=1` 或 `CI=true` → 直接退出。
- 本机没有任何 `runAll` ELF 源 → 跳过（不阻断提交）。
- 调用 `collect-deploy-binaries.sh` 时带 `COLLECT_SOFT=1`。
- 仅当 `COLLECT_DEPLOY_BINARIES_REQUIRED=1` 时归集失败才非零退出。
