# ADR-0021: 逻辑回退须经审批，禁止无用旧逻辑残留

- **Status:** accepted
- **Date:** 2026-08-19
- **Author:** Trae AI
- **Deciders:** 工程团队

---

## Context

Agent 改代码时经常在目标语义之外留下第二套逻辑：旧函数「备用」、注释块、无期限 feature flag、测试变红后的半次回退。版本库已能找回历史，工作树里的旧路径只会继续被调用、被复制、被当成真源。

第 16 条禁止未上线阶段的默认兼容层，但上线后、以及「看起来像小修复」的会话里，同样的残留仍在发生。需要一条**编码规范级**约束：回退/保留旧逻辑必须可审计，默认是删除。

## Decision

We will treat **any logic rollback or retention of superseded behavior** as a gated change:

1. **Default:** delete the old path. Deletion does not need human approval.
2. **Keep or restore** (dual path, resurrected implementation, commented-out old code, compatibility shim without a removal date) **requires in-session human approval**.
3. `/goal` auto-decide does **not** waive this gate. Silence is not approval; unapproved keep → delete.
4. Approved retention must be annotated `Logic-Rollback-OK: <reason>; remove-by: YYYY-MM-DD` in code **and** the commit message.
5. CI scans staged source diffs for high-signal rollback markers and blocks commits that lack the waiver.

## Alternatives Considered

### Alternative 1: Rely on code review only

- **Pros:** 无门禁成本
- **Cons:** Agent 直合 main（ADR-0019）；评审来不及拦「先留着」
- **Why rejected:** 需要提交前自动拦截 + Agent 行为规则

### Alternative 2: Ban all dual paths with no waiver

- **Pros:** 最简单
- **Cons:** 上线后短窗口兼容（外部回调、网关协议）有时合法
- **Why rejected:** 允许有期限的审批保留，而不是永久双写

### Alternative 3: Feature flags as the default migration tool

- **Pros:** 可灰度
- **Cons:** 未上线第 16 条已禁止把双逻辑当默认；flag 常变成永久分叉
- **Why rejected:** flag 若维持两套语义，仍须本条审批与 `remove-by`

## Consequences

### Positive

- 工作树只保留一套目标语义，减少死代码与错误调用
- 真正需要的短窗口兼容可审计、有截止日期
- 与第 16、31 条互补：未上线不堆兼容；迁移有测试基线；迁完不留旧路

### Negative / Trade-offs

- 高信号扫描会漏掉「无关键字的静默拷回」——主要靠 Agent 规则
- `git revert` 触及业务源码时须补 `Logic-Rollback-OK`，多一步
- 误标 waiver 会放行

### Mitigations

- Agent alwaysApply 规则：想保留必须先问；默认删
- CI 覆盖文件名/注释/双路径关键词 + revert 类 commit message
- 漏检模式记入 OPT，按 fix-one-search-all 补模式

## References

- `.ai/01_project_constraints/58_logic_rollback_requires_approval.md`
- `.ai/01_project_constraints/00_project_constraints.md` 第 16、31、53 条
- [ADR-0019: Agent 交付默认直接合入 main](0019-agent-ship-direct-merge-main.md)
