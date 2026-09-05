// @ts-check
/**
 * 登录页 E2E：现须勾选隐私政策（有 testid）后再提交，且依赖后端已发布 /api/privacy-policy/public/current/。
 */
export async function submitLoginWithEmailPassword(page, email, password) {
  const privacy = page.getByTestId('login-privacy-accept');
  if (await privacy.isVisible().catch(() => false)) {
    await privacy.check();
  }
  const license = page.getByTestId('login-license-accept');
  if (await license.isVisible().catch(() => false)) {
    await license.check();
  }
  await page.locator('#email').fill(email);
  await page.locator('#password').fill(password);
  await page.locator('form').first().evaluate((form) => form.requestSubmit());
}
