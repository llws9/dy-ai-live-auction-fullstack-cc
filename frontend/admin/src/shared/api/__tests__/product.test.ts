import { post } from '../request';
import { productApi } from '../product';

jest.mock('../request', () => ({
  get: jest.fn(),
  post: jest.fn(),
  put: jest.fn(),
  del: jest.fn(),
  buildQuery: jest.fn(() => ''),
}));

describe('productApi.generateCopywriting', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('posts to the Gateway AI copywriting route with a 70s timeout', async () => {
    (post as jest.Mock).mockResolvedValue({
      name: 'AI 标题',
      description: 'AI 描述',
      selling_points: ['卖点一'],
      suggested_start_price: '199.00',
    });

    const payload = {
      images: ['https://cdn.example.com/product.jpg'],
      keywords: '类目：艺术收藏',
    };

    const result = await productApi.generateCopywriting(payload);

    expect(post).toHaveBeenCalledWith('/products/ai/copywriting', payload, { timeout: 70000 });
    expect(result.name).toBe('AI 标题');
  });
});
