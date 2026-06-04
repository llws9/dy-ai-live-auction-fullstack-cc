import { test, expect } from '@playwright/test';
import { mockNewUiApis, seedAuthenticatedUser } from './utils/new-ui-fixtures';

test.describe('Live Page - 新版 H5 UI', () => {
  test.beforeEach(async ({ page }) => {
    await mockNewUiApis(page);
    await page.goto('/live?id=301&auction_id=101');
  });

  test('displays live auction room', async ({ page }) => {
    await expect(page.getByText('星河钻石腕表').first()).toBeVisible();
    await expect(page.getByText('当前最高价').first()).toBeVisible();
    await expect(page.getByText('正在竞拍')).toBeVisible();
  });

  test('displays back link', async ({ page }) => {
    await expect(page.getByRole('link', { name: '‹' })).toBeVisible();
  });

  test('displays host and viewer info', async ({ page }) => {
    await expect(page.getByText('拍卖师')).toBeVisible();
    await expect(page.getByText(/在线/)).toBeVisible();
  });

  test('displays product and ranking block', async ({ page }) => {
    await seedAuthenticatedUser(page);
    await page.goto('/live?id=301&auction_id=101&sheet=bid');

    await expect(page.getByRole('heading', { name: '星河钻石腕表' })).toBeVisible();
    await expect(page.getByRole('heading', { name: '出价排行' })).toBeVisible();
    await expect(page.getByText('测试用户')).toBeVisible();
  });

  test('bid controls are visible', async ({ page }) => {
    await seedAuthenticatedUser(page);
    await page.goto('/live?id=301&auction_id=101&sheet=bid');

    await expect(page.getByLabel('输入出价金额')).toBeVisible();
    await expect(page.getByRole('button', { name: '最低价' })).toBeVisible();
    await expect(page.getByRole('button', { name: '立即出价' })).toBeVisible();
  });

  test('unauthenticated bid shows login hint', async ({ page }) => {
    await page.getByRole('button', { name: '出价' }).click();
    await expect(page.getByRole('status')).toContainText('请先登录后出价');
  });

  test('authenticated bid succeeds', async ({ page }) => {
    await seedAuthenticatedUser(page);
    await page.goto('/live?id=301&auction_id=101&sheet=bid');

    await page.getByLabel('输入出价金额').fill('12900');
    await page.getByRole('button', { name: '立即出价' }).click();

    await expect(page.getByText('出价成功')).toBeVisible({ timeout: 10000 });
  });

  test('back link returns to home', async ({ page }) => {
    await page.getByRole('link', { name: '‹' }).click();

    await expect(page).toHaveURL(/\/$/);
    await expect(page.getByRole('heading', { name: '奢华竞拍' })).toBeVisible();
  });
});
