import type { CSSProperties, SVGProps } from 'react';
import { useCallback, useEffect, useLayoutEffect, useMemo, useRef } from 'react';
import { Link, useLocation } from 'react-router-dom';
import BadgeDot from '../BadgeDot';
import { useTouchpointNotifications } from '../../hooks/useTouchpointNotifications';
import { getCountBucket, trackEvent } from '../../utils/trackEvent';
import styles from './MobileShell.module.css';

type NavIconName = 'home' | 'live' | 'profile';

const hiddenNavPaths = new Set([
  '/detail',
  '/result',
  '/notifications',
  '/following',
  '/history',
  '/login',
]);

const navItems = [
  { path: '/', label: '首页', icon: 'home' as const },
  { path: '/live', label: '直播间', icon: 'live' as const },
  { path: '/profile', label: '我的', icon: 'profile' as const, badge: true },
];

const NAV_INDICATOR_WIDTH = 72;

const navIndicatorStyle = {
  '--nav-indicator-width': `${NAV_INDICATOR_WIDTH}px`,
  '--nav-indicator-x': '0px',
} as CSSProperties;

const iconProps: SVGProps<SVGSVGElement> = {
  viewBox: '0 0 24 24',
  fill: 'none',
  stroke: 'currentColor',
  strokeWidth: 1.85,
  strokeLinecap: 'round',
  strokeLinejoin: 'round',
  'aria-hidden': true,
};

function NavIcon({ name }: { name: NavIconName }) {
  if (name === 'home') {
    return (
      <svg {...iconProps}>
        <path d="M4 11.2 12 4l8 7.2" />
        <path d="M6.5 10.5v8h11v-8" />
        <path d="M10 18v-4h4v4" />
      </svg>
    );
  }

  if (name === 'live') {
    return (
      <svg {...iconProps}>
        <path d="M7 6.5v11l10-5.5-10-5.5Z" />
        <path d="M18 5v14" />
      </svg>
    );
  }

  return (
    <svg {...iconProps}>
      <path d="M12 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8Z" />
      <path d="M4.5 20a7.5 7.5 0 0 1 15 0" />
    </svg>
  );
}

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
  const navRef = useRef<HTMLElement | null>(null);
  const navItemRefs = useRef<Array<HTMLAnchorElement | null>>([]);
  const profileUnreadTotal = unreadTotal + wonNotPaid;
  const hidden = isHiddenPath(pathname);
  const activeIndex = useMemo(
    () => navItems.findIndex((item) => isActivePath(pathname, item.path)),
    [pathname],
  );
  const safeActiveIndex = activeIndex >= 0 ? activeIndex : 0;

  const placeIndicator = useCallback((animate: boolean) => {
    const nav = navRef.current;
    const activeItem = navItemRefs.current[safeActiveIndex];
    if (!nav || !activeItem) return;

    const navRect = nav.getBoundingClientRect();
    const itemRect = activeItem.getBoundingClientRect();
    const x = itemRect.left - navRect.left + (itemRect.width - NAV_INDICATOR_WIDTH) / 2;

    if (!animate) {
      nav.dataset.measuring = 'true';
    }

    nav.style.setProperty('--nav-indicator-width', `${NAV_INDICATOR_WIDTH}px`);
    nav.style.setProperty('--nav-indicator-x', `${Math.round(x)}px`);

    if (!animate) {
      requestAnimationFrame(() => {
        if (navRef.current) {
          delete navRef.current.dataset.measuring;
        }
      });
    }
  }, [safeActiveIndex]);

  useLayoutEffect(() => {
    if (hidden) return;
    placeIndicator(false);
  }, [hidden, placeIndicator]);

  useEffect(() => {
    if (hidden) return;

    let cancelled = false;
    const remeasure = () => {
      if (!cancelled) {
        placeIndicator(false);
      }
    };
    const fonts = 'fonts' in document
      ? (document as Document & { fonts?: { ready?: Promise<unknown> } }).fonts
      : undefined;

    fonts?.ready?.then(remeasure).catch(() => undefined);
    window.addEventListener('resize', remeasure);
    return () => {
      cancelled = true;
      window.removeEventListener('resize', remeasure);
    };
  }, [hidden, placeIndicator]);

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
    <nav ref={navRef} className={styles.bottomNav} style={navIndicatorStyle} aria-label="底部导航">
      <span className={styles.navIndicator} data-testid="bottom-nav-indicator" aria-hidden="true" />
      <span className={styles.navIndicatorLine} data-testid="bottom-nav-indicator-line" aria-hidden="true" />
      {navItems.map((item, index) => {
        const isActive = index === safeActiveIndex;

        return (
          <Link
            key={item.path}
            ref={(element) => {
              navItemRefs.current[index] = element;
            }}
            to={item.path}
            className={`${styles.navItem} ${isActive ? styles.navItemActive : ''}`}
            data-state={isActive ? 'active' : 'inactive'}
            aria-current={isActive ? 'page' : undefined}
            onClick={() => trackNavClick(item.path)}
          >
            <span className={styles.navIconWrap}>
              <span className={styles.navIcon} aria-hidden="true">
                <NavIcon name={item.icon} />
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
