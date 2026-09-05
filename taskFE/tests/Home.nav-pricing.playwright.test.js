// @ts-check
import { test, expect } from '@playwright/test';

const ACCESS = 'u824976301710503936';

test.describe('首页导航「价格」', () => {
  test('从带 accessCode 的首页进入价格页并展示资源收费项', async ({ page }) => {
    await page.goto(`/?accessCode=${ACCESS}`);

    await expect(page.getByTestId('nav-pricing')).toBeVisible();
    await page.getByTestId('nav-pricing').click();

    await expect(page).toHaveURL(new RegExp(`/pricing/\\?accessCode=${ACCESS}`));
    await expect(page.locator('[data-alias="view-pricing-page"]')).toBeVisible();
    await expect(page.getByRole('heading', { name: '资源收费标准' })).toBeVisible();
    await expect(page.getByRole('heading', { name: '资源收费标准' })).toHaveClass(/-mt-10/);
    await expect(page.getByText('以下为平台各项资源的计费标准。用户需先购买对应资源，后续使用时直接扣减资源配额。')).toBeVisible();
    await expect(page.getByText('与购买门槛')).toHaveCount(0);

    await expect(page.getByRole('heading', { name: '任务' })).toBeVisible();
    await expect(page.getByText(/12 个月存续期/)).toBeVisible();
    await expect(page.getByRole('heading', { name: 'GitLab 磁盘' })).toBeVisible();
    await expect(page.getByText(/GitLab 仓库磁盘空间/)).toBeVisible();
    await expect(page.getByRole('heading', { name: 'GitLab 流量费' })).toBeVisible();
    await expect(page.getByText(/同区域内网不计费/)).toBeVisible();

    await expect(page.getByText('会员等级体系')).toHaveCount(0);
    await expect(page.getByText('平台采用两级会员制度')).toHaveCount(0);
    await expect(page.getByText('可购买资源：')).toHaveCount(0);
    await expect(page.getByText('商品购买门槛')).toHaveCount(0);
    await expect(page.getByText('以下各商品价格、所需会员等级及购买限制。')).toHaveCount(0);

    // 购买与扣费说明可见；价格保留声明在其下方
    await expect(page.getByText('购买与扣费说明')).toBeVisible();
    await expect(page.getByText(/不提供退款退费服务/)).toBeVisible();
    const notice = page.getByTestId('pricing-change-notice');
    await expect(notice).toBeVisible();
    await expect(notice).toContainText('平台保留修改价格的权利');
    const notesHeading = page.getByRole('heading', { name: '购买与扣费说明' });
    const noticeBox = await notice.boundingBox();
    const notesBox = await notesHeading.boundingBox();
    expect(noticeBox && notesBox && noticeBox.y > notesBox.y).toBeTruthy();
  });
});
