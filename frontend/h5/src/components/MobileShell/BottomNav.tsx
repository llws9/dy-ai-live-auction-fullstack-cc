import { useEffect } from 'react';
import { Link, useLocation } from 'react-router-dom';
import BadgeDot from '../BadgeDot';
import { useTouchpointNotifications } from '../../hooks/useTouchpointNotifications';
import { getCountBucket, trackEvent } from '../../utils/trackEvent';
import styles from './MobileShell.module.css';

const hiddenNavPaths = new Set([
  '/detail',
  '/result',
  '/notifications',
  '/following',
  '/history',
  '/login',
]);

const navItems = [
  { path: '/', label: '首页', icon: '⌂' },
  { path: '/live', label: '直播间', icon: '▶' },
  { path: '/profile', label: '我的', icon: '○', badge: true },
];

function isHiddenPath(pathname: string) {
  return hiddenNavPaths.has(pathname);
}

function isActivePath(pathname: string, itemPath: string) {
  return itemPath === '/' ? pathname === '/' : pathname.startsWith(itemPath);
}

function trackNavClick(path: string) {
  if (path === '/profile') {
    trackEvent('entry_clicked', {
      source: 'bottom_nav',
      entry: 'profile_tab',
      type: 'all',
      result: 'clicked',
    });
  }
}

function BottomNav() {
  const { pathname } = useLocation();
  const { summaryLoaded, unreadTotal, wonNotPaid } = useTouchpointNotifications();
  const profileUnreadTotal = unreadTotal + wonNotPaid;
  const hidden = isHiddenPath(pathname);

  useEffect(() => {
    if (hidden || !summaryLoaded) return;

    trackEvent('summary_exposed', {
      source: 'bottom_nav',
      entry: 'profile_tab',
      type: 'all',
      result: 'success',
      countBucket: getCountBucket(profileUnreadTotal),
    });
  }, [hidden, summaryLoaded, profileUnreadTotal]);

  if (hidden) {
    return null;
  }

  return (
    <nav className={styles.bottomNav} aria-label="底部导航">
      {navItems.map((item) => {
        const isActive = isActivePath(pathname, item.path);

        return (
          <Link
            key={item.path}
            to={item.path}
            className={`${styles.navItem} ${isActive ? styles.navItemActive : ''}`}
            data-state={isActive ? 'active' : 'inactive'}
            aria-current={isActive ? 'page' : undefined}
            onClick={() => trackNavClick(item.path)}
          >
            <span className={styles.navIconWrap}>
              <span className={styles.navIcon} aria-hidden="true">
                {item.icon}
              </span>
              {item.badge && <BadgeDot count={profileUnreadTotal} />}
            </span>
            <span className={styles.navLabel}>{item.label}</span>
          </Link>
        );
      })}
    </nav>
  );
}

export default BottomNav;
