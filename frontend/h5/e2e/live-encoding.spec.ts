import { test, expect } from '@playwright/test';
import { mockNewUiApis } from './utils/new-ui-fixtures';

const toUtf8Mojibake = (text: string) =>
  encodeURIComponent(text).replace(/%([0-9A-F]{2})/g, (_, hex: string) => String.fromCharCode(parseInt(hex, 16)));

test.describe('直播页乱码修复', () => {
  test('收起态和展开态都显示修复后的中文', async ({ page }) => {
    await mockNewUiApis(page);

    await page.route('**/api/v1/live-streams?*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 200,
          message: 'success',
          data: {
            list: [
              {
                id: 301,
                name: toUtf8Mojibake('瓷器珍藏夜场'),
                host_name: '拍卖师',
                viewer_count: 88,
                current_auction_id: 101,
              },
            ],
            total: 1,
          },
        }),
      });
    });

    await page.route('**/api/v1/auctions/101', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 200,
          message: 'success',
          data: {
            id: 101,
            product_id: 201,
            live_stream_id: 301,
            status: 1,
            current_price: 12800,
            start_price: 12000,
            increment: 100,
            product: {
              id: 201,
              name: toUtf8Mojibake('明代紫砂壶'),
              description: toUtf8Mojibake('名家手作孤品'),
              images: [],
              rules: { start_price: 12000, increment: 100 },
            },
          },
        }),
      });
    });

    await page.route('**/api/v1/products/201', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 200,
          message: 'success',
          data: {
            id: 201,
            name: toUtf8Mojibake('明代紫砂壶'),
            description: toUtf8Mojibake('名家手作孤品'),
            images: [],
            rules: { start_price: 12000, increment: 100 },
          },
        }),
      });
    });

    await page.route('**/api/v1/live-streams/301', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 200,
          message: 'success',
          data: {
            id: 301,
            name: toUtf8Mojibake('瓷器珍藏夜场'),
            host_name: '拍卖师',
            viewer_count: 88,
            followers_count: 12,
            is_following: false,
          },
        }),
      });
    });

    await page.goto('/live?id=301&auction_id=101');

    await expect(page.getByText('明代紫砂壶').first()).toBeVisible();
    await expect(page.getByText('名家手作孤品').first()).toBeVisible();
    await expect(page.getByText(/æ|å|ç|è/)).toHaveCount(0);

    await page.getByText('明代紫砂壶').first().click();

    await expect(page.getByText('明代紫砂壶').nth(1)).toBeVisible();
    await expect(page.getByText('名家手作孤品').nth(1)).toBeVisible();
  });
});
