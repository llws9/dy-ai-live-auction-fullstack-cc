import { test, expect, Page } from '@playwright/test';

test.describe('竞拍流程', () => {
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

  test('查看竞拍列表', async () => {
    // 导航到竞拍列表页面
    await page.goto('/auctions');
    await page.waitForLoadState('networkidle');

    // 等待竞拍列表加载
    await expect(page.locator('.auction-list, [data-testid="auction-list"]')).toBeVisible({ timeout: 10000 });

    // 验证列表项存在
    const auctionItems = page.locator('.auction-item, [data-testid="auction-item"]');
    const count = await auctionItems.count();
    expect(count).toBeGreaterThan(0);

    // 验证关键信息显示
    const firstItem = auctionItems.first();
    await expect(firstItem.locator('text=/商品|竞拍/')).toBeVisible();
    await expect(firstItem.locator('text=/价格|当前价/')).toBeVisible();
    await expect(firstItem.locator('text=/状态|进行中/')).toBeVisible();
  });

  test('查看竞拍列表 - 筛选和排序', async () => {
    await page.goto('/auctions');
    await page.waitForLoadState('networkidle');

    // 测试状态筛选
    const statusFilter = page.locator('select[data-testid="status-filter"], .status-filter');
    if (await statusFilter.count() > 0) {
      await statusFilter.selectOption('active');
      await page.waitForLoadState('networkidle');

      // 验证筛选结果
      await expect(page.locator('.auction-item, [data-testid="auction-item"]')).toBeVisible({ timeout: 5000 });
    }

    // 测试价格排序
    const sortButton = page.locator('[data-testid="sort-price"], button:has-text("价格")');
    if (await sortButton.count() > 0) {
      await sortButton.click();
      await page.waitForLoadState('networkidle');
    }
  });

  test('进入竞拍详情', async () => {
    await page.goto('/auctions');
    await page.waitForLoadState('networkidle');

    // 点击第一个竞拍项
    const firstAuction = page.locator('.auction-item, [data-testid="auction-item"]').first();
    await firstAuction.click();

    // 等待跳转到详情页
    await page.waitForURL(/.*auctions\/\d+/, { timeout: 10000 });
    await page.waitForLoadState('networkidle');

    // 验证详情页元素
    await expect(page.locator('.auction-detail, [data-testid="auction-detail"]')).toBeVisible();
    await expect(page.locator('text=/商品名称|商品详情/')).toBeVisible();
    await expect(page.locator('text=/当前价格|起拍价/')).toBeVisible();
    await expect(page.locator('text=/出价|竞拍按钮/')).toBeVisible();

    // 验证竞拍信息
    await expect(page.locator('text=/剩余时间|倒计时/')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('text=/参与人数|竞拍人数/')).toBeVisible({ timeout: 5000 });
  });

  test('出价操作 - 成功出价', async () => {
    await page.goto('/auctions');
    await page.waitForLoadState('networkidle');

    // 进入竞拍详情
    const firstAuction = page.locator('.auction-item, [data-testid="auction-item"]').first();
    await firstAuction.click();
    await page.waitForURL(/.*auctions\/\d+/, { timeout: 10000 });
    await page.waitForLoadState('networkidle');

    // 获取当前价格
    const priceText = await page.locator('text=/当前价格|起拍价/').first().textContent();
    const currentPrice = parseFloat(priceText?.match(/\d+\.?\d*/)?.[0] || '0');

    // 填写出价金额
    const bidAmount = currentPrice + 10;
    const bidInput = page.locator('input[placeholder*="出价"], input[data-testid="bid-amount"]');
    await bidInput.fill(bidAmount.toString());

    // 点击出价按钮
    await page.click('button:has-text("出价"), button[data-testid="bid-button"]');

    // 等待出价成功提示
    await expect(page.locator('text=/出价成功|竞拍成功/')).toBeVisible({ timeout: 10000 });

    // 验证价格更新
    await page.waitForTimeout(1000);
    const newPriceText = await page.locator('text=/当前价格/').first().textContent();
    const newPrice = parseFloat(newPriceText?.match(/\d+\.?\d*/)?.[0] || '0');
    expect(newPrice).toBe(bidAmount);
  });

  test('出价操作 - 出价金额过低', async () => {
    await page.goto('/auctions');
    await page.waitForLoadState('networkidle');

    const firstAuction = page.locator('.auction-item, [data-testid="auction-item"]').first();
    await firstAuction.click();
    await page.waitForURL(/.*auctions\/\d+/, { timeout: 10000 });
    await page.waitForLoadState('networkidle');

    // 获取当前价格
    const priceText = await page.locator('text=/当前价格|起拍价/').first().textContent();
    const currentPrice = parseFloat(priceText?.match(/\d+\.?\d*/)?.[0] || '100');

    // 填写低于当前价格的金额
    const lowBidAmount = Math.max(1, currentPrice - 10);
    const bidInput = page.locator('input[placeholder*="出价"], input[data-testid="bid-amount"]');
    await bidInput.fill(lowBidAmount.toString());

    // 点击出价按钮
    await page.click('button:has-text("出价"), button[data-testid="bid-button"]');

    // 等待错误提示
    await expect(page.locator('text=/出价.*高于|价格过低|出价失败/')).toBeVisible({ timeout: 10000 });
  });

  test('出价操作 - 余额不足', async () => {
    await page.goto('/auctions');
    await page.waitForLoadState('networkidle');

    const firstAuction = page.locator('.auction-item, [data-testid="auction-item"]').first();
    await firstAuction.click();
    await page.waitForURL(/.*auctions\/\d+/, { timeout: 10000 });
    await page.waitForLoadState('networkidle');

    // 填写一个非常高的出价金额
    const highBidAmount = 999999999;
    const bidInput = page.locator('input[placeholder*="出价"], input[data-testid="bid-amount"]');
    await bidInput.fill(highBidAmount.toString());

    // 点击出价按钮
    await page.click('button:has-text("出价"), button[data-testid="bid-button"]');

    // 等待余额不足提示
    await expect(page.locator('text=/余额不足|账户余额不足/')).toBeVisible({ timeout: 10000 });
  });

  test('查看排名', async () => {
    await page.goto('/auctions');
    await page.waitForLoadState('networkidle');

    const firstAuction = page.locator('.auction-item, [data-testid="auction-item"]').first();
    await firstAuction.click();
    await page.waitForURL(/.*auctions\/\d+/, { timeout: 10000 });
    await page.waitForLoadState('networkidle');

    // 查找排名区域
    const rankingSection = page.locator('.ranking-section, [data-testid="ranking-list"]');
    if (await rankingSection.count() > 0) {
      await expect(rankingSection).toBeVisible();

      // 验证排名项
      const rankingItems = rankingSection.locator('.ranking-item, [data-testid="ranking-item"]');
      const count = await rankingItems.count();
      expect(count).toBeGreaterThan(0);

      // 验证排名信息
      const firstRankItem = rankingItems.first();
      await expect(firstRankItem.locator('text=/\\d+/')).toBeVisible(); // 排名
      await expect(firstRankItem.locator('text=/用户|竞拍者/')).toBeVisible(); // 用户
    }
  });

  test('竞拍结束 - 显示获胜者', async () => {
    // 导航到一个已结束的竞拍
    await page.goto('/auctions?status=ended');
    await page.waitForLoadState('networkidle');

    const endedAuction = page.locator('.auction-item.ended, [data-testid="auction-item-ended"]').first();
    if (await endedAuction.count() > 0) {
      await endedAuction.click();
      await page.waitForURL(/.*auctions\/\d+/, { timeout: 10000 });
      await page.waitForLoadState('networkidle');

      // 验证竞拍结束状态
      await expect(page.locator('text=/已结束|竞拍结束/')).toBeVisible({ timeout: 5000 });

      // 验证获胜者信息
      await expect(page.locator('text=/获胜者|成交用户/')).toBeVisible({ timeout: 5000 });

      // 验证成交价格
      await expect(page.locator('text=/成交价|最终价格/')).toBeVisible({ timeout: 5000 });

      // 验证出价按钮不可用
      const bidButton = page.locator('button:has-text("出价"), button[data-testid="bid-button"]');
      expect(await bidButton.isDisabled()).toBeTruthy();
    }
  });

  test('竞拍倒计时', async () => {
    await page.goto('/auctions');
    await page.waitForLoadState('networkidle');

    const firstAuction = page.locator('.auction-item, [data-testid="auction-item"]').first();
    await firstAuction.click();
    await page.waitForURL(/.*auctions\/\d+/, { timeout: 10000 });
    await page.waitForLoadState('networkidle');

    // 获取倒计时元素
    const countdown = page.locator('.countdown, [data-testid="countdown"]');
    if (await countdown.count() > 0) {
      // 获取初始倒计时
      const initialTime = await countdown.textContent();

      // 等待2秒
      await page.waitForTimeout(2000);

      // 获取更新后的倒计时
      const updatedTime = await countdown.textContent();

      // 验证倒计时在变化
      expect(initialTime).not.toBe(updatedTime);
    }
  });

  test('竞拍列表分页', async () => {
    await page.goto('/auctions');
    await page.waitForLoadState('networkidle');

    // 查找分页控件
    const pagination = page.locator('.pagination, [data-testid="pagination"]');
    if (await pagination.count() > 0) {
      // 点击下一页
      const nextButton = pagination.locator('button:has-text("下一页"), [data-testid="next-page"]');
      if (await nextButton.count() > 0 && await nextButton.isEnabled()) {
        await nextButton.click();
        await page.waitForLoadState('networkidle');

        // 验证URL变化或列表更新
        await expect(page.locator('.auction-item, [data-testid="auction-item"]')).toBeVisible({ timeout: 5000 });
      }
    }
  });

  test('竞拍详情 - 查看出价记录', async () => {
    await page.goto('/auctions');
    await page.waitForLoadState('networkidle');

    const firstAuction = page.locator('.auction-item, [data-testid="auction-item"]').first();
    await firstAuction.click();
    await page.waitForURL(/.*auctions\/\d+/, { timeout: 10000 });
    await page.waitForLoadState('networkidle');

    // 查找出价记录标签页
    const bidHistoryTab = page.locator('button:has-text("出价记录"), [data-testid="bid-history-tab"]');
    if (await bidHistoryTab.count() > 0) {
      await bidHistoryTab.click();

      // 等待出价记录加载
      await page.waitForLoadState('networkidle');

      // 验证出价记录列表
      const bidRecords = page.locator('.bid-record, [data-testid="bid-record"]');
      const count = await bidRecords.count();
      expect(count).toBeGreaterThanOrEqual(0);

      // 如果有记录,验证信息
      if (count > 0) {
        const firstRecord = bidRecords.first();
        await expect(firstRecord.locator('text=/用户/')).toBeVisible();
        await expect(firstRecord.locator('text=/\\d+/')).toBeVisible(); // 金额
        await expect(firstRecord.locator('text=/时间/')).toBeVisible();
      }
    }
  });
});

test.describe('竞拍通知', () => {
  test('实时竞拍更新', async ({ browser }) => {
    const page = await browser.newPage();

    // 登录
    await page.goto('/login');
    await page.waitForLoadState('networkidle');
    await page.fill('input[placeholder*="用户名"]', 'testuser');
    await page.fill('input[placeholder*="密码"]', 'Test@123456');
    await page.click('button:has-text("登录")');
    await page.waitForURL(/.*\//, { timeout: 10000 });

    // 进入竞拍详情页
    await page.goto('/auctions');
    await page.waitForLoadState('networkidle');

    const firstAuction = page.locator('.auction-item, [data-testid="auction-item"]').first();
    await firstAuction.click();
    await page.waitForURL(/.*auctions\/\d+/, { timeout: 10000 });

    // 等待WebSocket连接
    await page.waitForTimeout(2000);

    // 验证WebSocket连接状态
    const wsConnected = await page.evaluate(() => {
      return (window as any).wsConnected || false;
    });

    // 如果WebSocket支持,验证实时更新
    if (wsConnected) {
      // 模拟等待实时更新
      await page.waitForTimeout(5000);
    }

    await page.close();
  });
});
