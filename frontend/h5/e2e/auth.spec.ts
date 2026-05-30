import { test, expect } from '@playwright/test';
import { loginWithNewUi, mockNewUiApis, mockToken, mockUser, seedAuthenticatedUser } from './utils/new-ui-fixtures';

test.describe('用户认证流程 - 新版 H5 UI', () => {
  test.beforeEach(async ({ page }) => {
    await mockNewUiApis(page);
  });

  test('登录页展示手机号与密码表单', async ({ page }) => {
    await page.goto('/login');

    await expect(page.getByRole('heading', { name: '登录' })).toBeVisible();
    await expect(page.getByText('奢华竞拍')).toBeVisible();
    await expect(page.getByPlaceholder('请输入手机号')).toBeVisible();
    await expect(page.getByPlaceholder('请输入密码')).toBeVisible();
  });

  test('表单验证 - 空手机号', async ({ page }) => {
    await page.goto('/login');
    await page.getByPlaceholder('请输入密码').fill('Test@123456');
    await page.getByRole('button', { name: '登录' }).click();

    await expect(page.getByText('请输入手机号')).toBeVisible();
  });

  test('表单验证 - 空密码', async ({ page }) => {
    await page.goto('/login');
    await page.getByPlaceholder('请输入手机号').fill(mockUser.phone!);
    await page.getByRole('button', { name: '登录' }).click();

    await expect(page.getByText('请输入密码')).toBeVisible();
  });

  test('用户登录成功并写入新版 auth_token', async ({ page }) => {
    await loginWithNewUi(page);

    await expect(page).toHaveURL(/\/$/);
    await expect(page.getByRole('heading', { name: '奢华竞拍' })).toBeVisible();

    const stored = await page.evaluate(() => ({
      token: localStorage.getItem('auth_token'),
      user: JSON.parse(localStorage.getItem('auth_user') || 'null'),
    }));
    expect(stored.token).toBe(mockToken);
    expect(stored.user.name).toBe(mockUser.name);
  });

  test('用户登录失败 - 错误密码', async ({ page }) => {
    await loginWithNewUi(page, mockUser.phone!, 'wrongpassword');

    await expect(page.getByText('登录失败，请检查密码')).toBeVisible({ timeout: 10000 });
    await expect(page).toHaveURL(/\/login/);
  });

  test('未登录访问个人中心会跳转登录页', async ({ page }) => {
    await page.goto('/profile');
    await expect(page).toHaveURL(/\/login/);
  });

  test('已登录访问个人中心展示用户信息', async ({ page }) => {
    await seedAuthenticatedUser(page);
    await page.goto('/profile');

    await expect(page.getByRole('heading', { name: mockUser.name })).toBeVisible();
    await expect(page.getByText(`ID: ${mockUser.id}`)).toBeVisible();
    await expect(page.getByText('钱包余额')).toBeVisible();
  });

  test('退出登录清除 auth_token 并返回登录页', async ({ page }) => {
    await seedAuthenticatedUser(page);
    await page.goto('/profile');

    await page.getByRole('button', { name: /退出登录/ }).click();
    await expect(page).toHaveURL(/\/login/);

    const token = await page.evaluate(() => localStorage.getItem('auth_token'));
    expect(token).toBeNull();
  });

  test('认证状态持久化 - 刷新页面保持登录态', async ({ page }) => {
    await seedAuthenticatedUser(page);
    await page.goto('/profile');
    await expect(page.getByRole('heading', { name: mockUser.name })).toBeVisible();

    await page.reload();
    await expect(page.getByRole('heading', { name: mockUser.name })).toBeVisible();
  });
});
