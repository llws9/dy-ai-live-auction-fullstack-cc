import { productApi } from '../index';

describe('productApi.generateCopywriting', () => {
  beforeEach(() => {
    localStorage.clear();
    jest.clearAllMocks();
  });

  it('posts to the Gateway AI copywriting route and accepts the backend raw response shape', async () => {
    const fetchMock = jest.fn().mockResolvedValue({
      ok: true,
      headers: {
        get: (name: string) => (name.toLowerCase() === 'content-type' ? 'application/json' : null),
      },
      json: async () => ({
        name: 'AI 标题',
        description: 'AI 描述',
        selling_points: ['卖点一'],
        suggested_start_price: '199.00',
      }),
    });
    global.fetch = fetchMock;

    const payload = {
      images: ['https://cdn.example.com/product.jpg'],
      keywords: '类目：艺术收藏',
    };

    const result = await productApi.generateCopywriting(payload);

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/products/ai/copywriting',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify(payload),
      })
    );
    expect(result.name).toBe('AI 标题');
  });

  it('uses admin_auth_token as the Authorization bearer token', async () => {
    localStorage.setItem('admin_auth_token', 'admin-token');
    const fetchMock = jest.fn().mockResolvedValue({
      ok: true,
      headers: {
        get: (name: string) => (name.toLowerCase() === 'content-type' ? 'application/json' : null),
      },
      json: async () => ({ name: 'AI 标题', description: 'AI 描述' }),
    });
    global.fetch = fetchMock;

    await productApi.generateCopywriting({
      images: ['https://cdn.example.com/product.jpg'],
    });

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/products/ai/copywriting',
      expect.objectContaining({
        headers: expect.objectContaining({
          Authorization: 'Bearer admin-token',
        }),
      })
    );
  });

  it('does not use the legacy token key as the Authorization bearer token', async () => {
    localStorage.setItem('token', 'legacy-token');
    const fetchMock = jest.fn().mockResolvedValue({
      ok: true,
      headers: {
        get: (name: string) => (name.toLowerCase() === 'content-type' ? 'application/json' : null),
      },
      json: async () => ({ name: 'AI 标题', description: 'AI 描述' }),
    });
    global.fetch = fetchMock;

    await productApi.generateCopywriting({
      images: ['https://cdn.example.com/product.jpg'],
    });

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/products/ai/copywriting',
      expect.objectContaining({
        headers: expect.not.objectContaining({
          Authorization: 'Bearer legacy-token',
        }),
      })
    );
  });

  it('clears the admin auth keys when the API returns 401', async () => {
    localStorage.setItem('admin_auth_token', 'expired-admin-token');
    localStorage.setItem('admin_auth_user', JSON.stringify({ id: 999 }));
    localStorage.setItem('token', 'legacy-token');
    localStorage.setItem('userInfo', JSON.stringify({ id: 1 }));

    global.fetch = jest.fn().mockResolvedValue({
      ok: false,
      status: 401,
      headers: {
        get: (name: string) => (name.toLowerCase() === 'content-type' ? 'application/json' : null),
      },
      json: async () => ({ code: 401, message: '未授权' }),
    });

    await expect(productApi.generateCopywriting({
      images: ['https://cdn.example.com/product.jpg'],
    })).rejects.toMatchObject({ status: 401 });

    expect(localStorage.getItem('admin_auth_token')).toBeNull();
    expect(localStorage.getItem('admin_auth_user')).toBeNull();
    expect(localStorage.getItem('token')).toBeNull();
    expect(localStorage.getItem('userInfo')).toBeNull();
  });
});

describe('productApi.list', () => {
  beforeEach(() => {
    localStorage.clear();
    jest.clearAllMocks();
  });

  it('repairs mojibake product text fields before returning the list', async () => {
    global.fetch = jest.fn().mockResolvedValue({
      ok: true,
      headers: {
        get: (name: string) => (name.toLowerCase() === 'content-type' ? 'application/json' : null),
      },
      json: async () => ({
        code: 0,
        data: {
          list: [
            {
              id: 1,
              name: 'ç¨€æœ‰ç å®',
              description: 'ç²¾é€‰æ‹å“',
              category: 'ç¿¡ç¿ ',
              images: [],
              status: 1,
              created_at: '2026-06-02T00:00:00Z',
              updated_at: '2026-06-02T00:00:00Z',
            },
          ],
          total: 1,
          page: 1,
          page_size: 10,
        },
      }),
    });

    const result = await productApi.list({ page: 1, page_size: 10 });

    expect(result.list[0]).toMatchObject({
      name: '稀有珠宝',
      description: '精选拍品',
      category: '翡翠',
    });
  });
});
