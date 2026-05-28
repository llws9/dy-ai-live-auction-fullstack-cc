import { test, expect } from '@playwright/test';
import { clearStorage } from './utils/test-helpers';

test.describe('Admin Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    // Mock API 响应
    await page.route('/api/v1/auth/login', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          message: 'success',
          data: {
            token: 'mock-admin-jwt-token',
            user: {
              id: 1,
              name: 'Admin',
              email: 'admin@example.com',
              role: 2,
            },
          },
        }),
      });
    });

    // Mock 统计数据 API
    await page.route('/api/v1/statistics/**', async (route) => {
      const url = route.request().url();
      if (url.includes('overview')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            code: 0,
            data: {
              totalAuctions: 100,
              activeAuctions: 15,
              totalRevenue: 12500000,
              todayRevenue: 125000,
              totalUsers: 500,
              newUsersToday: 45,
              successRate: 85.5,
              avgBidPrice: 15000,
            },
          }),
        });
      } else if (url.includes('group_by=day')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            code: 0,
            data: {
              daily_stats: [
                { date: '2024-05-20', revenue: 15000, orders: 25 },
                { date: '2024-05-21', revenue: 18000, orders: 30 },
                { date: '2024-05-22', revenue: 22000, orders: 35 },
              ],
            },
          }),
        });
      } else if (url.includes('group_by=category')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            code: 0,
            data: {
              category_stats: [
                { category: '奢侈品', revenue: 50000, count: 15 },
                { category: '数码', revenue: 35000, count: 20 },
              ],
            },
          }),
        });
      } else {
        await route.continue();
      }
    });

    await clearStorage(page);

    // 登录流程
    await page.goto('/admin-login');
    await page.waitForLoadState('networkidle');
    await page.fill('input[placeholder*="邮箱"]', 'admin@example.com');
    await page.fill('input[placeholder*="密码"]', 'Admin@123456');
    await page.click('button:has-text("登录")');

    // 等待跳转到 dashboard
    await page.waitForURL(/.*dashboard/, { timeout: 10000 });
    await page.waitForLoadState('networkidle');
  });

  test.afterEach(async ({ page }) => {
    await clearStorage(page);
  });

  test('displays page title', async ({ page }) => {
    // 页面标题是"数据大屏"
    await expect(page.locator('h1.page-title')).toContainText('数据大屏');
  });

  test('displays statistics cards', async ({ page }) => {
    // Stats grid should be visible (实际类名是 stats-grid)
    await expect(page.locator('.stats-grid')).toBeVisible();

    // 检查统计卡片数量（应该有8个）
    const statCards = page.locator('.stats-grid > *');
    await expect(statCards).toHaveCount(8);
  });

  test('displays charts section', async ({ page }) => {
    // 图表区域应该在卡片中显示
    await expect(page.locator('.card').filter({ hasText: '收入趋势' })).toBeVisible();
    await expect(page.locator('.card').filter({ hasText: '类目收入分布' })).toBeVisible();
  });

  test('displays quick links section', async ({ page }) => {
    // 快捷入口区域
    await expect(page.locator('.card').filter({ hasText: '详细报表' })).toBeVisible();

    // 检查快捷链接 (使用 Link 组件内部的文本匹配)
    await expect(page.getByRole('link', { name: /竞拍统计/ })).toBeVisible();
    await expect(page.getByRole('link', { name: /收入统计/ })).toBeVisible();
    await expect(page.getByRole('link', { name: /用户统计/ })).toBeVisible();
  });

  test('sidebar navigation works', async ({ page }) => {
    // 验证侧边栏存在
    await expect(page.locator('.sidebar')).toBeVisible();

    // 点击商品管理 (使用文本匹配)
    await page.getByRole('link', { name: /商品管理/ }).click();
    await page.waitForLoadState('networkidle');
    await expect(page).toHaveURL(/.*products/);
  });
});
