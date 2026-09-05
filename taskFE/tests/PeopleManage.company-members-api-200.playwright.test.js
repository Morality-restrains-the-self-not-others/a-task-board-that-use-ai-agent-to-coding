// @ts-check
/**
 * 核验：租户用户登录后访问 people/manage 页面，company_members API 应返回 200 而非 500
 * 错误原因：LoginMethod 的 user 字段为 GenericForeignKey，不能直接用于 filter 反向查询
 * 测试账号：contact@daydaymoney.com / <env:PLAYWRIGHT_TEST_PASSWORD>
 */
import { test, expect } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';

test.describe('登录后访问 people/manage，company_members API 应返回 200', () => {
  test('租户用户登录后跳转 people/manage，company_members API 不应返回 500', async ({ page }) => {
    const peopleManageUrl = '/tenant/821976991517573120/people/manage/';

    await playwrightLoginWithLegalAccept(page, {
      email: 'contact@daydaymoney.com',
      password: process.env.PLAYWRIGHT_TEST_PASSWORD,
    });
    expect(page.url().includes('/auth/login/'), '登录后应离开登录页').toBe(false);

    const companyMembersUrlPart = 'company_members';
    const [membersResponse] = await Promise.all([
      page.waitForResponse(
        (res) => {
          const url = res.url();
          return (
            url.includes(companyMembersUrlPart) &&
            res.request().method() === 'GET'
          );
        },
        { timeout: 25_000 }
      ),
      page.goto(peopleManageUrl),
    ]);
    await page.waitForLoadState('networkidle');

    expect(
      membersResponse.status(),
      `company_members API ${membersResponse.url()} 应返回 200`
    ).toBe(200);

    await expect(page.getByRole('heading', { name: '管理人员' })).toBeVisible({
      timeout: 5000,
    });
  });
});
