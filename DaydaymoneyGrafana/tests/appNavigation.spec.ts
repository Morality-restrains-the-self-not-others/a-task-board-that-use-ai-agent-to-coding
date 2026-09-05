import { test, expect } from './fixtures';

test.describe('navigating app', () => {
  test('app root redirects to Daydaymoney console', async ({ gotoPage, page }) => {
    await gotoPage('/');
    await expect(page).toHaveURL(/\/plugins\/daydaymoney-grafana-app/);
  });
});
