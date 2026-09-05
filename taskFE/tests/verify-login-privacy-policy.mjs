/**
 * 核验登录页能加载当前生效的隐私条款与服务协议（无 blocking 错误提示）。
 * LOGIN_URL 默认 http://localhost:4000/auth/login/
 */
import { chromium } from '@playwright/test';

const LOGIN_URL = process.env.LOGIN_URL || 'http://localhost:4000/auth/login/';

async function main() {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();

  const privacyResp = page
    .waitForResponse(
      (r) => r.url().includes('/api/privacy-policy/public/current/') && r.request().method() === 'GET',
      { timeout: 30000 },
    )
    .catch(() => null);
  const licenseResp = page
    .waitForResponse(
      (r) => r.url().includes('/api/license-agreement/public/current/') && r.request().method() === 'GET',
      { timeout: 30000 },
    )
    .catch(() => null);

  await page.goto(LOGIN_URL, { waitUntil: 'domcontentloaded', timeout: 60000 });
  const [privacyResponse, licenseResponse] = await Promise.all([privacyResp, licenseResp]);

  const privacyStatus = privacyResponse?.status() ?? 'missing';
  const licenseStatus = licenseResponse?.status() ?? 'missing';
  console.log('privacy API status:', privacyStatus);
  console.log('license API status:', licenseStatus);

  if (privacyStatus !== 200) {
    const body = privacyResponse ? await privacyResponse.text().catch(() => '') : '';
    console.error('privacy API failed:', body);
    await browser.close();
    process.exit(1);
  }
  if (licenseStatus !== 200) {
    const body = licenseResponse ? await licenseResponse.text().catch(() => '') : '';
    console.error('license API failed:', body);
    await browser.close();
    process.exit(1);
  }

  await page.waitForSelector('[data-testid="login-privacy-accept"]', { state: 'visible', timeout: 30000 });
  await page.waitForSelector('[data-testid="login-license-accept"]', { state: 'visible', timeout: 30000 });

  const privacyError = page.getByText('系统尚未发布隐私条款，请联系管理员后再登录。');
  const licenseError = page.getByText('系统尚未发布服务协议，请联系管理员后再登录。');
  const privacyErrorVisible = await privacyError.isVisible().catch(() => false);
  const licenseErrorVisible = await licenseError.isVisible().catch(() => false);

  if (privacyErrorVisible || licenseErrorVisible) {
    console.error('blocking legal error visible on login page', {
      privacyErrorVisible,
      licenseErrorVisible,
    });
    await browser.close();
    process.exit(1);
  }

  console.log('OK: login page loaded privacy policy and license agreement');
  await browser.close();
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
