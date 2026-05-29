import { GrowthBook, GrowthBookProvider as GBProvider } from '@growthbook/growthbook-react';
import React, { createContext, useContext, useEffect, useMemo, useState } from 'react';
import { useAuth } from './authContext';

interface GrowthBookContextValue {
  growthbook: GrowthBook;
}

const GrowthBookContext = createContext<GrowthBookContextValue | null>(null);

export function useGrowthBook() {
  const context = useContext(GrowthBookContext);
  if (!context) {
    throw new Error('useGrowthBook must be used within GrowthBookContextProvider');
  }
  return context.growthbook;
}

interface GrowthBookContextProviderProps {
  children: React.ReactNode;
}

export function GrowthBookContextProvider({ children }: GrowthBookContextProviderProps) {
  const { user } = useAuth();
  const [loaded, setLoaded] = useState(false);

  // 使用 useMemo 进行懒初始化，符合 React Hooks 规则
  const gb = useMemo(() => new GrowthBook({
    apiHost: import.meta.env.VITE_GROWTHBOOK_API_HOST || 'http://localhost:3200',
    clientKey: import.meta.env.VITE_GROWTHBOOK_CLIENT_KEY || 'dev-client-key',
    enableDevMode: import.meta.env.DEV,
    trackingCallback: (experiment, result) => {
      console.log(`Experiment ${experiment.key} assigned variation ${result.variationId}`);
    },
  }), []);

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
  }, [user, gb]);

  // 初始加载特性配置并清理setInterval
  useEffect(() => {
    gb.loadFeatures().then(() => {
      setLoaded(true);
    }).catch((err) => {
      console.warn('Failed to load GrowthBook features:', err);
      setLoaded(true); // 即使失败也继续
    });

    // 自动刷新特性配置，组件卸载时清理
    const intervalId = setInterval(() => {
      gb.refreshFeatures();
    }, 60000);

    return () => {
      clearInterval(intervalId);
    };
  }, [gb]);

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