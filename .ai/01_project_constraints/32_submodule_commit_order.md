# 子仓库优先提交规则

## 基本信息

- 版本：2.0.0
- 创建日期：2026-07-24
- 最后修改：2026-07-25
- 维护者：Trae AI 团队

## 背景（为何是元规则）

本仓库为 meta-repo，包含 34 个 git submodule（注册于 `.gitmodules`，`git submodule status` 可列）。当一次变更同时涉及子仓库与主仓库（meta-repo 层级）时，若先提交主仓库而子仓库未推送，其他开发者或 CI 拉取后将无法引用子仓库 commit（submodule 处于 detached HEAD）。

因此必须**先提交并推送子仓库变更，再提交主仓库**，确保主仓库引用的子仓库 commit 始终存在于远端。

## 规则分类

### 核心规则

#### 子仓库优先提交

- **描述**：凡一次变更同时涉及子仓库（`.gitmodules` 中列出的 submodule 目录）与主仓库时，**必须严格按以下顺序操作**：
  1. **先提交子仓库**：进入每个有变更的子仓库目录，独立执行 `git add` → `git commit` → `git push`
  2. **再提交主仓库**：回到 meta-repo 根目录，提交主仓库变更（含 submodule 指针更新）
- **适用场景**：所有同时触及 submodule 目录内文件与 meta-repo 层级文件的变更；Agent 自动提交、/goal 交付、/ship 收尾等自动化流程
- **优先级**：高
- **规则类型**：禁止忽略

#### 子仓库变更识别

- **描述**：Agent 在提交前**必须**检查子仓库状态。识别方法（以下均可）：
  - `git submodule foreach --quiet 'git status --porcelain'` — 检测未提交变更
  - `git submodule status` — 检测 commit 偏移（前缀 `+` = 子仓库 HEAD 与 gitlink 不一致）
  - `git submodule foreach --quiet 'git rev-list --count @{u}..HEAD'` — 检测未推送
  - 快捷脚本：`python3 runAll/scripts/commit_with_submodules.py --dry-run`
- **适用场景**：每次 `git commit` 前的自动检查
- **优先级**：高

#### 自动化拦截（Git Hooks）

- **描述**：主仓库 `.githooks/pre-commit` 和 `.githooks/pre-push` 在每次 commit/push 时**自动执行**：
  - **pre-commit**：`git submodule foreach --quiet` 检测未提交变更 → 阻断
  - **pre-push**：`git submodule foreach --quiet` 检测未推送提交 → 阻断
  - **hooksPath**：`git config core.hooksPath .githooks` 指向版本控制的 hooks 目录（v13 起全仓统一：各子仓同约定，见 `docs/superpowers/specs/2026-08-06-git-hooks-version-control-design.md`）
  - **OPT-20260823-041**：gitlink 指针同步只把「已跟踪未提交变更」（` M`/`M `/`A `/`D ` 等）判定为 dirty 并阻断；**纯未跟踪文件（`??`）不阻断**。他会话留下的未跟踪意图文档等文件**不得被本会话 `git add`**——本会话只 `git add` 自己改动的文件，未跟踪文件交由归属会话处理（判定逻辑见 `scripts/lib/gitlink_dirty_check.sh`，自测 `scripts/lib/gitlink_dirty_check_selftest.sh`）
- **适用场景**：所有 `git commit` / `git push`
- **优先级**：高
- **规则类型**：自动执行

#### 推送顺序

- **描述**：推送时必须先推送所有有变更的子仓库到远端，再推送主仓库。禁止在子仓库未推送的情况下推送主仓库。
- **适用场景**：`git push` 操作
- **优先级**：高
- **规则类型**：禁止忽略

### 最佳实践

#### 自动化脚本

- **描述**：使用 `runAll/scripts/commit_with_submodules.py` 统一处理子仓库优先提交流程：
  - `--dry-run`：预览有变更的子仓库
  - `--apply`：执行提交推送
  - `--check-hooks`：检查子仓库 pre-commit 钩子部署状态
  - `--deploy-hooks`：自动部署缺失的 pre-commit 钩子
  - `--repo <name>`：仅处理指定子仓库
  - `--no-push`：仅本地提交，跳过推送
- **适用场景**：Agent 自动提交、开发者手动提交
- **优先级**：中

#### 提交信息规范

- **描述**：子仓库提交信息应清晰描述变更内容；主仓库提交信息在必要时注明子仓库指针变更
- **适用场景**：所有提交
- **优先级**：低

#### 新开发者环境搭建

```bash
# 克隆主仓库后初始化所有子仓库
git submodule update --init --recursive

# 配置 hooks（如克隆时未自动配置）——v13 起全仓一键激活：
bash runAll/scripts/install-hooks-all.sh   # 主仓 + 全部子仓幂等激活（--check 仅校验）
```

## 与其它规则的关系

- 与「Git 提交规范」（第 6 条）：子仓库和主仓库的每次提交均须遵守禁止 `--no-verify` 规则
- 与「提交时随机单元测试与遗留债 10% 修复」（第 29 条）：每个子仓库的独立提交触发各自的 pre-commit 门禁
- 与「已合入 feat 分支自动清理」（第 23 条）：子仓库分支清理独立执行
- 本规则优先于任何「一次性提交所有变更」的便利性倾向；正确性 > 便利性
