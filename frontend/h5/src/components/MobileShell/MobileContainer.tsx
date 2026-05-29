import { ReactNode } from 'react';
import BottomNav from './BottomNav';
import styles from './MobileShell.module.css';

interface MobileContainerProps {
  children: ReactNode;
}

function MobileContainer({ children }: MobileContainerProps) {
  return (
    <div className={styles.shell} data-testid="mobile-shell">
      <div className={styles.viewport}>
        <div className={styles.content}>{children}</div>
        <BottomNav />
      </div>
    </div>
  );
}

export default MobileContainer;
