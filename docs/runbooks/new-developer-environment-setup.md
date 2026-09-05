# 新开发者环境搭建 — Git Hooks 一键激活

> 适用：新克隆 meta-repo 后，让全部仓库的 git hooks 门禁立即生效（git-hooks-version-control v13）。

## 1. 克隆与子模块初始化

```bash
git clone <ram-work 远端地址> && cd ram-work
git submodule update --init --recursive   # 检出全部子仓
```

## 2. 一键激活全部 hooks（一条命令）

```bash
bash runAll/scripts/install-hooks-all.sh
```

- 遍历主仓 + 全部子仓，对每个含 `.githooks/` 的仓库执行 `git config core.hooksPath .githooks`
- 幂等：已激活的仓库显示 `✅ ... already active`，可重复执行
- 校验模式：`bash runAll/scripts/install-hooks-all.sh --check`（只检查不修改，exit 非 0 表示有未激活仓）

## 3. 验证

```bash
bash runAll/scripts/install-hooks-all.sh --check     # 全绿 = 37 个仓库已激活
python3 db/scripts/ci/check_subrepo_random_precommit_hooks.py   # CI 同标准校验
```

## 4. 门禁一览（激活后自动生效）

| 钩子 | 生效位置 | 作用 |
|------|---------|------|
| `pre-commit` | 主仓 + 全部子仓 | 随机单元测试抽测（30%）+ 子仓库优先提交门禁（主仓） |
| `commit-msg` | 主仓 + 全部子仓 | bug-fix 提交必须携带对应回归单测（`.ai/41`） |
| `pre-push` | 主仓 | 未推送的子仓库提交阻断（`.ai/32`） |
| docs 专属 | docs 子仓 | 架构视图自动归档（保留最近 5 版） |

钩子执行时打印版本戳（如 `▸ [hook:pre-commit v1.0.0]`），版本清单见各仓 `.githooks/HOOK_VERSION`。

## 5. 钩子源与分发

- **入库真源**：每个仓库的 `.githooks/`（git 跟踪，可审计）
- **模板 SSOT**：主仓 `scripts/hooks/templates/`（6 语言 pre-commit + commit-msg + 共享抽测库）
- **重新部署**（模板升级后）：`bash scripts/deploy_repo_random_precommit.sh`（渲染至各仓 `.githooks/`）

## 6. 注意事项

- 旧模式 `.git/hooks/` 副本已退役（v13）——不要在 `.git/hooks/` 手工放钩子，改动一律走 `.githooks/`
- 变更 hooks 后请提交对应子仓（钩子入库是本项目的强制约定）
- 相关规则：`.ai/01_project_constraints/32_submodule_commit_order.md`（子仓库优先提交）、`41_bug_fix_unit_test_required.md`（bug-fix 回归单测）、`42_ramsync_git_operation_order.md`（ramsync 操作顺序）
