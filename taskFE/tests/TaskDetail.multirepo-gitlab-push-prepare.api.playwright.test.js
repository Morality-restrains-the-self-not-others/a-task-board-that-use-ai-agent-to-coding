// @ts-check
/**
 * API 级验收（无需浏览器登录）：针对失败请求同构 payload，
 * 断言 taskCloudService prepare 返回 oauth-access-push + oauth_auth_by_repo，
 * 不再走无凭据裸 git/push。
 *
 *   cd taskFE
 *   npx playwright test --config=playwright.chromium.config.js \
 *     tests/TaskDetail.multirepo-gitlab-push-prepare.api.playwright.test.js
 */
import { test, expect } from '@playwright/test';

const CLOUD = (process.env.TASK_CLOUD_BASE || 'http://127.0.0.1:8018').replace(/\/$/, '');
const TENANT = '850256677331562496';
const WORKSPACE = '861623708318031872';
const TASK = 'task_12939023414154091865';
const LAYER = '20260712_050040_f0461b';
const USER = '850256676127797248';
const BRANCH =
  'feature/2026-07-12_example-user_qq.com_daydaymoneytask_12939023414154091865___taskTitle_';

test.describe('layer-git-push prepare API（多仓 GitLab OAuth）', () => {
  test('prefer_container_remote=true 仍应换票并 use_oauth_access_push', async ({ request }) => {
    const resp = await request.post(`${CLOUD}/api/internal/layer-git-push/prepare`, {
      data: {
        tenant_id: TENANT,
        workspace_id: WORKSPACE,
        task_id: TASK,
        layer_id: LAYER,
        user_id: USER,
        prefer_container_remote: true,
        identity_id: '',
        repo_url: '',
        target_branch: BRANCH,
      },
      timeout: 60000,
    });
    expect(resp.status(), await resp.text()).toBe(200);
    const body = await resp.json();
    expect(body.ok).toBe(true);
    expect(body.use_oauth_access_push).toBe(true);
    const oauth = body.push_body?.oauth_auth_by_repo || {};
    expect(Object.keys(oauth).length).toBeGreaterThanOrEqual(2);
    const somanyad = oauth['https://gitlab.daydaymoney.com/example-user/somanyad'];
    expect(somanyad?.provider).toBe('gitlab');
    expect(String(somanyad?.access_token || '').length).toBeGreaterThan(10);
    expect(String(somanyad?.provider_key || '')).toMatch(/^gitlab:/);
  });
});
