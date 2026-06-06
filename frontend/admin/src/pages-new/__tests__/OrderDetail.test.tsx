import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import OrderDetail from '../OrderDetail'
import { orderApi } from '@/shared/api'

jest.mock('@/shared/api', () => ({
  orderApi: {
    get: jest.fn(),
    ship: jest.fn(),
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

function renderOrderDetail() {
  return render(
    <MemoryRouter initialEntries={['/order/detail?id=101']}>
      <OrderDetail />
    </MemoryRouter>
  )
}

const baseOrder = {
  id: 101,
  auction_id: 201,
  product_id: 11,
  product_name: '宋代玉镯',
  product_image: '',
  user_id: 901,
  final_price: 3000,
  status: 1,
  created_at: '2026-06-06T10:00:00Z',
}

beforeEach(() => {
  jest.clearAllMocks()
})

describe('OrderDetail', () => {
  it('renders buyer nickname and avatar when provided by backend', async () => {
    mockedOrderApi.get.mockResolvedValue({
      ...baseOrder,
      user_name: '张三',
      user_avatar: 'https://cdn/u901.png',
    })

    renderOrderDetail()

    expect(await screen.findByText('买家：张三')).toBeInTheDocument()
    expect(screen.getByAltText('张三')).toHaveAttribute('src', 'https://cdn/u901.png')
  })

  it('falls back to user id when buyer nickname is missing', async () => {
    mockedOrderApi.get.mockResolvedValue({
      ...baseOrder,
      user_name: '',
      user_avatar: '',
    })

    renderOrderDetail()

    expect(await screen.findByText('买家：用户 #901')).toBeInTheDocument()
    expect(screen.queryByAltText('用户 #901')).not.toBeInTheDocument()
  })
})

