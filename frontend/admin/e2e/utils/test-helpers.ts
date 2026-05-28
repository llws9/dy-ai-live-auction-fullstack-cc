import { Page } from '@playwright/test';

/**
 * Admin端测试工具函数
 */

/**
 * 管理员登录函数
 */
export async function adminLogin(
  page: Page,
  email: string = 'admin@example.com',
  password: string = 'Admin@123456'
) {
  await page.goto('/admin-login');
  await page.waitForLoadState('networkidle');

  await page.fill('input[placeholder*="邮箱"]', email);
  await page.fill('input[placeholder*="密码"]', password);
  await page.click('button:has-text("登录")');

  // 等待登录成功后跳转
  await page.waitForURL(/.*dashboard/, { timeout: 10000 });

  // 验证登录成功
  const token = await page.evaluate(() => localStorage.getItem('admin_auth_token'));
  return token !== null;
}

/**
 * 退出登录函数
 */
export async function logout(page: Page) {
  await page.click('button:has-text("退出"), [data-testid="logout-button"]');
  await page.waitForURL(/.*admin-login/, { timeout: 10000 });

  const token = await page.evaluate(() => localStorage.getItem('admin_auth_token'));
  return token === null;
}

/**
 * 导航到菜单项
 */
export async function navigateToMenu(page: Page, menuText: string) {
  const menu = page.locator(`.menu-item:has-text("${menuText}"), [data-testid="menu-${menuText}"]`);
  await menu.click();
  await page.waitForLoadState('networkidle');
}

/**
 * 等待Toast消息
 */
export async function waitForToast(page: Page, text: string, timeout = 10000) {
  await page.waitForSelector(`.toast:has-text("${text}"), [data-testid="toast"]:has-text("${text}")`, {
    timeout,
  });
}

/**
 * 确认对话框
 */
export async function confirmDialog(page: Page, confirm = true) {
  const dialog = page.locator('.dialog, [data-testid="confirm-dialog"]');
  if (await dialog.count() > 0) {
    const button = confirm
      ? dialog.locator('button:has-text("确认"), button:has-text("确定")')
      : dialog.locator('button:has-text("取消")');
    await button.click();
  }
}

/**
 * 填写表单
 */
export async function fillForm(page: Page, fields: Record<string, string>) {
  for (const [selector, value] of Object.entries(fields)) {
    const element = page.locator(selector);
    await element.fill(value);
  }
}

/**
 * 生成随机字符串
 */
export function randomString(length = 8) {
  return Math.random().toString(36).substring(2, length + 2);
}

/**
 * 生成随机商品名称
 */
export function randomProductName() {
  return `测试商品_${Date.now()}_${randomString(4)}`;
}

/**
 * 生成随机竞拍名称
 */
export function randomAuctionName() {
  return `测试竞拍_${Date.now()}_${randomString(4)}`;
}

/**
 * 等待API响应
 */
export async function waitForAPI(page: Page, urlPattern: string | RegExp, timeout = 10000) {
  return page.waitForResponse(response => {
    const url = response.url();
    if (typeof urlPattern === 'string') {
      return url.includes(urlPattern);
    }
    return urlPattern.test(url);
  }, { timeout });
}

/**
 * 清除存储
 */
export async function clearStorage(page: Page) {
  try {
    await page.evaluate(() => {
      localStorage.clear();
      sessionStorage.clear();
    });
  } catch {
    // 如果页面还没有加载完成或无法访问 localStorage，忽略错误
  }
}

/**
 * 获取表格行数
 */
export async function getTableRowCount(page: Page, tableSelector: string) {
  const rows = page.locator(`${tableSelector} tr, ${tableSelector} [data-testid="table-row"]`);
  return rows.count();
}

/**
 * 检查权限
 */
export async function checkPermission(page: Page, permissionText: string) {
  const element = page.locator(`text=${permissionText}`);
  return element.count() > 0;
}

/**
 * 切换侧边栏
 */
export async function toggleSidebar(page: Page) {
  const toggleButton = page.locator('.sidebar-toggle, [data-testid="sidebar-toggle"]');
  if (await toggleButton.count() > 0) {
    await toggleButton.click();
  }
}

/**
 * 获取统计数据
 */
export async function getStatValue(page: Page, statName: string) {
  const statCard = page.locator(`.stat-card:has-text("${statName}"), [data-testid="stat-${statName}"]`);
  if (await statCard.count() > 0) {
    const valueText = await statCard.locator('.stat-value, [data-testid="stat-value"]').textContent();
    return valueText?.match(/[\d,]+/)?.[0] || '0';
  }
  return '0';
}

/**
 * 等待加载完成
 */
export async function waitForLoading(page: Page, timeout = 10000) {
  const loading = page.locator('.loading, [data-testid="loading"]');
  if (await loading.count() > 0) {
    await loading.waitFor({ state: 'hidden', timeout });
  }
}

/**
 * 选择日期范围
 */
export async function selectDateRange(
  page: Page,
  startDate: string,
  endDate: string
) {
  const datePicker = page.locator('.date-picker, [data-testid="date-picker"]');
  if (await datePicker.count() > 0) {
    await datePicker.click();

    const startInput = page.locator('input[name="startDate"], [data-testid="start-date"]');
    if (await startInput.count() > 0) {
      await startInput.fill(startDate);
    }

    const endInput = page.locator('input[name="endDate"], [data-testid="end-date"]');
    if (await endInput.count() > 0) {
      await endInput.fill(endDate);
    }

    const confirmButton = page.locator('button:has-text("确认"), button:has-text("确定")');
    if (await confirmButton.count() > 0) {
      await confirmButton.click();
    }
  }
}

/**
 * 导出数据
 */
export async function exportData(page: Page, format: 'excel' | 'csv' | 'pdf' = 'excel') {
  const exportButton = page.locator('button:has-text("导出"), [data-testid="export-button"]');
  if (await exportButton.count() > 0) {
    const downloadPromise = page.waitForEvent('download', { timeout: 15000 });

    await exportButton.click();

    // 选择导出格式
    const formatButton = page.locator(`button:has-text("${format}"), [data-testid="export-${format}"]`);
    if (await formatButton.count() > 0) {
      await formatButton.click();
    }

    const download = await downloadPromise;
    return download?.suggestedFilename();
  }
  return null;
}
