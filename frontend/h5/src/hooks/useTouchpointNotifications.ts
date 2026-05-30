export interface TouchpointNotifications {
  pendingPayment: number;
  unreadTotal: number;
}

export function useTouchpointNotifications(): TouchpointNotifications {
  return {
    pendingPayment: 1,
    unreadTotal: 3,
  };
}
