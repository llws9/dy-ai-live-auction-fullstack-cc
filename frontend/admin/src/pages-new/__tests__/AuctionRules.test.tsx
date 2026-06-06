import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import AuctionRules from '../AuctionRules'
import { auctionRuleTemplateApi } from '@/shared/api'

jest.mock('@/shared/api', () => ({
  auctionRuleTemplateApi: {
    list: jest.fn(),
    create: jest.fn(),
    update: jest.fn(),
    delete: jest.fn(),
  },
}))

const mockedAuctionRuleTemplateApi = auctionRuleTemplateApi as jest.Mocked<typeof auctionRuleTemplateApi>

describe('AuctionRules', () => {
  let confirmSpy: jest.SpyInstance

  beforeEach(() => {
    jest.clearAllMocks()
    confirmSpy = jest.spyOn(window, 'confirm').mockReturnValue(true)
    mockedAuctionRuleTemplateApi.list.mockResolvedValue({
      list: [
        {
          id: 101,
          name: '商家专场模板',
          start_price: '1000.00',
          increment: '50.00',
          cap_price: '5000.00',
          duration: 3600,
          delay_duration: 30,
          max_delay_time: 180,
          trigger_delay_before: 30,
          is_default: true,
        },
      ],
      total: 1,
      page: 1,
      page_size: 20,
    })
  })

  afterEach(() => {
    confirmSpy.mockRestore()
  })

  it('loads merchant rule templates from backend API instead of static cards', async () => {
    render(<AuctionRules />)

    await waitFor(() => {
      expect(mockedAuctionRuleTemplateApi.list).toHaveBeenCalledWith({ page: 1, page_size: 20 })
    })

    expect(await screen.findByText('商家专场模板')).toBeInTheDocument()
    expect(screen.getByText('起拍价 1000.00 元')).toBeInTheDocument()
    expect(screen.queryByText('高价值艺术品模板')).not.toBeInTheDocument()
  })

  it('creates a rule template with backend contract fields', async () => {
    const user = userEvent.setup()
    mockedAuctionRuleTemplateApi.create.mockResolvedValue({
      id: 102,
      name: '新模板',
      start_price: '200.00',
      increment: '20.00',
      cap_price: '',
      duration: 1800,
      delay_duration: 30,
      max_delay_time: 180,
      trigger_delay_before: 30,
      is_default: false,
    })

    render(<AuctionRules />)
    await screen.findByText('商家专场模板')

    await user.click(screen.getByRole('button', { name: /新建模板/ }))
    await user.clear(screen.getByLabelText('模板名称'))
    await user.type(screen.getByLabelText('模板名称'), '新模板')
    await user.clear(screen.getByLabelText('起拍价'))
    await user.type(screen.getByLabelText('起拍价'), '200.00')
    await user.clear(screen.getByLabelText('加价幅度'))
    await user.type(screen.getByLabelText('加价幅度'), '20.00')
    await user.clear(screen.getByLabelText('竞拍时长'))
    await user.type(screen.getByLabelText('竞拍时长'), '1800')
    await user.click(screen.getByRole('button', { name: '保存模板' }))

    await waitFor(() => {
      expect(mockedAuctionRuleTemplateApi.create).toHaveBeenCalledWith({
        name: '新模板',
        start_price: '200.00',
        increment: '20.00',
        cap_price: '',
        duration: 1800,
        delay_duration: 30,
        max_delay_time: 180,
        trigger_delay_before: 30,
        is_default: false,
      })
    })
    expect(mockedAuctionRuleTemplateApi.list).toHaveBeenCalledTimes(2)
  })

  it('updates, clones and deletes templates through backend API', async () => {
    const user = userEvent.setup()
    mockedAuctionRuleTemplateApi.update.mockResolvedValue({
      id: 101,
      name: '已修改模板',
      start_price: '1000.00',
      increment: '80.00',
      cap_price: '5000.00',
      duration: 3600,
      delay_duration: 30,
      max_delay_time: 180,
      trigger_delay_before: 30,
      is_default: true,
    })
    mockedAuctionRuleTemplateApi.create.mockResolvedValue({
      id: 103,
      name: '商家专场模板 副本',
      start_price: '1000.00',
      increment: '50.00',
      cap_price: '5000.00',
      duration: 3600,
      delay_duration: 30,
      max_delay_time: 180,
      trigger_delay_before: 30,
      is_default: false,
    })
    mockedAuctionRuleTemplateApi.delete.mockResolvedValue(undefined)

    render(<AuctionRules />)
    await screen.findByText('商家专场模板')

    await user.click(screen.getByRole('button', { name: /配置规则/ }))
    await user.clear(screen.getByLabelText('加价幅度'))
    await user.type(screen.getByLabelText('加价幅度'), '80.00')
    await user.click(screen.getByRole('button', { name: '保存模板' }))
    await waitFor(() => expect(mockedAuctionRuleTemplateApi.update).toHaveBeenCalledWith(101, expect.objectContaining({ increment: '80.00' })))

    await user.click(screen.getByRole('button', { name: /克隆/ }))
    await waitFor(() => {
      expect(mockedAuctionRuleTemplateApi.create).toHaveBeenCalledWith(expect.objectContaining({
        name: '商家专场模板 副本',
        is_default: false,
      }))
    })

    await user.click(screen.getByRole('button', { name: '删除 商家专场模板' }))
    await waitFor(() => expect(mockedAuctionRuleTemplateApi.delete).toHaveBeenCalledWith(101))
  })
})
