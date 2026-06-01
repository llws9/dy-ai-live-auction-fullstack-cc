import { Link, useLocation } from 'react-router-dom';
import BadgeDot from '../BadgeDot';
import { useTouchpointNotifications } from '../../hooks/useTouchpointNotifications';
import { trackEvent } from '../../utils/trackEvent';
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
  const { unreadTotal } = useTouchpointNotifications();

  if (isHiddenPath(pathname)) {
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
            aria-current={isActive ? 'page' : undefined}
            onClick={() => trackNavClick(item.path)}
          >
            <span className={styles.navIconWrap}>
              <span className={styles.navIcon} aria-hidden="true">
                {item.icon}
              </span>
              {item.badge && <BadgeDot count={unreadTotal} />}
            </span>
            <span>{item.label}</span>
          </Link>
        );
      })}
    </nav>
  );
}

export default BottomNav;
