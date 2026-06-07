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
          display_status: 'schedulable',
          display_status_label: '可排期',
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
    expect(mockedAuctionApi.create).toHaveBeenCalledWith(expect.objectContaining({
      product_id: 501,
      duration: 3600,
      start_time: expect.any(String),
    }))
    expect(mockedProductApi.applyRuleTemplate.mock.invocationCallOrder[0]).toBeLessThan(
      mockedAuctionApi.create.mock.invocationCallOrder[0]
    )
  })

  it('submits scheduled start time when merchant creates an auction', async () => {
    const user = userEvent.setup()

    render(
      <MemoryRouter>
        <AuctionList />
      </MemoryRouter>
    )

    await user.click(await screen.findByRole('button', { name: '创建竞拍场次' }))

    const startInput = await screen.findByLabelText('开拍时间')
    await user.clear(startInput)
    await user.type(startInput, '2026-06-08T10:30')
    await user.click(screen.getByRole('button', { name: '确认创建竞拍' }))

    await waitFor(() => {
      expect(mockedAuctionApi.create).toHaveBeenCalledWith({
        product_id: 501,
        duration: 3600,
        start_time: new Date('2026-06-08T10:30').toISOString(),
      })
    })
  })

  it('binds auction creation to live stream context from the live-room console', async () => {
    const user = userEvent.setup()

    render(
      <MemoryRouter initialEntries={['/auction/list?live_stream_id=501&create=1']}>
        <AuctionList />
      </MemoryRouter>
    )

    expect(await screen.findByText('当前直播间：#501')).toBeInTheDocument()
    await waitFor(() => {
      expect(mockedAuctionApi.list).toHaveBeenCalledWith(expect.objectContaining({ live_stream_id: 501 }))
    })

    await user.click(screen.getByRole('button', { name: '确认创建竞拍' }))

    await waitFor(() => {
      expect(mockedAuctionApi.create).toHaveBeenCalledWith(expect.objectContaining({
        product_id: 501,
        duration: 3600,
        live_stream_id: 501,
      }))
    })
  })

  it('shows empty schedulable products state', async () => {
    const user = userEvent.setup()
    mockedProductApi.list.mockResolvedValueOnce({ list: [], total: 0, page: 1, page_size: 100 })

    render(
      <MemoryRouter>
        <AuctionList />
      </MemoryRouter>
    )

    await user.click(await screen.findByRole('button', { name: '创建竞拍场次' }))

    expect(await screen.findByText('暂无可排期商品')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '确认创建竞拍' })).toBeDisabled()
  })

  it('only lists schedulable products in create auction selector', async () => {
    const user = userEvent.setup()
    mockedProductApi.list.mockResolvedValueOnce({
      list: [
        {
          id: 501,
          name: '可排期商品',
          description: '',
          images: [],
          category_id: null,
          category_name: '',
          status: 1,
          display_status: 'schedulable',
          display_status_label: '可排期',
          created_at: '2026-06-06T00:00:00Z',
          updated_at: '2026-06-06T00:00:00Z',
        },
        {
          id: 502,
          name: '已成交商品',
          description: '',
          images: [],
          category_id: null,
          category_name: '',
          status: 1,
          display_status: 'sold',
          display_status_label: '已拍卖',
          created_at: '2026-06-06T00:00:00Z',
          updated_at: '2026-06-06T00:00:00Z',
        },
        {
          id: 503,
          name: '竞拍中商品',
          description: '',
          images: [],
          category_id: null,
          category_name: '',
          status: 1,
          display_status: 'auctioning',
          display_status_label: '竞拍中',
          created_at: '2026-06-06T00:00:00Z',
          updated_at: '2026-06-06T00:00:00Z',
        },
      ],
      total: 3,
      page: 1,
      page_size: 100,
    })

    render(
      <MemoryRouter>
        <AuctionList />
      </MemoryRouter>
    )

    await user.click(await screen.findByRole('button', { name: '创建竞拍场次' }))

    expect(await screen.findByRole('option', { name: '可排期商品' })).toBeInTheDocument()
    expect(screen.queryByRole('option', { name: '已成交商品' })).not.toBeInTheDocument()
    expect(screen.queryByRole('option', { name: '竞拍中商品' })).not.toBeInTheDocument()
    expect(screen.getByLabelText('竞拍商品')).toHaveValue('501')
  })

  it('shows sold and unsold lifecycle tabs', async () => {
    render(
      <MemoryRouter>
        <AuctionList />
      </MemoryRouter>
    )

    expect(await screen.findByRole('tab', { name: '全部场次' })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: '竞拍中' })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: '已拍卖' })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: '流拍' })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: '已取消' })).toBeInTheDocument()
  })

  it('loads both ongoing and delayed auctions for active tab', async () => {
    const user = userEvent.setup()
    mockedAuctionApi.list.mockImplementation(async (params?: any) => {
      if (params?.status === 1) {
        return {
          list: [{
            id: 7001,
            product: { name: '正在竞拍商品' },
            status: 1,
            current_price: 100,
            start_time: '2026-06-06T00:00:00Z',
            bid_count: 1,
            live_stream_id: 8001,
          }],
          total: 1,
        }
      }
      if (params?.status === 2) {
        return {
          list: [{
            id: 7002,
            product: { name: '延时竞拍商品' },
            status: 2,
            current_price: 120,
            start_time: '2026-06-06T00:00:00Z',
            bid_count: 2,
            live_stream_id: 8002,
          }],
          total: 1,
        }
      }
      return { list: [], total: 0 }
    })

    render(
      <MemoryRouter>
        <AuctionList />
      </MemoryRouter>
    )

    await user.click(await screen.findByRole('tab', { name: '竞拍中' }))

    await waitFor(() => {
      expect(mockedAuctionApi.list).toHaveBeenCalledWith(expect.objectContaining({ status: 1 }))
      expect(mockedAuctionApi.list).toHaveBeenCalledWith(expect.objectContaining({ status: 2 }))
    })
    expect(await screen.findByText('正在竞拍商品')).toBeInTheDocument()
    expect(screen.getByText('延时竞拍商品')).toBeInTheDocument()
  })
})
