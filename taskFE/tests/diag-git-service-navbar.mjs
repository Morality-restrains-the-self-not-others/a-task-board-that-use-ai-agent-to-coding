// Diagnostic: check if "代码仓库" button appears on projects page
import { chromium } from 'playwright';

const CDP_URL = 'http://127.0.0.1:9222';
const PAGE_URL = 'http://183.250.1.132:4000/tenant/850256677331562496/projects';

async function main() {
  let browser;
  try {
    browser = await chromium.connectOverCDP(CDP_URL);
  } catch (e) {
    console.log('CDP连接失败，尝试直接启动浏览器...');
    browser = await chromium.launch({ headless: false });
  }

  const contexts = browser.contexts();
  const context = contexts[0] || await browser.newContext();
  const pages = context.pages();
  const page = pages[0] || await context.newPage();

  console.log('=== 导航到项目页面 ===');
  await page.goto(PAGE_URL, { waitUntil: 'networkidle', timeout: 30000 });
  await page.waitForTimeout(2000);

  console.log('\n=== 页面URL ===');
  console.log(page.url());

  console.log('\n=== 导航栏HTML (data-alias="cmp-navbar-main") ===');
  const navbarHtml = await page.evaluate(() => {
    const nav = document.querySelector('[data-alias="cmp-navbar-main"]');
    return nav ? nav.innerHTML.substring(0, 3000) : 'NOT FOUND';
  });
  console.log(navbarHtml);

  console.log('\n=== 查找 "代码仓库" 元素 ===');
  const gitServiceInfo = await page.evaluate(() => {
    const el = document.querySelector('[data-testid="nav-git-service"]');
    if (el) {
      return {
        found: true,
        tag: el.tagName,
        href: el.getAttribute('href'),
        target: el.getAttribute('target'),
        text: el.textContent.trim(),
        visible: el.offsetParent !== null,
      };
    }
    // Also search by text
    const allLinks = Array.from(document.querySelectorAll('nav a'));
    const codeRepo = allLinks.find(a => a.textContent.includes('代码仓库'));
    const allText = allLinks.map(a => a.textContent.trim()).join(', ');
    return { found: false, allNavLinks: allText };
  });
  console.log(JSON.stringify(gitServiceInfo, null, 2));

  console.log('\n=== 检查 VITE_GIT_SERVICE_PUBLIC_URL ===');
  const envInfo = await page.evaluate(() => {
    // Check if the env var is available in the built JS
    const scripts = Array.from(document.querySelectorAll('script[src]'));
    const mainScripts = scripts.filter(s => s.src.includes('main') || s.src.includes('app'));
    return {
      scriptCount: scripts.length,
      mainScripts: mainScripts.map(s => s.src),
    };
  });
  console.log(JSON.stringify(envInfo, null, 2));

  console.log('\n=== 认证状态 ===');
  const authInfo = await page.evaluate(() => {
    const logoutBtn = document.querySelector('nav button');
    const loginLink = Array.from(document.querySelectorAll('nav a')).find(a => a.textContent.includes('登录'));
    return {
      hasLogout: !!logoutBtn,
      hasLogin: !!loginLink,
      cookies: document.cookie.substring(0, 500),
    };
  });
  console.log(JSON.stringify(authInfo, null, 2));

  console.log('\n=== 诊断完成 ===');
  await browser.close();
}

main().catch(err => {
  console.error('诊断失败:', err.message);
  process.exit(1);
});
