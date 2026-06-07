import { Page, expect } from '@playwright/test';

export const mockUser = {
  id: 1001,
  name: '测试用户',
  phone: '13800138000',
  email: 'test@example.com',
  role: 0,
};

export const mockToken = 'e2e-auth-token';

const categories = [
  { id: 1, name: '珠宝腕表' },
  { id: 2, name: '艺术品' },
];

const auctions = [
  {
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
      category_id: 1,
      name: '星河钻石腕表',
      category_name: '珠宝腕表',
      description: '新版 H5 E2E 测试竞拍商品',
      images: ['https://copilot-cn.bytedance.net/api/ide/v1/text_to_image?prompt=luxury%20diamond%20watch%20on%20black%20velvet%2C%20realistic%20product%20photo&image_size=landscape_4_3'],
      rules: { start_price: 12000, increment: 100, cap_price: 50000, trigger_delay_before: 30 },
    },
  },
  {
    id: 102,
    product_id: 202,
    live_stream_id: 302,
    status: 3,
    current_price: 88000,
    start_price: 50000,
    increment: 500,
    bid_count: 16,
    product: {
      id: 202,
      category_id: 2,
      name: '宋代青瓷珍藏',
      category_name: '艺术品',
      description: '已结束竞拍商品',
      images: ['https://copilot-cn.bytedance.net/api/ide/v1/text_to_image?prompt=ancient%20celadon%20vase%20museum%20lighting%2C%20realistic%20auction%20catalog%20photo&image_size=landscape_4_3'],
      rules: { start_price: 50000, increment: 500, trigger_delay_before: 30 },
    },
  },
];

const bids = [
  { id: 1, user_id: 1001, user_name: '测试用户', amount: 12800, created_at: new Date().toISOString() },
  { id: 2, user_id: 1002, user_name: '收藏家A', amount: 12600, created_at: new Date().toISOString() },
];

const liveStreams = [
  {
    id: 301,
    name: '星河钻石腕表直播间',
    status: 1,
    host_name: '拍卖师',
    creator_name: '拍卖师',
    viewer_count: 1288,
    followers_count: 42,
    is_following: false,
    current_auction_id: 101,
    current_auctions_count: 1,
    current_price: '12800.00',
    recent_deals: [{ product_name: '和田玉牌', final_price: '500.00' }],
    auctions: [auctions[0]],
  },
  {
    id: 302,
    name: '宋代青瓷珍藏回放',
    status: 0,
    host_name: '拍卖师',
    viewer_count: 256,
    followers_count: 18,
    is_following: false,
    current_auction_id: null,
    current_auctions_count: 0,
    next_auction: {
      auction_id: 21,
      product_id: 9,
      product_name: '翡翠手镯',
      start_price: '300.00',
      start_time: '2026-06-08T10:00:00Z',
    },
    recent_deals: [],
    auctions: [auctions[1]],
  },
];

const history = [
  {
    id: 101,
    auction_id: 101,
    product_name: '星河钻石腕表',
    my_highest_bid: 12800,
    final_price: 12800,
    bid_count: 3,
    is_winner: true,
    ended_at: new Date().toISOString(),
  },
  {
    id: 102,
    auction_id: 102,
    product_name: '宋代青瓷珍藏',
    my_highest_bid: 82000,
    final_price: 88000,
    bid_count: 4,
    is_winner: false,
    ended_at: new Date().toISOString(),
  },
];

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

function getAuction(id: number) {
  return auctions.find((auction) => auction.id === id) || auctions[0];
}

function listAuctions(categoryId?: number) {
  if (!categoryId) {
    return auctions;
  }
  return auctions.filter((auction) => auction.product.category_id === categoryId);
}

export async function mockNewUiApis(page: Page) {
  await page.route('**/api/v1/**', async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname.replace('/api/v1', '');

    if (path === '/auth/login' && request.method() === 'POST') {
      const payload = request.postDataJSON() as { phone?: string; email?: string; password?: string };
      if (payload.password === 'wrongpassword') {
        await route.fulfill(json({ code: 401, message: '登录失败，请检查密码' }, 401));
        return;
      }
      await route.fulfill(json(success({ token: mockToken, user: mockUser })));
      return;
    }

    if (path === '/users/me') {
      await route.fulfill(json(success(mockUser)));
      return;
    }

    if (path === '/user/balance') {
      await route.fulfill(json(success({ balance: 200000, available_balance: 200000, frozen_amount: 0 })));
      return;
    }

    if (path === '/orders') {
      await route.fulfill(json(success({ items: history.slice(0, 2) })));
      return;
    }

    if (path === '/orders/history') {
      await route.fulfill(json(success({ items: history })));
      return;
    }

    if (path === '/categories') {
      await route.fulfill(json(success(categories)));
      return;
    }

    if (path === '/auctions') {
      const categoryId = Number(url.searchParams.get('category_id') || '');
      await route.fulfill(json(success({ items: listAuctions(Number.isNaN(categoryId) ? undefined : categoryId) })));
      return;
    }

    const auctionBidsMatch = path.match(/^\/auctions\/(\d+)\/bids$/);
    if (auctionBidsMatch) {
      await route.fulfill(json(success({ bids })));
      return;
    }

    const auctionBidMatch = path.match(/^\/auctions\/(\d+)\/bid$/);
    if (auctionBidMatch && request.method() === 'POST') {
      const payload = request.postDataJSON() as { amount?: number };
      await route.fulfill(json(success({ current_price: payload.amount ?? 12900 })));
      return;
    }

    const auctionMatch = path.match(/^\/auctions\/(\d+)$/);
    if (auctionMatch) {
      await route.fulfill(json(success(getAuction(Number(auctionMatch[1])))));
      return;
    }

    const productMatch = path.match(/^\/products\/(\d+)$/);
    if (productMatch) {
      const productId = Number(productMatch[1]);
      const auction = auctions.find((item) => item.product_id === productId) || auctions[0];
      await route.fulfill(json(success(auction.product)));
      return;
    }

    if (path === '/products') {
      await route.fulfill(json(success({ items: auctions.map((auction) => auction.product) })));
      return;
    }

    if (path.startsWith('/experiments/viewed')) {
      await route.fulfill(json(success({ ok: true })));
      return;
    }

    if (path.startsWith('/notifications')) {
      await route.fulfill(json(success({ items: [], count: 0 })));
      return;
    }

    if (path.match(/^\/live-streams\/\d+\/fixed-price\/items$/)) {
      await route.fulfill(json(success({ items: [] })));
      return;
    }

    const liveStreamFollowersStatsMatch = path.match(/^\/live-streams\/(\d+)\/followers\/stats$/);
    if (liveStreamFollowersStatsMatch) {
      await route.fulfill(json(success({ count: 42, followers_count: 42, total_count: 42 })));
      return;
    }

    const liveStreamFollowStatusMatch = path.match(/^\/live-streams\/(\d+)\/follow-status$/);
    if (liveStreamFollowStatusMatch) {
      await route.fulfill(json(success({ is_following: false })));
      return;
    }

    const liveStreamMatch = path.match(/^\/live-streams\/(\d+)$/);
    if (liveStreamMatch) {
      const liveStream = liveStreams.find((item) => item.id === Number(liveStreamMatch[1])) || liveStreams[0];
      await route.fulfill(json(success(liveStream)));
      return;
    }

    if (path === '/live-streams') {
      await route.fulfill(json(success({ list: liveStreams, items: liveStreams })));
      return;
    }

    await route.fulfill(json(success({})));
  });
}

export async function seedAuthenticatedUser(page: Page) {
  await page.addInitScript(({ token, user }) => {
    window.localStorage.setItem('auth_token', token);
    window.localStorage.setItem('auth_user', JSON.stringify(user));
  }, { token: mockToken, user: mockUser });
}

export async function loginWithNewUi(page: Page, phone = mockUser.phone!, password = 'Test@123456') {
  await page.goto('/login');
  await expect(page.getByRole('heading', { name: '登录' })).toBeVisible();
  await page.getByPlaceholder('请输入手机号').fill(phone);
  await page.getByPlaceholder('请输入密码').fill(password);
  await page.getByRole('button', { name: '登录' }).click();
}
