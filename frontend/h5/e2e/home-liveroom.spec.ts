import { test, expect } from '@playwright/test';
import { mockNewUiApis } from './utils/new-ui-fixtures';

test.describe('Home Page - 直播间维度卡片', () => {
  test.beforeEach(async ({ page }) => {
    await mockNewUiApis(page);
    await page.goto('/');
  });

  test('全部 tab 渲染直播间卡片并能进入直播间', async ({ page }) => {
    // 直播中卡片可见
    const card = page.locator('article').first();
    await expect(card).toBeVisible();
    await expect(page.getByText('直播中').first()).toBeVisible();

    // 点"进入直播间"落到 /live?id=
    await page.getByRole('button', { name: '进入直播间' }).first().click();
    await expect(page).toHaveURL(/\/live\?id=\d+/);
  });

  test('即将开始的直播间渲染 next_auction 商品名', async ({ page }) => {
    await expect(page.getByText('翡翠手镯')).toBeVisible();
  });
});
