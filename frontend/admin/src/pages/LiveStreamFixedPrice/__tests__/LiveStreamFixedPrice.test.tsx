import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import LiveStreamFixedPrice from '../index'
import { auctionApi, fixedPriceAdminApi, liveStreamApi, productApi } from '@/shared/api'

jest.mock('@/shared/api', () => ({
  auctionApi: {
    list: jest.fn(),
  },
  liveStreamApi: {
    adminList: jest.fn(),
  },
  fixedPriceAdminApi: {
    list: jest.fn(),
    listItem: jest.fn(),
    offline: jest.fn(),
  },
  productApi: {
    list: jest.fn(),
  },
}))

const listMock = fixedPriceAdminApi.list as jest.Mock
const listItemMock = fixedPriceAdminApi.listItem as jest.Mock
const offlineMock = fixedPriceAdminApi.offline as jest.Mock
const adminListMock = liveStreamApi.adminList as jest.Mock
const auctionListMock = auctionApi.list as jest.Mock
const productListMock = productApi.list as jest.Mock

const baseItems = [
  {
    id: 7001,
    live_stream_id: 1001,
    product_id: 5001,
    product_title: '福利翡翠手镯',
    price: '99.00',
    total_stock: 20,
    remaining_stock: 12,
    status: 'on_sale',
  },
]

function renderPage() {
  return render(
    <MemoryRouter>
      <LiveStreamFixedPrice liveStreamId={1001} />
    </MemoryRouter>
  )
}

describe('LiveStreamFixedPrice', () => {
  beforeEach(() => {
    jest.clearAllMocks()
    jest.spyOn(window, 'confirm').mockReturnValue(true)
    adminListMock.mockResolvedValue({ list: [], total: 0, page: 1, page_size: 1 })
    auctionListMock.mockResolvedValue({
      list: [{ id: 8001, live_stream_id: 1001, status: 1, product: { name: '当前竞拍商品' } }],
      total: 1,
    })
    listMock.mockResolvedValue({ items: baseItems, total: 1, page: 1, page_size: 20 })
    productListMock.mockResolvedValue({
      list: [
        { id: 5002, name: '搭售周边A', status: 1, display_status: 'schedulable' },
        { id: 5003, name: '搭售周边B', status: 1, display_status: 'schedulable' },
      ],
      total: 2,
      page: 1,
      page_size: 100,
    })
  })

  afterEach(() => {
    jest.restoreAllMocks()
  })

  it('loads fixed-price items for the live stream', async () => {
    renderPage()

    await waitFor(() => expect(listMock).toHaveBeenCalledWith(1001, { page: 1, page_size: 20 }))
    expect(auctionListMock).toHaveBeenCalledWith({ live_stream_id: 1001, page: 1, page_size: 100 })
    expect(await screen.findByLabelText('竞拍场次')).toHaveValue('8001')
    expect(await screen.findByText('福利翡翠手镯')).toBeInTheDocument()
    expect(screen.getByText('¥99.00')).toBeInTheDocument()
    expect(screen.getByText('12 / 20')).toBeInTheDocument()
  })

  it('auto-resolves the merchant live stream when route id is missing', async () => {
    adminListMock.mockResolvedValueOnce({
      list: [{ id: 1001, name: '商家直播间', status: 1 }],
      total: 1,
      page: 1,
      page_size: 1,
    })

    render(
      <MemoryRouter initialEntries={['/live/fixed-price']}>
        <LiveStreamFixedPrice />
      </MemoryRouter>
    )

    await waitFor(() => expect(adminListMock).toHaveBeenCalledWith({ page: 1, page_size: 1 }))
    await waitFor(() => expect(listMock).toHaveBeenCalledWith(1001, { page: 1, page_size: 20 }))
    expect(await screen.findByText('福利翡翠手镯')).toBeInTheDocument()
  })

  it('adds a listed row after listing succeeds', async () => {
    listItemMock.mockResolvedValue({
      id: 7002,
      auction_id: 8001,
      remaining_stock: 5,
      status: 'on_sale',
    })

    renderPage()

    await waitFor(() => expect(screen.getByLabelText('竞拍场次')).toHaveValue('8001'))
    await waitFor(() => expect(productListMock).toHaveBeenCalledWith({ display_status: 'schedulable', page: 1, page_size: 100 }))
    fireEvent.change(screen.getByLabelText('搭售商品'), { target: { value: '5002' } })
    fireEvent.change(screen.getByLabelText('一口价'), { target: { value: '199.00' } })
    fireEvent.change(screen.getByLabelText('库存'), { target: { value: '5' } })
    fireEvent.click(screen.getByRole('button', { name: '新增上架' }))

    await waitFor(() => {
      expect(listItemMock).toHaveBeenCalledWith(1001, {
        auction_id: 8001,
        product_id: 5002,
        price: '199.00',
        stock: 5,
      })
    })
    expect(await screen.findByText('搭售周边A')).toBeInTheDocument()
    expect(screen.getByText('¥199.00')).toBeInTheDocument()
    expect(screen.getByText('5 / 5')).toBeInTheDocument()
  })

  it('disables product select when no schedulable product available', async () => {
    productListMock.mockResolvedValueOnce({ list: [], total: 0, page: 1, page_size: 100 })

    renderPage()

    const select = await screen.findByLabelText('搭售商品')
    await waitFor(() => expect(select).toBeDisabled())
    expect(screen.getByText('暂无可搭售商品，请先创建并发布商品')).toBeInTheDocument()
  })

  it('confirms offline and updates the row status', async () => {
    offlineMock.mockResolvedValue({ id: 7001, status: 'offline' })

    renderPage()

    fireEvent.click(await screen.findByRole('button', { name: '下架' }))

    await waitFor(() => expect(offlineMock).toHaveBeenCalledWith(7001))
    expect(window.confirm).toHaveBeenCalledWith('确认下架该一口价商品？')
    expect(await screen.findByText('已下架')).toBeInTheDocument()
  })
})
