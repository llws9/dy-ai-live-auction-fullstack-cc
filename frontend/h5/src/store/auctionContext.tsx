// store/auctionContext.tsx

import React, { createContext, useContext, useReducer, ReactNode } from 'react';

// 状态类型
interface AuctionState {
  auctionId: number;
  currentPrice: number;
  winnerId: number | null;
  endTime: number;
  status: number;
  ranking: RankItem[];
  connected: boolean;
  delayUsed: number;
}

interface RankItem {
  rank: number;
  userId: number;
  userName?: string;
  amount: number;
}

// Action 类型
type AuctionAction =
  | { type: 'SET_AUCTION'; payload: Partial<AuctionState> }
  | { type: 'UPDATE_PRICE'; payload: { price: number; winnerId: number } }
  | { type: 'UPDATE_RANKING'; payload: RankItem[] }
  | { type: 'TRIGGER_DELAY'; payload: { newEndTime: number; delayUsed: number } }
  | { type: 'END_AUCTION'; payload: { winnerId: number; finalPrice: number } }
  | { type: 'SET_CONNECTED'; payload: boolean };

// 初始状态
const initialState: AuctionState = {
  auctionId: 0,
  currentPrice: 0,
  winnerId: null,
  endTime: 0,
  status: 0,
  ranking: [],
  connected: false,
  delayUsed: 0,
};

// Reducer
function auctionReducer(state: AuctionState, action: AuctionAction): AuctionState {
  switch (action.type) {
    case 'SET_AUCTION':
      return { ...state, ...action.payload };

    case 'UPDATE_PRICE':
      return {
        ...state,
        currentPrice: action.payload.price,
        winnerId: action.payload.winnerId,
      };

    case 'UPDATE_RANKING':
      return { ...state, ranking: action.payload };

    case 'TRIGGER_DELAY':
      return {
        ...state,
        endTime: action.payload.newEndTime,
        delayUsed: action.payload.delayUsed,
      };

    case 'END_AUCTION':
      return {
        ...state,
        status: 3, // ended
        winnerId: action.payload.winnerId,
        currentPrice: action.payload.finalPrice,
      };

    case 'SET_CONNECTED':
      return { ...state, connected: action.payload };

    default:
      return state;
  }
}

// Context
interface AuctionContextType {
  state: AuctionState;
  dispatch: React.Dispatch<AuctionAction>;
}

const AuctionContext = createContext<AuctionContextType | undefined>(undefined);

// Provider
interface AuctionProviderProps {
  children: ReactNode;
  initialAuctionId?: number;
}

export function AuctionProvider({ children, initialAuctionId = 0 }: AuctionProviderProps) {
  const [state, dispatch] = useReducer(auctionReducer, {
    ...initialState,
    auctionId: initialAuctionId,
  });

  return (
    <AuctionContext.Provider value={{ state, dispatch }}>
      {children}
    </AuctionContext.Provider>
  );
}

// Hook
export function useAuctionState() {
  const context = useContext(AuctionContext);
  if (context === undefined) {
    throw new Error('useAuctionState must be used within an AuctionProvider');
  }
  return context;
}
