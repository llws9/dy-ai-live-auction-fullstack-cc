import { fixedPriceAdminApi } from '..'
import { get, post } from '../request'

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
  ApiError: class ApiError extends Error {},
  setToastFunction: jest.fn(),
}))

describe('fixedPriceAdminApi', () => {
  beforeEach(() => {
    jest.clearAllMocks()
  })

  it('uses the admin all-status endpoint for fixed-price list management', () => {
    fixedPriceAdminApi.list(1001, { page: 1, page_size: 20 })

    expect(get).toHaveBeenCalledWith('/admin/live-streams/1001/fixed-price/items?page=1&page_size=20')
  })

  it('lists fixed-price items with auction binding', () => {
    fixedPriceAdminApi.listItem(1001, {
      auction_id: 8001,
      product_id: 5001,
      price: '99.00',
      stock: 20,
    })

    expect(post).toHaveBeenCalledWith('/fixed-price/items', {
      auction_id: 8001,
      live_stream_id: 1001,
      product_id: 5001,
      price: '99.00',
      total_stock: 20,
      max_per_user: 1,
    })
  })
})
