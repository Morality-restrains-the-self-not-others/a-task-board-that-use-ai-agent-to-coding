# 价值流 — auto_run 交付修复

```
bootstrap → 首指令 job → completed
    → AUTO_RUN_DELIVERY_BEGIN
    → sync identities → commit（含嵌套）→ oauth refresh push（+PR）
    → success? → write done + remirror ztree
    → fail? → bump retry（不写成功 done）→ 启动时可补跑
```

## 测试点

| ID | 点 | 用例 |
|----|----|------|
| VS-1 | 失败可重试 | `runAutoRunDelivery does not lock done on push failure` |
| VS-2 | 成功幂等 | `syncs commit and push; skips when done` |
| VS-3 | 启动补跑 | `retryPendingAutoRunDeliveries only retries unfinished…` |
| VS-4 | 分支回退 | `resolveAutoRunDeliveryTargetBranch falls back…` |
