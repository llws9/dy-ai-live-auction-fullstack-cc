import { GrowthBook, GrowthBookProvider as GBProvider } from '@growthbook/growthbook-react';
import React, { createContext, useContext, useEffect, useState } from 'react';
import { useAuth } from './authContext';

// GrowthBook 配置
const gb = new GrowthBook({
  apiHost: import.meta.env.VITE_GROWTHBOOK_API_HOST || 'http://localhost:3200',
  clientKey: import.meta.env.VITE_GROWTHBOOK_CLIENT_KEY || 'dev-client-key',
  enableDevMode: import.meta.env.DEV,
  trackingCallback: (experiment, result) => {
    console.log(`Experiment ${experiment.key} assigned variation ${result.variationId}`);
  },
});

// 自动刷新特性配置
setInterval(() => {
  gb.refreshFeatures();
}, 60000);

interface GrowthBookContextValue {
  growthbook: GrowthBook;
}

const GrowthBookContext = createContext<GrowthBookContextValue>({ growthbook: gb });

export function useGrowthBook() {
  return useContext(GrowthBookContext);
}

interface GrowthBookContextProviderProps {
  children: React.ReactNode;
}

export function GrowthBookContextProvider({ children }: GrowthBookContextProviderProps) {
  const { user } = useAuth();
  const [loaded, setLoaded] = useState(false);

  // 更新用户属性
  useEffect(() => {
    if (user) {
      gb.setAttributes({
        id: user.id.toString(),
        role: user.role,
        email: user.email,
        deviceType: 'mobile',
      });
    } else {
      gb.setAttributes({
        id: 'anonymous',
        deviceType: 'mobile',
      });
    }
  }, [user]);

  // 初始加载特性配置
  useEffect(() => {
    gb.loadFeatures().then(() => {
      setLoaded(true);
    }).catch((err) => {
      console.warn('Failed to load GrowthBook features:', err);
      setLoaded(true); // 即使失败也继续
    });
  }, []);

  if (!loaded) {
    return null; // 或者显示 loading
  }

  return (
    <GrowthBookContext.Provider value={{ growthbook: gb }}>
      <GBProvider growthbook={gb}>
        {children}
      </GBProvider>
    </GrowthBookContext.Provider>
  );
}

export default GrowthBookContextProvider;