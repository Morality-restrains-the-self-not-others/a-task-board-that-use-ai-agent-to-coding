// @ts-check
/**
 * 验证 GitLab Connection 页面磁盘用量显示修复
 * 用法: node verify-gitlab-disk.mjs
 */
import { chromium } from 'playwright';

const EMAIL = process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com';
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD;
if (!PASSWORD) {
  throw new Error('PLAYWRIGHT_TEST_PASSWORD is required; do not hardcode secrets');
}
const BASE = 'https://api.daydaymoney.com';
const TENANT_ID = '871321455093116928';

const browser = await chromium.connectOverCDP('http://127.0.0.1:9222');
const contexts = browser.contexts();
const context = contexts[0] || (await browser.newContext());
const pages = context.pages();
const page = pages.length > 0 ? pages[0] : (await context.newPage());

// 关闭多余的空白页
for (const p of pages) {
  if (p !== page && p.url() === 'about:blank') {
    await p.close().catch(() => {});
  }
}

try {
  // 1. 先检查是否已登录
  await page.goto(`${BASE}/projects/`, { waitUntil: 'domcontentloaded', timeout: 15000 });
  await page.waitForTimeout(2000);

  const currentUrl = page.url();
  console.log(`[1] 当前页面: ${currentUrl}`);

  // 如果被重定向到登录页，则执行登录
  if (currentUrl.includes('/auth/login')) {
    console.log('[2] 未登录，开始登录...');

    // 切换到邮箱/密码 tab
    const emailTab = page.getByRole('button', { name: /邮箱\/密码|邮箱.*密码/ }).first();
    if (await emailTab.isVisible().catch(() => false)) {
      await emailTab.click();
      await page.waitForTimeout(500);
    }

    // 勾选隐私协议
    const checkboxes = [
      page.getByTestId('login-accept-all'),
      page.getByTestId('login-terms-accept'),
      page.getByTestId('login-privacy-accept'),
      page.getByTestId('login-license-accept'),
    ];
    for (const cb of checkboxes) {
      if (await cb.isVisible().catch(() => false)) {
        if (!(await cb.isChecked().catch(() => false))) {
          await cb.check();
        }
      }
    }

    // 点击协议文本行（有些环境 checkbox 不是原生的）
    const legalTexts = [
      page.getByText('我已阅读并同意全部条款', { exact: false }).first(),
      page.getByText('《隐私政策》', { exact: false }).first(),
      page.getByText('《软件许可及服务协议》', { exact: false }).first(),
    ];
    for (const row of legalTexts) {
      if (await row.isVisible().catch(() => false)) {
        await row.click({ force: true }).catch(() => {});
      }
    }

    await page.locator('#email').fill(EMAIL);
    await page.locator('#password').fill(PASSWORD);

    const loginBtn = page.getByRole('button', { name: '登录' });
    await loginBtn.click();

    // 等待登录完成
    await page.waitForURL(/\/projects\/?/, { timeout: 30000 });
    console.log('[3] 登录成功');
  } else {
    console.log('[2] 已登录，跳过登录步骤');
  }

  // 2. 导航到 GitLab Connection 页面
  const targetUrl = `${BASE}/tenant/${TENANT_ID}/settings/gitlab-connection/`;
  console.log(`[3] 导航到: ${targetUrl}`);
  await page.goto(targetUrl, { waitUntil: 'networkidle', timeout: 30000 });
  await page.waitForTimeout(3000);

  // 3. 读取磁盘用量元素
  const diskUsedEl = page.getByTestId('gitlab-disk-used-gb');
  const diskCurrentEl = page.getByTestId('gitlab-current-disk-gb');
  const balanceEl = page.getByTestId('gitlab-balance-points');

  const diskUsedText = await diskUsedEl.textContent().catch(() => 'ELEMENT_NOT_FOUND');
  const diskCurrentText = await diskCurrentEl.textContent().catch(() => 'ELEMENT_NOT_FOUND');
  const balanceText = await balanceEl.textContent().catch(() => 'ELEMENT_NOT_FOUND');

  console.log('\n========== 验证结果 ==========');
  console.log(`gitlab-disk-used-gb    : "${diskUsedText}"`);
  console.log(`gitlab-current-disk-gb : "${diskCurrentText}"`);
  console.log(`gitlab-balance-points  : "${balanceText}"`);

  // 4. 检查是否仍是 "0 GB / 0 GB"
  const wasBug = diskUsedText?.includes('0 GB / 0 GB') || (diskUsedText?.includes('0 GB') && diskCurrentText?.includes('0 GB'));

  if (wasBug) {
    console.log('\n⚠️  磁盘用量仍显示 0/0 — 可能需要等待 taskProjectService 完成首次计量');
    console.log('   请检查: taskProjectService 是否可达、GitLab admin token 是否配置');
  } else {
    console.log('\n✅ 磁盘用量显示已更新！不再显示 "0 GB / 0 GB"');
  }

  // 5. 截图保存
  const screenshotPath = '/tmp/gitlab-connection-verify.png';
  await page.screenshot({ path: screenshotPath, fullPage: true });
  console.log(`\n📸 截图已保存: ${screenshotPath}`);

  // 6. 同时输出完整 dl 内容供参考
  const dlText = await page.locator('dl.grid').first().textContent().catch(() => 'N/A');
  console.log(`\n📋 完整配额区域文本:\n${dlText}`);

} catch (e) {
  console.error('验证失败:', e.message);
  // 出错也截图
  await page.screenshot({ path: '/tmp/gitlab-connection-error.png', fullPage: true }).catch(() => {});
} finally {
  // 不关闭 browser，因为是 CDP 连接的
  console.log('\n完成');
}
