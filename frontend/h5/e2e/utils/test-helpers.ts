import { Page } from '@playwright/test';

/**
 * 测试工具函数
 */

/**
 * 登录函数
 */
export async function login(
  page: Page,
  username: string = 'testuser',
  password: string = 'Test@123456'
) {
  await page.goto('/login');
  await page.waitForLoadState('networkidle');

  await page.fill('input[placeholder*="用户名"]', username);
  await page.fill('input[placeholder*="密码"]', password);
  await page.click('button:has-text("登录")');

  await page.waitForURL(/.*\//, { timeout: 10000 });

  // 验证登录成功
  const token = await page.evaluate(() => localStorage.getItem('token'));
  return token !== null;
}

/**
 * 管理员登录函数
 */
export async function adminLogin(page: Page) {
  return login(page, 'admin', 'Admin@123456');
}

/**
 * 退出登录函数
 */
export async function logout(page: Page) {
  await page.click('button:has-text("退出"), [data-testid="logout-button"]');
  await page.waitForURL(/.*login/, { timeout: 10000 });

  // 验证Token已清除
  const token = await page.evaluate(() => localStorage.getItem('token'));
  return token === null;
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
 * 关闭弹窗
 */
export async function closeModal(page: Page) {
  const closeButton = page.locator('.modal-close, [data-testid="modal-close"]');
  if (await closeButton.count() > 0) {
    await closeButton.click();
  }
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
 * 生成随机邮箱
 */
export function randomEmail() {
  return `test_${Date.now()}@test.com`;
}

/**
 * 生成随机用户名
 */
export function randomUsername() {
  return `user_${Date.now()}_${randomString(4)}`;
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
 * 模拟网络离线
 */
export async function goOffline(page: Page) {
  await page.context().setOffline(true);
}

/**
 * 恢复网络
 */
export async function goOnline(page: Page) {
  await page.context().setOffline(false);
}

/**
 * 清除存储
 */
export async function clearStorage(page: Page) {
  await page.evaluate(() => {
    localStorage.clear();
    sessionStorage.clear();
  });
}

/**
 * 获取元素文本
 */
export async function getElementText(page: Page, selector: string) {
  const element = page.locator(selector);
  if (await element.count() > 0) {
    return element.textContent();
  }
  return null;
}

/**
 * 检查元素是否存在
 */
export async function elementExists(page: Page, selector: string) {
  const count = await page.locator(selector).count();
  return count > 0;
}

/**
 * 等待元素消失
 */
export async function waitForElementToDisappear(page: Page, selector: string, timeout = 10000) {
  await page.waitForSelector(selector, { state: 'hidden', timeout });
}

/**
 * 截图保存
 */
export async function takeScreenshot(page: Page, filename: string) {
  await page.screenshot({ path: `screenshots/${filename}.png`, fullPage: true });
}

/**
 * 模拟文件上传
 */
export async function uploadFile(
  page: Page,
  selector: string,
  filename: string,
  content: string = 'test content',
  mimeType: string = 'text/plain'
) {
  const fileInput = page.locator(selector);
  await fileInput.setInputFiles({
    name: filename,
    mimeType,
    buffer: Buffer.from(content),
  });
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
 * 检查是否在指定页面
 */
export async function isOnPage(page: Page, urlPattern: string | RegExp) {
  const url = page.url();
  if (typeof urlPattern === 'string') {
    return url.includes(urlPattern);
  }
  return urlPattern.test(url);
}

/**
 * 获取表格行数
 */
export async function getTableRowCount(page: Page, tableSelector: string) {
  const rows = page.locator(`${tableSelector} tr, ${tableSelector} [data-testid="table-row"]`);
  return rows.count();
}

/**
 * 点击表格行
 */
export async function clickTableRow(page: Page, tableSelector: string, rowIndex: number) {
  const rows = page.locator(`${tableSelector} tr, ${tableSelector} [data-testid="table-row"]`);
  await rows.nth(rowIndex).click();
}

/**
 * 选择下拉选项
 */
export async function selectOption(
  page: Page,
  selector: string,
  value: string | { label?: string; value?: string; index?: number }
) {
  const select = page.locator(selector);
  if (typeof value === 'string') {
    await select.selectOption(value);
  } else {
    await select.selectOption(value);
  }
}
