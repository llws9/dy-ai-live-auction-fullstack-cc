import { test, expect } from '@playwright/test';

test.describe('Admin Visual Regression Tests', () => {
  test('Dashboard page snapshot', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Wait for charts and data to load
    await page.waitForTimeout(1500);

    await expect(page).toHaveScreenshot('admin-dashboard.png', {
      fullPage: true,
      maxDiffPixels: 500,
    });
  });

  test('Product list page snapshot', async ({ page }) => {
    await page.goto('/product');
    await page.waitForLoadState('networkidle');

    // Wait for table to load
    await page.waitForTimeout(1000);

    await expect(page).toHaveScreenshot('admin-product-list.png', {
      fullPage: true,
      maxDiffPixels: 500,
    });
  });
});
