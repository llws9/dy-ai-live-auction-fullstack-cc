import { ReactNode, useEffect, useState } from 'react';
import { notificationApi } from '../../services/notification';
import { useAuth } from '../../store/authContext';
import LiveReminderModal, { StreamInfo } from '../LiveReminderModal';
import BottomNav from './BottomNav';
import styles from './MobileShell.module.css';

interface MobileContainerProps {
  children: ReactNode;
}

function MobileContainer({ children }: MobileContainerProps) {
  const [isReminderOpen, setIsReminderOpen] = useState(false);
  const [reminderStream, setReminderStream] = useState<StreamInfo | null>(null);
  const { isAuthenticated, loading: authLoading } = useAuth();

  useEffect(() => {
    if (authLoading || !isAuthenticated) {
      return;
    }

    let alive = true;

    notificationApi
      .getPendingLiveReminder()
      .then((result) => {
        if (!alive || !result.hasReminder || !result.stream) {
          return;
        }
        setReminderStream(result.stream);
        setIsReminderOpen(true);
      })
      .catch(() => {
        // 不在后端不可用时继续消费历史 mock 弹窗标记。
        localStorage.removeItem('pending_live_reminder');
      });

    return () => {
      alive = false;
    };
  }, [authLoading, isAuthenticated]);

  return (
    <div className={styles.shell} data-testid="mobile-shell">
      <div className={styles.viewport}>
        <div className={styles.content}>{children}</div>
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
