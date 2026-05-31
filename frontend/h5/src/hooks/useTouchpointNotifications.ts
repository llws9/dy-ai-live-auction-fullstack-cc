import { useEffect, useState } from 'react';
import { notificationApi, TouchpointSummary } from '../services/notification';
import { useAuth } from '../store/authContext';

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
  const { isAuthenticated, loading: authLoading } = useAuth();

  useEffect(() => {
    if (authLoading || !isAuthenticated) {
      setSummary(EMPTY);
      return;
    }

    let alive = true;

    notificationApi
      .getTouchpointSummary()
      .then((next) => {
        if (alive) {
          setSummary(next);
        }
      })
      .catch(() => {
        if (alive) {
          setSummary(EMPTY);
        }
      });

    return () => {
      alive = false;
    };
  }, [authLoading, isAuthenticated]);

  return {
    pendingPayment: summary.pendingPayment,
    unreadTotal: summary.unreadTotal,
  };
}
