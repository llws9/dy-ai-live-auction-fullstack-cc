import { Link, useLocation } from 'react-router-dom';
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
  { path: '/profile', label: '我的', icon: '○' },
];

function isHiddenPath(pathname: string) {
  return hiddenNavPaths.has(pathname);
}

function isActivePath(pathname: string, itemPath: string) {
  return itemPath === '/' ? pathname === '/' : pathname.startsWith(itemPath);
}

function BottomNav() {
  const { pathname } = useLocation();

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
          >
            <span className={styles.navIcon} aria-hidden="true">
              {item.icon}
            </span>
            <span>{item.label}</span>
          </Link>
        );
      })}
    </nav>
  );
}

export default BottomNav;
