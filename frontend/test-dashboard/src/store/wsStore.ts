import { create } from 'zustand';

// WS 推送的进度消息（与后端 ws.Message 对应）
export interface ProgressMsg {
  test_id: string;
  progress: number;
  step: string;
  metrics?: Record<string, unknown>;
  ts: number;
}

interface WSStoreState {
  connected: boolean;
  testID: string | null;
  progress: number;
  step: string;
  metrics: Record<string, unknown>;
  history: ProgressMsg[]; // 本次连接收到的全部消息
  socket: WebSocket | null;

  connect: (wsURL: string, testID: string) => void;
  disconnect: () => void;
}

export const useWSStore = create<WSStoreState>((set, get) => ({
  connected: false,
  testID: null,
  progress: 0,
  step: '',
  metrics: {},
  history: [],
  socket: null,

  connect: (wsURL, testID) => {
    // 先关旧连接
    get().disconnect();

    console.info('[ws] connecting', { wsURL, testID });
    const ws = new WebSocket(wsURL);
    set({
      socket: ws,
      testID,
      progress: 0,
      step: '',
      metrics: {},
      history: [],
      connected: false,
    });

    ws.onopen = () => {
      console.info('[ws] open', { testID });
      set({ connected: true });
    };
    ws.onclose = (e) => {
      console.info('[ws] close', { testID, code: e.code, reason: e.reason });
      set({ connected: false });
    };
    ws.onerror = (e) => {
      console.error('[ws] error', e);
      set({ connected: false });
    };
    ws.onmessage = (e) => {
      try {
        const msg = JSON.parse(e.data) as ProgressMsg;
        console.debug('[ws] message', msg);
        set((s) => ({
          progress: msg.progress,
          step: msg.step,
          metrics: msg.metrics || {},
          history: [...s.history, msg],
        }));
      } catch (err) {
        console.warn('[ws] non-json message', e.data, err);
      }
    };
  },

  disconnect: () => {
    const s = get().socket;
    if (s) {
      console.info('[ws] disconnect requested');
      try { s.close(); } catch { /* noop */ }
    }
    set({ socket: null, connected: false });
  },
}));
