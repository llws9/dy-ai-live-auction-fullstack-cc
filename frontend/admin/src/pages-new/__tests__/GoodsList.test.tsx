import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import GoodsList from '../GoodsList'
import { productApi } from '@/shared/api'

jest.mock('@/shared/api', () => ({
  productApi: {
    list: jest.fn(),
    delete: jest.fn(),
    publish: jest.fn(),
    unpublish: jest.fn(),
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

const mockedProductApi = productApi as jest.Mocked<typeof productApi>

function renderGoodsList() {
  return render(
    <MemoryRouter>
      <GoodsList />
    </MemoryRouter>
  )
}

describe('GoodsList', () => {
  beforeEach(() => {
    jest.clearAllMocks()
    mockedProductApi.list.mockResolvedValue({
      list: [
        {
          id: 1,
          name: '青花瓷瓶',
          description: '明代青花瓷瓶',
          images: [],
          category_id: 10,
          category_name: '艺术收藏',
          status: 1,
          created_at: '2026-06-04T10:00:00Z',
          updated_at: '2026-06-04T10:00:00Z',
        },
        {
          id: 2,
          name: '匿名藏品',
          description: '未归类商品',
          images: [],
          category_id: null,
          category_name: '',
          status: 0,
          created_at: '2026-06-04T11:00:00Z',
          updated_at: '2026-06-04T11:00:00Z',
        },
      ],
      total: 2,
      page: 1,
      page_size: 10,
    })
  })

  it('shows category_name and keeps 未分类 fallback when category_name is missing', async () => {
    renderGoodsList()

    await waitFor(() => {
      expect(mockedProductApi.list).toHaveBeenCalledWith({
        status: undefined,
        page: 1,
        page_size: 10,
      })
    })

    expect(await screen.findByText('艺术收藏')).toBeInTheDocument()
    expect(screen.getByText('未分类')).toBeInTheDocument()
  })

  it('shows schedulable wording instead of publish wording', async () => {
    mockedProductApi.list.mockResolvedValueOnce({
      list: [
        {
          id: 11,
          name: '青花瓷',
          description: '',
          images: [],
          category_id: null,
          category_name: '',
          status: 0,
          display_status_label: '草稿',
          created_at: '2026-06-07T00:00:00Z',
          updated_at: '2026-06-07T00:00:00Z',
        },
      ],
      total: 1,
      page: 1,
      page_size: 10,
    } as any)

    renderGoodsList()

    expect(await screen.findByText('草稿')).toBeInTheDocument()
    expect(screen.getByTitle('设为可排期')).toBeInTheDocument()
    expect(screen.queryByTitle('发布')).not.toBeInTheDocument()
  })

  it('uses derived status tabs for product lifecycle filtering', async () => {
    const user = userEvent.setup()
    renderGoodsList()

    expect(await screen.findByRole('tab', { name: '草稿' })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: '可排期' })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: '竞拍中' })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: '已拍卖' })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: '流拍' })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: '已下架' })).toBeInTheDocument()
    expect(screen.queryByRole('tab', { name: '未发布' })).not.toBeInTheDocument()
    expect(screen.queryByRole('tab', { name: '已发布' })).not.toBeInTheDocument()

    await user.click(screen.getByRole('tab', { name: '竞拍中' }))

    await waitFor(() => {
      expect(mockedProductApi.list).toHaveBeenLastCalledWith({
        display_status: 'auctioning',
        page: 1,
        page_size: 10,
      })
    })
  })
})
