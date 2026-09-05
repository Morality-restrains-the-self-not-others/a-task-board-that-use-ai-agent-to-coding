---
name: git-workflow-and-versioning
description: Git 工作流与版本管理规范。适用于所有代码变更的提交、分支、合并、打标签和发版。特别适合并行 AI agent 工作场景（worktrees + atomic commits）。
source: adapted from addyosmani/agent-skills
---

# Git 工作流与版本管理

## 概述

Git 是安全网。将提交视为保存点，分支视为沙盒，历史视为文档。在 AI agent 高速生成代码的场景下，严格的版本控制是使变更可管理、可审查、可回滚的机制。

## 核心原则

### 基于主干的开发（Trunk-Based Development）

`main` 始终可部署。工作在短生命周期（1-3 天）的特性分支上进行，快速合并回主干。

```
main ──●──●──●──●──●──●──●──●──●──  (始终可部署)
        ╲      ╱  ╲    ╱
         ●──●─╱    ●──╱    ← 短生命周期特性分支（1-3天）
```

### 1. 尽早提交、频繁提交

每个成功的增量切片获得自己的提交。不要积累大量未提交的变更。

```
工作模式:
  实现切片 → 测试 → 验证 → 提交 → 下一片

不是这样:
  实现所有 → 希望它工作 → 巨型提交
```

### 2. 原子提交

每个提交只做一件事：

```
# Good: 自我包含
a1b2c3d feat: add task creation endpoint with validation
d4e5f6g feat: add task creation form component
h7i8j9k feat: connect form to API with loading state
m1n2o3p test: add task creation tests (unit + integration)

# Bad: 全部混在一起
x1y2z3a feat: add task feature, fix sidebar, update deps, refactor utils
```

### 3. 描述性提交消息

提交消息解释**为什么**，不只是**什么**：

```
<type>: <简短描述>

<可选正文—解释为什么，不是什么>
```

**类型：**
- `feat` — 新功能
- `fix` — Bug 修复
- `refactor` — 既不修复 bug 也不添加功能的代码变更
- `test` — 添加或更新测试
- `docs` — 仅文档
- `chore` — 工具、依赖、配置

### 4. 分离关注点

不把格式化变更与行为变更混在一起。不把重构与功能混在一起。每个类型的变更应该是独立的提交（本仓 Agent 交付默认合入 `main` 并 push origin，见 `/10-ship` / ADR-0019；用户显式要求审查时再用 PR）：

```
# Good: 分离关注点
git commit -m "refactor: extract validation logic to shared utility"
git commit -m "feat: add phone number validation to registration"

# Bad: 混合关注点
git commit -m "refactor validation and add phone number field"
```

### 5. 控制变更大小

目标 ~100 行/提交。超过 ~1000 行的变更应该拆分。

## 分支策略

### 分支命名

```
feature/<简短描述>   → feature/task-creation
fix/<简短描述>       → fix/duplicate-tasks
chore/<简短描述>     → chore/update-deps
refactor/<简短描述>  → refactor/auth-module
```

- 从 `main` 分支
- 保持短生命周期（1-3 天内合并）
- 合并后删除分支
- 不完整的功能用 feature flag 而非长生命周期分支

## Worktrees — 并行 Agent 工作

对于并行 AI agent 工作，使用 git worktrees 同时运行多个分支：

```bash
# 为特性分支创建 worktree
git worktree add ../project-feature-a feature/task-creation
git worktree add ../project-feature-b feature/user-settings

# 每个 worktree 是独立目录，有自己检出的分支
ls ../
  project/              ← main 分支
  project-feature-a/    ← task-creation 分支
  project-feature-b/    ← user-settings 分支

# 完成后合并，清理（ship 强制；见约束 21）
git worktree remove ../project-feature-a
python3 runAll/scripts/cleanup_stale_worktrees.py --apply --shipped-branch feat/<name>
```

**优点：**
- 多个 agent 可同时在不同功能上工作
- 无需切换分支（每个目录有自己的分支）
- 如果一个实验失败，删除 worktree — 不会丢失任何东西
- 变更在显式合并前保持隔离

## 保存点模式

```
Agent 开始工作
    │
    ├── 做变更 → 测试通过？→ 提交 → 继续
    ├── 做变更 → 测试失败？→ 回退到上次提交 → 调查
    └── 功能完成 → 所有提交形成干净历史
```

此模式保证你永远不会丢失超过一个增量的工作。

## 变更摘要

任何修改后，提供结构化摘要：

```
CHANGES MADE:
- src/routes/tasks.go: 向 POST 端点添加了验证中间件
- src/lib/validation.go: 添加了 TaskCreateSchema

THINGS I DIDN'T TOUCH (intentionally):
- src/routes/auth.go: 有类似的验证缺口但超出范围
- src/middleware/error.go: 错误格式可改进（独立任务）

POTENTIAL CONCERNS:
- Zod 严格模式 — 拒绝额外字段。确认是否需要
```

## 版本与发版

### 语义化版本

对于有消费者的任何项目，版本 `MAJOR.MINOR.PATCH`：

```
MAJOR  破坏性变更 — 消费者必须修改代码才能升级
MINOR  新功能，向后兼容 — 安全升级
PATCH  Bug 修复，向后兼容 — 安全升级
```

### 标签是真相来源

```bash
git tag -a v1.4.0 -m "Release 1.4.0"
git push origin v1.4.0
```

从标签推导版本号，而非手改散落文件。

### 保持人类可读的 Changelog

Changelog 不是 `git log`。按 `Added / Changed / Fixed / Deprecated / Removed / Security` 分组，最新在上，每条围绕用户影响而非内部机制。

```markdown
## [1.4.0] - 2026-08-02
### Added
- CSV 批量任务导入
### Fixed
- 周期性任务截止日的时区偏移
```

**写这个条目时就和变更一起写**

## 提交前检查

```bash
# 1. 检查即将提交的内容
git diff --staged

# 2. 确保无密钥
git diff --staged | grep -i "password\|secret\|api_key\|token"

# 3. 运行测试（按项目约定）
go test ./...

# 4. 运行 lint
golangci-lint run
```

## 红旗

- 大量未提交变更积累
- 提交消息如 "fix", "update", "misc"
- 格式化变更与行为变更混合
- 项目中无 `.gitignore`
- 提交 `node_modules/`, `.env`, 或构建产物
- 长生命周期分支与 main 大幅偏离
- 向共享分支 force-push
- 破坏性变更以 minor/patch 版本发版
