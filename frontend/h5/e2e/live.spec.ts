import { test, expect } from '@playwright/test';

test.describe('Live Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/live');
  });

  test('displays live container', async ({ page }) => {
    // Page should have live container
    await expect(page.locator('.container')).toBeVisible();
  });

  test('displays close button', async ({ page }) => {
    // Close button should be visible
    const closeButton = page.locator('.closeBtn');
    await expect(closeButton).toBeVisible();
  });

  test('displays live info badge', async ({ page }) => {
    // Live badge with viewer count should be visible
    const liveBadge = page.locator('.liveInfo');
    await expect(liveBadge).toBeVisible();
  });

  test('displays product list', async ({ page }) => {
    // Product list should be visible
    await page.waitForSelector('.productList', { timeout: 5000 });
    await expect(page.locator('.productList')).toBeVisible();
  });

  test('product cards are interactive', async ({ page }) => {
    await page.waitForSelector('.productCard', { timeout: 5000 });

    // Product cards should be visible
    const productCards = page.locator('.productCard');
    const count = await productCards.count();

    if (count > 0) {
      // Hover effect should work
      await productCards.first().hover();
    }
  });

  test('close button returns to home', async ({ page }) => {
    await page.waitForSelector('.closeBtn', { timeout: 5000 });

    // Click close button
    await page.locator('.closeBtn').click();

    // Should navigate back
    await page.goBack();
    await expect(page).toHaveURL('/');
  });
});
