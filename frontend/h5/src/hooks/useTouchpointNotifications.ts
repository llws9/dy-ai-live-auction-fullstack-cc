import { useEffect, useState } from 'react';
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

interface TouchpointSummarySnapshot {
  identityKey: string | null;
  summary: TouchpointSummary;
  summaryLoaded: boolean;
}

const EMPTY_SNAPSHOT: TouchpointSummarySnapshot = {
  identityKey: null,
  summary: EMPTY,
  summaryLoaded: false,
};

let sharedSnapshot: TouchpointSummarySnapshot = EMPTY_SNAPSHOT;
let inFlight: Promise<void> | null = null;
let inFlightIdentityKey: string | null = null;
let requestVersion = 0;
const listeners = new Set<() => void>();

function getIdentityKey(token: string, userId: number | string) {
  return `${userId}:${token}`;
}

function getSharedSnapshot() {
  return sharedSnapshot;
}

function emitSharedSnapshot() {
  listeners.forEach((listener) => listener());
}

function setSharedSnapshot(next: TouchpointSummarySnapshot) {
  sharedSnapshot = next;
  emitSharedSnapshot();
}

function subscribeSharedSummary(listener: () => void) {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
    if (listeners.size === 0) {
      requestVersion += 1;
      inFlight = null;
      inFlightIdentityKey = null;
      sharedSnapshot = EMPTY_SNAPSHOT;
    }
  };
}

function clearSharedSummary() {
  requestVersion += 1;
  inFlight = null;
  inFlightIdentityKey = null;
  setSharedSnapshot(EMPTY_SNAPSHOT);
}

function loadSharedSummary(identityKey: string, force = false) {
  if (inFlight && inFlightIdentityKey === identityKey) {
    return;
  }

  if (!force && sharedSnapshot.identityKey === identityKey && sharedSnapshot.summaryLoaded) {
    return;
  }

  if (sharedSnapshot.identityKey !== identityKey || force) {
    setSharedSnapshot({
      identityKey,
      summary: EMPTY,
      summaryLoaded: false,
    });
  }

  const version = requestVersion + 1;
  requestVersion = version;
  inFlightIdentityKey = identityKey;
  inFlight = notificationApi
    .getTouchpointSummary()
    .then((next) => {
      if (requestVersion !== version || sharedSnapshot.identityKey !== identityKey) return;
      setSharedSnapshot({
        identityKey,
        summary: next,
        summaryLoaded: true,
      });
    })
    .catch(() => {
      if (requestVersion !== version || sharedSnapshot.identityKey !== identityKey) return;
      setSharedSnapshot({
        identityKey,
        summary: EMPTY,
        summaryLoaded: false,
      });
    })
    .finally(() => {
      if (requestVersion !== version || inFlightIdentityKey !== identityKey) return;
      inFlight = null;
      inFlightIdentityKey = null;
    });
}

export function useTouchpointNotifications(): TouchpointNotifications {
  const [snapshot, setSnapshot] = useState(getSharedSnapshot);
  const [refreshSignal, setRefreshSignal] = useState(0);
  const { isAuthenticated, loading: authLoading, token, user } = useAuth();
  const userId = user?.id ?? null;

  useEffect(() => {
    const refresh = () => setRefreshSignal((value) => value + 1);

    window.addEventListener(TOUCHPOINT_SUMMARY_INVALIDATED_EVENT, refresh);
    return () => {
      window.removeEventListener(TOUCHPOINT_SUMMARY_INVALIDATED_EVENT, refresh);
    };
  }, []);

  useEffect(() => {
    return subscribeSharedSummary(() => setSnapshot(getSharedSnapshot()));
  }, []);

  useEffect(() => {
    if (authLoading || !isAuthenticated || !token || userId === null) {
      clearSharedSummary();
      return;
    }

    loadSharedSummary(getIdentityKey(token, userId), refreshSignal > 0);
  }, [authLoading, isAuthenticated, token, userId, refreshSignal]);

  const identityKey =
    !authLoading && isAuthenticated && token && userId !== null ? getIdentityKey(token, userId) : null;
  const summary = snapshot.identityKey === identityKey ? snapshot.summary : EMPTY;
  const summaryLoaded = snapshot.identityKey === identityKey ? snapshot.summaryLoaded : false;

  return {
    pendingPayment: summary.pendingPayment,
    unreadTotal: summary.unreadTotal,
    summaryLoaded,
  };
}
