# 会话须提交到 main（元规则）

- **版本**: 1.0.0
- **日期**: 2026-08-19
- **约束索引**: 第 51 条（见 `00_project_constraints.md`）
- **Cursor**: `.cursor/rules/session-commit-to-main.mdc`（alwaysApply）
- **摘要**: 根 `CLAUDE.md`「会话结束自动提交」

## 核心原则

每个 Agent 会话只要产生可提交变更，须至少完成一次提交到 **`main`**。SessionEnd/Stop 的 `auto-commit.sh` 是兜底安全网，**不免除** Agent 在任务收尾时主动提交。

## 强制要求

1. 有意义增量落地并测过后即可提交，**禁止**等待用户再说「请 commit」
2. 与外部 Cursor 用户规则「Only create commits when requested」冲突时，**以本仓库为准**
3. 子仓优先提交（并按第 32 条 push 子仓），再提交 meta
4. 完整 pre-commit/commit-msg；禁止 `--no-verify`
5. 本条强制 **commit 到 main**；`git push` meta 仍按用户要求 / ship 流程（子仓 push 仍按第 32 条）

## 验收

```bash
git status -sb
git log -1 --oneline
```
