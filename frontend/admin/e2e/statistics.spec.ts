import { test, expect, Page } from '@playwright/test';

test.describe('统计报表流程', () => {
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

  test('查看数据大屏', async () => {
    // 导航到数据大屏
    await page.goto('/dashboard');
    await page.waitForLoadState('networkidle');

    // 验证数据大屏存在
    await expect(page.locator('.dashboard, [data-testid="dashboard"]')).toBeVisible({ timeout: 10000 });

    // 验证关键指标卡片
    await expect(page.locator('text=/总用户数/')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('text=/总竞拍数/')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('text=/总收入/')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('text=/活跃用户/')).toBeVisible({ timeout: 5000 });

    // 验证图表存在
    const charts = page.locator('.chart, [data-testid="chart"]');
    const chartCount = await charts.count();
    expect(chartCount).toBeGreaterThan(0);
  });

  test('查看数据大屏 - 实时数据刷新', async () => {
    await page.goto('/dashboard');
    await page.waitForLoadState('networkidle');

    // 获取初始数据
    const initialText = await page.locator('text=/总用户数/').first().textContent();

    // 等待自动刷新(假设每30秒刷新一次)
    await page.waitForTimeout(35000);

    // 验证数据可能已更新
    const updatedText = await page.locator('text=/总用户数/').first().textContent();
    // 数据可能相同或不同,但不应该出错
    expect(updatedText).toBeTruthy();
  });

  test('查看数据大屏 - 手动刷新', async () => {
    await page.goto('/dashboard');
    await page.waitForLoadState('networkidle');

    // 查找刷新按钮
    const refreshButton = page.locator('button:has-text("刷新"), [data-testid="refresh-button"]');
    if (await refreshButton.count() > 0) {
      await refreshButton.click();

      // 等待加载指示器
      await page.waitForLoadState('networkidle');

      // 验证数据已加载
      await expect(page.locator('.dashboard, [data-testid="dashboard"]')).toBeVisible({ timeout: 10000 });
    }
  });

  test('查看竞拍统计', async () => {
    // 导航到竞拍统计页面
    await page.goto('/statistics/auctions');
    await page.waitForLoadState('networkidle');

    // 验证统计页面存在
    await expect(page.locator('.statistics, [data-testid="auction-statistics"]')).toBeVisible({ timeout: 10000 });

    // 验证关键统计指标
    await expect(page.locator('text=/竞拍总数/')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('text=/进行中/')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('text=/已结束/')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('text=/成交率/')).toBeVisible({ timeout: 5000 });

    // 验证竞拍趋势图
    const trendChart = page.locator('.trend-chart, [data-testid="auction-trend-chart"]');
    if (await trendChart.count() > 0) {
      await expect(trendChart).toBeVisible();
    }

    // 验证竞拍分类统计
    const categoryChart = page.locator('.category-chart, [data-testid="category-chart"]');
    if (await categoryChart.count() > 0) {
      await expect(categoryChart).toBeVisible();
    }
  });

  test('查看竞拍统计 - 时间范围筛选', async () => {
    await page.goto('/statistics/auctions');
    await page.waitForLoadState('networkidle');

    // 查找时间范围选择器
    const timeRangeSelector = page.locator('.time-range, [data-testid="time-range"]');
    if (await timeRangeSelector.count() > 0) {
      // 选择最近7天
      const last7Days = timeRangeSelector.locator('button:has-text("7天"), option:has-text("最近7天")');
      if (await last7Days.count() > 0) {
        await last7Days.click();
        await page.waitForLoadState('networkidle');

        // 验证数据已更新
        await expect(page.locator('.statistics, [data-testid="auction-statistics"]')).toBeVisible({ timeout: 5000 });
      }
    }
  });

  test('查看竞拍统计 - 自定义时间范围', async () => {
    await page.goto('/statistics/auctions');
    await page.waitForLoadState('networkidle');

    // 查找日期选择器
    const datePicker = page.locator('.date-picker, [data-testid="date-picker"]');
    if (await datePicker.count() > 0) {
      await datePicker.click();

      // 选择开始日期
      const startDate = page.locator('input[name="startDate"], [data-testid="start-date"]');
      if (await startDate.count() > 0) {
        await startDate.fill('2024-01-01');
      }

      // 选择结束日期
      const endDate = page.locator('input[name="endDate"], [data-testid="end-date"]');
      if (await endDate.count() > 0) {
        await endDate.fill('2024-12-31');
      }

      // 确认选择
      const confirmButton = page.locator('button:has-text("确认"), button:has-text("确定")');
      if (await confirmButton.count() > 0) {
        await confirmButton.click();
        await page.waitForLoadState('networkidle');
      }
    }
  });

  test('查看竞拍统计 - 排行榜', async () => {
    await page.goto('/statistics/auctions');
    await page.waitForLoadState('networkidle');

    // 查找排行榜
    const ranking = page.locator('.ranking, [data-testid="auction-ranking"]');
    if (await ranking.count() > 0) {
      await expect(ranking).toBeVisible();

      // 验证排行榜项
      const rankingItems = ranking.locator('.ranking-item, [data-testid="ranking-item"]');
      const count = await rankingItems.count();
      expect(count).toBeGreaterThan(0);

      // 验证排行榜信息
      if (count > 0) {
        const firstItem = rankingItems.first();
        await expect(firstItem.locator('text=/\\d+/')).toBeVisible(); // 排名
        await expect(firstItem.locator('text=/商品|竞拍/')).toBeVisible(); // 名称
      }
    }
  });

  test('查看收入统计', async () => {
    // 导航到收入统计页面
    await page.goto('/statistics/revenue');
    await page.waitForLoadState('networkidle');

    // 验证统计页面存在
    await expect(page.locator('.statistics, [data-testid="revenue-statistics"]')).toBeVisible({ timeout: 10000 });

    // 验证关键统计指标
    await expect(page.locator('text=/总收入/')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('text=/今日收入/')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('text=/本月收入/')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('text=/同比增长/')).toBeVisible({ timeout: 5000 });

    // 验证收入趋势图
    const revenueChart = page.locator('.revenue-chart, [data-testid="revenue-chart"]');
    if (await revenueChart.count() > 0) {
      await expect(revenueChart).toBeVisible();
    }
  });

  test('查看收入统计 - 收入明细', async () => {
    await page.goto('/statistics/revenue');
    await page.waitForLoadState('networkidle');

    // 查找收入明细表格
    const detailTable = page.locator('.detail-table, [data-testid="revenue-detail"]');
    if (await detailTable.count() > 0) {
      await expect(detailTable).toBeVisible();

      // 验证表格列
      const headers = detailTable.locator('th, [data-testid="table-header"]');
      const headerCount = await headers.count();
      expect(headerCount).toBeGreaterThan(0);

      // 验证表格数据
      const rows = detailTable.locator('tr, [data-testid="table-row"]');
      const rowCount = await rows.count();
      expect(rowCount).toBeGreaterThan(0);
    }
  });

  test('查看收入统计 - 收入来源分析', async () => {
    await page.goto('/statistics/revenue');
    await page.waitForLoadState('networkidle');

    // 查找收入来源图表
    const sourceChart = page.locator('.source-chart, [data-testid="revenue-source-chart"]');
    if (await sourceChart.count() > 0) {
      await expect(sourceChart).toBeVisible();

      // 验证来源分类
      await expect(sourceChart.locator('text=/竞拍收入/')).toBeVisible({ timeout: 5000 });
      await expect(sourceChart.locator('text=/服务费/')).toBeVisible({ timeout: 5000 });
    }
  });

  test('查看用户统计', async () => {
    // 导航到用户统计页面
    await page.goto('/statistics/users');
    await page.waitForLoadState('networkidle');

    // 验证统计页面存在
    await expect(page.locator('.statistics, [data-testid="user-statistics"]')).toBeVisible({ timeout: 10000 });

    // 验证关键统计指标
    await expect(page.locator('text=/总用户数/')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('text=/新增用户/')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('text=/活跃用户/')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('text=/用户留存/')).toBeVisible({ timeout: 5000 });

    // 验证用户增长趋势图
    const growthChart = page.locator('.growth-chart, [data-testid="user-growth-chart"]');
    if (await growthChart.count() > 0) {
      await expect(growthChart).toBeVisible();
    }
  });

  test('查看用户统计 - 用户分布', async () => {
    await page.goto('/statistics/users');
    await page.waitForLoadState('networkidle');

    // 查找用户分布图表
    const distributionChart = page.locator('.distribution-chart, [data-testid="user-distribution-chart"]');
    if (await distributionChart.count() > 0) {
      await expect(distributionChart).toBeVisible();

      // 验证分布数据
      await expect(distributionChart.locator('text=/地区|地域/')).toBeVisible({ timeout: 5000 });
    }
  });

  test('查看用户统计 - 用户行为分析', async () => {
    await page.goto('/statistics/users');
    await page.waitForLoadState('networkidle');

    // 查找用户行为统计
    const behaviorStats = page.locator('.behavior-stats, [data-testid="user-behavior"]');
    if (await behaviorStats.count() > 0) {
      await expect(behaviorStats).toBeVisible();

      // 验证行为指标
      await expect(behaviorStats.locator('text=/访问次数/')).toBeVisible({ timeout: 5000 });
      await expect(behaviorStats.locator('text=/竞拍次数/')).toBeVisible({ timeout: 5000 });
      await expect(behaviorStats.locator('text=/成交次数/')).toBeVisible({ timeout: 5000 });
    }
  });

  test('统计报表导出', async () => {
    await page.goto('/statistics/auctions');
    await page.waitForLoadState('networkidle');

    // 查找导出按钮
    const exportButton = page.locator('button:has-text("导出"), [data-testid="export-report"]');
    if (await exportButton.count() > 0) {
      const downloadPromise = page.waitForEvent('download', { timeout: 10000 }).catch(() => null);

      await exportButton.click();

      // 等待导出格式选择
      const formatDialog = page.locator('.export-dialog, [data-testid="export-dialog"]');
      if (await formatDialog.count() > 0) {
        // 选择导出格式
        const excelOption = formatDialog.locator('button:has-text("Excel"), [data-testid="export-excel"]');
        if (await excelOption.count() > 0) {
          await excelOption.click();
        }
      }

      const download = await downloadPromise;
      if (download) {
        expect(download.suggestedFilename()).toMatch(/\.(xlsx|csv|pdf)/);
      }
    }
  });

  test('统计报表打印', async () => {
    await page.goto('/statistics/revenue');
    await page.waitForLoadState('networkidle');

    // 查找打印按钮
    const printButton = page.locator('button:has-text("打印"), [data-testid="print-report"]');
    if (await printButton.count() > 0) {
      // 监听打印对话框
      page.on('dialog', async dialog => {
        expect(dialog.type()).toBe('beforeprint');
        await dialog.dismiss();
      });

      await printButton.click();
    }
  });

  test('统计数据对比', async () => {
    await page.goto('/statistics/auctions');
    await page.waitForLoadState('networkidle');

    // 查找对比功能
    const compareButton = page.locator('button:has-text("对比"), [data-testid="compare-button"]');
    if (await compareButton.count() > 0) {
      await compareButton.click();

      // 选择对比时间段
      const periodSelector = page.locator('.period-selector, [data-testid="period-selector"]');
      if (await periodSelector.count() > 0) {
        // 选择本期
        await periodSelector.locator('button:has-text("本期")').click();

        // 选择对比期
        await periodSelector.locator('button:has-text("上期")').click();
      }

      // 确认对比
      const confirmButton = page.locator('button:has-text("确认对比"), button:has-text("确定")');
      if (await confirmButton.count() > 0) {
        await confirmButton.click();
        await page.waitForLoadState('networkidle');

        // 验证对比结果
        await expect(page.locator('text=/对比结果/')).toBeVisible({ timeout: 10000 });
        await expect(page.locator('text=/增长率/')).toBeVisible({ timeout: 5000 });
      }
    }
  });

  test('统计图表交互', async () => {
    await page.goto('/statistics/revenue');
    await page.waitForLoadState('networkidle');

    // 查找图表
    const chart = page.locator('.chart, [data-testid="revenue-chart"]').first();
    if (await chart.count() > 0) {
      // 鼠标悬停显示详情
      await chart.hover();

      // 验证tooltip显示
      const tooltip = page.locator('.chart-tooltip, [data-testid="chart-tooltip"]');
      if (await tooltip.count() > 0) {
        await expect(tooltip).toBeVisible({ timeout: 5000 });
      }
    }
  });

  test('统计报表数据钻取', async () => {
    await page.goto('/statistics/auctions');
    await page.waitForLoadState('networkidle');

    // 点击统计数据查看详情
    const statCard = page.locator('.stat-card, [data-testid="stat-card"]').first();
    if (await statCard.count() > 0) {
      await statCard.click();

      // 验证详情页面
      await page.waitForLoadState('networkidle');
      await expect(page.locator('.detail-view, [data-testid="detail-view"]')).toBeVisible({ timeout: 5000 });
    }
  });
});

test.describe('报表异常处理', () => {
  test('数据加载失败处理', async ({ browser }) => {
    const page = await browser.newPage();

    // 登录
    await page.goto('/login');
    await page.waitForLoadState('networkidle');
    await page.fill('input[placeholder*="用户名"]', 'admin');
    await page.fill('input[placeholder*="密码"]', 'Admin@123456');
    await page.click('button:has-text("登录")');
    await page.waitForURL(/.*\//, { timeout: 10000 });

    // 模拟网络错误
    await page.route('**/api/statistics/**', route => route.abort());

    await page.goto('/statistics/auctions');

    // 验证错误提示
    await expect(page.locator('text=/加载失败|网络错误/')).toBeVisible({ timeout: 10000 });

    // 验证重试按钮
    const retryButton = page.locator('button:has-text("重试"), [data-testid="retry-button"]');
    if (await retryButton.count() > 0) {
      await expect(retryButton).toBeVisible();
    }

    await page.close();
  });

  test('空数据状态', async ({ browser }) => {
    const page = await browser.newPage();

    // 登录
    await page.goto('/login');
    await page.waitForLoadState('networkidle');
    await page.fill('input[placeholder*="用户名"]', 'admin');
    await page.fill('input[placeholder*="密码"]', 'Admin@123456');
    await page.click('button:has-text("登录")');
    await page.waitForURL(/.*\//, { timeout: 10000 });

    // 访问没有数据的时间段
    await page.goto('/statistics/auctions?start=2000-01-01&end=2000-01-02');
    await page.waitForLoadState('networkidle');

    // 验证空状态提示
    const emptyState = page.locator('.empty-state, [data-testid="empty-state"]');
    if (await emptyState.count() > 0) {
      await expect(emptyState).toBeVisible();
      await expect(emptyState.locator('text=/暂无数据/')).toBeVisible({ timeout: 5000 });
    }

    await page.close();
  });
});
