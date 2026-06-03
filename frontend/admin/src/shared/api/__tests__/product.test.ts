import { productApi } from '../product';

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
});
