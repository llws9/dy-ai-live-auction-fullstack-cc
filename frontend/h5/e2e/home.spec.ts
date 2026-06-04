import { test, expect } from '@playwright/test';
import { mockNewUiApis } from './utils/new-ui-fixtures';

test.describe('Home Page - 首页分类闭环', () => {
  test.beforeEach(async ({ page }) => {
    await mockNewUiApis(page);
    await page.goto('/');
  });

  test('渲染固定 tab 与来自 /categories 的动态分类 tab', async ({ page }) => {
    await expect(page.getByRole('button', { name: '全部' })).toBeVisible();
    await expect(page.getByRole('button', { name: '收藏', exact: true })).toBeVisible();
    await expect(page.getByRole('button', { name: '珠宝腕表' })).toBeVisible();
    await expect(page.getByRole('button', { name: '艺术品' })).toBeVisible();
  });

  test('点击动态分类 tab 时透传 category_id 并仅展示匹配拍品', async ({ page }) => {
    await expect(page.getByRole('heading', { name: '星河钻石腕表' })).toBeVisible();
    await expect(page.getByText('宋代青瓷珍藏')).toBeVisible();

    const filteredRequest = page.waitForRequest((request) => {
      const url = new URL(request.url());
      return url.pathname === '/api/v1/auctions' && url.searchParams.get('category_id') === '1';
    });

    await page.getByRole('button', { name: '珠宝腕表' }).click();
    await filteredRequest;

    await expect(page.getByRole('heading', { name: '星河钻石腕表' })).toBeVisible();
    await expect(page.getByText('宋代青瓷珍藏')).not.toBeVisible();
  });
});
