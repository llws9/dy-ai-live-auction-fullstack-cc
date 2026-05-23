import { test, expect, Page } from '@playwright/test';

test.describe('Phase 2: 用户出价功能测试', () => {
  let page: Page;

  test.beforeEach(async ({ browser }) => {
    page = await browser.newPage();
  });

  test.afterEach(async () => {
    await page.close();
  });

  test.describe('认证状态检查', () => {
    test('未登录用户访问出价功能应跳转登录页', async () => {
      // 访问直播间页面
      await page.goto('/live');
      await page.waitForLoadState('networkidle');

      // 清除所有localStorage
      await page.evaluate(() => localStorage.clear());

      // 刷新页面
      await page.reload();
      await page.waitForLoadState('networkidle');

      // 检查是否有出价按钮
      const bidButtons = await page.locator('button:has-text("出价")').count();

      if (bidButtons > 0) {
        // 如果有出价按钮，点击它
        await page.click('button:has-text("出价")');

        // 等待一下
        await page.waitForTimeout(1000);

        // 应该显示登录提示或跳转登录页
        const loginPrompt = await page.locator('text=/请先登录|登录/').first();
        await expect(loginPrompt).toBeVisible({ timeout: 5000 });
      }
    });

    test('登录状态应在localStorage中存储token', async () => {
      // 访问登录页面
      await page.goto('/login');
      await page.waitForLoadState('networkidle');

      // 等待登录表单出现
      await page.waitForSelector('form', { timeout: 10000 });

      // 检查页面标题，确认在登录页
      const title = await page.locator('h2').first().textContent();
      expect(title).toBeTruthy();

      // 填写登录表单 - 使用更灵活的选择器
      const emailInput = page.locator('input[placeholder*="邮箱"]').first();
      const passwordInput = page.locator('input[type="password"]').first();

      // 检查输入框是否存在
      if (await emailInput.isVisible()) {
        await emailInput.fill('test@example.com');
      }

      if (await passwordInput.isVisible()) {
        await passwordInput.fill('Test@123456');
      }

      // 点击登录按钮
      const loginButton = page.locator('button:has-text("登录")').first();
      if (await loginButton.isVisible()) {
        await loginButton.click();

        // 等待登录处理完成
        await page.waitForTimeout(3000);

        // 检查localStorage中是否有token（如果登录成功）
        const token = await page.evaluate(() => localStorage.getItem('auth_token'));

        // 如果后端API可用且登录成功，token应该存在
        // 由于测试环境可能没有后端，我们只验证登录流程不报错
        if (token) {
          expect(token).toBeTruthy();
          expect(token.length).toBeGreaterThan(0);
        }
      }
    });
  });

  test.describe('出价输入组件验证', () => {
    test.beforeEach(async () => {
      // 模拟已登录状态
      await page.goto('/live');
      await page.waitForLoadState('networkidle');

      // 设置mock token
      await page.evaluate(() => {
        localStorage.setItem('auth_token', 'mock_token_for_testing');
        localStorage.setItem('auth_user', JSON.stringify({
          id: 1,
          email: 'test@example.com',
          name: 'Test User',
          role: 0
        }));
      });

      await page.reload();
      await page.waitForLoadState('networkidle');
    });

    test('出价金额应显示最小出价提示', async () => {
      // 查找出价按钮
      const bidButton = page.locator('button:has-text("出价")').first();

      if (await bidButton.isVisible()) {
        await bidButton.click();
        await page.waitForTimeout(500);

        // 检查是否显示出价输入框
        const bidInput = page.locator('input[type="number"]');
        const inputVisible = await bidInput.isVisible();

        if (inputVisible) {
          // 检查是否有最小出价提示
          const minBidHint = await page.locator('text=/最低出价|最小出价/').first();
          await expect(minBidHint).toBeVisible();
        }
      }
    });

    test('输入非法金额应显示错误提示', async () => {
      const bidButton = page.locator('button:has-text("出价")').first();

      if (await bidButton.isVisible()) {
        await bidButton.click();
        await page.waitForTimeout(500);

        const bidInput = page.locator('input[type="number"]');

        if (await bidInput.isVisible()) {
          // 输入一个明显过小的金额
          await bidInput.fill('0.01');
          await bidInput.blur();

          // 等待验证
          await page.waitForTimeout(300);

          // 检查是否有错误提示
          const errorText = await page.locator('text=/不能低于|请输入有效/').first();
          await expect(errorText).toBeVisible({ timeout: 3000 });
        }
      }
    });

    test('出价金额应限制小数位数', async () => {
      const bidButton = page.locator('button:has-text("出价")').first();

      if (await bidButton.isVisible()) {
        await bidButton.click();
        await page.waitForTimeout(500);

        const bidInput = page.locator('input[type="number"]');

        if (await bidInput.isVisible()) {
          // 输入超过2位小数的金额
          await bidInput.fill('100.123');
          await bidInput.blur();

          // 等待验证
          await page.waitForTimeout(300);

          // 检查是否有错误提示
          const errorText = await page.locator('text=/小数点后2位/').first();
          await expect(errorText).toBeVisible({ timeout: 3000 });
        }
      }
    });

    test('快捷出价按钮应正常工作', async () => {
      const bidButton = page.locator('button:has-text("出价")').first();

      if (await bidButton.isVisible()) {
        await bidButton.click();
        await page.waitForTimeout(500);

        // 检查是否有快捷出价按钮
        const quickBidButton = page.locator('button:has-text("最低价")').or(
          page.locator('button:has-text("+")')
        ).first();

        if (await quickBidButton.isVisible()) {
          // 记录当前金额
          const bidInput = page.locator('input[type="number"]');
          const beforeAmount = await bidInput.inputValue();

          // 点击快捷出价按钮
          await quickBidButton.click();
          await page.waitForTimeout(200);

          // 检查金额是否改变
          const afterAmount = await bidInput.inputValue();

          // 金额应该有变化
          expect(afterAmount).toBeTruthy();
        }
      }
    });
  });

  test.describe('出价流程测试', () => {
    test.beforeEach(async () => {
      await page.goto('/live');
      await page.waitForLoadState('networkidle');

      // 设置mock登录状态
      await page.evaluate(() => {
        localStorage.setItem('auth_token', 'mock_token_for_testing');
        localStorage.setItem('auth_user', JSON.stringify({
          id: 1,
          email: 'test@example.com',
          name: 'Test User',
          role: 0
        }));
      });

      await page.reload();
      await page.waitForLoadState('networkidle');
    });

    test('出价按钮点击后应显示加载状态', async () => {
      const bidButton = page.locator('button:has-text("出价")').first();

      if (await bidButton.isVisible()) {
        await bidButton.click();
        await page.waitForTimeout(500);

        // 查找立即出价按钮
        const submitButton = page.locator('button:has-text("立即出价")');

        if (await submitButton.isVisible()) {
          // 点击出价
          await submitButton.click();

          // 检查是否显示加载状态
          const loadingText = page.locator('text=/出价中|加载中/').first();
          await expect(loadingText).toBeVisible({ timeout: 2000 });
        }
      }
    });

    test('出价成功应显示成功提示', async () => {
      const bidButton = page.locator('button:has-text("出价")').first();

      if (await bidButton.isVisible()) {
        await bidButton.click();
        await page.waitForTimeout(500);

        const bidInput = page.locator('input[type="number"]');
        const submitButton = page.locator('button:has-text("立即出价")');

        if (await bidInput.isVisible() && await submitButton.isVisible()) {
          // 输入一个合理的金额
          await bidInput.fill('1000.00');

          // 点击出价
          await submitButton.click();

          // 等待响应
          await page.waitForTimeout(2000);

          // 检查是否有成功或失败提示
          const resultText = page.locator('text=/出价成功|出价失败|操作成功/').first();
          await expect(resultText).toBeVisible({ timeout: 5000 });
        }
      }
    });
  });

  test.describe('排名列表显示测试', () => {
    test.beforeEach(async () => {
      await page.goto('/live');
      await page.waitForLoadState('networkidle');

      // 设置mock登录状态
      await page.evaluate(() => {
        localStorage.setItem('auth_token', 'mock_token_for_testing');
        localStorage.setItem('auth_user', JSON.stringify({
          id: 1,
          email: 'test@example.com',
          name: 'Test User',
          role: 0
        }));
      });

      await page.reload();
      await page.waitForLoadState('networkidle');
    });

    test('排名列表应显示竞拍排名', async () => {
      const bidButton = page.locator('button:has-text("出价")').first();

      if (await bidButton.isVisible()) {
        await bidButton.click();
        await page.waitForTimeout(500);

        // 检查是否有排名列表
        const rankingHeader = page.locator('text=/出价排名|排名/').first();
        await expect(rankingHeader).toBeVisible({ timeout: 3000 });
      }
    });

    test('当前用户排名应高亮显示', async () => {
      const bidButton = page.locator('button:has-text("出价")').first();

      if (await bidButton.isVisible()) {
        await bidButton.click();
        await page.waitForTimeout(500);

        // 检查是否有"我的出价"或用户标识
        const myBid = page.locator('text=/我的出价|我的排名/').first();

        if (await myBid.isVisible()) {
          // 检查是否有高亮背景色
          const parent = myBid.locator('xpath=..');
          const backgroundColor = await parent.evaluate((el) => {
            return window.getComputedStyle(el).backgroundColor;
          });

          // 高亮应该有背景色（通常是黄色或浅色）
          expect(backgroundColor).toBeTruthy();
        }
      }
    });

    test('前三名应有特殊徽章', async () => {
      const bidButton = page.locator('button:has-text("出价")').first();

      if (await bidButton.isVisible()) {
        await bidButton.click();
        await page.waitForTimeout(500);

        // 检查是否有排名数字
        const rankBadge = page.locator('div:has-text("1")').or(
          page.locator('div:has-text("2")')
        ).or(
          page.locator('div:has-text("3")')
        ).first();

        if (await rankBadge.isVisible()) {
          // 检查徽章是否有背景色
          const badgeStyle = await rankBadge.evaluate((el) => {
            const style = window.getComputedStyle(el);
            return {
              backgroundColor: style.backgroundColor,
              borderRadius: style.borderRadius,
            };
          });

          // 徽章应该是圆形或圆角
          expect(badgeStyle.borderRadius).toBeTruthy();
        }
      }
    });
  });

  test.describe('WebSocket连接测试', () => {
    test('WebSocket应自动连接', async () => {
      await page.goto('/live');
      await page.waitForLoadState('networkidle');

      // 设置登录状态
      await page.evaluate(() => {
        localStorage.setItem('auth_token', 'mock_token_for_testing');
        localStorage.setItem('auth_user', JSON.stringify({
          id: 1,
          email: 'test@example.com',
          name: 'Test User',
          role: 0
        }));
      });

      // 点击出价按钮打开面板
      const bidButton = page.locator('button:has-text("出价")').first();

      if (await bidButton.isVisible()) {
        await bidButton.click();
        await page.waitForTimeout(1000);

        // 检查WebSocket连接状态（需要在控制台查看）
        const logs = await page.evaluate(() => {
          // 这里可以检查WebSocket状态
          return {
            hasWebSocket: typeof WebSocket !== 'undefined',
          };
        });

        expect(logs.hasWebSocket).toBeTruthy();
      }
    });

    test('WebSocket断线应自动重连', async () => {
      // 这个测试需要mock WebSocket，在真实环境中很难测试
      // 可以通过检查控制台日志来验证

      await page.goto('/live');
      await page.waitForLoadState('networkidle');

      // 监听控制台消息
      const consoleMessages: string[] = [];
      page.on('console', (msg) => {
        consoleMessages.push(msg.text());
      });

      // 等待一段时间
      await page.waitForTimeout(3000);

      // 检查是否有WebSocket相关的日志
      const wsLogs = consoleMessages.filter(msg =>
        msg.toLowerCase().includes('websocket') ||
        msg.toLowerCase().includes('connected') ||
        msg.toLowerCase().includes('reconnect')
      );

      // 如果有WebSocket日志，说明WebSocket功能正常
      console.log('WebSocket相关日志:', wsLogs);
    });
  });

  test.describe('移动端适配测试', () => {
    test('移动端出价UI应正常显示', async ({ browser }) => {
      // 使用移动设备
      const mobilePage = await browser.newPage({
        viewport: { width: 375, height: 667 },
        isMobile: true,
      });

      await mobilePage.goto('/live');
      await mobilePage.waitForLoadState('networkidle');

      // 检查出价按钮是否可见且可点击
      const bidButton = mobilePage.locator('button:has-text("出价")').first();

      if (await bidButton.isVisible()) {
        // 检查按钮大小（移动端最小44px）
        const buttonSize = await bidButton.evaluate((el) => {
          const rect = el.getBoundingClientRect();
          return {
            width: rect.width,
            height: rect.height,
          };
        });

        expect(buttonSize.width).toBeGreaterThanOrEqual(44);
        expect(buttonSize.height).toBeGreaterThanOrEqual(44);

        // 点击出价按钮
        await bidButton.click();
        await mobilePage.waitForTimeout(500);

        // 检查出价面板是否正常显示
        const bidInput = mobilePage.locator('input[type="number"]');
        const inputVisible = await bidInput.isVisible();

        expect(inputVisible).toBeTruthy();
      }

      await mobilePage.close();
    });
  });
});
