# Value Stream: runAll failed 下游级联关闭

> 源自设计：`docs/superpowers/specs/2026-05-27-runall-cascade-failed-downstream-stop-design.md`

## Value Summary

开发者在 runAll 控制台对 **git-oauth** 点「关闭」时，即使 **saas-backend 已 failed 但进程仍在**，也能一次级联停掉 platform 依赖链，不再出现「API accepted 但 git-oauth 仍 healthy」的 silent failure。

## End-to-End Flow

```text
[用户在 http://localhost:9999/ 对 git-oauth 点「关闭」]
  → POST /api/stop { cascade: true, session_id }
  → PlanStopCascade 生成计划（含 failed 下游）
  → resolveCascadeStepActor 逐步委托 owner
  → stopService 依次 SIGTERM / 端口兜底
  → 【交付点】git-oauth / saas-backend / ai-provider 均 status=stopped
```

## Value Increments

### Increment 1: failed 下游纳入级联停止计划（薄切片）

**Value to user:** git-oauth 在 saas-backend failed 场景下可一次关闭  
**Scope:** `isCascadeStopCandidateStatus` + `filterStoppable` + domain/runner 测试  
**Depends on:** `runall-cascade-mixed-ownership`（actor 委托已就绪）

### Increment 2: Playwright 回归

**Value to user:** UI 路径可自动化验证，防止回归  
**Scope:** 扩展 Playwright fixture platform 链 + `runall-stop-git-oauth.playwright.test.js`  
**Depends on:** Increment 1

## 价值流影响

| 现有 stream | 影响 |
|-------------|------|
| `runall-cascade-lifecycle` / `runall-stop-cascade` | 补充 failed+PID 验收 |
| `runall-cascade-mixed-ownership` | Playwright 覆盖 git-oauth 关闭 |
