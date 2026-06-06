import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import AuctionList from '../AuctionList'
import { auctionApi, auctionRuleTemplateApi, productApi, statisticsApi } from '@/shared/api'
import { useAuth } from '@/shared/auth'
import { MERCHANT_ROLE } from '@/shared/auth/roles'

jest.mock('@/shared/api', () => ({
  auctionApi: {
    list: jest.fn(),
    create: jest.fn(),
  },
  auctionRuleTemplateApi: {
    list: jest.fn(),
  },
  productApi: {
    list: jest.fn(),
    applyRuleTemplate: jest.fn(),
  },
  statisticsApi: {
    getOverview: jest.fn(),
  },
}))

jest.mock('@/shared/auth', () => ({
  useAuth: jest.fn(),
}))

const mockedAuctionApi = auctionApi as jest.Mocked<typeof auctionApi>
const mockedProductApi = productApi as jest.Mocked<typeof productApi>
const mockedRuleTemplateApi = auctionRuleTemplateApi as jest.Mocked<typeof auctionRuleTemplateApi>
const mockedStatisticsApi = statisticsApi as jest.Mocked<typeof statisticsApi>
const mockedUseAuth = useAuth as jest.Mock

describe('AuctionList create auction with rule template', () => {
  beforeEach(() => {
    jest.clearAllMocks()
    mockedUseAuth.mockReturnValue({
      user: { id: 1001, name: '商家用户', role: MERCHANT_ROLE },
    })
    mockedAuctionApi.list.mockResolvedValue({ list: [], total: 0 })
    mockedStatisticsApi.getOverview.mockResolvedValue({ total_auctions: 0, total_users: 0, today_revenue: 0 })
    mockedProductApi.list.mockResolvedValue({
      list: [
        {
          id: 501,
          name: '青花瓷瓶',
          description: '',
          images: [],
          category_id: null,
          category_name: '',
          status: 1,
          created_at: '2026-06-06T00:00:00Z',
          updated_at: '2026-06-06T00:00:00Z',
        },
      ],
      total: 1,
      page: 1,
      page_size: 100,
    })
    mockedRuleTemplateApi.list.mockResolvedValue({
      list: [
        {
          id: 301,
          name: '默认竞拍模板',
          start_price: '100.00',
          increment: '10.00',
          cap_price: '',
          duration: 3600,
          delay_duration: 30,
          max_delay_time: 180,
          trigger_delay_before: 30,
          is_default: true,
        },
      ],
      total: 1,
      page: 1,
      page_size: 100,
    })
    mockedProductApi.applyRuleTemplate.mockResolvedValue({
      id: 9001,
      product_id: 501,
      start_price: 100,
      increment: 10,
      cap_price: 0,
      duration: 3600,
      delay_duration: 30,
      max_delay_time: 180,
      trigger_delay_before: 30,
    })
    mockedAuctionApi.create.mockResolvedValue({
      id: 7001,
      product_id: 501,
      live_stream_id: 0,
      status: 0,
      current_price: 0,
      start_time: '2026-06-06T00:00:00Z',
      bid_count: 0,
      created_at: '2026-06-06T00:00:00Z',
    })
  })

  it('applies selected rule template before creating auction', async () => {
    const user = userEvent.setup()

    render(
      <MemoryRouter>
        <AuctionList />
      </MemoryRouter>
    )

    const createButton = await screen.findByRole('button', { name: '创建竞拍场次' })
    expect(createButton).toBeEnabled()
    await user.click(createButton)

    await waitFor(() => expect(mockedProductApi.list).toHaveBeenCalled())
    expect(await screen.findByLabelText('竞拍商品')).toHaveValue('501')
    expect(screen.getByLabelText('规则模板')).toHaveValue('301')

    await user.click(screen.getByRole('button', { name: '确认创建竞拍' }))

    await waitFor(() => {
      expect(mockedProductApi.applyRuleTemplate).toHaveBeenCalledWith(501, 301)
    })
    expect(mockedAuctionApi.create).toHaveBeenCalledWith({ product_id: 501, duration: 3600 })
    expect(mockedProductApi.applyRuleTemplate.mock.invocationCallOrder[0]).toBeLessThan(
      mockedAuctionApi.create.mock.invocationCallOrder[0]
    )
  })
})
