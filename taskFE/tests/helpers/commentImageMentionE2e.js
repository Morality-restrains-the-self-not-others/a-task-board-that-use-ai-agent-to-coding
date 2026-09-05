// @ts-check
import { expect } from '@playwright/test';

/**
 * 在评论输入框 @ 选镜像，硬件配置卡出现在智能体资源配置下方（composer 直挂，非 Teleport）。
 * @param {import('@playwright/test').Page} page
 * @param {{ imageText?: string }} [opts]
 */
export async function mentionInstalledImageInComment(page, opts = {}) {
  const editor = page.getByTestId('comment-content-editor');
  await editor.waitFor({ state: 'visible', timeout: 20_000 });
  await editor.click();
  await page.keyboard.type('@');
  const picker = page.getByTestId('comment-image-mention-picker');
  await expect(picker).toBeVisible({ timeout: 15_000 });
  const item = opts.imageText
    ? picker.locator('li').filter({ hasText: opts.imageText }).first()
    : picker.locator('li').first();
  await item.click();
  await expect(page.getByTestId('comment-composer-env-hardware-slot')).toBeVisible({ timeout: 15_000 });
}

/**
 * @param {import('@playwright/test').Page} page
 * @param {number} [timeout]
 */
export function waitForCommentCreateRequest(page, timeout = 30_000) {
  return page.waitForRequest(
    (r) => r.method() === 'POST' && r.url().includes('/comments/'),
    { timeout },
  );
}
