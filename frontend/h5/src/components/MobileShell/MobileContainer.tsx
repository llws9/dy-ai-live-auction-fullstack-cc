import { ReactNode, useEffect, useState } from 'react';
import { useLocation } from 'react-router-dom';
import LiveReminderModal, { StreamInfo } from '../LiveReminderModal';
import ThemeToggle from '../ThemeToggle';
import BottomNav from './BottomNav';
import styles from './MobileShell.module.css';

interface MobileContainerProps {
  children: ReactNode;
}

const mockLiveReminderStream: StreamInfo = {
  id: 'mock-live-reminder',
  name: '云端珍藏直播间',
  avatarUrl: 'data:image/svg+xml,%3Csvg xmlns=%22http://www.w3.org/2000/svg%22 width=%22120%22 height=%22120%22 viewBox=%220 0 120 120%22%3E%3Crect width=%22120%22 height=%22120%22 rx=%2232%22 fill=%22%2327272a%22/%3E%3Ccircle cx=%2260%22 cy=%2252%22 r=%2222%22 fill=%22%23d4af37%22/%3E%3Cpath d=%22M28 104c5-20 18-30 32-30s27 10 32 30%22 fill=%22%23f5f0e8%22/%3E%3C/svg%3E',
  statusText: '正在直播',
};

function MobileContainer({ children }: MobileContainerProps) {
  const [isReminderOpen, setIsReminderOpen] = useState(false);
  const { pathname } = useLocation();

  useEffect(() => {
    if (localStorage.getItem('pending_live_reminder') !== '1') {
      return;
    }

    localStorage.removeItem('pending_live_reminder');
    setIsReminderOpen(true);
  }, []);

  return (
    <div className={styles.shell} data-testid="mobile-shell">
      <div className={styles.viewport}>
        {pathname !== '/login' && (
          <div className={styles.themeToggleSlot}>
            <ThemeToggle />
          </div>
        )}
        <div className={styles.content}>{children}</div>
        <BottomNav />
        <LiveReminderModal
          isOpen={isReminderOpen}
          onClose={() => setIsReminderOpen(false)}
          stream={mockLiveReminderStream}
        />
      </div>
    </div>
  );
}

export default MobileContainer;
