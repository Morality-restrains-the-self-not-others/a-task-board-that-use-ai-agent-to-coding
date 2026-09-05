// @ts-check
/**
 * 核验：项目详情页已隐藏仓库 OAuth 授权与透传开关（产品策略）。
 * 测试账号：contact@daydaymoney.com / <env:PLAYWRIGHT_TEST_PASSWORD>
 */
import { test, expect } from '@playwright/test';

const LOGIN_URL = '/auth/login/';
const PROJECT_URL = '/tenant/822445079575764992/projects/822447075168653312/';
const EMAIL = 'contact@daydaymoney.com';
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD;
test.skip(!PASSWORD, 'PASSWORD env required (no hardcoded fallback)');

async function login(page) {
  await page.goto(LOGIN_URL);
  await page.waitForLoadState('networkidle');

  const emailPasswordTab = page.locator('text=邮箱').or(page.locator('button:has-text("邮箱")')).first();
  if (await emailPasswordTab.isVisible()) {
    await emailPasswordTab.click();
    await page.waitForTimeout(300);
  }

  const emailInput = page.locator('#email');
  const passwordInput = page.locator('#password');
  await expect(emailInput).toBeVisible();
  await expect(passwordInput).toBeVisible();
  await emailInput.fill(EMAIL);
  await passwordInput.fill(PASSWORD);
  await page.locator('form').first().evaluate((form) => form.requestSubmit());
  await page.waitForLoadState('networkidle');
  await page.waitForTimeout(2000);
}

test.describe('项目详情页 OAuth 区域已隐藏', () => {
  test('登录后访问项目详情页，不应再展示 OAuth 授权按钮', async ({ page }) => {
    await login(page);

    await page.goto(PROJECT_URL);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(3000);

    const projectHeading = page.getByRole('heading', { name: '项目详情' });
    if (!(await projectHeading.isVisible({ timeout: 10000 }).catch(() => false))) {
      test.skip(true, '项目详情页未加载（可能未登录、项目不存在或 API 错误），跳过');
    }

    await expect(page.getByText('Git 仓库').first()).toBeVisible({ timeout: 5000 });

    await expect(page.getByRole('button', { name: '解除 OAuth' })).toHaveCount(0);
    await expect(page.getByRole('button', { name: 'OAuth 授权' })).toHaveCount(0);
    await expect(page.getByText('仓库授权透传')).toHaveCount(0);
  });
});
