import { http, HttpResponse, delay } from 'msw';

// 管理端模拟数据
const mockAuctions = [
  {
    id: 1,
    product_id: 1,
    product_name: '限定款奢侈品包包',
    status: 1,
    current_price: 150,
    start_time: new Date().toISOString(),
    end_time: new Date(Date.now() + 3600000).toISOString(),
    delay_used: 0,
    winner_id: null,
  },
  {
    id: 2,
    product_id: 2,
    product_name: '签名版限量球鞋',
    status: 1,
    current_price: 280,
    start_time: new Date().toISOString(),
    end_time: new Date(Date.now() + 1800000).toISOString(),
    delay_used: 0,
    winner_id: null,
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
    stock: 10,
    created_at: new Date().toISOString(),
  },
  {
    id: 2,
    name: '签名版限量球鞋',
    description: '球星亲笔签名',
    image: 'https://images.unsplash.com/photo-1542291026-7eec264c27ff?w=400',
    base_price: 200,
    status: 1,
    stock: 5,
    created_at: new Date().toISOString(),
  },
];

const mockOrders = [
  {
    id: 1,
    user_id: 1,
    product_name: '限定款奢侈品包包',
    amount: 150,
    status: 'paid',
    created_at: new Date().toISOString(),
  },
  {
    id: 2,
    user_id: 2,
    product_name: '签名版限量球鞋',
    amount: 280,
    status: 'pending',
    created_at: new Date().toISOString(),
  },
];

export const handlers = [
  // 管理员登录
  http.post('/api/v1/admin/login', async () => {
    await delay(100);
    return HttpResponse.json({
      success: true,
      token: 'mock-admin-jwt-token',
      user: {
        id: 1,
        username: 'admin',
        role: 'admin',
      },
    });
  }),

  // 获取统计数据
  http.get('/api/v1/admin/statistics', async () => {
    await delay(100);
    return HttpResponse.json({
      totalAuctions: 100,
      activeAuctions: 15,
      totalOrders: 250,
      totalRevenue: 125000,
      todayUsers: 45,
      todayBids: 120,
    });
  }),

  // 获取竞拍列表
  http.get('/api/v1/admin/auctions', async () => {
    await delay(100);
    return HttpResponse.json({
      auctions: mockAuctions,
      total: mockAuctions.length,
      page: 1,
      pageSize: 10,
    });
  }),

  // 获取竞拍详情
  http.get('/api/v1/admin/auctions/:id', async ({ params }) => {
    await delay(50);
    const { id } = params;
    const auction = mockAuctions.find((a) => a.id === Number(id));

    if (!auction) {
      return new HttpResponse(null, { status: 404 });
    }

    return HttpResponse.json(auction);
  }),

  // 更新竞拍状态
  http.put('/api/v1/admin/auctions/:id', async ({ params, request }) => {
    await delay(50);
    const { id } = params;
    const body = await request.json();
    const auction = mockAuctions.find((a) => a.id === Number(id));

    if (!auction) {
      return new HttpResponse(null, { status: 404 });
    }

    return HttpResponse.json({
      ...auction,
      ...(body as object),
    });
  }),

  // 获取商品列表
  http.get('/api/v1/admin/products', async () => {
    await delay(100);
    return HttpResponse.json({
      products: mockProducts,
      total: mockProducts.length,
      page: 1,
      pageSize: 10,
    });
  }),

  // 创建商品
  http.post('/api/v1/admin/products', async ({ request }) => {
    await delay(100);
    const body = await request.json();
    return HttpResponse.json({
      id: Date.now(),
      ...(body as object),
      created_at: new Date().toISOString(),
    });
  }),

  // 更新商品
  http.put('/api/v1/admin/products/:id', async ({ params, request }) => {
    await delay(50);
    const { id } = params;
    const body = await request.json();
    const product = mockProducts.find((p) => p.id === Number(id));

    if (!product) {
      return new HttpResponse(null, { status: 404 });
    }

    return HttpResponse.json({
      ...product,
      ...(body as object),
    });
  }),

  // 删除商品
  http.delete('/api/v1/admin/products/:id', async ({ params }) => {
    await delay(50);
    const { id } = params;
    return HttpResponse.json({ success: true, id });
  }),

  // 获取订单列表
  http.get('/api/v1/admin/orders', async () => {
    await delay(100);
    return HttpResponse.json({
      orders: mockOrders,
      total: mockOrders.length,
      page: 1,
      pageSize: 10,
    });
  }),

  // 更新订单状态
  http.put('/api/v1/admin/orders/:id', async ({ params, request }) => {
    await delay(50);
    const { id } = params;
    const body = await request.json();
    const order = mockOrders.find((o) => o.id === Number(id));

    if (!order) {
      return new HttpResponse(null, { status: 404 });
    }

    return HttpResponse.json({
      ...order,
      ...(body as object),
    });
  }),
];
