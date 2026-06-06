import { productApi as sharedProductApi } from '..';
import { get, post, put } from '../request';
import { productApi } from '../product';
import { Category, Product } from '../types';

jest.mock('../request', () => ({
  get: jest.fn(),
  post: jest.fn(),
  put: jest.fn(),
  del: jest.fn(),
  buildQuery: (params: Record<string, string | number | undefined>) =>
    new URLSearchParams(
      Object.entries(params)
        .filter(([, value]) => value !== undefined)
        .map(([key, value]) => [key, String(value)])
    ).toString(),
}));

describe('productApi category contract', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('uses category_id/category_name types instead of legacy category strings', () => {
    const category: Category = {
      id: 9,
      name: '艺术收藏',
      code: 'art-collection',
      status: 1,
    };
    const productRecord: Product = {
      id: 101,
      name: '明代瓷器',
      description: '馆藏级藏品',
      images: ['https://cdn.example.com/product.jpg'],
      category_id: category.id,
      category_name: category.name,
      status: 1,
      created_at: '2026-06-04T00:00:00Z',
      updated_at: '2026-06-04T00:00:00Z',
    };
    const createPayload: Parameters<typeof productApi.create>[0] = {
      name: productRecord.name,
      description: productRecord.description,
      images: productRecord.images,
      category_id: category.id,
    };
    const updatePayload: Parameters<typeof sharedProductApi.update>[1] = {
      category_id: category.id,
    };

    expect(createPayload.category_id).toBe(category.id);
    expect(updatePayload.category_id).toBe(category.id);
    expect(productRecord.category_name).toBe(category.name);
  });

  it('module productApi creates and updates products with category_id payloads', async () => {
    const createPayload: Parameters<typeof productApi.create>[0] = {
      name: '明代瓷器',
      description: '馆藏级藏品',
      images: ['https://cdn.example.com/product.jpg'],
      category_id: 9,
    };
    const updatePayload: Parameters<typeof productApi.update>[1] = {
      category_id: 12,
    };
    (post as jest.Mock).mockResolvedValue(createPayload);
    (put as jest.Mock).mockResolvedValue(updatePayload);

    await productApi.create(createPayload);
    await productApi.update(101, updatePayload);

    expect(post).toHaveBeenCalledWith('/admin/products', createPayload);
    expect(put).toHaveBeenCalledWith('/admin/products/101', updatePayload);
  });

  it('shared productApi creates and updates products with category_id payloads', async () => {
    const createPayload: Parameters<typeof sharedProductApi.create>[0] = {
      name: '明代瓷器',
      description: '馆藏级藏品',
      images: ['https://cdn.example.com/product.jpg'],
      category_id: 9,
    };
    const updatePayload: Parameters<typeof sharedProductApi.update>[1] = {
      category_id: 12,
    };
    (post as jest.Mock).mockResolvedValue(createPayload);
    (put as jest.Mock).mockResolvedValue(updatePayload);

    await sharedProductApi.create(createPayload);
    await sharedProductApi.update(101, updatePayload);

    expect(post).toHaveBeenCalledWith('/admin/products', createPayload);
    expect(put).toHaveBeenCalledWith('/admin/products/101', updatePayload);
  });

  it('module productApi unwraps category list responses from GET /categories', async () => {
    const categories: Category[] = [
      { id: 9, name: '艺术收藏', code: 'art-collection', status: 1 },
    ];
    (get as jest.Mock).mockResolvedValue({ list: categories, total: 1, page: 1, page_size: 20 });

    const result = await productApi.listCategories();

    expect(get).toHaveBeenCalledWith('/categories');
    expect(result).toEqual(categories);
  });

  it('shared productApi unwraps category list responses from GET /categories', async () => {
    const categories: Category[] = [
      { id: 11, name: '珠宝名表', code: 'jewelry-watch', status: 1 },
    ];
    (get as jest.Mock).mockResolvedValue({ list: categories, total: 1, page: 1, page_size: 20 });

    const result = await sharedProductApi.listCategories();

    expect(get).toHaveBeenCalledWith('/categories');
    expect(result).toEqual(categories);
  });
});

describe('productApi.list', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('repairs mojibake product text fields before returning the list', async () => {
    (get as jest.Mock).mockResolvedValue({
      list: [
        {
          id: 1,
          name: 'ç¨€æœ‰ç å®',
          description: 'ç²¾é€‰æ‹å“',
          category_name: 'ç¿¡ç¿ ',
          images: [],
          status: 1,
          created_at: '2026-06-02T00:00:00Z',
          updated_at: '2026-06-02T00:00:00Z',
        },
      ],
      total: 1,
      page: 1,
      page_size: 10,
    });

    const result = await productApi.list({ page: 1, page_size: 10 });

    expect(result.list[0]).toMatchObject({
      name: '稀有珠宝',
      description: '精选拍品',
      category_name: '翡翠',
    });
  });

  it('uses the admin product endpoint so merchants can manage all statuses without widening public APIs', async () => {
    (get as jest.Mock).mockResolvedValue({ list: [], total: 0, page: 1, page_size: 10 });

    await productApi.list({ page: 1, page_size: 10 });

    expect(get).toHaveBeenCalledWith('/admin/products?page=1&page_size=10');
  });
});
