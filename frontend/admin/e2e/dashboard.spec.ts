import { test, expect } from '@playwright/test';

test.describe('Admin Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('displays page title', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /仪表盘|Dashboard/i })).toBeVisible();
  });

  test('displays statistics cards', async ({ page }) => {
    // Stats grid should be visible
    await expect(page.locator('.statsGrid')).toBeVisible();
  });

  test('displays charts section', async ({ page }) => {
    // Charts section should be visible
    await expect(page.locator('.chartsSection')).toBeVisible();
  });

  test('displays activity section', async ({ page }) => {
    // Activity list should be visible
    await expect(page.locator('.activitySection')).toBeVisible();
  });
});
