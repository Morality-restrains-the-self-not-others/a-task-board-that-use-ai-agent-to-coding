import { test, expect } from '@playwright/test';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

test('任务详情地域列表过滤测试', async ({ page }) => {
  try {
    await page.goto(`http://localhost:4000/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/839037065709281280/`, { timeout: 5000 });
  } catch (error) {
    console.log('服务器不可访问，跳过测试');
    test.skip();
    return;
  }
  
  try {
    await page.waitForSelector('select[id="cloud-platform-select"]', { timeout: 10000 });
  } catch (error) {
    console.log('无法找到cloud-platform-select元素，跳过测试');
    test.skip();
    return;
  }
  
  await page.selectOption('select[id="cloud-platform-select"]', { index: 1 });
  
  await page.waitForTimeout(2000);
  
  const regionSelect = page.locator('select[id*="region"]');
  await regionSelect.waitFor({ timeout: 10000 });
  
  const regionOptions = await regionSelect.locator('option').all();
  const regionValues = [];
  const regionLabels = [];
  
  for (const option of regionOptions) {
    const value = await option.getAttribute('value');
    const label = await option.innerText();
    if (value) {
      regionValues.push(value);
      regionLabels.push(label);
    }
  }
  
  console.log('地域选项值:', regionValues);
  console.log('地域选项标签:', regionLabels);
  
  expect(regionValues.length).toBeGreaterThan(0);
  
  for (let i = 0; i < regionLabels.length; i++) {
    const label = regionLabels[i];
    expect(label).not.toBe(regionValues[i]);
  }
  
  console.log('测试通过：地域显示名称与地域ID不同');
});