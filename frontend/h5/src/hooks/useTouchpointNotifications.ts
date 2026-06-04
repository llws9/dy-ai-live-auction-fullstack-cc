import { useEffect, useRef, useState } from 'react';
import { notificationApi, TouchpointSummary } from '../services/notification';
import { useAuth } from '../store/authContext';
import { TOUCHPOINT_SUMMARY_INVALIDATED_EVENT } from '../utils/touchpointSummaryEvents';

export interface TouchpointNotifications {
  pendingPayment: number;
  unreadTotal: number;
  summaryLoaded: boolean;
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
  const [summaryLoaded, setSummaryLoaded] = useState(false);
  const [refreshSignal, setRefreshSignal] = useState(0);
  const { isAuthenticated, loading: authLoading, token, user } = useAuth();
  const userId = user?.id ?? null;
  const identityRef = useRef({ token, userId });

  identityRef.current = { token, userId };

  useEffect(() => {
    const refresh = () => setRefreshSignal((value) => value + 1);

    window.addEventListener(TOUCHPOINT_SUMMARY_INVALIDATED_EVENT, refresh);
    return () => {
      window.removeEventListener(TOUCHPOINT_SUMMARY_INVALIDATED_EVENT, refresh);
    };
  }, []);

  useEffect(() => {
    if (authLoading || !isAuthenticated || !token || userId === null) {
      setSummary(EMPTY);
      setSummaryLoaded(false);
      return;
    }

    let alive = true;
    const identitySnapshot = { token, userId };
    setSummary(EMPTY);
    setSummaryLoaded(false);

    const isCurrentIdentity = () => {
      const latest = identityRef.current;
      return latest.token === identitySnapshot.token && latest.userId === identitySnapshot.userId;
    };

    notificationApi
      .getTouchpointSummary()
      .then((next) => {
        if (alive && isCurrentIdentity()) {
          setSummary(next);
          setSummaryLoaded(true);
        }
      })
      .catch(() => {
        if (alive && isCurrentIdentity()) {
          setSummary(EMPTY);
          setSummaryLoaded(false);
        }
      });

    return () => {
      alive = false;
    };
  }, [authLoading, isAuthenticated, token, userId, refreshSignal]);

  return {
    pendingPayment: summary.pendingPayment,
    unreadTotal: summary.unreadTotal,
    summaryLoaded,
  };
}
