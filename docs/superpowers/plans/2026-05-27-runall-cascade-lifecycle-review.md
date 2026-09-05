# Code Review: runAll 链式启停（Cascade Lifecycle）

> Plan: `docs/superpowers/plans/2026-05-27-runall-cascade-lifecycle-implementation-plan.md`  
> Date: 2026-05-27  
> Verdict: **通过（可合并）**

## 验证

```bash
cd runAll && go test ./... -count=1
# ok — 全部通过（含 ~81s 的 src 集成测试）
```

## 计划对照

| 增量 | 状态 |
|------|------|
| Increment 1 启动链式 + UI 默认 cascade | 完成 |
| Increment 2 关闭链式 | 完成 |
| Increment 3 启动本组 | 完成 |
| Increment 4 失败诊断 + cascade=false 回归 | 完成 |
| Increment 5 全局启停 | 按设计留二期，未实现（符合范围） |

## 发现问题

### 已修复（审查中）

- 领域服务 `ServiceCascadeOrchestrationService.stopPolicy` 字段未使用 → 已删除。

### 建议（非阻塞）

1. **空计划启动**：目标已 `healthy` 且无可启动上游时返回 `nil`（no-op），与设计一致；可在 UI 增加轻提示（二期）。
2. **同步 HTTP**：长链启动会阻塞请求直至结束，符合 NFR L1；若 platform 全链超时，考虑二期异步 job。
3. **无 git 仓库**：工作区非 git checkout，Step 9 无法 `gh pr create`，需用户本地初始化仓库后提交。

## 安全 / 所有权

- 链式每步仍走 `*WithActor` 与 `EnsureOperableBySession`。
- 首步失败（如 ownership）不包装 `CascadeFailure`，错误文案与单点一致。

## 结论

无 Critical/Important 阻塞项。可进入交付（本地验证 + 用户侧 git/PR）。
