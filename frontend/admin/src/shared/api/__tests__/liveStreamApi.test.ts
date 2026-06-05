import { get } from '../request';
import { liveStreamApi } from '../index';

jest.mock('../request', () => ({
  get: jest.fn(),
  post: jest.fn(),
  put: jest.fn(),
  del: jest.fn(),
  buildQuery: (params: Record<string, string | number | undefined>) =>
    new URLSearchParams(
      Object.entries(params)
        .filter(([, value]) => value !== undefined)
        .map(([key, value]) => [key, String(value)])
    ).toString(),
}));

describe('liveStreamApi', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('uses scoped admin endpoint for management detail pages', async () => {
    (get as jest.Mock).mockResolvedValue({ id: 501 });

    await liveStreamApi.adminGet(501);

    expect(get).toHaveBeenCalledWith('/admin/live-streams/501');
  });
});
