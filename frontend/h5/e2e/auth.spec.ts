import { test, expect, Page } from '@playwright/test';

test.describe('用户认证流程', () => {
  let page: Page;

  test.beforeEach(async ({ browser }) => {
    page = await browser.newPage();
  });

  test.afterEach(async () => {
    await page.close();
  });

  test('用户注册成功', async () => {
    // 导航到注册页面
    await page.goto('/register');

    // 等待页面加载
    await page.waitForLoadState('networkidle');

    // 生成唯一的用户名
    const uniqueUsername = `testuser_${Date.now()}`;
    const password = 'Test@123456';
    const email = `${uniqueUsername}@test.com`;

    // 填写注册表单
    await page.fill('input[placeholder*="用户名"]', uniqueUsername);
    await page.fill('input[placeholder*="密码"]', password);
    await page.fill('input[placeholder*="确认密码"]', password);
    await page.fill('input[placeholder*="邮箱"]', email);

    // 点击注册按钮
    await page.click('button:has-text("注册")');

    // 等待注册成功提示
    await expect(page.locator('text=注册成功')).toBeVisible({ timeout: 10000 });

    // 验证跳转到登录页面
    await expect(page).toHaveURL(/.*login/, { timeout: 5000 });
  });

  test('用户注册失败 - 用户名已存在', async () => {
    await page.goto('/register');
    await page.waitForLoadState('networkidle');

    // 使用已存在的用户名
    const existingUsername = 'admin';
    const password = 'Test@123456';

    await page.fill('input[placeholder*="用户名"]', existingUsername);
    await page.fill('input[placeholder*="密码"]', password);
    await page.fill('input[placeholder*="确认密码"]', password);

    await page.click('button:has-text("注册")');

    // 等待错误提示
    await expect(page.locator('text=/已存在|注册失败/')).toBeVisible({ timeout: 10000 });
  });

  test('用户登录成功', async () => {
    await page.goto('/login');
    await page.waitForLoadState('networkidle');

    // 使用测试账号登录
    await page.fill('input[placeholder*="用户名"]', 'testuser');
    await page.fill('input[placeholder*="密码"]', 'Test@123456');

    await page.click('button:has-text("登录")');

    // 等待登录成功并跳转
    await expect(page).toHaveURL(/.*\//, { timeout: 10000 });

    // 验证Token存储
    const token = await page.evaluate(() => localStorage.getItem('token'));
    expect(token).toBeTruthy();

    // 验证用户信息显示
    await expect(page.locator('text=testuser')).toBeVisible({ timeout: 5000 });
  });

  test('用户登录失败 - 错误的密码', async () => {
    await page.goto('/login');
    await page.waitForLoadState('networkidle');

    await page.fill('input[placeholder*="用户名"]', 'testuser');
    await page.fill('input[placeholder*="密码"]', 'wrongpassword');

    await page.click('button:has-text("登录")');

    // 等待错误提示
    await expect(page.locator('text=/密码错误|登录失败/')).toBeVisible({ timeout: 10000 });

    // 验证没有Token存储
    const token = await page.evaluate(() => localStorage.getItem('token'));
    expect(token).toBeFalsy();
  });

  test('Token验证 - 已登录用户访问受保护页面', async () => {
    // 先登录
    await page.goto('/login');
    await page.waitForLoadState('networkidle');

    await page.fill('input[placeholder*="用户名"]', 'testuser');
    await page.fill('input[placeholder*="密码"]', 'Test@123456');
    await page.click('button:has-text("登录")');

    await page.waitForURL(/.*\//, { timeout: 10000 });

    // 访问需要认证的页面
    await page.goto('/profile');
    await page.waitForLoadState('networkidle');

    // 验证用户信息显示
    await expect(page.locator('text=testuser')).toBeVisible({ timeout: 5000 });
  });

  test('Token验证 - 未登录用户重定向到登录页', async () => {
    // 清除所有存储
    await page.goto('/');
    await page.evaluate(() => {
      localStorage.clear();
      sessionStorage.clear();
    });

    // 尝试访问受保护页面
    await page.goto('/profile');

    // 等待重定向到登录页
    await expect(page).toHaveURL(/.*login/, { timeout: 10000 });
  });

  test('退出登录', async () => {
    // 先登录
    await page.goto('/login');
    await page.waitForLoadState('networkidle');

    await page.fill('input[placeholder*="用户名"]', 'testuser');
    await page.fill('input[placeholder*="密码"]', 'Test@123456');
    await page.click('button:has-text("登录")');

    await page.waitForURL(/.*\//, { timeout: 10000 });

    // 点击退出按钮
    await page.click('button:has-text("退出"), [data-testid="logout-button"]');

    // 等待退出成功
    await page.waitForURL(/.*login/, { timeout: 10000 });

    // 验证Token已清除
    const token = await page.evaluate(() => localStorage.getItem('token'));
    expect(token).toBeFalsy();
  });

  test('表单验证 - 空用户名', async () => {
    await page.goto('/login');
    await page.waitForLoadState('networkidle');

    // 不填写用户名,直接点击登录
    await page.fill('input[placeholder*="密码"]', 'Test@123456');
    await page.click('button:has-text("登录")');

    // 等待验证提示
    await expect(page.locator('text=/请输入用户名|用户名不能为空/')).toBeVisible({ timeout: 5000 });
  });

  test('表单验证 - 密码格式错误', async () => {
    await page.goto('/register');
    await page.waitForLoadState('networkidle');

    const uniqueUsername = `testuser_${Date.now()}`;

    await page.fill('input[placeholder*="用户名"]', uniqueUsername);
    await page.fill('input[placeholder*="密码"]', '123'); // 太短的密码
    await page.fill('input[placeholder*="确认密码"]', '123');

    await page.click('button:has-text("注册")');

    // 等待验证提示
    await expect(page.locator('text=/密码.*位|密码格式错误/')).toBeVisible({ timeout: 5000 });
  });
});

test.describe('认证状态持久化', () => {
  test('刷新页面保持登录状态', async ({ browser }) => {
    const page = await browser.newPage();

    // 登录
    await page.goto('/login');
    await page.waitForLoadState('networkidle');

    await page.fill('input[placeholder*="用户名"]', 'testuser');
    await page.fill('input[placeholder*="密码"]', 'Test@123456');
    await page.click('button:has-text("登录")');

    await page.waitForURL(/.*\//, { timeout: 10000 });

    // 刷新页面
    await page.reload();
    await page.waitForLoadState('networkidle');

    // 验证仍然保持登录状态
    await expect(page.locator('text=testuser')).toBeVisible({ timeout: 5000 });

    await page.close();
  });
});
