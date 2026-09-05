# 实施计划: relay 本地 GitLab 推送卡住

> **For Claude:** 按计划逐条执行并勾选。

**Goal:** 修复 relay somanyad 任务 zTree 推送卡住，并附回归测试。

**Architecture:** SaaS 换票不变；容器 `oauth-access-push` 用 `oauth_auth_by_repo` canonical key 驱动 push；git 命令加超时。

**Tech Stack:** onlineServiceJS (Node), Django (已有), Playwright

---

### Task 1: 容器 resolveOAuthPushRepoContext

**Files:**
- Modify: `trae-agent/onlineServiceJS/src/layerGitOauthPush.mjs`
- Create: `trae-agent/onlineServiceJS/src/layerGitOauthPush.test.mjs`
- Modify: `trae-agent/onlineServiceJS/package.json` (test:unit)

**Step 1:** 写失败单测 localhost GitLab  
**Step 2:** 实现 `parseOwnerRepoFromPathUrl` + `resolveOAuthPushRepoContext`  
**Step 3:** `npm run test:unit` → 绿

- [x] Task 1

### Task 2: git push 超时

**Files:**
- Modify: `trae-agent/onlineServiceJS/src/layerGitOauthPush.mjs`

**Step 1:** `GIT_PUSH_TIMEOUT_MS` + timer kill  
**Step 2:** 单测仍绿

- [x] Task 2

### Task 3: Playwright relay E2E

**Files:**
- Create: `task2app/playwright/front_project/tests/TaskDetail.relay-to-trae-somanyad-hello-world-push.playwright.test.js`

**Step 1:** 全流程 + push trace  
**Step 2:** 本地凭据齐全时 `npx playwright test`（可选 CI skip）

- [x] Task 3

### Task 4: 验证与交付

**Commands:**
```bash
cd trae-agent/onlineServiceJS && npm run test:unit
cd task2app/Saas_project && python -m pytest tests/test_layer_git_push_policy.py::test_layer_git_push_with_gitlab_oauth_only -q
cd task2app/front_project/app && npm run test -- taskDetailLayerActions.test.js  # 若 vitest 可用
```

- [x] Task 4（Step 7–9 执行）
