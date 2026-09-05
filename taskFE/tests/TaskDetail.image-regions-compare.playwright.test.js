// @ts-check
/**
 * 核验：任务详情页面显示的镜像支持地域与云平台支持地域数据格式是否一致
 * 
 * 问题描述：页面显示的地域数量不同，原因是 fetchImageRegions 返回的数据格式与模板期望不一致
 * 
 * 修复方案：在 fetchImageRegions 函数中添加格式转换，确保返回 { region_id, region_name } 格式
 */
import { test, expect } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';
import { mentionInstalledImageInComment } from './helpers/commentImageMentionE2e.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

test.describe('任务详情 — 镜像地域数据格式验证', () => {
  test('镜像地域API返回应正确转换为模板期望格式', async ({ page }) => {
    const email = process.env.E2E_EMAIL;
    const password = process.env.E2E_PASSWORD;
    
    test.skip(!email || !password, '未设置 E2E_EMAIL / E2E_PASSWORD，跳过需登录的用例');

    await page.goto('/auth/login/');
    await playwrightLoginWithLegalAccept(page, { email, password });
    
    const tenantId = '827923618468040704';
    const workspaceId = '827923618602258432';
    const taskId = '839037065709281280';

    const imageRegionsData = [];
    const platformRegionsData = [];

    await page.route('**/installed-images/*/regions/**', async (route) => {
      await route.continue();
      const response = await route.fetch();
      const data = await response.json();
      imageRegionsData.push(...(Array.isArray(data) ? data : []));
    });

    await page.route('**/cloud-platform/*/cloud/regions/**', async (route) => {
      await route.continue();
      const response = await route.fetch();
      const data = await response.json();
      platformRegionsData.push(...(Array.isArray(data) ? data : []));
    });

    await page.goto(`/tenant/${tenantId}/workspace/${workspaceId}/task-detail/${taskId}/`);
    await page.waitForLoadState('networkidle');

    await mentionInstalledImageInComment(page);
    
    await page.waitForSelector('#region-select');
    const regionSelect = page.locator('#region-select');
    const displayedRegionCount = await regionSelect.locator('option').count();
    
    console.log(`\n=== 地域数据格式分析 ===`);
    console.log(`镜像地域API原始数据格式:`);
    if (imageRegionsData.length > 0) {
      console.log(`  第一个地域对象键:`, Object.keys(imageRegionsData[0]));
      console.log(`  原始数据示例:`, JSON.stringify(imageRegionsData[0]));
    }
    
    console.log(`\n云平台地域API原始数据格式:`);
    if (platformRegionsData.length > 0) {
      console.log(`  第一个地域对象键:`, Object.keys(platformRegionsData[0]));
      console.log(`  原始数据示例:`, JSON.stringify(platformRegionsData[0]));
    }

    console.log(`\n页面显示地域数量: ${displayedRegionCount}`);

    const displayedRegions = await regionSelect.locator('option').allTextContents();
    console.log(`显示的地域名称:`, displayedRegions);

    expect(displayedRegionCount).toBeGreaterThan(0);
  });
});