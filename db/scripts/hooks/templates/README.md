# 子仓随机单元测试 pre-commit 模板

源自 `task2app/scripts/hooks/pre-commit` 策略：暂存相关测例优先跑，其余按比例随机抽测；失败则阻止提交。

| 模板 | 适用 |
| --- | --- |
| `pre-commit.go` | 纯 Go（`*_test.go`） |
| `pre-commit.python` | 纯 Python（`test_*.py`） |
| `pre-commit.node` | Node / vitest（`*.test.js` / `*.unit.test.js`） |
| `pre-commit.go_js` | Go + frontend unit |
| `pre-commit.go_python` | Go + Python |
| `pre-commit.placeholder` | 暂无单测资产 |

部署（仓库根）：

```bash
bash scripts/deploy_repo_random_precommit.sh          # 全部子仓
bash scripts/deploy_repo_random_precommit.sh taskAuth # 指定子仓
```

规则：`.ai/01_project_constraints/28_commit_random_unit_test_debt_fix.md`

## 共享抽测库（v65 起）

`lib/random_test_runner.sh` — 随机抽测逻辑 SSOT（收集/随机选择/运行，含多模块 go 仓 fallback）。
重构后的模板 `source` 该库；夜间巡检（`scripts/nightly_test_sweep.py`）通过 CLI 子命令复用同一逻辑。
分发：`deploy_repo_random_precommit.sh` 将 lib 复制到各子仓 `scripts/hooks/lib/`。

自测：`bash scripts/lib/random_test_runner.sh self-test`（CI 门禁调用同一自测）。
