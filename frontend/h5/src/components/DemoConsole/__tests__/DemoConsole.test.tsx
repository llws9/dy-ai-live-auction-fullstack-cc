import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import DemoConsole from '../index';
import { useAuth } from '../../../store/authContext';
import { useDemo } from '../../../store/demoContext';
import { useToast } from '../../Toast';
import { rechargeDemoUser, triggerFollowBid } from '../../../services/demoApi';

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
  triggerFollowBid: jest.fn(),
  rechargeDemoUser: jest.fn(),
}));

const mockedUseAuth = useAuth as jest.MockedFunction<typeof useAuth>;
const mockedUseDemo = useDemo as jest.MockedFunction<typeof useDemo>;
const mockedUseToast = useToast as jest.MockedFunction<typeof useToast>;
const mockedTriggerFollowBid = triggerFollowBid as jest.MockedFunction<typeof triggerFollowBid>;
const mockedRechargeDemoUser = rechargeDemoUser as jest.MockedFunction<typeof rechargeDemoUser>;
const mockLogin = jest.fn();
const mockShowToast = jest.fn();

function renderConsole(currentAuctionId: number | null = 12345) {
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
    mockLogin.mockResolvedValue(undefined);
    mockedTriggerFollowBid.mockResolvedValue({ ok: true });
    mockedRechargeDemoUser.mockResolvedValue({ ok: true });
  });

  it('shows the assistive touch menu and second-level skeletons', async () => {
    const user = userEvent.setup();
    renderConsole();

    await user.click(screen.getByTestId('demo-console-fab'));

    expect(screen.getByTestId('demo-console-menu')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '账号' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '演示' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '充值' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '关闭' })).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: '账号' }));

    expect(screen.getByRole('button', { name: '买家A' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '商家' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '管理员' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '返回' })).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: '返回' }));
    await user.click(screen.getByRole('button', { name: '演示' }));

    expect(screen.getByRole('button', { name: '他人跟价' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '并发压测' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '竞拍延时' })).toBeInTheDocument();
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

  it('recharges demo buyer B with a fixed amount for background bidding', async () => {
    const user = userEvent.setup();
    renderConsole();

    await user.click(screen.getByTestId('demo-console-fab'));
    await user.click(screen.getByRole('button', { name: '充值' }));

    expect(mockedRechargeDemoUser).toHaveBeenCalledWith({ userId: 9102, amount: '10000.00' });
    expect(mockShowToast).toHaveBeenCalledWith('已为B账户充值', 'success', 2500);
  });

  it('shows a short error toast when recharge fails', async () => {
    const user = userEvent.setup();
    mockedRechargeDemoUser.mockRejectedValueOnce(new Error('余额服务不可用'));
    renderConsole();

    await user.click(screen.getByTestId('demo-console-fab'));
    await user.click(screen.getByRole('button', { name: '充值' }));

    expect(mockShowToast).toHaveBeenCalledWith('充值失败：余额服务不可用', 'error', 2500);
  });

  it('keeps pressure and delay as prompt-only demo actions', async () => {
    const user = userEvent.setup();
    renderConsole();

    await user.click(screen.getByTestId('demo-console-fab'));
    await user.click(screen.getByRole('button', { name: '演示' }));
    await user.click(screen.getByRole('button', { name: '并发压测' }));
    await user.click(screen.getByRole('button', { name: '竞拍延时' }));

    expect(mockShowToast).toHaveBeenCalledWith('并发压测暂未接入后端链路', 'info', 2500);
    expect(mockShowToast).toHaveBeenCalledWith('竞拍延时请通过临近结束出价触发', 'info', 2500);
    expect(mockedTriggerFollowBid).not.toHaveBeenCalled();
    expect(mockedRechargeDemoUser).not.toHaveBeenCalled();
  });
});
