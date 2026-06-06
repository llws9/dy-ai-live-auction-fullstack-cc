// 商品API

import { get, post, put, del, buildQuery } from './request';
import { Product, AuctionRule, PaginatedResponse, Category } from './types';
import { normalizeProductListResponse, normalizeProductText } from './productEncoding';

export interface ProductListParams {
  status?: number;
  page?: number;
  page_size?: number;
}

export interface ProductCreateData {
  name: string;
  description: string;
  images: string[];
  category_id?: number | null;
}

export interface RuleCreateData {
  start_price: number;
  increment: number;
  cap_price: number;
  duration: number;
  delay_duration: number;
  max_delay_time: number;
  trigger_delay_before: number;
}

export interface CopywritingGenerateData {
  images: string[];
  category_id?: number;
  keywords?: string;
}

export interface CopywritingDraft {
  name: string;
  description: string;
  selling_points: string[];
  suggested_start_price: string;
}

function normalizeCategoryList(response: Category[] | { list?: Category[] }): Category[] {
  if (Array.isArray(response)) {
    return response;
  }

  if (Array.isArray(response.list)) {
    return response.list;
  }

  return [];
}

export const productApi = {
  // 获取商品列表
  list: (params?: ProductListParams) => {
    const query = buildQuery(params || {});
    return get<PaginatedResponse<Product>>(`/admin/products?${query}`)
      .then(normalizeProductListResponse);
  },

  // 获取商品详情
  get: (id: number) => get<Product>(`/admin/products/${id}`).then(normalizeProductText),

  // 获取分类列表
  listCategories: () => get<Category[] | { list?: Category[] }>('/categories').then(normalizeCategoryList),

  // 创建商品
  create: (data: ProductCreateData) => post<Product>('/admin/products', data).then(normalizeProductText),

  // AI 一键文案
  generateCopywriting: (data: CopywritingGenerateData) =>
    post<CopywritingDraft>('/products/ai/copywriting', data, { timeout: 70000 }),

  // 更新商品
  update: (id: number, data: Partial<ProductCreateData>) => put<Product>(`/admin/products/${id}`, data).then(normalizeProductText),

  // 删除商品
  delete: (id: number) => del<void>(`/admin/products/${id}`),

  // 发布商品
  publish: (id: number) => post<Product>(`/products/${id}/publish`).then(normalizeProductText),

  // 下架商品
  unpublish: (id: number, reason?: string) => post<Product>(`/products/${id}/unpublish`, { reason }).then(normalizeProductText),

  // 获取竞拍规则
  getRules: (productId: number) => get<AuctionRule>(`/products/${productId}/rules`),

  // 创建竞拍规则
  createRules: (productId: number, data: RuleCreateData) => post<AuctionRule>(`/products/${productId}/rules`, data),
};
