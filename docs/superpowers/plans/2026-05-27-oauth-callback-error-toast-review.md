# Code Review: OAuth 回调失败 Toast

> 对照：`docs/superpowers/plans/2026-05-27-oauth-callback-error-toast-implementation-plan.md`  
> 验证：`cd task2app/front_project/app && npm test -- --run utils/gitSiteOAuthCallbackUtils.test.js tests/domain/oauth_callback/oauth_callback_domain_model.test.js` → **14 passed**

## 结论

**通过（可交付）** — 无 Critical/Important 阻塞项。

## 符合计划项

| 需求 | 状态 |
|------|------|
| 非 ok 回调 → `toastService.error` 5s | ✅ guard 测试覆盖 |
| URL 清除 `gitlab`/`github` | ✅ `applyOAuthCallbackFromRoute` 测试 |
| 设置页 skip Toast | ✅ skip 策略 + guard 测试 |
| 共享 hints | ✅ `OAuthCallbackHintCatalog` + Settings import |
| CreateTaskModal return_key | ✅ 已实现 |
| 领域层无 router/toast 依赖 | ✅ `rg` 无匹配 |

## 发现问题

### Minor

1. **guard 测试异步等待** — `setupOAuthCallbackToastGuard` 测试使用 `vi.waitFor` 等待 `void applyOAuthCallbackFromRoute`；若未来改为同步可简化。
2. **valueStream 薄切片 test_file 仍为 view_test.md** — Vitest 在 `front_project`，与 pytest `working_dir` 分离；已在 md 中记录 Vitest 签收。
3. **Increment 4 多入口** — 任务详情/PR 凭据/创建任务仅依赖全局 guard，无独立组件测；接受（与设计方案一致）。

## 风险与后续

| 风险 | 下一步 |
|------|--------|
| `profile_failed` 后端根因未修 | 运维/GitLab 配置排查（非本 PR 范围） |
| 无 git 仓库无法 PR | 在真实 git 根目录提交后执行 `gh pr create` |

## DDD 合规（前端）

- 领域目录 `domain/oauth_callback/` 无基础设施导入 ✅
- 应用层 `gitSiteOAuthCallbackUtils.js` 负责组装 ✅
