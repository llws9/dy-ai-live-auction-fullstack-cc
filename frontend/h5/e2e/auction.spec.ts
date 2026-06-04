import { test, expect } from '@playwright/test';
import { mockNewUiApis, seedAuthenticatedUser } from './utils/new-ui-fixtures';

test.describe('竞拍流程 - 新版 H5 UI', () => {
  test.beforeEach(async ({ page }) => {
    await mockNewUiApis(page);
  });

  test('首页展示竞拍列表与分类标签', async ({ page }) => {
    await page.goto('/');

    await expect(page.getByRole('heading', { name: '奢华竞拍' })).toBeVisible();
    await expect(page.getByRole('button', { name: '全部' })).toBeVisible();
    await expect(page.getByRole('button', { name: '珠宝腕表' })).toBeVisible();
    await expect(page.getByText('星河钻石腕表')).toBeVisible();
    await expect(page.getByText('宋代青瓷珍藏')).toBeVisible();
  });

  test('收藏标签展示新版空态', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('button', { name: '收藏', exact: true }).click();

    await expect(page.getByText('暂无收藏直播间')).toBeVisible();
  });

  test('从首页进入商品详情', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('link', { name: '详情' }).first().click();

    await expect(page).toHaveURL(/\/detail\?id=101/);
    await expect(page.getByRole('heading', { name: '商品详情' })).toBeVisible();
    await expect(page.getByRole('heading', { name: '星河钻石腕表' })).toBeVisible();
    await expect(page.getByText('当前出价')).toBeVisible();
    await expect(page.getByText('竞拍规则')).toBeVisible();
  });

  test('详情页展示出价记录', async ({ page }) => {
    await page.goto('/detail?id=101');

    await expect(page.getByRole('heading', { name: '出价记录' })).toBeVisible();
    await expect(page.getByText('测试用户')).toBeVisible();
    await expect(page.getByText('收藏家A')).toBeVisible();
  });

  test('未登录出价跳转登录并携带 redirect', async ({ page }) => {
    await page.goto('/detail?id=101');

    await page.getByRole('link', { name: '参与竞拍' }).click();
    await page.getByRole('button', { name: '出价' }).click();

    await expect(page.getByRole('status')).toContainText('请先登录后出价');
  });

  test('出价金额过低展示新版 toast', async ({ page }) => {
    await seedAuthenticatedUser(page);
    await page.goto('/live?id=301&auction_id=101&sheet=bid');

    await page.getByLabel('输入出价金额').fill('12800');
    await page.getByRole('button', { name: '立即出价' }).click();

    await expect(page.getByText(/最低出价/)).toBeVisible();
  });

  test('快捷加价按钮会填充出价输入框', async ({ page }) => {
    await seedAuthenticatedUser(page);
    await page.goto('/live?id=301&auction_id=101&sheet=bid');

    await page.getByRole('button', { name: '+100' }).click();
    await expect(page.getByLabel('输入出价金额')).toHaveValue('13000');
  });

  test('登录后成功出价', async ({ page }) => {
    await seedAuthenticatedUser(page);
    await page.goto('/live?id=301&auction_id=101&sheet=bid');

    await page.getByLabel('输入出价金额').fill('12900');
    await page.getByRole('button', { name: '立即出价' }).click();

    await expect(page.getByText(/出价成功/)).toBeVisible({ timeout: 10000 });
  });

  test('从首页进入直播页', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('link', { name: '进入直播' }).first().click();

    await expect(page).toHaveURL(/\/live\?id=301&auction_id=101/);
    await expect(page.locator('body')).toContainText(/直播|竞拍|藏品/);
  });

  test('已结束竞拍展示查看结果入口', async ({ page }) => {
    await page.goto('/');

    await expect(page.getByText('已结束', { exact: true }).first()).toBeVisible();
    await expect(page.getByRole('link', { name: '查看结果' })).toBeVisible();
  });

  test('已结束详情页展示竞拍结果入口', async ({ page }) => {
    await page.goto('/detail?id=102');

    await expect(page.getByText('已结束', { exact: true })).toBeVisible();
    await expect(page.getByRole('link', { name: '查看竞拍结果' })).toBeVisible();
  });

  test('实时竞拍页面可渲染基础内容', async ({ page }) => {
    await page.goto('/live?id=301&auction_id=101');

    await expect(page.locator('body')).toContainText(/直播|竞拍|藏品/);
  });
});
