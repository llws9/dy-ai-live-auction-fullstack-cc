import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import LiveStreamFixedPrice from '../index'
import { fixedPriceAdminApi } from '@/shared/api'

jest.mock('@/shared/api', () => ({
  fixedPriceAdminApi: {
    list: jest.fn(),
    listItem: jest.fn(),
    offline: jest.fn(),
  },
}))

const listMock = fixedPriceAdminApi.list as jest.Mock
const listItemMock = fixedPriceAdminApi.listItem as jest.Mock
const offlineMock = fixedPriceAdminApi.offline as jest.Mock

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
    listMock.mockResolvedValue({ items: baseItems, total: 1, page: 1, page_size: 20 })
  })

  afterEach(() => {
    jest.restoreAllMocks()
  })

  it('loads fixed-price items for the live stream', async () => {
    renderPage()

    await waitFor(() => expect(listMock).toHaveBeenCalledWith(1001, { page: 1, page_size: 20 }))
    expect(await screen.findByText('福利翡翠手镯')).toBeInTheDocument()
    expect(screen.getByText('¥99.00')).toBeInTheDocument()
    expect(screen.getByText('12 / 20')).toBeInTheDocument()
  })

  it('adds a listed row after listing succeeds', async () => {
    listItemMock.mockResolvedValue({
      id: 7002,
      remaining_stock: 5,
      status: 'on_sale',
    })

    renderPage()

    fireEvent.change(screen.getByLabelText('商品 ID'), { target: { value: '5002' } })
    fireEvent.change(screen.getByLabelText('一口价'), { target: { value: '199.00' } })
    fireEvent.change(screen.getByLabelText('库存'), { target: { value: '5' } })
    fireEvent.click(screen.getByRole('button', { name: '新增上架' }))

    await waitFor(() => {
      expect(listItemMock).toHaveBeenCalledWith(1001, {
        product_id: 5002,
        price: '199.00',
        stock: 5,
      })
    })
    expect(await screen.findByText('商品 #5002')).toBeInTheDocument()
    expect(screen.getByText('¥199.00')).toBeInTheDocument()
    expect(screen.getByText('5 / 5')).toBeInTheDocument()
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
