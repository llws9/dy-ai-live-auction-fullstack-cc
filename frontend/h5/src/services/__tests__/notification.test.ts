import { get, post } from '../api';
import { notificationApi } from '../notification';

jest.mock('../api', () => ({
  get: jest.fn(),
  post: jest.fn(),
  put: jest.fn(),
}));

const mockedGet = get as jest.MockedFunction<typeof get>;
const mockedPost = post as jest.MockedFunction<typeof post>;

describe('notificationApi touchpoint contracts', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('loads touchpoint summary from backend', async () => {
    mockedGet.mockResolvedValue({
      unreadTotal: 7,
      pendingPayment: 2,
      wonNotPaid: 1,
      outbid: 3,
      endingSoon: 1,
    });

    await expect(notificationApi.getTouchpointSummary()).resolves.toMatchObject({
      unreadTotal: 7,
      pendingPayment: 2,
    });

    expect(mockedGet).toHaveBeenCalledWith('/notifications/summary');
  });

  it('marks touchpoint categories as read through backend', async () => {
    mockedPost.mockResolvedValue(undefined);

    await notificationApi.markCategoryAsRead('pendingPayment');

    expect(mockedPost).toHaveBeenCalledWith('/notifications/read-category', { category: 'pendingPayment' });
  });

  it('loads pending live reminder from backend', async () => {
    mockedGet.mockResolvedValue({
      hasReminder: true,
      stream: { id: 1, name: '云端珍藏直播间', avatarUrl: '', statusText: '正在直播' },
    });

    await expect(notificationApi.getPendingLiveReminder()).resolves.toMatchObject({
      hasReminder: true,
      stream: expect.objectContaining({ id: 1 }),
    });

    expect(mockedGet).toHaveBeenCalledWith('/live/pending-reminder');
  });
});
