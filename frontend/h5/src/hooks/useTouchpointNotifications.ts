import { useEffect, useRef, useState } from 'react';
import { notificationApi, TouchpointSummary } from '../services/notification';
import { useAuth } from '../store/authContext';
import { getCountBucket, trackEvent } from '../utils/trackEvent';

export interface TouchpointNotifications {
  pendingPayment: number;
  unreadTotal: number;
}

const EMPTY: TouchpointSummary = {
  unreadTotal: 0,
  pendingPayment: 0,
  wonNotPaid: 0,
  outbid: 0,
  endingSoon: 0,
};

export function useTouchpointNotifications(): TouchpointNotifications {
  const [summary, setSummary] = useState<TouchpointSummary>(EMPTY);
  const { isAuthenticated, loading: authLoading, token, user } = useAuth();
  const userId = user?.id ?? null;
  const identityRef = useRef({ token, userId });

  identityRef.current = { token, userId };

  useEffect(() => {
    if (authLoading || !isAuthenticated || !token || userId === null) {
      setSummary(EMPTY);
      return;
    }

    let alive = true;
    const identitySnapshot = { token, userId };
    setSummary(EMPTY);

    const isCurrentIdentity = () => {
      const latest = identityRef.current;
      return latest.token === identitySnapshot.token && latest.userId === identitySnapshot.userId;
    };

    notificationApi
      .getTouchpointSummary()
      .then((next) => {
        if (alive && isCurrentIdentity()) {
          setSummary(next);
          trackEvent('summary_exposed', {
            source: 'bottom_nav',
            entry: 'profile_tab',
            type: 'all',
            result: 'success',
            countBucket: getCountBucket(next.unreadTotal ?? 0),
          });
        }
      })
      .catch(() => {
        if (alive && isCurrentIdentity()) {
          setSummary(EMPTY);
        }
      });

    return () => {
      alive = false;
    };
  }, [authLoading, isAuthenticated, token, userId]);

  return {
    pendingPayment: summary.pendingPayment,
    unreadTotal: summary.unreadTotal,
  };
}
