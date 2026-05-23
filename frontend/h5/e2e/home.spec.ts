import { test, expect } from '@playwright/test';

test.describe('Home Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('displays header with navigation', async ({ page }) => {
    await expect(page.getByText('直播竞拍')).toBeVisible();
    await expect(page.getByRole('button', { name: /关注/ })).toBeVisible();
    await expect(page.getByRole('button', { name: /历史/ })).toBeVisible();
  });

  test('displays live entry card', async ({ page }) => {
    await expect(page.getByText('进入直播间')).toBeVisible();
    await expect(page.getByText('实时竞拍 · 互动体验')).toBeVisible();
  });

  test('displays auction tabs', async ({ page }) => {
    await expect(page.getByRole('button', { name: '全部' })).toBeVisible();
    await expect(page.getByRole('button', { name: '进行中' })).toBeVisible();
    await expect(page.getByRole('button', { name: '已结束' })).toBeVisible();
  });

  test('filters auctions by tab', async ({ page }) => {
    // Wait for auctions to load
    await page.waitForSelector('.card', { timeout: 5000 });

    // Click "进行中" tab
    await page.getByRole('button', { name: '进行中' }).click();

    // Verify tab is active
    await expect(page.getByRole('button', { name: '进行中' })).toHaveClass(/tabActive/);
  });

  test('navigates to auction detail', async ({ page }) => {
    await page.waitForSelector('.card', { timeout: 5000 });

    // Click first auction card
    await page.locator('.card').first().click();

    // Should navigate to auction detail page
    await expect(page).toHaveURL(/\/auction\/\d+/);
  });

  test('navigates to live page', async ({ page }) => {
    await page.getByText('进入直播间').click();
    await expect(page).toHaveURL('/live');
  });
});
