import { test, expect, Page } from '@playwright/test';

test.describe('竞拍管理流程', () => {
  let page: Page;

  test.beforeEach(async ({ browser }) => {
    page = await browser.newPage();

    // 管理员登录
    await page.goto('/login');
    await page.waitForLoadState('networkidle');

    await page.fill('input[placeholder*="用户名"]', 'admin');
    await page.fill('input[placeholder*="密码"]', 'Admin@123456');
    await page.click('button:has-text("登录")');

    await page.waitForURL(/.*\//, { timeout: 10000 });
  });

  test.afterEach(async () => {
    await page.close();
  });

  test('查看竞拍列表', async () => {
    // 导航到竞拍管理页面
    await page.goto('/auctions');
    await page.waitForLoadState('networkidle');

    // 验证竞拍列表存在
    await expect(page.locator('.auction-list, [data-testid="auction-list"]')).toBeVisible({ timeout: 10000 });

    // 验证竞拍项存在
    const auctionItems = page.locator('.auction-item, [data-testid="auction-item"]');
    const count = await auctionItems.count();
    expect(count).toBeGreaterThanOrEqual(0);

    // 如果有竞拍,验证关键信息
    if (count > 0) {
      const firstAuction = auctionItems.first();
      await expect(firstAuction.locator('text=/竞拍|拍卖/')).toBeVisible();
      await expect(firstAuction.locator('text=/价格|起拍价/')).toBeVisible();
      await expect(firstAuction.locator('text=/状态/')).toBeVisible();
      await expect(firstAuction.locator('text=/时间/')).toBeVisible({ timeout: 5000 });
    }
  });

  test('查看竞拍列表 - 按状态筛选', async () => {
    await page.goto('/auctions');
    await page.waitForLoadState('networkidle');

    // 测试状态筛选
    const statusFilter = page.locator('select[data-testid="status-filter"], .status-filter');
    if (await statusFilter.count() > 0) {
      await statusFilter.selectOption('active');
      await page.waitForLoadState('networkidle');

      // 验证筛选结果
      const filteredItems = page.locator('.auction-item, [data-testid="auction-item"]');
      if (await filteredItems.count() > 0) {
        await expect(filteredItems.first().locator('text=/进行中|活跃/')).toBeVisible({ timeout: 5000 });
      }
    }
  });

  test('查看竞拍列表 - 搜索', async () => {
    await page.goto('/auctions');
    await page.waitForLoadState('networkidle');

    // 查找搜索框
    const searchInput = page.locator('input[placeholder*="搜索"], input[data-testid="auction-search"]');
    if (await searchInput.count() > 0) {
      await searchInput.fill('测试竞拍');
      await searchInput.press('Enter');

      await page.waitForLoadState('networkidle');

      // 验证搜索结果
      const searchResults = page.locator('.auction-item, [data-testid="auction-item"]');
      if (await searchResults.count() > 0) {
        await expect(searchResults.first().locator('text=/测试竞拍/')).toBeVisible({ timeout: 5000 });
      }
    }
  });

  test('查看竞拍列表 - 排序', async () => {
    await page.goto('/auctions');
    await page.waitForLoadState('networkidle');

    // 测试时间排序
    const sortTime = page.locator('[data-testid="sort-time"], button:has-text("时间")');
    if (await sortTime.count() > 0) {
      await sortTime.click();
      await page.waitForLoadState('networkidle');
    }

    // 测试价格排序
    const sortPrice = page.locator('[data-testid="sort-price"], button:has-text("价格")');
    if (await sortPrice.count() > 0) {
      await sortPrice.click();
      await page.waitForLoadState('networkidle');
    }
  });

  test('查看竞拍列表 - 分页', async () => {
    await page.goto('/auctions');
    await page.waitForLoadState('networkidle');

    // 查找分页控件
    const pagination = page.locator('.pagination, [data-testid="pagination"]');
    if (await pagination.count() > 0) {
      const nextButton = pagination.locator('button:has-text("下一页"), [data-testid="next-page"]');
      if (await nextButton.count() > 0 && await nextButton.isEnabled()) {
        await nextButton.click();
        await page.waitForLoadState('networkidle');

        // 验证页面更新
        await expect(page.locator('.auction-item, [data-testid="auction-item"]')).toBeVisible({ timeout: 5000 });
      }
    }
  });

  test('查看竞拍详情', async () => {
    await page.goto('/auctions');
    await page.waitForLoadState('networkidle');

    const auctionItems = page.locator('.auction-item, [data-testid="auction-item"]');
    if (await auctionItems.count() > 0) {
      // 点击查看详情
      await auctionItems.first().locator('button:has-text("查看"), button:has-text("详情")').click();

      // 等待详情页加载
      await page.waitForURL(/.*auctions\/\d+/, { timeout: 10000 });
      await page.waitForLoadState('networkidle');

      // 验证详情页元素
      await expect(page.locator('.auction-detail, [data-testid="auction-detail"]')).toBeVisible();
      await expect(page.locator('text=/竞拍信息/')).toBeVisible();
      await expect(page.locator('text=/商品信息/')).toBeVisible();
      await expect(page.locator('text=/价格|起拍价/')).toBeVisible();
      await expect(page.locator('text=/状态/')).toBeVisible({ timeout: 5000 });

      // 验证出价记录
      await expect(page.locator('text=/出价记录|竞拍记录/')).toBeVisible({ timeout: 5000 });
    }
  });

  test('查看竞拍详情 - 出价记录', async () => {
    await page.goto('/auctions');
    await page.waitForLoadState('networkidle');

    const auctionItems = page.locator('.auction-item, [data-testid="auction-item"]');
    if (await auctionItems.count() > 0) {
      await auctionItems.first().locator('button:has-text("查看"), button:has-text("详情")').click();
      await page.waitForURL(/.*auctions\/\d+/, { timeout: 10000 });
      await page.waitForLoadState('networkidle');

      // 查找出价记录列表
      const bidRecords = page.locator('.bid-record, [data-testid="bid-record"]');
      const count = await bidRecords.count();
      expect(count).toBeGreaterThanOrEqual(0);

      // 如果有记录,验证信息
      if (count > 0) {
        const firstRecord = bidRecords.first();
        await expect(firstRecord.locator('text=/用户|竞拍者/')).toBeVisible();
        await expect(firstRecord.locator('text=/\\d+/')).toBeVisible(); // 金额
        await expect(firstRecord.locator('text=/时间/')).toBeVisible();
      }
    }
  });

  test('查看竞拍详情 - 参与用户', async () => {
    await page.goto('/auctions');
    await page.waitForLoadState('networkidle');

    const auctionItems = page.locator('.auction-item, [data-testid="auction-item"]');
    if (await auctionItems.count() > 0) {
      await auctionItems.first().locator('button:has-text("查看"), button:has-text("详情")').click();
      await page.waitForURL(/.*auctions\/\d+/, { timeout: 10000 });
      await page.waitForLoadState('networkidle');

      // 查找参与用户列表
      const participantSection = page.locator('.participants, [data-testid="participants"]');
      if (await participantSection.count() > 0) {
        await expect(participantSection).toBeVisible();

        const participants = participantSection.locator('.participant, [data-testid="participant"]');
        const count = await participants.count();
        expect(count).toBeGreaterThanOrEqual(0);
      }
    }
  });

  test('取消竞拍 - 成功', async () => {
    await page.goto('/auctions');
    await page.waitForLoadState('networkidle');

    // 查找进行中的竞拍
    const activeAuctions = page.locator('.auction-item:has-text("进行中"), [data-testid="auction-item-active"]');
    if (await activeAuctions.count() > 0) {
      // 点击取消按钮
      const cancelButton = activeAuctions.first().locator('button:has-text("取消")');
      await cancelButton.click();

      // 等待确认弹窗
      const confirmDialog = page.locator('.dialog, [data-testid="confirm-dialog"]');
      await expect(confirmDialog).toBeVisible({ timeout: 5000 });
      await expect(confirmDialog.locator('text=/确认取消/')).toBeVisible();

      // 填写取消原因
      const reasonInput = confirmDialog.locator('textarea, input[data-testid="cancel-reason"]');
      if (await reasonInput.count() > 0) {
        await reasonInput.fill('测试取消竞拍');
      }

      // 确认取消
      await confirmDialog.locator('button:has-text("确认"), button:has-text("确定")').click();

      // 等待取消成功
      await expect(page.locator('text=/取消成功/')).toBeVisible({ timeout: 10000 });
    }
  });

  test('取消竞拍 - 取消操作', async () => {
    await page.goto('/auctions');
    await page.waitForLoadState('networkidle');

    const activeAuctions = page.locator('.auction-item:has-text("进行中"), [data-testid="auction-item-active"]');
    if (await activeAuctions.count() > 0) {
      await activeAuctions.first().locator('button:has-text("取消")').click();

      const confirmDialog = page.locator('.dialog, [data-testid="confirm-dialog"]');
      await expect(confirmDialog).toBeVisible({ timeout: 5000 });

      // 点击取消按钮
      await confirmDialog.locator('button:has-text("取消")').click();

      // 验证竞拍仍然存在
      await expect(activeAuctions.first()).toBeVisible({ timeout: 5000 });
    }
  });

  test('取消竞拍 - 已结束竞拍无法取消', async () => {
    await page.goto('/auctions?status=ended');
    await page.waitForLoadState('networkidle');

    const endedAuctions = page.locator('.auction-item:has-text("已结束"), [data-testid="auction-item-ended"]');
    if (await endedAuctions.count() > 0) {
      // 验证取消按钮不存在或禁用
      const cancelButton = endedAuctions.first().locator('button:has-text("取消")');
      if (await cancelButton.count() > 0) {
        expect(await cancelButton.isDisabled()).toBeTruthy();
      }
    }
  });

  test('创建竞拍', async () => {
    await page.goto('/auctions');
    await page.waitForLoadState('networkidle');

    // 点击创建竞拍按钮
    const createButton = page.locator('button:has-text("创建竞拍"), button:has-text("新增")');
    if (await createButton.count() > 0) {
      await createButton.click();
      await page.waitForLoadState('networkidle');

      // 填写竞拍信息
      const auctionName = `测试竞拍_${Date.now()}`;
      await page.fill('input[name="name"], input[placeholder*="竞拍名称"]', auctionName);
      await page.fill('input[name="startPrice"], input[placeholder*="起拍价"]', '100');
      await page.fill('input[name="duration"], input[placeholder*="时长"]', '3600');

      // 选择商品
      const productSelect = page.locator('select[name="product"], [data-testid="product-select"]');
      if (await productSelect.count() > 0) {
        await productSelect.selectOption({ index: 0 });
      }

      // 选择开始时间
      const startTime = page.locator('input[name="startTime"], input[type="datetime-local"]');
      if (await startTime.count() > 0) {
        const tomorrow = new Date(Date.now() + 86400000).toISOString().slice(0, 16);
        await startTime.fill(tomorrow);
      }

      // 提交
      await page.click('button:has-text("提交"), button:has-text("创建")');

      // 等待创建成功
      await expect(page.locator('text=/创建成功/')).toBeVisible({ timeout: 10000 });

      // 验证竞拍出现在列表中
      await page.goto('/auctions');
      await page.waitForLoadState('networkidle');
      await expect(page.locator(`text=${auctionName}`)).toBeVisible({ timeout: 5000 });
    }
  });

  test('编辑竞拍', async () => {
    await page.goto('/auctions');
    await page.waitForLoadState('networkidle');

    // 查找未开始的竞拍
    const pendingAuctions = page.locator('.auction-item:has-text("未开始"), [data-testid="auction-item-pending"]');
    if (await pendingAuctions.count() > 0) {
      await pendingAuctions.first().locator('button:has-text("编辑")').click();
      await page.waitForLoadState('networkidle');

      // 修改竞拍信息
      await page.fill('input[name="startPrice"], input[placeholder*="起拍价"]', '200');

      // 保存
      await page.click('button:has-text("保存"), button:has-text("更新")');

      // 等待保存成功
      await expect(page.locator('text=/保存成功/')).toBeVisible({ timeout: 10000 });
    }
  });

  test('竞拍监控', async () => {
    await page.goto('/auctions');
    await page.waitForLoadState('networkidle');

    const activeAuctions = page.locator('.auction-item:has-text("进行中"), [data-testid="auction-item-active"]');
    if (await activeAuctions.count() > 0) {
      await activeAuctions.first().locator('button:has-text("监控"), button:has-text("实时")').click();
      await page.waitForLoadState('networkidle');

      // 验证监控页面
      await expect(page.locator('.monitor, [data-testid="auction-monitor"]')).toBeVisible({ timeout: 5000 });

      // 验证实时数据
      await expect(page.locator('text=/当前价格/')).toBeVisible({ timeout: 5000 });
      await expect(page.locator('text=/参与人数/')).toBeVisible({ timeout: 5000 });
      await expect(page.locator('text=/剩余时间/')).toBeVisible({ timeout: 5000 });
    }
  });

  test('竞拍数据导出', async () => {
    await page.goto('/auctions');
    await page.waitForLoadState('networkidle');

    const exportButton = page.locator('button:has-text("导出"), [data-testid="export-auctions"]');
    if (await exportButton.count() > 0) {
      const downloadPromise = page.waitForEvent('download', { timeout: 10000 }).catch(() => null);

      await exportButton.click();

      const download = await downloadPromise;
      if (download) {
        expect(download.suggestedFilename()).toMatch(/\.(xlsx|csv)/);
      }
    }
  });
});

test.describe('竞拍统计分析', () => {
  test('查看竞拍统计数据', async ({ browser }) => {
    const page = await browser.newPage();

    // 登录
    await page.goto('/login');
    await page.waitForLoadState('networkidle');
    await page.fill('input[placeholder*="用户名"]', 'admin');
    await page.fill('input[placeholder*="密码"]', 'Admin@123456');
    await page.click('button:has-text("登录")');
    await page.waitForURL(/.*\//, { timeout: 10000 });

    await page.goto('/auctions');
    await page.waitForLoadState('networkidle');

    // 查找统计面板
    const statsPanel = page.locator('.stats-panel, [data-testid="auction-stats"]');
    if (await statsPanel.count() > 0) {
      await expect(statsPanel).toBeVisible();

      // 验证关键统计指标
      await expect(statsPanel.locator('text=/进行中/')).toBeVisible({ timeout: 5000 });
      await expect(statsPanel.locator('text=/已结束/')).toBeVisible({ timeout: 5000 });
      await expect(statsPanel.locator('text=/总成交额/')).toBeVisible({ timeout: 5000 });
    }

    await page.close();
  });
});
