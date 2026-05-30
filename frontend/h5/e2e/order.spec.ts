import { test, expect } from '@playwright/test';
import { mockNewUiApis, mockUser, seedAuthenticatedUser } from './utils/new-ui-fixtures';

test.describe('竞拍记录与个人中心 - 新版 H5 UI', () => {
  test.beforeEach(async ({ page }) => {
    await mockNewUiApis(page);
  });

  test('个人中心展示账户信息与最近订单', async ({ page }) => {
    await seedAuthenticatedUser(page);
    await page.goto('/profile');

    await expect(page.getByRole('heading', { name: mockUser.name })).toBeVisible();
    await expect(page.getByText('钱包余额')).toBeVisible();
    await expect(page.getByText('最近订单')).toBeVisible();
    await expect(page.getByText('星河钻石腕表')).toBeVisible();
  });

  test('个人中心可进入关注和历史记录', async ({ page }) => {
    await seedAuthenticatedUser(page);
    await page.goto('/profile');

    await expect(page.getByRole('link', { name: /关注/ }).first()).toBeVisible();
    await expect(page.getByRole('link', { name: /竞拍记录|全部记录|历史/ }).first()).toBeVisible();
  });

  test('历史记录页面展示统计与记录列表', async ({ page }) => {
    await seedAuthenticatedUser(page);
    await page.goto('/history');

    await expect(page.getByRole('heading', { name: '我的竞拍记录' })).toBeVisible();
    await expect(page.getByText('参与场次')).toBeVisible();
    await expect(page.getByText('成交总额')).toBeVisible();
    await expect(page.getByText('星河钻石腕表')).toBeVisible();
    await expect(page.getByText('宋代青瓷珍藏')).toBeVisible();
  });

  test('历史记录支持竞拍成功筛选', async ({ page }) => {
    await seedAuthenticatedUser(page);
    await page.goto('/history');
    await page.getByRole('button', { name: '竞拍成功' }).click();

    await expect(page.getByText('星河钻石腕表')).toBeVisible();
    await expect(page.getByText('宋代青瓷珍藏')).not.toBeVisible();
  });

  test('历史记录支持未中标筛选', async ({ page }) => {
    await seedAuthenticatedUser(page);
    await page.goto('/history');
    await page.getByRole('button', { name: '未中标' }).click();

    await expect(page.getByText('宋代青瓷珍藏')).toBeVisible();
    await expect(page.getByText('星河钻石腕表')).not.toBeVisible();
  });

  test('历史记录卡片可跳转结果或详情', async ({ page }) => {
    await seedAuthenticatedUser(page);
    await page.goto('/history');

    await page.getByRole('link', { name: '查看结果' }).first().click();
    await expect(page).toHaveURL(/\/result\?id=101/);
  });

  test('未登录访问历史记录会重定向登录', async ({ page }) => {
    await page.goto('/');
    await page.evaluate(() => localStorage.clear());
    await page.goto('/history');

    await expect(page).toHaveURL(/\/login/);
  });
});
