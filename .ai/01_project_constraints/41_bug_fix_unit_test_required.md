# Bug 修复必须携带对应回归单元测试（元规则）

## 基本信息

- 版本：1.0.0
- 创建日期：2026-08-05
- 最后修改：2026-08-05
- 维护者：Trae AI 团队
- 适用范围：整个 monorepo（根仓提交 + 各子仓提交）

## 背景（为何是元规则）

AI 辅助开发中 bug 修复最常见的反模式是：**只改源码、不写回归测试**。后果是同类错误在下一次重构、迁移、或类似场景中被原样复现，且没有任何自动化屏障可以拦截——「同样的错误无法提交」无从谈起。

本规则与既有规则组成闭环：

| 规则 | 管什么 | 与本条关系 |
|------|--------|-----------|
| 第 6 条（Git 提交规范） | 禁止 `--no-verify` | 禁止跳过本门禁 |
| 第 29 条（随机单测 + 10% 债） | 暂存触及的测试**必须 100% 通过** | 保证本条要求的测试是**真实运行且通过**的，而非摆设 |
| 第 34 条（Bug 修复 Playwright 验收） | 用户可感知 bug 的 E2E 验收 | 本条管**单元回归测试的存在与对应**；34 条管 E2E 验收；叠加适用 |
| 第 31 条（迁移/重构测试先行） | 重构前先建行为基线 | 本条管修复场景；互不替代 |

**本条核心承诺：凡 `fix` 类提交，暂存区必须包含与被修源码对应的回归单元测试；门禁缺失即阻断提交。** 这样「将来代码可能有同样的错误」时，回归测试会失败——而被第 29 条门禁强制修复后才能提交。

## 规则分类

### 核心规则

#### fix 提交必须携带对应回归单元测试

- **描述**：凡 commit message 判定为 bug 修复类型（见「触发识别」），暂存区**必须**包含至少一个**与被修改源码文件对应**的**单元测试**文件变更（新增/修改）。判定为 fix 且无对应单测 → commit-msg 门禁 **exit 1 阻断提交**。
- **适用场景**：所有 Git 提交（含 Agent 代提）的 commit-msg 阶段（消息已确定的唯一可靠检测点）；Agent 在提交前自检。
- **优先级**：高
- **规则类型**：禁止忽略

#### 测试必须与被修源码对应

- **描述**：回归测试文件须与被修源码**同目录 / 同包 / tests 镜像目录**，或文件名含被修文件 stem（如 `parser.go` ↔ `parser_test.go`、`models.py` ↔ `test_models.py`、`Button.tsx` ↔ `Button.unit.test.ts`）。与任何源码变更目录无关的「凑数」测试视为违规。
- **原因**：防止用无关测试文件冒充回归覆盖。

#### 测试必须真实覆盖被修复缺陷

- **描述**：回归测试必须断言**被修复缺陷本身**（修复前失败、修复后通过的用例），不得是无关行为测试。
- **运行强制**：本规则只负责「存在 + 对应」静态检查；**测试真实运行并 100% 通过**由第 29 条门禁（暂存触及的测例必须全部通过）强制执行。Agent / 开发者须在提交前本地跑通新增测试。
- **推荐顺序**：Red（写测试复现缺陷，确认失败）→ Green（修复，测试转绿）→ Refactor。若缺陷已修复，须补写测试并确认其能复现原缺陷（可通过临时还原修复验证），并在 commit message 标注回归测试已通过。

### 例外（豁免，门禁放行）

以下情形允许 fix 提交不携带单元测试，门禁不阻断：

1. **显式豁免**：commit message 含 `no-test: <理由>`（如 `no-test: infra-only`）。理由须具体（如「纯 CI 配置修复，无业务逻辑」）；含糊理由（如 `no-test: 没时间`）违反本规则精神，Agent 应劝阻。
2. **纯文档/配置/脚本/工具类修复**：暂存变更仅含文档（`*.md`）、配置（`*.yaml/yml/json/toml/conf`）、CI/脚本（`db/scripts/`、`scripts/`、`.github/`）、`.ai/`、`.cursor/` 等**非业务源码**文件，且无任何业务源码（Go/Python/JS/TS/Vue）变更。示例：`fix: 修正 README 拼写`。
3. **仅测试类提交**：暂存变更全部为测试文件（为存量代码补充测试、修复测试本身）。
4. **`revert` 类提交**：不属于 bug 修复，不触发本条单测门禁。触及业务源码的 revert 仍须遵守第 53 条（逻辑回退审批）。

### 强制与禁止

1. **禁止**以「测试写不了」「时间紧」「这个 bug 太简单」为由跳过测试——豁免仅限上述例外，且例外 1 必须显式 `no-test:` 标注。
2. **禁止**把既有测试改名/挪位/复制冒充新回归测试（对应性检查会拒绝与源码变更无关的测试）。
3. **禁止**删除、注释、`skip` 失败测试来规避运行门禁（与第 29 条一致）。
4. **禁止** `--no-verify` 跳过本门禁（与第 6 条叠加）。
5. 例外 2/3 由门禁自动判定；人工无法判定时**默认阻断**（保守原则，宁可多补测试）。

## 触发识别（fix 判定）

commit message 首行命中以下任一模式即判定为 bug 修复提交：

| 模式 | 示例 |
|------|------|
| Conventional commit `fix:` / `hotfix:` / `bugfix:`（可带 scope） | `fix(parser): NPE on empty input`、`hotfix(api): 400 on GET` |
| 中文修复前缀 | `修复：xxx`、`修正 xxx`、`修 bug：xxx`、`bug 修复：xxx`、`补丁：xxx` |
| 首行含 fix/bug 关键词 | `fix typo in auth flow`、`xxx bug 修复` |

**不触发**：`feat` / `docs` / `refactor` / `test` / `chore` / `revert` 开头；纯文档修复命中 fix 关键词时由例外 2 放行。

## 门禁与配置

| 项 | 路径 |
| --- | --- |
| 细则（本文） | `.ai/01_project_constraints/41_bug_fix_unit_test_required.md` |
| 约束索引 | `00_project_constraints.md` 第 39 条 |
| 根摘要 | 仓库根 [`.ai.md`](../../.ai.md) |
| Cursor 元规则 | `.cursor/rules/bug-fix-unit-test-required.mdc` |
| 门禁脚本（根仓） | `db/scripts/ci/check_bug_fix_unit_tests.py` |
| 门禁自测 | `db/scripts/ci/test_check_bug_fix_unit_tests.py` |
| **commit-msg 权威门禁** | `.githooks/commit-msg`（git 以 `$1` 传入最终 message 文件；commit-msg 是**唯一可靠检测点**） |
| pre-commit 框架注册 | `.pre-commit-config.yaml`（`stages: [commit-msg]`；需 `pre-commit install --hook-type commit-msg`） |
| CI 自测 | `.github/workflows/repo-quality-gates.yml` |

> **为什么不做 pre-commit 阶段检测**：`git commit -m` 时，pre-commit 阶段 `.git/COMMIT_EDITMSG` 仍为**上次成功提交**的陈旧内容（git 仅在 commit-msg 前重写该文件），读它会产生**误拦截**（把上一笔 fix 消息误套到本次 feat 提交上）。故门禁只挂 commit-msg 阶段，pre-commit 不读 message 文件。

### 验收命令

```bash
# 门禁自测（必须全绿）
python3 db/scripts/ci/test_check_bug_fix_unit_tests.py

# 手动验证门禁（--message / --staged-files 可注入）
python3 db/scripts/ci/check_bug_fix_unit_tests.py --message "fix: x" \
  --staged-files "app/parser.go"

# 真实提交路径：git commit 时自动触发（.githooks/commit-msg 权威 + pre-commit 框架 commit-msg 阶段）
```

## Agent 执行要点

```
识别到 bug 修复任务
  → 修复前（如可）：先写回归单测复现缺陷（Red）
  → 修复源码（Green）
  → 本地运行新增测试确认通过
  → 同一次提交携带源码 + 测试（暂存区必须同时包含）
  → 若确属例外：在 commit message 标注 no-test: <理由>
  → 提交（门禁校验通过后放行）
```

## 与相关规则的关系

- **第 6 条**：本门禁是第 6 条「禁止 `--no-verify`、测试通过后再提交」在 bug 修复场景的具体化。
- **第 29 条**：随机单测门禁强制**运行**暂存测试；本条保证 fix 提交**有**对应测试。二者缺一不可。
- **第 34 条**：用户可感知 bug 仍须 Playwright 验收（本条不豁免 34 条）；本条管单元回归测试。
- **第 31 条**：迁移/重构场景由 31 条管理，与本条互不替代（排除条款：31 条明确排除「修复单一 bug」）。

## 变更日志

- 2026-08-05：1.0.0 初版；fix 提交必须携带对应回归单元测试；门禁脚本 + 自测 + 全量注册。
