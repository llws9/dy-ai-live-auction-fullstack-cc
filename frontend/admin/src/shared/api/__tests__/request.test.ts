import { ApiError, post } from '../request'

function jsonResponse(status: number, body: unknown) {
  return {
    ok: status >= 200 && status < 300,
    status,
    headers: {
      get: (key: string) => (key.toLowerCase() === 'content-type' ? 'application/json' : null),
    },
    json: async () => body,
  }
}

describe('request response code handling', () => {
  const originalFetch = global.fetch

  beforeEach(() => {
    localStorage.clear()
    global.fetch = jest.fn()
  })

  afterEach(() => {
    global.fetch = originalFetch
  })

  it('treats HTTP 201 with business code 201 as success', async () => {
    ;(global.fetch as jest.Mock).mockResolvedValue(
      jsonResponse(201, {
          code: 201,
          message: 'success',
          data: { id: 1, name: '默认模板' },
      })
    )

    await expect(post('/admin/auction-rule-templates', { name: '默认模板' }, { showError: false })).resolves.toEqual({
      id: 1,
      name: '默认模板',
    })
  })

  it('still rejects non-success business codes on 2xx HTTP responses', async () => {
    ;(global.fetch as jest.Mock).mockResolvedValue(
      jsonResponse(200, {
          code: 400,
          message: '模板名称不能为空',
      })
    )

    await expect(post('/admin/auction-rule-templates', {}, { showError: false })).rejects.toMatchObject<ApiError>({
      status: 200,
      code: 400,
      message: '模板名称不能为空',
    })
  })
})
