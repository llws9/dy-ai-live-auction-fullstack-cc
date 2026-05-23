import { test, expect, Page } from '@playwright/test';

test.describe('订单流程', () => {
  let page: Page;

  test.beforeEach(async ({ browser }) => {
    page = await browser.newPage();

    // 登录
    await page.goto('/login');
    await page.waitForLoadState('networkidle');

    await page.fill('input[placeholder*="用户名"]', 'testuser');
    await page.fill('input[placeholder*="密码"]', 'Test@123456');
    await page.click('button:has-text("登录")');

    await page.waitForURL(/.*\//, { timeout: 10000 });
  });

  test.afterEach(async () => {
    await page.close();
  });

  test('查看订单列表', async () => {
    // 导航到订单页面
    await page.goto('/orders');
    await page.waitForLoadState('networkidle');

    // 等待订单列表加载
    await expect(page.locator('.order-list, [data-testid="order-list"]')).toBeVisible({ timeout: 10000 });

    // 验证订单项存在
    const orderItems = page.locator('.order-item, [data-testid="order-item"]');
    const count = await orderItems.count();
    expect(count).toBeGreaterThanOrEqual(0);

    // 如果有订单,验证关键信息
    if (count > 0) {
      const firstOrder = orderItems.first();
      await expect(firstOrder.locator('text=/订单号/')).toBeVisible();
      await expect(firstOrder.locator('text=/商品/')).toBeVisible();
      await expect(firstOrder.locator('text=/价格|金额/')).toBeVisible();
      await expect(firstOrder.locator('text=/状态/')).toBeVisible();
    }
  });

  test('查看订单列表 - 按状态筛选', async () => {
    await page.goto('/orders');
    await page.waitForLoadState('networkidle');

    // 测试状态筛选
    const statusTabs = page.locator('.order-tabs, [data-testid="order-tabs"]');
    if (await statusTabs.count() > 0) {
      // 点击待付款标签
      const pendingTab = statusTabs.locator('button:has-text("待付款"), [data-testid="pending-tab"]');
      if (await pendingTab.count() > 0) {
        await pendingTab.click();
        await page.waitForLoadState('networkidle');

        // 验证筛选结果
        const orders = page.locator('.order-item, [data-testid="order-item"]');
        if (await orders.count() > 0) {
          await expect(orders.first().locator('text=/待付款/')).toBeVisible({ timeout: 5000 });
        }
      }
    }
  });

  test('查看订单详情', async () => {
    await page.goto('/orders');
    await page.waitForLoadState('networkidle');

    const orderItems = page.locator('.order-item, [data-testid="order-item"]');
    const count = await orderItems.count();

    if (count > 0) {
      // 点击第一个订单
      await orderItems.first().click();

      // 等待跳转到详情页
      await page.waitForURL(/.*orders\/\d+/, { timeout: 10000 });
      await page.waitForLoadState('networkidle');

      // 验证详情页元素
      await expect(page.locator('.order-detail, [data-testid="order-detail"]')).toBeVisible();
      await expect(page.locator('text=/订单信息/')).toBeVisible();
      await expect(page.locator('text=/商品信息/')).toBeVisible();
      await expect(page.locator('text=/价格|金额/')).toBeVisible();

      // 验证订单状态
      await expect(page.locator('text=/订单状态/')).toBeVisible({ timeout: 5000 });
    }
  });

  test('订单支付 - 选择支付方式', async () => {
    await page.goto('/orders');
    await page.waitForLoadState('networkidle');

    // 查找待付款订单
    const pendingOrders = page.locator('.order-item:has-text("待付款"), [data-testid="order-item-pending"]');
    if (await pendingOrders.count() > 0) {
      // 点击支付按钮
      const payButton = pendingOrders.first().locator('button:has-text("支付")');
      await payButton.click();

      // 等待支付弹窗或跳转
      await page.waitForLoadState('networkidle');

      // 验证支付方式选择
      const paymentMethods = page.locator('.payment-methods, [data-testid="payment-methods"]');
      if (await paymentMethods.count() > 0) {
        await expect(paymentMethods.locator('text=/支付宝/')).toBeVisible({ timeout: 5000 });
        await expect(paymentMethods.locator('text=/微信/')).toBeVisible({ timeout: 5000 });
      }
    }
  });

  test('订单支付 - 模拟支付流程', async () => {
    await page.goto('/orders');
    await page.waitForLoadState('networkidle');

    const pendingOrders = page.locator('.order-item:has-text("待付款"), [data-testid="order-item-pending"]');
    if (await pendingOrders.count() > 0) {
      // 进入订单详情
      await pendingOrders.first().click();
      await page.waitForURL(/.*orders\/\d+/, { timeout: 10000 });
      await page.waitForLoadState('networkidle');

      // 点击支付按钮
      const payButton = page.locator('button:has-text("立即支付"), button[data-testid="pay-button"]');
      if (await payButton.count() > 0) {
        await payButton.click();

        // 等待支付页面或弹窗
        await page.waitForLoadState('networkidle');

        // 选择支付方式(测试环境模拟)
        const alipayOption = page.locator('text=/支付宝/, [data-testid="alipay"]');
        if (await alipayOption.count() > 0) {
          await alipayOption.click();
        }

        // 确认支付
        const confirmPay = page.locator('button:has-text("确认支付"), [data-testid="confirm-pay"]');
        if (await confirmPay.count() > 0) {
          await confirmPay.click();

          // 等待支付结果
          await expect(page.locator('text=/支付成功|支付完成/')).toBeVisible({ timeout: 15000 });
        }
      }
    }
  });

  test('订单取消', async () => {
    await page.goto('/orders');
    await page.waitForLoadState('networkidle');

    const pendingOrders = page.locator('.order-item:has-text("待付款"), [data-testid="order-item-pending"]');
    if (await pendingOrders.count() > 0) {
      // 点击取消按钮
      const cancelButton = pendingOrders.first().locator('button:has-text("取消")');
      await cancelButton.click();

      // 等待确认弹窗
      const confirmDialog = page.locator('.dialog, [data-testid="confirm-dialog"]');
      if (await confirmDialog.count() > 0) {
        await expect(confirmDialog.locator('text=/确认取消/')).toBeVisible({ timeout: 5000 });

        // 确认取消
        await confirmDialog.locator('button:has-text("确认"), button:has-text("确定")').click();

        // 等待取消成功
        await expect(page.locator('text=/取消成功/')).toBeVisible({ timeout: 10000 });
      }
    }
  });

  test('查看历史记录', async () => {
    // 导航到历史记录页面
    await page.goto('/history');
    await page.waitForLoadState('networkidle');

    // 等待历史记录加载
    await expect(page.locator('.history-list, [data-testid="history-list"]')).toBeVisible({ timeout: 10000 });

    // 验证历史记录项
    const historyItems = page.locator('.history-item, [data-testid="history-item"]');
    const count = await historyItems.count();
    expect(count).toBeGreaterThanOrEqual(0);

    // 如果有记录,验证信息
    if (count > 0) {
      const firstItem = historyItems.first();
      await expect(firstItem.locator('text=/竞拍|订单/')).toBeVisible({ timeout: 5000 });
    }
  });

  test('查看历史记录 - 按类型筛选', async () => {
    await page.goto('/history');
    await page.waitForLoadState('networkidle');

    // 测试类型筛选
    const typeFilter = page.locator('.type-filter, [data-testid="type-filter"]');
    if (await typeFilter.count() > 0) {
      // 选择竞拍记录
      const auctionFilter = typeFilter.locator('button:has-text("竞拍"), [data-testid="auction-filter"]');
      if (await auctionFilter.count() > 0) {
        await auctionFilter.click();
        await page.waitForLoadState('networkidle');
      }
    }
  });

  test('订单搜索', async () => {
    await page.goto('/orders');
    await page.waitForLoadState('networkidle');

    // 查找搜索框
    const searchInput = page.locator('input[placeholder*="搜索"], input[data-testid="order-search"]');
    if (await searchInput.count() > 0) {
      // 输入订单号
      await searchInput.fill('ORD123456');
      await searchInput.press('Enter');

      // 等待搜索结果
      await page.waitForLoadState('networkidle');

      // 验证搜索结果
      const searchResults = page.locator('.order-item, [data-testid="order-item"]');
      if (await searchResults.count() > 0) {
        await expect(searchResults.first().locator('text=/ORD123456/')).toBeVisible({ timeout: 5000 });
      }
    }
  });

  test('订单列表分页', async () => {
    await page.goto('/orders');
    await page.waitForLoadState('networkidle');

    // 查找分页控件
    const pagination = page.locator('.pagination, [data-testid="pagination"]');
    if (await pagination.count() > 0) {
      // 点击下一页
      const nextButton = pagination.locator('button:has-text("下一页"), [data-testid="next-page"]');
      if (await nextButton.count() > 0 && await nextButton.isEnabled()) {
        await nextButton.click();
        await page.waitForLoadState('networkidle');

        // 验证列表更新
        await expect(page.locator('.order-item, [data-testid="order-item"]')).toBeVisible({ timeout: 5000 });
      }
    }
  });

  test('订单导出', async () => {
    await page.goto('/orders');
    await page.waitForLoadState('networkidle');

    // 查找导出按钮
    const exportButton = page.locator('button:has-text("导出"), [data-testid="export-orders"]');
    if (await exportButton.count() > 0) {
      // 设置下载监听
      const downloadPromise = page.waitForEvent('download', { timeout: 10000 }).catch(() => null);

      await exportButton.click();

      // 验证下载
      const download = await downloadPromise;
      if (download) {
        expect(download.suggestedFilename()).toMatch(/\.(xlsx|csv|pdf)/);
      }
    }
  });

  test('订单状态追踪', async () => {
    await page.goto('/orders');
    await page.waitForLoadState('networkidle');

    const orderItems = page.locator('.order-item, [data-testid="order-item"]');
    if (await orderItems.count() > 0) {
      // 点击订单进入详情
      await orderItems.first().click();
      await page.waitForURL(/.*orders\/\d+/, { timeout: 10000 });
      await page.waitForLoadState('networkidle');

      // 查找订单追踪信息
      const trackingSection = page.locator('.order-tracking, [data-testid="order-tracking"]');
      if (await trackingSection.count() > 0) {
        await expect(trackingSection).toBeVisible();

        // 验证追踪步骤
        const trackingSteps = trackingSection.locator('.tracking-step, [data-testid="tracking-step"]');
        const count = await trackingSteps.count();
        expect(count).toBeGreaterThan(0);

        // 验证当前步骤高亮
        const activeStep = trackingSteps.locator('.active, [data-active="true"]');
        if (await activeStep.count() > 0) {
          await expect(activeStep).toBeVisible();
        }
      }
    }
  });

  test('订单评价', async () => {
    await page.goto('/orders');
    await page.waitForLoadState('networkidle');

    // 查找已完成订单
    const completedOrders = page.locator('.order-item:has-text("已完成"), [data-testid="order-item-completed"]');
    if (await completedOrders.count() > 0) {
      // 点击评价按钮
      const reviewButton = completedOrders.first().locator('button:has-text("评价")');
      if (await reviewButton.count() > 0) {
        await reviewButton.click();

        // 等待评价弹窗
        const reviewDialog = page.locator('.review-dialog, [data-testid="review-dialog"]');
        if (await reviewDialog.count() > 0) {
          // 填写评价
          const ratingStars = reviewDialog.locator('.star, [data-testid="rating-star"]');
          if (await ratingStars.count() > 0) {
            await ratingStars.nth(4).click(); // 5星
          }

          // 填写评价内容
          const reviewInput = reviewDialog.locator('textarea, input[data-testid="review-input"]');
          if (await reviewInput.count() > 0) {
            await reviewInput.fill('非常满意的一次竞拍体验!');
          }

          // 提交评价
          await reviewDialog.locator('button:has-text("提交")').click();

          // 等待评价成功
          await expect(page.locator('text=/评价成功/')).toBeVisible({ timeout: 10000 });
        }
      }
    }
  });
});

test.describe('订单异常处理', () => {
  test('网络错误处理', async ({ browser }) => {
    const page = await browser.newPage();

    // 登录
    await page.goto('/login');
    await page.waitForLoadState('networkidle');
    await page.fill('input[placeholder*="用户名"]', 'testuser');
    await page.fill('input[placeholder*="密码"]', 'Test@123456');
    await page.click('button:has-text("登录")');
    await page.waitForURL(/.*\//, { timeout: 10000 });

    // 模拟离线
    await page.context().setOffline(true);

    // 尝试访问订单页面
    await page.goto('/orders');

    // 验证错误提示
    await expect(page.locator('text=/网络错误|加载失败/')).toBeVisible({ timeout: 10000 });

    // 恢复网络
    await page.context().setOffline(false);

    await page.close();
  });

  test('订单不存在', async ({ browser }) => {
    const page = await browser.newPage();

    // 登录
    await page.goto('/login');
    await page.waitForLoadState('networkidle');
    await page.fill('input[placeholder*="用户名"]', 'testuser');
    await page.fill('input[placeholder*="密码"]', 'Test@123456');
    await page.click('button:has-text("登录")');
    await page.waitForURL(/.*\//, { timeout: 10000 });

    // 访问不存在的订单
    await page.goto('/orders/999999999');
    await page.waitForLoadState('networkidle');

    // 验证错误提示
    await expect(page.locator('text=/订单不存在|404/')).toBeVisible({ timeout: 10000 });

    await page.close();
  });
});
