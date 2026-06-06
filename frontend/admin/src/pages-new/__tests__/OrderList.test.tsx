import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import OrderList from '../OrderList'
import { orderApi, statisticsApi } from '@/shared/api'

jest.mock('@/shared/api', () => ({
  orderApi: {
    list: jest.fn(),
    ship: jest.fn(),
  },
  statisticsApi: {
    getOverview: jest.fn(),
  },
}))

const mockNavigate = jest.fn()

jest.mock('react-router-dom', () => {
  const actual = jest.requireActual('react-router-dom')
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  }
})

const mockedOrderApi = orderApi as jest.Mocked<typeof orderApi>
const mockedStatisticsApi = statisticsApi as jest.Mocked<typeof statisticsApi>

function renderOrderList() {
  return render(
    <MemoryRouter>
      <OrderList />
    </MemoryRouter>
  )
}

beforeEach(() => {
  jest.clearAllMocks()
  mockedStatisticsApi.getOverview.mockResolvedValue({
    today_revenue: 1200,
    total_revenue: 9800,
  })
  mockedOrderApi.list.mockResolvedValue({
    list: [
      {
        id: 101,
        product_id: 11,
        product_name: '宋代玉镯',
        user_id: 901,
        user_name: '张三',
        user_avatar: 'https://cdn/u901.png',
        final_price: 3000,
        status: 1,
        created_at: '2026-06-06T10:00:00Z',
      },
    ],
    total: 1,
    page: 1,
    page_size: 20,
    summary: {
      pending_payment_count: 3,
      paid_count: 2,
      shipped_count: 1,
      completed_count: 4,
    },
  })
})

describe('OrderList', () => {
  it('renders backend summary and buyer nickname', async () => {
    renderOrderList()

    expect(await screen.findByText('张三')).toBeInTheDocument()
    expect(screen.getByText('3')).toBeInTheDocument()
    expect(screen.getByText('2')).toBeInTheDocument()
    expect(screen.getByText('宋代玉镯')).toBeInTheDocument()
  })

  it('submits search to backend instead of filtering the current page only', async () => {
    renderOrderList()

    await waitFor(() => {
      expect(mockedOrderApi.list).toHaveBeenCalledWith({
        status: undefined,
        page: 1,
        page_size: 20,
      })
    })

    fireEvent.change(screen.getByPlaceholderText('搜索订单号/商品/买家ID'), {
      target: { value: '玉镯' },
    })
    fireEvent.click(screen.getByRole('button', { name: '搜索' }))

    await waitFor(() => {
      expect(mockedOrderApi.list).toHaveBeenLastCalledWith({
        status: undefined,
        search: '玉镯',
        page: 1,
        page_size: 20,
      })
    })
  })

  it('does not expose no-op filter or more action buttons', async () => {
    const { container } = renderOrderList()

    await screen.findByText('张三')

    expect(container.querySelector('.lucide-filter')).not.toBeInTheDocument()
    expect(container.querySelector('.lucide-more-horizontal')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: '查看详情' })).toBeInTheDocument()
  })
})
