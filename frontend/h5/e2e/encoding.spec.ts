import { test, expect } from '@playwright/test';
import { mockNewUiApis } from './utils/new-ui-fixtures';

test.describe('乱码修复', () => {
  test('商品详情页显示修复后的中文', async ({ page }) => {
    await mockNewUiApis(page);
    await page.route('**/api/v1/products/201', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 200,
          message: 'success',
          data: {
            id: 201,
            name: 'è€è±é’»çŸ³æˆ’æŒ‡',
            description: 'ç²¾é€‰ä¸»çŸ³ï¼Œç«å½©å‡ºè‰²',
            images: [],
            rules: { start_price: 12000, increment: 100 },
          },
        }),
      });
    });

    await page.goto('/detail?id=101');

    await expect(page.getByRole('heading', { name: '老花钻石戒指' })).toBeVisible();
    await expect(page.getByText('精选主石，火彩出色')).toBeVisible();
    await expect(page.getByText('è€è±é’»çŸ³æˆ’æŒ‡')).toHaveCount(0);
  });
});
