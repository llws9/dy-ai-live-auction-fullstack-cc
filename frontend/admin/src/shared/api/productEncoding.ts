import { repairUtf8Mojibake } from '../../utils/textEncoding';

const PRODUCT_TEXT_FIELDS = ['name', 'description', 'category', 'brand'] as const;

type ProductTextField = (typeof PRODUCT_TEXT_FIELDS)[number];
type ProductLike = object & Partial<Record<ProductTextField, string | null>>;

export function normalizeProductText<T extends ProductLike>(product: T): T {
  return PRODUCT_TEXT_FIELDS.reduce<T>((normalized, field) => {
    const value = normalized[field];
    if (typeof value !== 'string') {
      return normalized;
    }
    return {
      ...normalized,
      [field]: repairUtf8Mojibake(value),
    };
  }, { ...product });
}

export function normalizeProductListResponse<T extends ProductLike, R extends { list?: T[] }>(response: R): R {
  if (!Array.isArray(response.list)) {
    return response;
  }

  return {
    ...response,
    list: response.list.map(normalizeProductText),
  };
}
