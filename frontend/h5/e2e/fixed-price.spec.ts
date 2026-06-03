import { test, expect, type Page, type Route } from '@playwright/test';
import { mockToken, mockUser, seedAuthenticatedUser } from './utils/new-ui-fixtures';

const auction = {
  id: 101,
  product_id: 201,
  live_stream_id: 301,
  status: 1,
  current_price: 12800,
  start_price: 12000,
  increment: 100,
  bid_count: 8,
  product: {
    id: 201,
    name: '星河钻石腕表',
    category_name: '珠宝腕表',
    description: '一口价 E2E 直播间主拍商品',
    images: [
      'https://copilot-cn.bytedance.net/api/ide/v1/text_to_image?prompt=luxury%20diamond%20watch%20auction%20livestream%2C%20realistic%20product%20photo&image_size=landscape_4_3',
    ],
    rules: { start_price: 12000, increment: 100, cap_price: 50000, trigger_delay_before: 30 },
  },
};

const fixedPriceItem = {
  id: 7001,
  product_id: 5001,
  price: '88.00',
  total_stock: 10,
  remaining_stock: 5,
  status: 'on_sale',
  product_brief: {
    id: 5001,
    title: '一口价翡翠手串',
    cover_image: 'https://copilot-cn.bytedance.net/api/ide/v1/text_to_image?prompt=jade%20bracelet%20on%20auction%20table%2C%20realistic%20product%20photo&image_size=square',
  },
};

function json(body: unknown, status = 200) {
  return {
    status,
    contentType: 'application/json',
    body: JSON.stringify(body),
  };
}

function success(data: unknown) {
  return { code: 200, message: 'success', data };
}

async function installMockWebSocket(page: Page) {
  await page.addInitScript(() => {
    type Handler = ((event: unknown) => void) | null;

    class MockWebSocket {
      static CONNECTING = 0;
      static OPEN = 1;
      static CLOSING = 2;
      static CLOSED = 3;

      readyState = MockWebSocket.OPEN;
      onopen: Handler = null;
      onmessage: Handler = null;
      onerror: Handler = null;
      onclose: Handler = null;

      constructor(public url: string) {
        const bag = ((window as any).__fixedPriceWsInstances ||= []);
        bag.push(this);
        window.setTimeout(() => this.onopen?.({ type: 'open' }), 0);
      }

      send() {}

      close() {
        this.readyState = MockWebSocket.CLOSED;
        this.onclose?.({ code: 1000 });
      }

      emit(type: string, data: unknown) {
        this.onmessage?.({
          data: JSON.stringify({ type, data, timestamp: Date.now() }),
        });
      }
    }

    (window as any).WebSocket = MockWebSocket;
    (window as any).__fixedPriceWsEmit = (type: string, data: unknown) => {
      const instances = (window as any).__fixedPriceWsInstances || [];
      instances.forEach((instance: MockWebSocket) => instance.emit(type, data));
    };
  });
}

async function mockFixedPriceApis(page: Page, purchaseMode: 'success' | 'insufficient' = 'success') {
  await page.route('**/api/v1/**', async (route: Route) => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname.replace('/api/v1', '');

    if (path === '/live-streams/301/fixed-price/items') {
      await route.fulfill(json(success({ items: [fixedPriceItem] })));
      return;
    }

    if (path === '/fixed-price/items/7001/purchase' && request.method() === 'POST') {
      if (purchaseMode === 'insufficient') {
        await route.fulfill(json({ code: 'FP_INSUFFICIENT_BALANCE', message: '余额不足' }, 402));
        return;
      }
      await route.fulfill(json(success({
        order_id: 9001,
        item_id: 7001,
        price: '88.00',
        remaining_stock: 4,
        status: 'success',
      })));
      return;
    }

    if (path === '/user/profile') {
      await route.fulfill(json(success(mockUser)));
      return;
    }

    if (path === '/user/balance') {
      await route.fulfill(json(success({ balance: 200000, available_balance: 200000, frozen_amount: 0 })));
      return;
    }

    if (path === '/auctions/101') {
      await route.fulfill(json(success(auction)));
      return;
    }

    if (path === '/auctions/101/bids') {
      await route.fulfill(json(success({
        bids: [{ id: 1, user_id: 1001, user_name: '测试用户', amount: 12800, created_at: new Date().toISOString() }],
      })));
      return;
    }

    if (path.startsWith('/experiments/viewed') || path.startsWith('/notifications')) {
      await route.fulfill(json(success({ items: [], count: 0 })));
      return;
    }

    if (path.startsWith('/live-streams')) {
      await route.fulfill(json(success({ items: [], followed: false, follower_count: 0 })));
      return;
    }

    await route.fulfill(json(success({})));
  });
}

async function openFixedPriceLive(page: Page, purchaseMode: 'success' | 'insufficient' = 'success') {
  await installMockWebSocket(page);
  await seedAuthenticatedUser(page);
  await page.addInitScript((token) => {
    window.localStorage.setItem('token', token);
  }, mockToken);
  await mockFixedPriceApis(page, purchaseMode);
  await page.goto('/live?id=301&auction_id=101');
  await expect(page.getByText('一口价翡翠手串')).toBeVisible();
}

test.describe('Fixed Price Live Smoke', () => {
  test('user can purchase fixed-price item and navigate to order detail', async ({ page }) => {
    await openFixedPriceLive(page);

    await page.getByRole('button', { name: '立即抢' }).click();
    await expect(page.getByRole('dialog', { name: '确认抢购' })).toBeVisible();
    await page.getByRole('button', { name: '确认抢购' }).click();

    await expect(page.getByRole('status')).toContainText('抢到了！');
    await expect(page).toHaveURL(/\/order\/9001/);
  });

  test('insufficient balance shows recharge guidance', async ({ page }) => {
    await openFixedPriceLive(page, 'insufficient');

    await page.getByRole('button', { name: '立即抢' }).click();
    await page.getByRole('button', { name: '确认抢购' }).click();

    await expect(page.getByRole('alertdialog', { name: '余额不足，去充值' })).toBeVisible();
    await page.getByRole('button', { name: '去充值' }).click();
    await expect(page).toHaveURL(/\/wallet\/recharge/);
  });

  test('offline realtime event removes fixed-price card within one second', async ({ page }) => {
    await openFixedPriceLive(page);

    await page.evaluate(() => {
      (window as any).__fixedPriceWsEmit('fixed_price_offline', { item_id: 7001 });
    });

    await expect(page.getByText('一口价翡翠手串')).toBeHidden({ timeout: 1000 });
  });

  test('listed and flair realtime events render new card and buyer flair', async ({ page }) => {
    await openFixedPriceLive(page);

    await page.evaluate(() => {
      (window as any).__fixedPriceWsEmit('fixed_price_listed', {
        item: {
          id: 7002,
          product_id: 5002,
          price: '128.00',
          total_stock: 8,
          remaining_stock: 8,
          status: 'on_sale',
          product_brief: { id: 5002, title: '返场珍珠项链' },
        },
      });
      (window as any).__fixedPriceWsEmit('fixed_price_flair', {
        buyer_nickname: '王女士',
        product_title: '返场珍珠项链',
        price: '128.00',
      });
    });

    await expect(page.getByRole('heading', { name: '返场珍珠项链' })).toBeVisible();
    await expect(page.getByLabel('一口价购买飘屏')).toContainText('王女士');
    await expect(page.getByLabel('一口价购买飘屏')).toContainText('刚刚抢到 返场珍珠项链');
  });

  test('flair fallback renders buyer_id and item_id when backend sends only IDs', async ({ page }) => {
    await openFixedPriceLive(page);

    await page.evaluate(() => {
      (window as any).__fixedPriceWsEmit('fixed_price_flair', {
        item_id: 7003,
        buyer_id: 1001,
        price: '88.00',
      });
    });

    await expect(page.getByLabel('一口价购买飘屏')).toContainText('用户 #1001');
    await expect(page.getByLabel('一口价购买飘屏')).toContainText('刚刚抢到 商品 #7003');
    await expect(page.getByLabel('一口价购买飘屏')).toContainText('¥88.00');
  });
});
