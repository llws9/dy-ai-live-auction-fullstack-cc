import React from 'react'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import GoodsEdit from '../GoodsEdit'
import { productApi } from '@/shared/api'

jest.mock('@/shared/api', () => ({
  productApi: {
    get: jest.fn(),
    listCategories: jest.fn(),
    create: jest.fn(),
    update: jest.fn(),
    publish: jest.fn(),
    generateCopywriting: jest.fn(),
  },
}))

const mockNavigate = jest.fn()

jest.mock('react-router-dom', () => {
  const { TextEncoder } = jest.requireActual('util')
  global.TextEncoder = global.TextEncoder || TextEncoder
  const actual = jest.requireActual('react-router-dom')
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  }
})

function renderGoodsEdit(route = '/goods/edit') {
  return render(
    <MemoryRouter initialEntries={[route]}>
      <GoodsEdit />
    </MemoryRouter>
  )
}

describe('GoodsEdit AI copywriting integration', () => {
  let consoleErrorSpy: jest.SpyInstance

  beforeEach(() => {
    jest.clearAllMocks()
    window.alert = jest.fn()
    consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation(() => undefined)
    ;(productApi.listCategories as jest.Mock).mockResolvedValue([
      { id: 1, name: '艺术收藏', code: 'art' },
      { id: 2, name: '珠宝名表', code: 'jewelry' },
    ])
  })

  afterEach(() => {
    consoleErrorSpy.mockRestore()
  })

  it('does not call AI copywriting without a valid image URL', () => {
    renderGoodsEdit()

    fireEvent.click(screen.getByRole('button', { name: /AI 一键文案/ }))

    expect(productApi.generateCopywriting).not.toHaveBeenCalled()
    expect(window.alert).toHaveBeenCalledWith('请先添加至少一张商品图片')
  })

  it('shows an invalid image URL message when some added images are not http or https', () => {
    renderGoodsEdit()

    fireEvent.change(screen.getByPlaceholderText('输入图片URL'), {
      target: { value: 'ftp://cdn.example.com/bad.jpg' },
    })
    fireEvent.click(screen.getByRole('button', { name: '添加图片 URL' }))
    fireEvent.change(screen.getByPlaceholderText('输入图片URL'), {
      target: { value: 'https://cdn.example.com/good.jpg' },
    })
    fireEvent.click(screen.getByRole('button', { name: '添加图片 URL' }))

    fireEvent.click(screen.getByRole('button', { name: /AI 一键文案/ }))

    expect(productApi.generateCopywriting).not.toHaveBeenCalled()
    expect(window.alert).toHaveBeenCalledWith('图片 URL 必须以 http:// 或 https:// 开头')
  })

  it('generates AI copywriting and applies the draft to the form', async () => {
    ;(productApi.generateCopywriting as jest.Mock).mockResolvedValue({
      name: 'AI 复古相机',
      description: '这是一台适合直播竞拍的复古相机。',
      selling_points: ['复古外观', '成色良好', '适合收藏'],
      suggested_start_price: '199.00',
    })

    renderGoodsEdit()

    fireEvent.change(screen.getByPlaceholderText('输入图片URL'), {
      target: { value: 'https://cdn.example.com/camera.jpg' },
    })
    fireEvent.click(screen.getByRole('button', { name: '添加图片 URL' }))

    await waitFor(() => {
      expect(screen.getByRole('option', { name: '艺术收藏' })).toBeInTheDocument()
    })

    fireEvent.change(screen.getByRole('combobox'), {
      target: { value: '1' },
    })
    fireEvent.click(screen.getByRole('button', { name: /AI 一键文案/ }))

    await waitFor(() => {
      expect(productApi.generateCopywriting).toHaveBeenCalledWith({
        images: ['https://cdn.example.com/camera.jpg'],
        keywords: expect.stringContaining('类目：艺术收藏'),
      })
    })

    expect(screen.getByDisplayValue('AI 复古相机')).toBeInTheDocument()
    expect(screen.getByDisplayValue(/这是一台适合直播竞拍的复古相机/)).toBeInTheDocument()
    expect(screen.getByText('复古外观')).toBeInTheDocument()
    expect(screen.getByText(/AI 建议起拍价：¥199.00/)).toBeInTheDocument()
    expect(screen.getByText('AI 仅生成草稿，请确认后再保存或发布。')).toBeInTheDocument()
  })

  it('loads categories, backfills category_id in edit mode, submits category_id, and uses selected category name for AI keywords', async () => {
    ;(productApi.get as jest.Mock).mockResolvedValue({
      id: 12,
      name: '旧商品',
      description: '旧描述',
      images: ['https://cdn.example.com/old.jpg'],
      category_id: 2,
    })
    ;(productApi.update as jest.Mock).mockResolvedValue({
      id: 12,
    })
    ;(productApi.generateCopywriting as jest.Mock).mockResolvedValue({
      name: 'AI 珠宝',
      description: '适合直播竞拍的珠宝。',
      selling_points: ['宝石闪耀'],
      suggested_start_price: '299.00',
    })

    renderGoodsEdit('/goods/edit?id=12')

    expect(productApi.listCategories).toHaveBeenCalledTimes(1)
    expect(productApi.get).toHaveBeenCalledWith(12)

    await waitFor(() => {
      expect(screen.getByDisplayValue('珠宝名表')).toBeInTheDocument()
    })

    fireEvent.change(screen.getByDisplayValue('珠宝名表'), {
      target: { value: '1' },
    })

    fireEvent.click(screen.getByRole('button', { name: /AI 一键文案/ }))

    await waitFor(() => {
      expect(productApi.generateCopywriting).toHaveBeenCalledWith({
        images: ['https://cdn.example.com/old.jpg'],
        keywords: expect.stringContaining('类目：艺术收藏'),
      })
    })

    fireEvent.click(screen.getByRole('button', { name: '保存为草稿' }))

    await waitFor(() => {
      expect(productApi.update).toHaveBeenCalledWith(
        12,
        expect.objectContaining({
          category_id: 1,
        }),
      )
    })
    expect(productApi.update).not.toHaveBeenCalledWith(
      12,
      expect.objectContaining({
        category: expect.anything(),
      }),
    )
  })

  it('does not overwrite current form values when AI generation fails', async () => {
    ;(productApi.generateCopywriting as jest.Mock).mockRejectedValue({ status: 504 })

    renderGoodsEdit()

    fireEvent.change(screen.getByPlaceholderText('输入商品完整名称'), {
      target: { value: '用户手写标题' },
    })
    fireEvent.change(screen.getByPlaceholderText('详细介绍商品的来源、年代、成色等信息...'), {
      target: { value: '用户手写描述' },
    })
    fireEvent.change(screen.getByPlaceholderText('输入图片URL'), {
      target: { value: 'https://cdn.example.com/item.jpg' },
    })
    fireEvent.click(screen.getByRole('button', { name: '添加图片 URL' }))
    fireEvent.click(screen.getByRole('button', { name: /AI 一键文案/ }))

    await waitFor(() => {
      expect(window.alert).toHaveBeenCalledWith('AI 生成超时，请换一张更稳定的公网图片或稍后重试')
    })

    expect(screen.getByDisplayValue('用户手写标题')).toBeInTheDocument()
    expect(screen.getByDisplayValue('用户手写描述')).toBeInTheDocument()
  })
})
