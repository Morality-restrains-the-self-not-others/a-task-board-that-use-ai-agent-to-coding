# runall-stop-cascade-failed-downstream

## 场景

platform 链中 `saas-backend` 处于 **failed** 且 **pid > 0**，用户对 `git-oauth` 在 runAll UI 点击「关闭」（cascade 默认 true）。

## 期望

1. 级联计划包含 `saas-backend`（在 `git-oauth` 之前）
2. `git-oauth`、`saas-backend`、`ai-provider` 最终均为 `stopped`
3. 无 takeover / Stop failed 弹窗

## 自动化

- Go: `TestRunner_StopServiceCascade_StopsFailedDownstreamBeforeUpstream`
- Playwright: `runAll/playwright/tests/runall-stop-git-oauth.playwright.test.js`

## 手工（可选）

1. 打开 http://localhost:9999/
2. 确认 saas-backend=failed 且 git-oauth=healthy
3. 对 git-oauth 点「关闭」
4. 刷新后 git-oauth 应为 stopped
