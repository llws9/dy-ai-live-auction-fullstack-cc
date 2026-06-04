import { http, HttpResponse, delay } from 'msw';

// 模拟数据
const mockAuctions = [
  {
    id: 1,
    product_id: 1,
    product_name: '限定款奢侈品包包',
    product_image: 'https://images.unsplash.com/photo-1548036328-c9fa89d128fa?w=400',
    status: 1,
    current_price: 150,
    end_time: new Date(Date.now() + 3600000).toISOString(),
    start_time: new Date().toISOString(),
    bidder_count: 12,
  },
  {
    id: 2,
    product_id: 2,
    product_name: '签名版限量球鞋',
    product_image: 'https://images.unsplash.com/photo-1542291026-7eec264c27ff?w=400',
    status: 1,
    current_price: 280,
    end_time: new Date(Date.now() + 1800000).toISOString(),
    start_time: new Date().toISOString(),
    bidder_count: 8,
  },
  {
    id: 3,
    product_id: 3,
    product_name: '古董怀表收藏品',
    product_image: 'https://images.unsplash.com/photo-1509048191080-d2984bad6ae5?w=400',
    status: 3,
    current_price: 520,
    end_time: new Date(Date.now() - 3600000).toISOString(),
    start_time: new Date(Date.now() - 7200000).toISOString(),
    bidder_count: 25,
  },
];

const mockProducts = [
  {
    id: 1,
    name: '限定款奢侈品包包',
    description: '限量发售，品质保证',
    image: 'https://images.unsplash.com/photo-1548036328-c9fa89d128fa?w=400',
    base_price: 100,
    status: 1,
    created_at: new Date().toISOString(),
  },
  {
    id: 2,
    name: '签名版限量球鞋',
    description: '球星亲笔签名',
    image: 'https://images.unsplash.com/photo-1542291026-7eec264c27ff?w=400',
    base_price: 200,
    status: 1,
    created_at: new Date().toISOString(),
  },
];

export const handlers = [
  // 获取竞拍列表
  http.get('/api/v1/auctions', async () => {
    await delay(100);
    return HttpResponse.json({
      auctions: mockAuctions,
      total: mockAuctions.length,
    });
  }),

  // 获取竞拍详情
  http.get('/api/v1/auctions/:id', async ({ params }) => {
    await delay(50);
    const { id } = params;
    const auction = mockAuctions.find((a) => a.id === Number(id));

    if (!auction) {
      return new HttpResponse(null, { status: 404 });
    }

    return HttpResponse.json(auction);
  }),

  // 获取出价记录
  http.get('/api/v1/auctions/:id/bids', async () => {
    await delay(50);
    return HttpResponse.json({
      bids: [
        { id: 1, user_id: 2, user_name: '用户A', amount: 150, created_at: new Date().toISOString() },
        { id: 2, user_id: 3, user_name: '用户B', amount: 140, created_at: new Date().toISOString() },
        { id: 3, user_id: 4, user_name: '用户C', amount: 130, created_at: new Date().toISOString() },
      ],
    });
  }),

  // 出价
  http.post('/api/v1/auctions/:id/bid', async ({ request }) => {
    await delay(100);
    const body = await request.json();
    return HttpResponse.json({
      success: true,
      new_price: (body as any).amount || 160,
      message: '出价成功',
    });
  }),

  // 获取商品列表
  http.get('/api/v1/products', async () => {
    await delay(100);
    return HttpResponse.json({
      products: mockProducts,
      total: mockProducts.length,
    });
  }),

  // 获取商品详情
  http.get('/api/v1/products/:id', async ({ params }) => {
    await delay(50);
    const { id } = params;
    const product = mockProducts.find((p) => p.id === Number(id));

    if (!product) {
      return new HttpResponse(null, { status: 404 });
    }

    return HttpResponse.json(product);
  }),

  // 用户登录
  http.post('/api/v1/auth/login', async () => {
    await delay(100);
    return HttpResponse.json({
      success: true,
      token: 'mock-jwt-token',
      user: {
        id: 1,
        username: 'testuser',
        role: 'user',
      },
    });
  }),

  // 获取用户信息
  http.get('/api/v1/users/me', async () => {
    await delay(50);
    return HttpResponse.json({
      id: 1,
      username: 'testuser',
      avatar: 'https://images.unsplash.com/photo-1535713875002-d1d0cf377fde?w=100',
      balance: 1000,
    });
  }),
];
