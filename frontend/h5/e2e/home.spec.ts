import { test, expect } from '@playwright/test';
import { mockNewUiApis } from './utils/new-ui-fixtures';

test.describe('Home Page - 新版 H5 UI', () => {
  test.beforeEach(async ({ page }) => {
    await mockNewUiApis(page);
    await page.goto('/');
  });

  test('displays header with navigation', async ({ page }) => {
    await expect(page.getByRole('heading', { name: '奢华竞拍' })).toBeVisible();
    await expect(page.getByLabel('我的关注')).toBeVisible();
    await expect(page.getByLabel('消息通知')).toBeVisible();
  });

  test('displays auction cards', async ({ page }) => {
    await expect(page.getByRole('heading', { name: '星河钻石腕表' })).toBeVisible();
    await expect(page.getByText('宋代青瓷珍藏')).toBeVisible();
    await expect(page.getByRole('link', { name: '详情' }).first()).toBeVisible();
    await expect(page.getByRole('link', { name: '进入直播' }).first()).toBeVisible();
  });

  test('displays auction category tabs', async ({ page }) => {
    await expect(page.getByRole('button', { name: '全部' })).toBeVisible();
    await expect(page.getByRole('button', { name: '收藏', exact: true })).toBeVisible();
    await expect(page.getByRole('button', { name: '珠宝腕表' })).toBeVisible();
    await expect(page.getByRole('button', { name: '艺术品' })).toBeVisible();
  });

  test('filters auctions by tab', async ({ page }) => {
    await page.getByRole('button', { name: '珠宝腕表' }).click();

    await expect(page.getByRole('heading', { name: '星河钻石腕表' })).toBeVisible();
    await expect(page.getByText('宋代青瓷珍藏')).not.toBeVisible();
  });

  test('navigates to auction detail', async ({ page }) => {
    await page.getByRole('link', { name: '详情' }).first().click();

    await expect(page).toHaveURL(/\/detail\?id=101/);
    await expect(page.getByRole('heading', { name: '商品详情' })).toBeVisible();
  });

  test('navigates to live page', async ({ page }) => {
    await page.getByRole('link', { name: '进入直播' }).first().click();

    await expect(page).toHaveURL(/\/live\?id=301&auction_id=101/);
    await expect(page.getByRole('heading', { name: '星河钻石腕表' })).toBeVisible();
  });
});

test.describe('Home Page - mobile shell layout', () => {
  test.use({
    viewport: { width: 512, height: 768 },
    isMobile: true,
    hasTouch: true,
  });

  test('keeps bottom navigation fixed on wide mobile browser viewport', async ({ page }) => {
    await mockNewUiApis(page);
    await page.goto('/');

    const nav = page.getByRole('navigation', { name: '底部导航' });
    await expect(nav).toBeVisible();

    const metrics = await nav.evaluate((element) => {
      const rect = element.getBoundingClientRect();
      const style = window.getComputedStyle(element);
      return {
        position: style.position,
        top: rect.top,
        bottom: rect.bottom,
        viewportHeight: window.innerHeight,
      };
    });

    expect(metrics.position).toBe('fixed');
    expect(metrics.top).toBeGreaterThanOrEqual(0);
    expect(metrics.bottom).toBeLessThanOrEqual(metrics.viewportHeight);
  });
});
