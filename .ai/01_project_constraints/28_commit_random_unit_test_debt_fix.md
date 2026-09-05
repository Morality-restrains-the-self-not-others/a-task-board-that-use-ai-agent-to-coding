# 提交时随机单元测试与遗留债 10% 修复

## 基本信息

- 版本：1.1.0
- 创建日期：2026-07-19
- 最后修改：2026-07-19
- 维护者：Trae AI 团队

## 背景（为何是元规则）

全量单测在每次提交上不可行；若不抽样，遗留失败会长期潜伏。Agent / 开发者在随机抽测撞上与本次改动无关的失败时，常以「非本次引入」为由跳过（或试图 `--no-verify`），导致债只增不减。

本专文约定：**每次提交必须执行随机单元测试**；一旦发现遗留失败，**同次提交至少修复发现失败集合的 10%**（向上取整，且至少 1 个），再允许放行剩余已知债并记入清单。与 [00_project_constraints.md](./00_project_constraints.md) 第 6 条（禁止 `--no-verify`）叠加：不得用跳过钩子逃避；对本规则定义的「遗留随机发现」允许在完成 10% 配额后记录余债继续提交。

## 规则分类

### 核心规则

#### 提交必须执行随机单元测试

- **描述**：凡 `git commit`（含 Agent 代提），pre-commit **必须**运行 monorepo 随机单元测试门禁（见下方脚本）。抽测池为**单元测试**（非 Playwright / e2e / 集成全栈）。每次提交至少抽到 **1** 个用例文件（池非空时）；抽样比例与上下限见配置。
- **适用场景**：所有 Git 提交的 pre-commit 阶段；Agent 在提交前自检。
- **优先级**：高
- **规则类型**：禁止忽略

#### 遗留失败须按 10% 配额修复

- **描述**：随机抽测若发现失败用例文件集合大小为 **N（N≥1）**，则本次提交必须**至少修复**  
  `required = max(1, ceil(N × 0.10))`  
  个失败文件（修复后复跑须通过）。其余失败可记入遗留债清单，不强制同批清零。
- **适用场景**：随机门禁发现的、且**非本次暂存区引入**的失败（遗留债）。
- **优先级**：高
- **规则类型**：禁止忽略

##### 强制与禁止

1. **暂存相关测例 100% 通过**：本次暂存触及的测试文件、或其直接覆盖的实现所对应的单测，**不得**适用 10% 配额；必须全部通过。
2. **禁止** `--no-verify` / `--no-gpg-sign` 等方式跳过本门禁（与第 6 条一致）。
3. **禁止**仅删除/跳过测例、或把失败标成「忽略」而不修复来凑配额；配额以「先前失败 → 现已通过」计数。
4. **禁止**在未达配额时口头承诺「下次再修」并强行提交。
5. 达配额后，**余下失败**须写入 [`.learnings/UNIT_TEST_DEBT.md`](../../.learnings/UNIT_TEST_DEBT.md)（路径 + 最近发现时间 + 简述），供后续随机抽测继续压迫。

##### 配额示例

| 本次发现失败文件数 N | 至少修复数 |
| --- | --- |
| 1 | 1 |
| 9 | 1 |
| 10 | 1 |
| 11 | 2 |
| 25 | 3 |

##### Agent 执行要点

```
pre-commit / 门禁报随机单测失败
  → 列出失败文件集合（N）与 required = max(1, ceil(N*0.1))
  → 修复 ≥ required 个，并复跑确认通过
  → 余债写入 .learnings/UNIT_TEST_DEBT.md
  → 再提交（门禁校验配额后放行本批余债）
```

### 最佳实践

- 优先修复与当前改动同域、或修复成本低的失败，提高债下降速度。
- 新建功能仍遵循 BDD/测例先行；本规则不降低新代码的测试要求。
- `task2app/scripts/hooks/pre-commit` 内既有 Vitest 随机抽测与本 monorepo 门禁**叠加**适用；不得以子仓钩子替代根门禁。

### 子仓随机单测 pre-commit（强制覆盖）

- **描述**：monorepo 内**每一个**独立 git 子仓（见 `.gitmodules`）提交时，必须安装并执行与 task2app 同策略的随机单元测试 pre-commit：暂存相关测例优先，其余按比例（默认 30%）随机抽测；池非空时若随机未命中则**至少再跑 1 个**。无单测资产的仓使用 placeholder 钩子（明示跳过，不得省略 hooks 目录）。
- **模板 SSOT**：仓库根 `scripts/hooks/templates/`（`pre-commit.go` / `.python` / `.node` / `.go_js` / `.go_python` / `.placeholder`）
- **部署**：`bash scripts/deploy_repo_random_precommit.sh`（可跟子仓名）；各仓 `.githooks/` 入库真源 + `git config core.hooksPath .githooks` 激活（v13，见 2026-08-06-git-hooks-version-control-design.md）
- **自举**：仅暂存 `.githooks/*` / `scripts/hooks/*` / `README.md` 时可跳过抽测以便提交钩子本身（**禁止**借此跳过业务改动的抽测）
- **优先级**：高；规则类型：禁止忽略

## 门禁与配置

| 项 | 路径 |
| --- | --- |
| 细则（本文） | `.ai/01_project_constraints/28_commit_random_unit_test_debt_fix.md` |
| 约束索引 | [00_project_constraints.md](./00_project_constraints.md) 第 29 条 |
| Cursor 元规则 | `.cursor/rules/commit-random-unit-test-debt-fix.mdc` |
| 子仓模板 | `scripts/hooks/templates/` |
| 子仓部署脚本 | `scripts/deploy_repo_random_precommit.sh`（渲染至各仓 `.githooks/`） |
| 门禁脚本（根） | `db/scripts/ci/run_commit_random_unit_tests.py` |
| 配置 | `db/scripts/ci/commit_random_unit_tests.yaml` |
| 运行报告 | `commitResult/random_unit_tests/last_run.json` |
| 余债清单 | `.learnings/UNIT_TEST_DEBT.md` |

### 验收命令

```bash
# 子仓钩子部署 / 复装
bash scripts/deploy_repo_random_precommit.sh
test -f taskAuth/.githooks/pre-commit && test "$(git -C taskAuth config core.hooksPath)" = .githooks

# 根仓随机门禁
python3 db/scripts/ci/run_commit_random_unit_tests.py
python3 db/scripts/ci/run_commit_random_unit_tests.py --discover-only
python3 db/scripts/ci/test_run_commit_random_unit_tests.py
```

## 与第 6 条（Git 提交规范）的关系

- **仍然禁止** `--no-verify`。
- **本次改动引入或暂存触及的失败**：必须 100% 修复。
- **随机抽测发现的遗留失败**：适用本文 10% 配额；完成配额并登记余债后，不要求同批清零全部遗留失败。
- 若环境/权限导致某失败无法在本机修复：按第 6 条提示用户后中断，**不得**自行跳过钩子。

## 变更日志

- 2026-07-19：1.1.0 全量子仓复制 task2app 随机单测 pre-commit；模板与 `deploy_repo_random_precommit.sh`。
- 2026-07-19：1.0.0 初版；提交随机单测 + 遗留失败 10% 修复配额；门禁脚本与 Cursor alwaysApply 元规则。
