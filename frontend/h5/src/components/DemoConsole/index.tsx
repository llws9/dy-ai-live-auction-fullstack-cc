import { useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import {
  createDemoFixedPriceItem,
  createDemoMerchantAuction,
  rechargeDemoUser,
  shortenDemoAuction,
  triggerOtherSkyLamp,
  triggerFollowBid,
} from '../../services/demoApi';
import type { DemoMerchantAuctionMode } from '../../services/demoApi';
import { useAuth } from '../../store/authContext';
import { useDemo } from '../../store/demoContext';
import { useToast } from '../Toast';
import './DemoConsole.css';

type MenuView = 'root' | 'accounts' | 'demo' | 'recharge' | 'merchant';

interface DemoAccount {
  label: string;
  phone: string;
}

const DEMO_PASSWORD = 'Demo@123456';
const BUYER_A_USER_ID = 9101;
const BUYER_B_USER_ID = 9102;
const DEMO_RECHARGE_AMOUNT = '10000.00';
const TOAST_DURATION_MS = 2500;
const SHORTEN_AUCTION_REMAINING_SECONDS = 10;

const DEMO_ACCOUNTS: DemoAccount[] = [
  { label: '买家A', phone: '13800138001' },
  { label: '商家', phone: '13800138002' },
  { label: '管理员', phone: '13800138003' },
];

const RECHARGE_TARGETS = [
  { label: '演示账户A', userID: BUYER_A_USER_ID },
  { label: '演示账户B', userID: BUYER_B_USER_ID },
];

export default function DemoConsole() {
  const { login } = useAuth();
  const { currentAuctionId, currentLiveStreamId } = useDemo();
  const { showToast } = useToast();
  const location = useLocation();
  const navigate = useNavigate();
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
      if (location.pathname === '/login') {
        navigate('/', { replace: true });
      }
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

  const handleOtherSkyLamp = async () => {
    if (!currentAuctionId) {
      showToast('请先进入直播间', 'warning', TOAST_DURATION_MS);
      return;
    }

    setRunningAction('other-sky-lamp');
    try {
      await triggerOtherSkyLamp({ auctionId: currentAuctionId });
      showToast('已触发他人天灯', 'success', TOAST_DURATION_MS);
    } catch (error) {
      const message = error instanceof Error ? error.message : '请稍后重试';
      showToast(`天灯失败：${message}`, 'error', TOAST_DURATION_MS);
    } finally {
      setRunningAction(null);
    }
  };

  const handleRecharge = async (target: (typeof RECHARGE_TARGETS)[number]) => {
    const actionKey = `recharge-${target.userID}`;
    setRunningAction(actionKey);
    try {
      await rechargeDemoUser({ userId: target.userID, amount: DEMO_RECHARGE_AMOUNT });
      showToast(`已为${target.label}充值`, 'success', TOAST_DURATION_MS);
    } catch (error) {
      const message = error instanceof Error ? error.message : '请稍后重试';
      showToast(`充值失败：${message}`, 'error', TOAST_DURATION_MS);
    } finally {
      setRunningAction(null);
    }
  };

  const handleShortenAuction = async () => {
    if (!currentAuctionId) {
      showToast('请先进入直播间', 'warning', TOAST_DURATION_MS);
      return;
    }

    setRunningAction('shorten-auction');
    try {
      await shortenDemoAuction({
        auctionId: currentAuctionId,
        remainingSeconds: SHORTEN_AUCTION_REMAINING_SECONDS,
      });
      showToast('竞拍将在10秒后结束', 'success', TOAST_DURATION_MS);
    } catch (error) {
      const message = error instanceof Error ? error.message : '请稍后重试';
      showToast(`竞拍延时失败：${message}`, 'error', TOAST_DURATION_MS);
    } finally {
      setRunningAction(null);
    }
  };

  const handleMerchantAuction = async (mode: DemoMerchantAuctionMode) => {
    const actionKey = `merchant-auction-${mode}`;
    setRunningAction(actionKey);
    try {
      await createDemoMerchantAuction(mode);
      showToast(mode === 'upcoming' ? '已创建1分钟后开播的竞拍' : '已创建正在竞拍场次', 'success', TOAST_DURATION_MS);
    } catch (error) {
      const message = error instanceof Error ? error.message : '请稍后重试';
      showToast(`商家动作失败：${message}`, 'error', TOAST_DURATION_MS);
    } finally {
      setRunningAction(null);
    }
  };

  const handleMerchantFixedPrice = async () => {
    if (!currentAuctionId || !currentLiveStreamId) {
      showToast('请先进入正在竞拍的直播间', 'warning', TOAST_DURATION_MS);
      return;
    }

    setRunningAction('merchant-fixed-price');
    try {
      await createDemoFixedPriceItem({ auctionId: currentAuctionId, liveStreamId: currentLiveStreamId });
      showToast('已为当前场次创建一口价商品', 'success', TOAST_DURATION_MS);
    } catch (error) {
      const message = error instanceof Error ? error.message : '请稍后重试';
      showToast(`商家动作失败：${message}`, 'error', TOAST_DURATION_MS);
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
                onClick={() => setView('recharge')}
              >
                充值
              </button>
              <button type="button" className="demo-console__item" onClick={() => setView('merchant')}>
                商家
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
                className="demo-console__item"
                onClick={handleOtherSkyLamp}
                disabled={runningAction === 'other-sky-lamp'}
              >
                他人天灯
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
                className="demo-console__item"
                onClick={handleShortenAuction}
                disabled={runningAction === 'shorten-auction'}
              >
                倒计时
              </button>
              <button type="button" className="demo-console__item demo-console__item--secondary" onClick={() => setView('root')}>
                返回
              </button>
            </>
          )}

          {view === 'recharge' && (
            <>
              {RECHARGE_TARGETS.map((target) => (
                <button
                  key={target.userID}
                  type="button"
                  className="demo-console__item"
                  onClick={() => handleRecharge(target)}
                  disabled={runningAction === `recharge-${target.userID}`}
                >
                  {target.label}
                </button>
              ))}
              <button type="button" className="demo-console__item demo-console__item--secondary" onClick={() => setView('root')}>
                返回
              </button>
            </>
          )}

          {view === 'merchant' && (
            <>
              <button
                type="button"
                className="demo-console__item"
                onClick={() => handleMerchantAuction('upcoming')}
                disabled={runningAction === 'merchant-auction-upcoming'}
              >
                即将开播
              </button>
              <button
                type="button"
                className="demo-console__item"
                onClick={() => handleMerchantAuction('ongoing')}
                disabled={runningAction === 'merchant-auction-ongoing'}
              >
                正在竞拍
              </button>
              <button
                type="button"
                className="demo-console__item"
                onClick={handleMerchantFixedPrice}
                disabled={runningAction === 'merchant-fixed-price'}
              >
                一口价
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
