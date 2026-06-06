import { auctionRuleTemplateApi } from '..'
import { get, post, put, del } from '../request'

jest.mock('../request', () => ({
  get: jest.fn(),
  post: jest.fn(),
  put: jest.fn(),
  del: jest.fn(),
  buildQuery: (params: Record<string, string | number | boolean | undefined>) =>
    new URLSearchParams(
      Object.entries(params)
        .filter(([, value]) => value !== undefined)
        .map(([key, value]) => [key, String(value)])
    ).toString(),
}))

describe('auctionRuleTemplateApi', () => {
  beforeEach(() => {
    jest.clearAllMocks()
  })

  it('uses merchant rule template backend endpoints', async () => {
    const payload = {
      name: '默认模板',
      start_price: '100.00',
      increment: '10.00',
      cap_price: '',
      duration: 3600,
      delay_duration: 30,
      max_delay_time: 180,
      trigger_delay_before: 30,
      is_default: true,
    }
    ;(get as jest.Mock).mockResolvedValue({ list: [], total: 0, page: 1, page_size: 20 })
    ;(post as jest.Mock).mockResolvedValue({ id: 1, ...payload })
    ;(put as jest.Mock).mockResolvedValue({ id: 1, ...payload, increment: '20.00' })
    ;(del as jest.Mock).mockResolvedValue(undefined)

    await auctionRuleTemplateApi.list({ page: 1, page_size: 20 })
    await auctionRuleTemplateApi.create(payload)
    await auctionRuleTemplateApi.update(1, { ...payload, increment: '20.00' })
    await auctionRuleTemplateApi.delete(1)

    expect(get).toHaveBeenCalledWith('/admin/auction-rule-templates?page=1&page_size=20')
    expect(post).toHaveBeenCalledWith('/admin/auction-rule-templates', payload)
    expect(put).toHaveBeenCalledWith('/admin/auction-rule-templates/1', { ...payload, increment: '20.00' })
    expect(del).toHaveBeenCalledWith('/admin/auction-rule-templates/1')
  })
})
