# ADR-0019: Agent 交付默认直接合入 main 并推送 origin

- **Status:** accepted
- **Date:** 2026-08-19
- **Author:** Agent / Cursor
- **Deciders:** 仓库维护者

---

## Context

`/0-auto-flow` 与 `/10-ship` 曾要求推送 `feat/*` 后执行 `gh pr create`、且「不直接 merge」。实际交付却常直接推 `main` 或另路径合入，导致：

1. 开放 PR 堆积、零审查、与 `main` 冲突或等价重复；
2. 清理脚本删分支但不关 PR，幽灵 PR 长期存在；
3. Agent 交付路径与约束 21「合入即删」语义不一致。

需要统一 **Agent 默认交付路径**，避免再产生「功能已在 main、PR 未关」的状态。

## Decision

We will:

1. Agent / `/10-ship` / `/0-auto-flow` Step 10 **默认**将变更合入 **`main` 并 `git push origin main`**（多仓遵循子仓优先规则 32）。
2. **禁止**默认 `gh pr create` 或「只开 PR、不 merge」。
3. 用户显式要求开 PR 做人工审查时除外；审查通过后仍须合入 `main`、推送并执行约束 21 清理；同主题残留开放 PR 须 `gh pr close`。
4. 规范落点：`.claude/skills/10-ship/SKILL.md`、`.claude/skills/0-auto-flow/SKILL.md`、`.ai/01_project_constraints/21_merged_feat_branch_cleanup.md`。

## Alternatives Considered

### Alternative 1: 继续默认开 PR、人工 merge

- **Pros:** 保留 GitHub review 门禁与 CI 在 PR 上的可见性
- **Cons:** 与本仓「会话直接推 main」实践冲突；已证明会堆积未合 PR
- **Why rejected:** 用户明确要求改为直接合入并推送；存量未合 PR 已造成噪音

### Alternative 2: 开 PR 后 Agent 自动 squash-merge

- **Pros:** 仍有 PR 审计轨迹
- **Cons:** 多仓 monorepo 下 PR 串联成本高；meta 子模块指针 PR CI 易在 checkout 失败
- **Why rejected:** 直接 push `main` 更短路径，且与现有 auto-commit / 子仓指针同步一致

## Consequences

### Positive

- 交付路径单一：合入 = 推送 = 可清理 feat
- 减少幽灵开放 PR
- 与约束 21、会话提交到 main 规则一致

### Negative

- 失去默认 PR 作为强制 review 界面（可用用户显式要求 PR 弥补）
- `main` 上 CI 失败需靠推送后修复而非 PR 门禁拦截

### Mitigations

- 本地 pre-commit / commit-msg 门禁不可 `--no-verify`
- 大改或敏感变更时用户可显式要求开 PR
- 推送后仍须跑 `cleanup_stale_worktrees` / `delete_merged_feat_branches`
