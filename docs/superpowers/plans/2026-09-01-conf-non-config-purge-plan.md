# Plan: conf.git 非配置资产收口

- **设计**: `docs/superpowers/specs/2026-09-01-conf-non-config-purge-design.md`（accepted）
- **goal-mode 裁剪**: Step 2/4/5/6 SKIP（无新 endpoint、无价值流步骤变更、无领域模型）；Step 3 SKIP；Step 1 已完成。

## Tasks

- [x] 允许清单门禁 `check_conf_tracked_allowlist.py` + 自测
- [x] `check_subrepo_random_precommit_hooks.py` 对 conf 豁免全家桶、要求薄 pre-commit
- [x] 迁表征测试与 `check_conf_sync.sh` 到 `db/scripts/ci/`
- [x] tencent-sh-1 compose → `gitService/docker-compose.tencent-sh-1.yml`
- [x] `generateKey.sh` → taskChromePlugin；删除 plist
- [x] SKIP_HOOK_REPOS + 瘦身 conf `.githooks`
- [x] 更新 README / ai.md / 元规则 47 验收命令
