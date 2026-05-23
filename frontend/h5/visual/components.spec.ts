import { test, expect } from '@playwright/test';

test.describe('Visual Regression Tests - Components', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('Button component snapshot', async ({ page }) => {
    // Navigate to a page with buttons or create test elements
    await page.goto('/');

    // Wait for page to be stable
    await page.waitForLoadState('networkidle');

    // Find any buttons on the page
    const buttons = page.locator('button');
    const count = await buttons.count();

    if (count > 0) {
      // Take snapshot of first button
      await expect(buttons.first()).toHaveScreenshot('button-component.png', {
        maxDiffPixels: 100,
      });
    }
  });

  test('Card component snapshot', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    const cards = page.locator('.card');
    const count = await cards.count();

    if (count > 0) {
      await expect(cards.first()).toHaveScreenshot('card-component.png', {
        maxDiffPixels: 100,
      });
    }
  });
});

test.describe('Visual Regression Tests - Pages', () => {
  test('Home page snapshot', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Wait for content to load
    await page.waitForTimeout(1000);

    // Take full page snapshot
    await expect(page).toHaveScreenshot('home-page.png', {
      fullPage: true,
      maxDiffPixels: 500,
    });
  });

  test('Auction detail page snapshot', async ({ page }) => {
    await page.goto('/auction/1');
    await page.waitForLoadState('networkidle');

    // Wait for content to load
    await page.waitForTimeout(1000);

    // Take page snapshot
    await expect(page).toHaveScreenshot('auction-page.png', {
      fullPage: true,
      maxDiffPixels: 500,
    });
  });

  test('Live page snapshot', async ({ page }) => {
    await page.goto('/live');
    await page.waitForLoadState('networkidle');

    // Wait for content to load
    await page.waitForTimeout(1000);

    // Take page snapshot (above fold only for live page)
    await expect(page).toHaveScreenshot('live-page.png', {
      maxDiffPixels: 500,
    });
  });
});
