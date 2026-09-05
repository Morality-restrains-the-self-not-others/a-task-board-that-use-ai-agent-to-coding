# test_collect_on_commit.py Companion

断言提交钩子入口：`SKIP_COLLECT_DEPLOY_BINARIES` 跳过、无 ELF 源跳过、有源则写入 `$META/deploy-binaries/`。用 `SESSION_META_ROOT` 隔离 live 树。
