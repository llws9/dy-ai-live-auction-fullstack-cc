import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import DemoConsole from '../index';
import { useAuth } from '../../../store/authContext';
import { useDemo } from '../../../store/demoContext';
import { useToast } from '../../Toast';
import {
  createDemoFixedPriceItem,
  createDemoMerchantAuction,
  rechargeDemoUser,
  shortenDemoAuction,
  triggerOtherSkyLamp,
  triggerFollowBid,
} from '../../../services/demoApi';

jest.mock('../../../store/authContext', () => ({
  useAuth: jest.fn(),
}));

jest.mock('../../../store/demoContext', () => ({
  useDemo: jest.fn(),
}));

jest.mock('../../Toast', () => ({
  useToast: jest.fn(),
}));

jest.mock('../../../services/demoApi', () => ({
  createDemoFixedPriceItem: jest.fn(),
  createDemoMerchantAuction: jest.fn(),
  shortenDemoAuction: jest.fn(),
  triggerOtherSkyLamp: jest.fn(),
  triggerFollowBid: jest.fn(),
  rechargeDemoUser: jest.fn(),
}));

const mockNavigate = jest.fn();
let mockPathname = '/';

jest.mock('react-router-dom', () => ({
  useLocation: () => ({ pathname: mockPathname }),
  useNavigate: () => mockNavigate,
}));

const mockedUseAuth = useAuth as jest.MockedFunction<typeof useAuth>;
const mockedUseDemo = useDemo as jest.MockedFunction<typeof useDemo>;
const mockedUseToast = useToast as jest.MockedFunction<typeof useToast>;
const mockedTriggerFollowBid = triggerFollowBid as jest.MockedFunction<typeof triggerFollowBid>;
const mockedTriggerOtherSkyLamp = triggerOtherSkyLamp as jest.MockedFunction<typeof triggerOtherSkyLamp>;
const mockedRechargeDemoUser = rechargeDemoUser as jest.MockedFunction<typeof rechargeDemoUser>;
const mockedCreateDemoMerchantAuction = createDemoMerchantAuction as jest.MockedFunction<typeof createDemoMerchantAuction>;
const mockedCreateDemoFixedPriceItem = createDemoFixedPriceItem as jest.MockedFunction<typeof createDemoFixedPriceItem>;
const mockedShortenDemoAuction = shortenDemoAuction as jest.MockedFunction<typeof shortenDemoAuction>;
const mockLogin = jest.fn();
const mockShowToast = jest.fn();

function renderConsole(currentAuctionId: number | null = 12345, currentLiveStreamId: number | null = 88) {
  mockedUseAuth.mockReturnValue({
    isAuthenticated: true,
    user: { id: 1, email: 'buyer@example.com', name: '买家A', role: 0 },
    token: 'token-1',
    loading: false,
    login: mockLogin,
    setAuth: jest.fn(),
    logout: jest.fn(),
    isAdmin: jest.fn(() => false),
    isMerchant: jest.fn(() => false),
  });
  mockedUseDemo.mockReturnValue({
    currentAuctionId,
    setCurrentAuctionId: jest.fn(),
    currentLiveStreamId,
    setCurrentLiveStreamId: jest.fn(),
  });
  mockedUseToast.mockReturnValue({
    showToast: mockShowToast,
    showLoading: jest.fn(),
  });

  return render(<DemoConsole />);
}

describe('DemoConsole', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockPathname = '/';
    mockLogin.mockResolvedValue(undefined);
    mockedTriggerFollowBid.mockResolvedValue({ ok: true });
    mockedTriggerOtherSkyLamp.mockResolvedValue({ ok: true });
    mockedRechargeDemoUser.mockResolvedValue({ ok: true });
    mockedCreateDemoMerchantAuction.mockResolvedValue({ ok: true });
    mockedCreateDemoFixedPriceItem.mockResolvedValue({ ok: true });
    mockedShortenDemoAuction.mockResolvedValue({ ok: true });
  });

  it('shows the assistive touch menu and second-level skeletons', async () => {
    const user = userEvent.setup();
    renderConsole();

    await user.click(screen.getByTestId('demo-console-fab'));

    expect(screen.getByTestId('demo-console-menu')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '账号' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '演示' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '充值' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '商家' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '关闭' })).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: '账号' }));

    expect(screen.getByRole('button', { name: '买家A' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '商家' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '管理员' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '返回' })).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: '返回' }));
    await user.click(screen.getByRole('button', { name: '演示' }));

    expect(screen.getByRole('button', { name: '他人跟价' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '他人天灯' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '并发压测' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '倒计时' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '返回' })).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: '返回' }));
    await user.click(screen.getByRole('button', { name: '充值' }));

    expect(screen.getByRole('button', { name: '演示账户A' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '演示账户B' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '返回' })).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: '返回' }));
    await user.click(screen.getByRole('button', { name: '商家' }));

    expect(screen.getByRole('button', { name: '即将开播' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '正在竞拍' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '一口价' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '返回' })).toBeInTheDocument();
  });

  it('switches demo accounts through useAuth login with unified 138 accounts', async () => {
    const user = userEvent.setup();
    renderConsole();

    await user.click(screen.getByTestId('demo-console-fab'));
    await user.click(screen.getByRole('button', { name: '账号' }));
    await user.click(screen.getByRole('button', { name: '买家A' }));
    await user.click(screen.getByRole('button', { name: '商家' }));
    await user.click(screen.getByRole('button', { name: '管理员' }));

    expect(mockLogin).toHaveBeenNthCalledWith(1, {
      phone: '13800138001',
      password: 'Demo@123456',
    });
    expect(mockLogin).toHaveBeenNthCalledWith(2, {
      phone: '13800138002',
      password: 'Demo@123456',
    });
    expect(mockLogin).toHaveBeenNthCalledWith(3, {
      phone: '13800138003',
      password: 'Demo@123456',
    });
    expect(mockNavigate).not.toHaveBeenCalled();
  });

  it('navigates from login page to home after switching a demo account', async () => {
    const user = userEvent.setup();
    mockPathname = '/login';
    renderConsole();

    await user.click(screen.getByTestId('demo-console-fab'));
    await user.click(screen.getByRole('button', { name: '账号' }));
    await user.click(screen.getByRole('button', { name: '买家A' }));

    expect(mockLogin).toHaveBeenCalledWith({
      phone: '13800138001',
      password: 'Demo@123456',
    });
    expect(mockNavigate).toHaveBeenCalledWith('/', { replace: true });
  });

  it('shows an error toast when account switching fails', async () => {
    const user = userEvent.setup();
    mockLogin.mockRejectedValueOnce(new Error('密码错误'));
    renderConsole();

    await user.click(screen.getByTestId('demo-console-fab'));
    await user.click(screen.getByRole('button', { name: '账号' }));
    await user.click(screen.getByRole('button', { name: '买家A' }));

    expect(mockShowToast).toHaveBeenCalledWith('切换账号失败：密码错误', 'error', 2500);
  });

  it('collapses the menu when closed while keeping the floating entry available', async () => {
    const user = userEvent.setup();
    renderConsole();

    await user.click(screen.getByTestId('demo-console-fab'));
    await user.click(screen.getByRole('button', { name: '关闭' }));

    expect(screen.getByTestId('demo-console-fab')).toBeInTheDocument();
    expect(screen.queryByTestId('demo-console-menu')).not.toBeInTheDocument();
  });

  it('warns and skips the follow-bid api when there is no current auction', async () => {
    const user = userEvent.setup();
    renderConsole(null);

    await user.click(screen.getByTestId('demo-console-fab'));
    await user.click(screen.getByRole('button', { name: '演示' }));
    await user.click(screen.getByRole('button', { name: '他人跟价' }));

    expect(mockShowToast).toHaveBeenCalledWith('请先进入直播间', 'warning', 2500);
    expect(mockedTriggerFollowBid).not.toHaveBeenCalled();
  });

  it('triggers buyer B follow-bid for the current auction and reports success', async () => {
    const user = userEvent.setup();
    renderConsole(777);

    await user.click(screen.getByTestId('demo-console-fab'));
    await user.click(screen.getByRole('button', { name: '演示' }));
    await user.click(screen.getByRole('button', { name: '他人跟价' }));

    expect(mockedTriggerFollowBid).toHaveBeenCalledWith({ auctionId: 777 });
    expect(mockShowToast).toHaveBeenCalledWith('已触发他人跟价', 'success', 2500);
  });

  it('shows a short error toast when follow-bid fails', async () => {
    const user = userEvent.setup();
    mockedTriggerFollowBid.mockRejectedValueOnce(new Error('跟价冲突，请重试'));
    renderConsole(777);

    await user.click(screen.getByTestId('demo-console-fab'));
    await user.click(screen.getByRole('button', { name: '演示' }));
    await user.click(screen.getByRole('button', { name: '他人跟价' }));

    expect(mockShowToast).toHaveBeenCalledWith('跟价失败：跟价冲突，请重试', 'error', 2500);
  });

  it('triggers buyer B sky lamp for the current auction and reports success', async () => {
    const user = userEvent.setup();
    renderConsole(777);

    await user.click(screen.getByTestId('demo-console-fab'));
    await user.click(screen.getByRole('button', { name: '演示' }));
    await user.click(screen.getByRole('button', { name: '他人天灯' }));

    expect(mockedTriggerOtherSkyLamp).toHaveBeenCalledWith({ auctionId: 777 });
    expect(mockShowToast).toHaveBeenCalledWith('已触发他人天灯', 'success', 2500);
  });

  it('warns and skips other sky lamp when there is no current auction', async () => {
    const user = userEvent.setup();
    renderConsole(null);

    await user.click(screen.getByTestId('demo-console-fab'));
    await user.click(screen.getByRole('button', { name: '演示' }));
    await user.click(screen.getByRole('button', { name: '他人天灯' }));

    expect(mockShowToast).toHaveBeenCalledWith('请先进入直播间', 'warning', 2500);
    expect(mockedTriggerOtherSkyLamp).not.toHaveBeenCalled();
  });

  it('shows a short error toast when other sky lamp fails', async () => {
    const user = userEvent.setup();
    mockedTriggerOtherSkyLamp.mockRejectedValueOnce(new Error('已有活跃的点天灯订阅'));
    renderConsole(777);

    await user.click(screen.getByTestId('demo-console-fab'));
    await user.click(screen.getByRole('button', { name: '演示' }));
    await user.click(screen.getByRole('button', { name: '他人天灯' }));

    expect(mockShowToast).toHaveBeenCalledWith('天灯失败：已有活跃的点天灯订阅', 'error', 2500);
  });

  it('recharges demo buyer A and B with a fixed amount from the second-level menu', async () => {
    const user = userEvent.setup();
    renderConsole();

    await user.click(screen.getByTestId('demo-console-fab'));
    await user.click(screen.getByRole('button', { name: '充值' }));
    await user.click(screen.getByRole('button', { name: '演示账户A' }));
    await user.click(screen.getByRole('button', { name: '演示账户B' }));

    expect(mockedRechargeDemoUser).toHaveBeenNthCalledWith(1, { userId: 9101, amount: '10000.00' });
    expect(mockedRechargeDemoUser).toHaveBeenNthCalledWith(2, { userId: 9102, amount: '10000.00' });
    expect(mockShowToast).toHaveBeenCalledWith('已为演示账户A充值', 'success', 2500);
    expect(mockShowToast).toHaveBeenCalledWith('已为演示账户B充值', 'success', 2500);
  });

  it('shows a short error toast when recharge fails', async () => {
    const user = userEvent.setup();
    mockedRechargeDemoUser.mockRejectedValueOnce(new Error('余额服务不可用'));
    renderConsole();

    await user.click(screen.getByTestId('demo-console-fab'));
    await user.click(screen.getByRole('button', { name: '充值' }));
    await user.click(screen.getByRole('button', { name: '演示账户B' }));

    expect(mockShowToast).toHaveBeenCalledWith('充值失败：余额服务不可用', 'error', 2500);
  });

  it('creates upcoming and ongoing merchant auctions from the merchant menu', async () => {
    const user = userEvent.setup();
    renderConsole();

    await user.click(screen.getByTestId('demo-console-fab'));
    await user.click(screen.getByRole('button', { name: '商家' }));
    await user.click(screen.getByRole('button', { name: '即将开播' }));
    await user.click(screen.getByRole('button', { name: '正在竞拍' }));

    expect(mockedCreateDemoMerchantAuction).toHaveBeenNthCalledWith(1, 'upcoming');
    expect(mockedCreateDemoMerchantAuction).toHaveBeenNthCalledWith(2, 'ongoing');
    expect(mockShowToast).toHaveBeenCalledWith('已创建1分钟后开播的竞拍', 'success', 2500);
    expect(mockShowToast).toHaveBeenCalledWith('已创建正在竞拍场次', 'success', 2500);
  });

  it('warns and skips fixed-price creation outside a live room', async () => {
    const user = userEvent.setup();
    renderConsole(12345, null);

    await user.click(screen.getByTestId('demo-console-fab'));
    await user.click(screen.getByRole('button', { name: '商家' }));
    await user.click(screen.getByRole('button', { name: '一口价' }));

    expect(mockShowToast).toHaveBeenCalledWith('请先进入正在竞拍的直播间', 'warning', 2500);
    expect(mockedCreateDemoFixedPriceItem).not.toHaveBeenCalled();
  });

  it('creates a fixed-price item for the current live room', async () => {
    const user = userEvent.setup();
    renderConsole(12345, 880301);

    await user.click(screen.getByTestId('demo-console-fab'));
    await user.click(screen.getByRole('button', { name: '商家' }));
    await user.click(screen.getByRole('button', { name: '一口价' }));

    expect(mockedCreateDemoFixedPriceItem).toHaveBeenCalledWith({ auctionId: 12345, liveStreamId: 880301 });
    expect(mockShowToast).toHaveBeenCalledWith('已为当前场次创建一口价商品', 'success', 2500);
  });

  it('shows a short error toast when a merchant action fails', async () => {
    const user = userEvent.setup();
    mockedCreateDemoMerchantAuction.mockRejectedValueOnce(new Error('创建失败'));
    renderConsole();

    await user.click(screen.getByTestId('demo-console-fab'));
    await user.click(screen.getByRole('button', { name: '商家' }));
    await user.click(screen.getByRole('button', { name: '即将开播' }));

    expect(mockShowToast).toHaveBeenCalledWith('商家动作失败：创建失败', 'error', 2500);
  });

  it('shortens the current auction to ten seconds from the demo menu', async () => {
    const user = userEvent.setup();
    renderConsole(777);

    await user.click(screen.getByTestId('demo-console-fab'));
    await user.click(screen.getByRole('button', { name: '演示' }));
    await user.click(screen.getByRole('button', { name: '倒计时' }));

    expect(mockedShortenDemoAuction).toHaveBeenCalledWith({ auctionId: 777, remainingSeconds: 10 });
    expect(mockShowToast).toHaveBeenCalledWith('竞拍将在10秒后结束', 'success', 2500);
  });

  it('warns and skips auction shorten when there is no current auction', async () => {
    const user = userEvent.setup();
    renderConsole(null);

    await user.click(screen.getByTestId('demo-console-fab'));
    await user.click(screen.getByRole('button', { name: '演示' }));
    await user.click(screen.getByRole('button', { name: '倒计时' }));

    expect(mockShowToast).toHaveBeenCalledWith('请先进入直播间', 'warning', 2500);
    expect(mockedShortenDemoAuction).not.toHaveBeenCalled();
  });

  it('shows a short error toast when auction shorten fails', async () => {
    const user = userEvent.setup();
    mockedShortenDemoAuction.mockRejectedValueOnce(new Error('竞拍已结束'));
    renderConsole(777);

    await user.click(screen.getByTestId('demo-console-fab'));
    await user.click(screen.getByRole('button', { name: '演示' }));
    await user.click(screen.getByRole('button', { name: '倒计时' }));

    expect(mockShowToast).toHaveBeenCalledWith('竞拍延时失败：竞拍已结束', 'error', 2500);
  });

  it('keeps pressure as a prompt-only demo action', async () => {
    const user = userEvent.setup();
    renderConsole();

    await user.click(screen.getByTestId('demo-console-fab'));
    await user.click(screen.getByRole('button', { name: '演示' }));
    await user.click(screen.getByRole('button', { name: '并发压测' }));

    expect(mockShowToast).toHaveBeenCalledWith('并发压测暂未接入后端链路', 'info', 2500);
    expect(mockedTriggerFollowBid).not.toHaveBeenCalled();
    expect(mockedTriggerOtherSkyLamp).not.toHaveBeenCalled();
    expect(mockedShortenDemoAuction).not.toHaveBeenCalled();
    expect(mockedRechargeDemoUser).not.toHaveBeenCalled();
  });
});
