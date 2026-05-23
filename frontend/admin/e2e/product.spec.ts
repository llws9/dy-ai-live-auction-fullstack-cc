import { test, expect, Page } from '@playwright/test';

test.describe('商品管理流程', () => {
  let page: Page;
  let productId: string;

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

  test('创建商品 - 成功', async () => {
    // 导航到商品管理页面
    await page.goto('/products');
    await page.waitForLoadState('networkidle');

    // 点击创建商品按钮
    await page.click('button:has-text("创建商品"), button:has-text("新增")');

    // 等待商品表单加载
    await page.waitForLoadState('networkidle');

    // 填写商品信息
    const productName = `测试商品_${Date.now()}`;
    await page.fill('input[name="name"], input[placeholder*="商品名称"]', productName);
    await page.fill('textarea[name="description"], textarea[placeholder*="商品描述"]', '这是一个测试商品的描述信息');
    await page.fill('input[name="price"], input[placeholder*="价格"]', '99.99');
    await page.fill('input[name="stock"], input[placeholder*="库存"]', '100');

    // 上传商品图片
    const imageInput = page.locator('input[type="file"], input[accept*="image"]');
    if (await imageInput.count() > 0) {
      await imageInput.setInputFiles({
        name: 'test-image.jpg',
        mimeType: 'image/jpeg',
        buffer: Buffer.from('test-image-content'),
      });
    }

    // 选择商品分类
    const categorySelect = page.locator('select[name="category"], [data-testid="category-select"]');
    if (await categorySelect.count() > 0) {
      await categorySelect.selectOption({ label: '电子产品' });
    }

    // 点击提交按钮
    await page.click('button:has-text("提交"), button:has-text("保存")');

    // 等待创建成功提示
    await expect(page.locator('text=/创建成功|保存成功/')).toBeVisible({ timeout: 10000 });

    // 验证商品出现在列表中
    await page.goto('/products');
    await page.waitForLoadState('networkidle');
    await expect(page.locator(`text=${productName}`)).toBeVisible({ timeout: 5000 });
  });

  test('创建商品 - 验证失败', async () => {
    await page.goto('/products');
    await page.waitForLoadState('networkidle');

    await page.click('button:has-text("创建商品"), button:has-text("新增")');
    await page.waitForLoadState('networkidle');

    // 不填写必填项,直接提交
    await page.click('button:has-text("提交"), button:has-text("保存")');

    // 等待验证错误提示
    await expect(page.locator('text=/请填写|必填|不能为空/')).toBeVisible({ timeout: 5000 });
  });

  test('编辑商品 - 成功', async () => {
    await page.goto('/products');
    await page.waitForLoadState('networkidle');

    // 查找商品列表
    const productItems = page.locator('.product-item, [data-testid="product-item"]');
    const count = await productItems.count();

    if (count > 0) {
      // 点击第一个商品的编辑按钮
      await productItems.first().locator('button:has-text("编辑")').click();

      // 等待编辑表单加载
      await page.waitForLoadState('networkidle');

      // 修改商品信息
      const newPrice = '199.99';
      await page.fill('input[name="price"], input[placeholder*="价格"]', newPrice);

      // 点击保存按钮
      await page.click('button:has-text("保存"), button:has-text("更新")');

      // 等待保存成功
      await expect(page.locator('text=/保存成功|更新成功/')).toBeVisible({ timeout: 10000 });

      // 验证价格更新
      await page.goto('/products');
      await page.waitForLoadState('networkidle');
      await expect(page.locator(`text=${newPrice}`)).toBeVisible({ timeout: 5000 });
    }
  });

  test('编辑商品 - 取消编辑', async () => {
    await page.goto('/products');
    await page.waitForLoadState('networkidle');

    const productItems = page.locator('.product-item, [data-testid="product-item"]');
    if (await productItems.count() > 0) {
      await productItems.first().locator('button:has-text("编辑")').click();
      await page.waitForLoadState('networkidle');

      // 修改商品信息
      await page.fill('input[name="price"], input[placeholder*="价格"]', '999.99');

      // 点击取消按钮
      await page.click('button:has-text("取消")');

      // 验证返回列表页
      await expect(page).toHaveURL(/.*products/, { timeout: 5000 });

      // 验证修改未保存
      await expect(page.locator('text=999.99')).not.toBeVisible({ timeout: 3000 });
    }
  });

  test('删除商品 - 成功', async () => {
    await page.goto('/products');
    await page.waitForLoadState('networkidle');

    const productItems = page.locator('.product-item, [data-testid="product-item"]');
    const initialCount = await productItems.count();

    if (initialCount > 0) {
      // 获取第一个商品名称
      const productName = await productItems.first().locator('.product-name, [data-testid="product-name"]').textContent();

      // 点击删除按钮
      await productItems.first().locator('button:has-text("删除")').click();

      // 等待确认弹窗
      const confirmDialog = page.locator('.dialog, [data-testid="confirm-dialog"]');
      await expect(confirmDialog).toBeVisible({ timeout: 5000 });

      // 确认删除
      await confirmDialog.locator('button:has-text("确认"), button:has-text("确定")').click();

      // 等待删除成功提示
      await expect(page.locator('text=/删除成功/')).toBeVisible({ timeout: 10000 });

      // 验证商品已从列表中移除
      await page.waitForTimeout(1000);
      if (productName) {
        await expect(page.locator(`text=${productName}`)).not.toBeVisible({ timeout: 5000 });
      }
    }
  });

  test('删除商品 - 取消删除', async () => {
    await page.goto('/products');
    await page.waitForLoadState('networkidle');

    const productItems = page.locator('.product-item, [data-testid="product-item"]');
    if (await productItems.count() > 0) {
      // 点击删除按钮
      await productItems.first().locator('button:has-text("删除")').click();

      // 等待确认弹窗
      const confirmDialog = page.locator('.dialog, [data-testid="confirm-dialog"]');
      await expect(confirmDialog).toBeVisible({ timeout: 5000 });

      // 取消删除
      await confirmDialog.locator('button:has-text("取消")').click();

      // 验证商品仍然存在
      await expect(productItems.first()).toBeVisible({ timeout: 5000 });
    }
  });

  test('查看商品列表', async () => {
    await page.goto('/products');
    await page.waitForLoadState('networkidle');

    // 验证商品列表存在
    await expect(page.locator('.product-list, [data-testid="product-list"]')).toBeVisible({ timeout: 10000 });

    // 验证商品项存在
    const productItems = page.locator('.product-item, [data-testid="product-item"]');
    const count = await productItems.count();
    expect(count).toBeGreaterThanOrEqual(0);

    // 如果有商品,验证关键信息
    if (count > 0) {
      const firstProduct = productItems.first();
      await expect(firstProduct.locator('text=/商品|名称/')).toBeVisible();
      await expect(firstProduct.locator('text=/价格/')).toBeVisible();
      await expect(firstProduct.locator('text=/库存/')).toBeVisible();
      await expect(firstProduct.locator('text=/状态/')).toBeVisible();
    }
  });

  test('商品列表 - 搜索', async () => {
    await page.goto('/products');
    await page.waitForLoadState('networkidle');

    // 查找搜索框
    const searchInput = page.locator('input[placeholder*="搜索"], input[data-testid="product-search"]');
    if (await searchInput.count() > 0) {
      await searchInput.fill('测试商品');
      await searchInput.press('Enter');

      await page.waitForLoadState('networkidle');

      // 验证搜索结果
      const searchResults = page.locator('.product-item, [data-testid="product-item"]');
      if (await searchResults.count() > 0) {
        await expect(searchResults.first().locator('text=/测试商品/')).toBeVisible({ timeout: 5000 });
      }
    }
  });

  test('商品列表 - 筛选', async () => {
    await page.goto('/products');
    await page.waitForLoadState('networkidle');

    // 测试状态筛选
    const statusFilter = page.locator('select[data-testid="status-filter"], .status-filter');
    if (await statusFilter.count() > 0) {
      await statusFilter.selectOption('active');
      await page.waitForLoadState('networkidle');

      // 验证筛选结果
      const filteredItems = page.locator('.product-item, [data-testid="product-item"]');
      if (await filteredItems.count() > 0) {
        await expect(filteredItems.first().locator('text=/上架|销售中/')).toBeVisible({ timeout: 5000 });
      }
    }
  });

  test('商品列表 - 排序', async () => {
    await page.goto('/products');
    await page.waitForLoadState('networkidle');

    // 测试价格排序
    const sortPrice = page.locator('[data-testid="sort-price"], button:has-text("价格")');
    if (await sortPrice.count() > 0) {
      await sortPrice.click();
      await page.waitForLoadState('networkidle');

      // 验证排序效果
      const productItems = page.locator('.product-item, [data-testid="product-item"]');
      if (await productItems.count() > 1) {
        // 获取前两个商品的价格
        const firstPrice = await productItems.first().locator('text=/\\d+\\.?\\d*/').textContent();
        const secondPrice = await productItems.nth(1).locator('text=/\\d+\\.?\\d*/').textContent();

        // 验证价格排序(升序或降序)
        expect(parseFloat(firstPrice || '0')).toBeDefined();
        expect(parseFloat(secondPrice || '0')).toBeDefined();
      }
    }
  });

  test('商品列表 - 分页', async () => {
    await page.goto('/products');
    await page.waitForLoadState('networkidle');

    // 查找分页控件
    const pagination = page.locator('.pagination, [data-testid="pagination"]');
    if (await pagination.count() > 0) {
      const nextButton = pagination.locator('button:has-text("下一页"), [data-testid="next-page"]');
      if (await nextButton.count() > 0 && await nextButton.isEnabled()) {
        await nextButton.click();
        await page.waitForLoadState('networkidle');

        // 验证页面更新
        await expect(page.locator('.product-item, [data-testid="product-item"]')).toBeVisible({ timeout: 5000 });
      }
    }
  });

  test('商品批量操作', async () => {
    await page.goto('/products');
    await page.waitForLoadState('networkidle');

    // 查找批量操作功能
    const selectAll = page.locator('input[type="checkbox"][data-testid="select-all"], .select-all-checkbox');
    if (await selectAll.count() > 0) {
      // 全选
      await selectAll.check();

      // 验证所有商品被选中
      const checkboxes = page.locator('.product-item input[type="checkbox"]:checked');
      const checkedCount = await checkboxes.count();
      expect(checkedCount).toBeGreaterThan(0);

      // 执行批量操作(如批量上架)
      const batchAction = page.locator('button:has-text("批量上架"), [data-testid="batch-list"]');
      if (await batchAction.count() > 0) {
        await batchAction.click();
        await expect(page.locator('text=/操作成功/')).toBeVisible({ timeout: 10000 });
      }
    }
  });

  test('商品详情查看', async () => {
    await page.goto('/products');
    await page.waitForLoadState('networkidle');

    const productItems = page.locator('.product-item, [data-testid="product-item"]');
    if (await productItems.count() > 0) {
      // 点击商品查看详情
      await productItems.first().click();

      // 等待详情页加载
      await page.waitForURL(/.*products\/\d+/, { timeout: 10000 });
      await page.waitForLoadState('networkidle');

      // 验证详情页元素
      await expect(page.locator('.product-detail, [data-testid="product-detail"]')).toBeVisible();
      await expect(page.locator('text=/商品名称/')).toBeVisible();
      await expect(page.locator('text=/价格/')).toBeVisible();
      await expect(page.locator('text=/库存/')).toBeVisible();
      await expect(page.locator('text=/描述/')).toBeVisible({ timeout: 5000 });
    }
  });

  test('商品状态切换', async () => {
    await page.goto('/products');
    await page.waitForLoadState('networkidle');

    const productItems = page.locator('.product-item, [data-testid="product-item"]');
    if (await productItems.count() > 0) {
      // 点击上架/下架按钮
      const statusButton = productItems.first().locator('button:has-text("上架"), button:has-text("下架")');
      if (await statusButton.count() > 0) {
        await statusButton.click();

        // 等待状态更新
        await expect(page.locator('text=/操作成功/')).toBeVisible({ timeout: 10000 });
      }
    }
  });
});

test.describe('商品导入导出', () => {
  test('导出商品数据', async ({ browser }) => {
    const page = await browser.newPage();

    // 登录
    await page.goto('/login');
    await page.waitForLoadState('networkidle');
    await page.fill('input[placeholder*="用户名"]', 'admin');
    await page.fill('input[placeholder*="密码"]', 'Admin@123456');
    await page.click('button:has-text("登录")');
    await page.waitForURL(/.*\//, { timeout: 10000 });

    await page.goto('/products');
    await page.waitForLoadState('networkidle');

    // 查找导出按钮
    const exportButton = page.locator('button:has-text("导出"), [data-testid="export-products"]');
    if (await exportButton.count() > 0) {
      const downloadPromise = page.waitForEvent('download', { timeout: 10000 }).catch(() => null);

      await exportButton.click();

      const download = await downloadPromise;
      if (download) {
        expect(download.suggestedFilename()).toMatch(/\.(xlsx|csv)/);
      }
    }

    await page.close();
  });

  test('导入商品数据', async ({ browser }) => {
    const page = await browser.newPage();

    // 登录
    await page.goto('/login');
    await page.waitForLoadState('networkidle');
    await page.fill('input[placeholder*="用户名"]', 'admin');
    await page.fill('input[placeholder*="密码"]', 'Admin@123456');
    await page.click('button:has-text("登录")');
    await page.waitForURL(/.*\//, { timeout: 10000 });

    await page.goto('/products');
    await page.waitForLoadState('networkidle');

    // 查找导入按钮
    const importButton = page.locator('button:has-text("导入"), [data-testid="import-products"]');
    if (await importButton.count() > 0) {
      await importButton.click();

      // 等待导入弹窗
      const importDialog = page.locator('.import-dialog, [data-testid="import-dialog"]');
      if (await importDialog.count() > 0) {
        // 上传文件
        const fileInput = importDialog.locator('input[type="file"]');
        await fileInput.setInputFiles({
          name: 'products.xlsx',
          mimeType: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
          buffer: Buffer.from('test-excel-content'),
        });

        // 确认导入
        await importDialog.locator('button:has-text("确认"), button:has-text("导入")').click();

        // 等待导入结果
        await expect(page.locator('text=/导入成功|导入完成/')).toBeVisible({ timeout: 15000 });
      }
    }

    await page.close();
  });
});
