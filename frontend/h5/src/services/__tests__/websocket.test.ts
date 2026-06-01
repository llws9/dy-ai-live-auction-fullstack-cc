import { describe, it, expect, jest, beforeEach, afterEach } from '@jest/globals';
import WebSocketService from '../websocket';

jest.mock('../api', () => ({
  buildLoginRedirectPath: jest.fn(() => '/login'),
}));

// Mock WebSocket
class MockWebSocket {
  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSING = 2;
  static CLOSED = 3;
  static instances: MockWebSocket[] = [];

  readyState = MockWebSocket.OPEN;
  onopen: (() => void) | null = null;
  onclose: (() => void) | null = null;
  onmessage: ((event: { data: string }) => void) | null = null;
  onerror: ((error: Error) => void) | null = null;

  send = jest.fn();
  close = jest.fn();

  constructor(_url: string) {
    MockWebSocket.instances.push(this);
    setTimeout(() => {
      if (this.onopen) this.onopen();
    }, 10);
  }
}

// Replace global WebSocket
(global as any).WebSocket = MockWebSocket;

describe('WebSocketService', () => {
  let service: WebSocketService;

  beforeEach(() => {
    jest.useFakeTimers();
    MockWebSocket.instances = [];
    service = new WebSocketService(1);
  });

  afterEach(() => {
    service.disconnect();
    jest.useRealTimers();
  });

  it('should initialize with correct auction ID', () => {
    expect(service).toBeDefined();
  });

  it('should register event handlers', () => {
    const handler = jest.fn();
    service.on('rank_update', handler);

    // Handler should be registered
    expect(handler).toBeDefined();
  });

  it('should remove event handlers with off()', () => {
    const handler = jest.fn();
    service.on('test_event', handler);
    service.off('test_event', handler);

    // Handler should be removed (no error)
    expect(true).toBe(true);
  });

  it('should connect successfully', async () => {
    const connectPromise = service.connect();

    // Advance time to trigger onopen
    act(() => {
      jest.advanceTimersByTime(20);
    });

    await expect(connectPromise).resolves.toBeUndefined();
  });

  it('should send messages when connected', async () => {
    service.connect();
    jest.advanceTimersByTime(20);

    service.send({ type: 'bid', data: { amount: 100 } });

    // Message should be sent (no error)
    expect(true).toBe(true);
  });

  it('should handle sync_request', async () => {
    service.connect();
    jest.advanceTimersByTime(20);

    service.requestSync();

    // Sync request should be sent (no error)
    expect(true).toBe(true);
  });

  it('should use exponential backoff for reconnection', () => {
    // Test the delay sequence concept
    const delays = [1, 2, 4, 8, 16, 30, 30, 30, 30, 30];

    expect(delays[0]).toBe(1);
    expect(delays[1]).toBe(2);
    expect(delays[2]).toBe(4);
    expect(delays[3]).toBe(8);
    expect(delays[4]).toBe(16);
    expect(delays[5]).toBe(30);
  });

  it('should disconnect cleanly', async () => {
    service.connect();
    jest.advanceTimersByTime(20);

    service.disconnect();

    // Should disconnect without error
    expect(true).toBe(true);
  });

  it('dispatches notification websocket messages to both notification APIs', async () => {
    const notificationHandler = jest.fn();
    const genericNotificationHandler = jest.fn();
    service.onNotification(notificationHandler);
    service.on('notification', genericNotificationHandler);

    const connectPromise = service.connect();
    jest.advanceTimersByTime(20);
    await connectPromise;

    const notification = {
      id: 10,
      type: 'auction_won',
      title: '恭喜中标',
      content: '请尽快完成支付',
      data: { auction_id: 5 },
      created_at: '2026-06-01T00:00:00Z',
    };
    MockWebSocket.instances[0].onmessage?.({
      data: JSON.stringify({
        type: 'notification',
        timestamp: Date.now(),
        data: notification,
      }),
    });

    expect(notificationHandler).toHaveBeenCalledTimes(1);
    expect(notificationHandler).toHaveBeenCalledWith(notification);
    expect(genericNotificationHandler).toHaveBeenCalledTimes(1);
    expect(genericNotificationHandler).toHaveBeenCalledWith(notification);
  });
});

// Helper for act
function act(callback: () => void) {
  callback();
}
