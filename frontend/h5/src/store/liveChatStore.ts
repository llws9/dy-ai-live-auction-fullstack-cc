import { create } from 'zustand';

export interface ChatMessage {
  live_stream_id: number;
  user_id: number;
  user_name: string;
  avatar_url?: string;
  text: string;
  sent_at: number;
  client_msg_id?: string;
}

const MAX_HISTORY = 200;
const COOLDOWN_MS = 1000;

interface LiveChatState {
  history: ChatMessage[];
  lastSentAt: number;

  receive: (msg: ChatMessage) => void;
  markSent: () => void;
  isCoolingDown: () => boolean;
  reset: () => void;
}

export const useLiveChatStore = create<LiveChatState>((set, get) => ({
  history: [],
  lastSentAt: 0,

  receive: (msg) =>
    set((s) => ({
      history: [...s.history, msg].slice(-MAX_HISTORY),
    })),

  markSent: () => set({ lastSentAt: Date.now() }),

  isCoolingDown: () => Date.now() - get().lastSentAt < COOLDOWN_MS,

  reset: () => set({ history: [], lastSentAt: 0 }),
}));
