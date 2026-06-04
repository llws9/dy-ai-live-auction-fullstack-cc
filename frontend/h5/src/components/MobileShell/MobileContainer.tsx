import { ReactNode, useEffect, useRef, useState } from 'react';
import { useLocation } from 'react-router-dom';
import { notificationApi } from '../../services/notification';
import { useAuth } from '../../store/authContext';
import { trackEvent } from '../../utils/trackEvent';
import LiveReminderModal, { StreamInfo } from '../LiveReminderModal';
import BottomNav from './BottomNav';
import styles from './MobileShell.module.css';

interface MobileContainerProps {
  children: ReactNode;
}

function MobileContainer({ children }: MobileContainerProps) {
  const { pathname } = useLocation();
  const [isReminderOpen, setIsReminderOpen] = useState(false);
  const [reminderStream, setReminderStream] = useState<StreamInfo | null>(null);
  const { isAuthenticated, loading: authLoading, token, user } = useAuth();
  const userId = user?.id ?? null;
  const identityRef = useRef({ token, userId });
  const isLiveRoute = pathname.startsWith('/live');

  identityRef.current = { token, userId };

  useEffect(() => {
    if (authLoading || !isAuthenticated || !token || userId === null) {
      setIsReminderOpen(false);
      setReminderStream(null);
      return;
    }

    let alive = true;
    const identitySnapshot = { token, userId };
    setIsReminderOpen(false);
    setReminderStream(null);

    const isCurrentIdentity = () => {
      const latest = identityRef.current;
      return latest.token === identitySnapshot.token && latest.userId === identitySnapshot.userId;
    };

    notificationApi
      .getPendingLiveReminder()
      .then((result) => {
        if (!alive || !isCurrentIdentity() || !result.hasReminder || !result.stream) {
          return;
        }
        setReminderStream(result.stream);
        setIsReminderOpen(true);
        trackEvent('live_reminder_exposed', {
          source: 'mobile_shell',
          entry: 'live_reminder_modal',
          type: 'live_start',
          result: 'success',
        });
      })
      .catch(() => {
        // 不在后端不可用时继续消费历史 mock 弹窗标记。
        localStorage.removeItem('pending_live_reminder');
      });

    return () => {
      alive = false;
    };
  }, [authLoading, isAuthenticated, token, userId, pathname]);

  return (
    <div className={styles.shell} data-testid="mobile-shell">
      <div className={styles.viewport}>
        <div className={`${styles.content} ${isLiveRoute ? styles.contentLive : ''}`}>{children}</div>
        <BottomNav />
        <LiveReminderModal
          isOpen={isReminderOpen}
          onClose={() => setIsReminderOpen(false)}
          stream={reminderStream}
        />
      </div>
    </div>
  );
}

export default MobileContainer;
