import { createContext, ReactNode, useContext, useState } from 'react';

interface DemoContextType {
  currentAuctionId: number | null;
  setCurrentAuctionId: (auctionId: number | null) => void;
}

const DemoContext = createContext<DemoContextType | undefined>(undefined);

interface DemoProviderProps {
  children: ReactNode;
}

export function DemoProvider({ children }: DemoProviderProps) {
  const [currentAuctionId, setCurrentAuctionId] = useState<number | null>(null);

  const value: DemoContextType = {
    currentAuctionId,
    setCurrentAuctionId,
  };

  return <DemoContext.Provider value={value}>{children}</DemoContext.Provider>;
}

export function useDemo(): DemoContextType {
  const context = useContext(DemoContext);
  if (context === undefined) {
    throw new Error('useDemo must be used within a DemoProvider');
  }
  return context;
}
