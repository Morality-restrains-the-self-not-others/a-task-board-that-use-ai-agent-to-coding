// @ts-check
import { expect } from '@playwright/test'

/**
 * 选中层后默认是「文件变动」Tab；仍点一次以确保面板可见（幂等）。
 * @param {import('@playwright/test').Page} page
 */
export async function openLayerFilesChangesTab(page) {
  const tab = page.getByTestId('layer-files-tab-changes')
  await expect(tab).toBeVisible({ timeout: 15000 })
  await tab.click()
  const panel = page.getByTestId('task-detail-layer-changes')
  await expect(panel).toBeVisible({ timeout: 15000 })
  return panel
}
