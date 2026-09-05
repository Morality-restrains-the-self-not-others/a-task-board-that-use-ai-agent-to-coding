# Value Stream: runAll 链式关闭混合所有权

> Derived from design: `docs/superpowers/specs/2026-05-27-runall-cascade-mixed-ownership-design.md`

## Value Summary

本地开发者在 bootstrap 与 UI session 混用所有权时，仍能通过一次「关闭」完成 platform 依赖链停服，无需对每个下游服务手动 takeover。

## End-to-End Flow

```text
[bootstrap 拉起 taskFE 等 → runall-bootstrap 所有权]
  → [UI 启动 git-oauth → 浏览器 session 所有权]
  → [用户点击 git-oauth「关闭」 cascade=true]
  → [Runner 为每步解析有效 actor：下游用 bootstrap，目标用 UI session]
  → [整链 stopped，无 takeover 错误]
```

## Value Increments

### Increment 1: 级联步骤 Actor 委托（Thin Slice）

**Value to user:** 混合所有权下关闭 `git-oauth` 成功。

**Scope:** `CascadeStepActorResolver` + `executeLifecyclePlan` 接线 + 集成测试。

**Depends on:** `runall-cascade-lifecycle` 已实现基础链式关闭。

### Increment 2: 关闭本组混用所有权

**Value to user:** `StopGroupWithActor` 同样按服务 owner 委托。

**Scope:** `stopGroup` 回调使用 `resolveCascadeStepActor`。

**Depends on:** Increment 1.
