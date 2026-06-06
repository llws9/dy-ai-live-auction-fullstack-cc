import { useState } from 'react';
import { rechargeDemoUser, triggerFollowBid } from '../../services/demoApi';
import { useAuth } from '../../store/authContext';
import { useDemo } from '../../store/demoContext';
import { useToast } from '../Toast';
import './DemoConsole.css';

type MenuView = 'root' | 'accounts' | 'demo';

interface DemoAccount {
  label: string;
  phone: string;
}

const DEMO_PASSWORD = 'Demo@123456';
const BUYER_B_USER_ID = 9102;
const BUYER_B_RECHARGE_AMOUNT = '10000.00';
const TOAST_DURATION_MS = 2500;

const DEMO_ACCOUNTS: DemoAccount[] = [
  { label: '买家A', phone: '13800138001' },
  { label: '商家', phone: '13800138002' },
  { label: '管理员', phone: '13800138003' },
];

export default function DemoConsole() {
  const { login } = useAuth();
  const { currentAuctionId } = useDemo();
  const { showToast } = useToast();
  const [open, setOpen] = useState(false);
  const [view, setView] = useState<MenuView>('root');
  const [switchingPhone, setSwitchingPhone] = useState<string | null>(null);
  const [runningAction, setRunningAction] = useState<string | null>(null);

  const handleClose = () => {
    setOpen(false);
    setView('root');
  };

  const handleSwitchAccount = async (account: DemoAccount) => {
    setSwitchingPhone(account.phone);
    try {
      await login({ phone: account.phone, password: DEMO_PASSWORD });
      showToast(`已切换到${account.label}`, 'success', TOAST_DURATION_MS);
    } catch (error) {
      const message = error instanceof Error ? error.message : '请稍后重试';
      showToast(`切换账号失败：${message}`, 'error', TOAST_DURATION_MS);
    } finally {
      setSwitchingPhone(null);
    }
  };

  const handleFollowBid = async () => {
    if (!currentAuctionId) {
      showToast('请先进入直播间', 'warning', TOAST_DURATION_MS);
      return;
    }

    setRunningAction('follow-bid');
    try {
      await triggerFollowBid({ auctionId: currentAuctionId });
      showToast('已触发他人跟价', 'success', TOAST_DURATION_MS);
    } catch (error) {
      const message = error instanceof Error ? error.message : '请稍后重试';
      showToast(`跟价失败：${message}`, 'error', TOAST_DURATION_MS);
    } finally {
      setRunningAction(null);
    }
  };

  const handleRechargeBuyerB = async () => {
    setRunningAction('recharge');
    try {
      await rechargeDemoUser({ userId: BUYER_B_USER_ID, amount: BUYER_B_RECHARGE_AMOUNT });
      showToast('已为B账户充值', 'success', TOAST_DURATION_MS);
    } catch (error) {
      const message = error instanceof Error ? error.message : '请稍后重试';
      showToast(`充值失败：${message}`, 'error', TOAST_DURATION_MS);
    } finally {
      setRunningAction(null);
    }
  };

  const showPromptOnlyAction = (message: string) => {
    showToast(message, 'info', TOAST_DURATION_MS);
  };

  return (
    <div className={`demo-console ${open ? 'demo-console--open' : ''}`} data-testid="demo-console">
      {open && (
        <div className="demo-console__menu" data-testid="demo-console-menu" role="menu" aria-label="演示控制菜单">
          {view === 'root' && (
            <>
              <button type="button" className="demo-console__item" onClick={() => setView('accounts')}>
                账号
              </button>
              <button type="button" className="demo-console__item" onClick={() => setView('demo')}>
                演示
              </button>
              <button
                type="button"
                className="demo-console__item"
                onClick={handleRechargeBuyerB}
                disabled={runningAction === 'recharge'}
              >
                充值
              </button>
              <button type="button" className="demo-console__item demo-console__item--danger" onClick={handleClose}>
                关闭
              </button>
            </>
          )}

          {view === 'accounts' && (
            <>
              {DEMO_ACCOUNTS.map((account) => (
                <button
                  key={account.phone}
                  type="button"
                  className="demo-console__item"
                  onClick={() => handleSwitchAccount(account)}
                  disabled={switchingPhone === account.phone}
                >
                  {account.label}
                </button>
              ))}
              <button type="button" className="demo-console__item demo-console__item--secondary" onClick={() => setView('root')}>
                返回
              </button>
            </>
          )}

          {view === 'demo' && (
            <>
              <button
                type="button"
                className="demo-console__item"
                onClick={handleFollowBid}
                disabled={runningAction === 'follow-bid'}
              >
                他人跟价
              </button>
              <button
                type="button"
                className="demo-console__item demo-console__item--placeholder"
                onClick={() => showPromptOnlyAction('并发压测暂未接入后端链路')}
              >
                并发压测
              </button>
              <button
                type="button"
                className="demo-console__item demo-console__item--placeholder"
                onClick={() => showPromptOnlyAction('竞拍延时请通过临近结束出价触发')}
              >
                竞拍延时
              </button>
              <button type="button" className="demo-console__item demo-console__item--secondary" onClick={() => setView('root')}>
                返回
              </button>
            </>
          )}
        </div>
      )}

      <button
        type="button"
        className="demo-console__fab"
        data-testid="demo-console-fab"
        aria-label={open ? '收起演示控制台' : '打开演示控制台'}
        onClick={() => {
          setOpen((nextOpen) => !nextOpen);
          setView('root');
        }}
      >
        Demo
      </button>
    </div>
  );
}
