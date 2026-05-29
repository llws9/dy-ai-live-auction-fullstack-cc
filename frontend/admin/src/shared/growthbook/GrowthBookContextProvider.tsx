import { GrowthBook, GrowthBookProvider as GBProvider } from '@growthbook/growthbook-react';
import React, { useEffect, useMemo } from 'react';
import { useAuth } from '../auth';

interface GrowthBookContextProviderProps {
  children: React.ReactNode;
}

export function GrowthBookContextProvider({ children }: GrowthBookContextProviderProps) {
  const { user } = useAuth();

  // 使用 useMemo 进行懒初始化，符合 React Hooks 规则
  const gb = useMemo(() => new GrowthBook({
    apiHost: import.meta.env.VITE_GROWTHBOOK_API_HOST || 'http://localhost:3200',
    clientKey: import.meta.env.VITE_GROWTHBOOK_CLIENT_KEY || 'dev-client-key',
    enableDevMode: import.meta.env.DEV,
    trackingCallback: (experiment, result) => {
      // 可选：发送实验数据到后端
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
      });
    } else {
      gb.setAttributes({
        id: 'anonymous',
      });
    }
  }, [user, gb]);

  // 初始加载特性配置并清理setInterval
  useEffect(() => {
    gb.loadFeatures();

    // 自动刷新特性配置，组件卸载时清理
    const intervalId = setInterval(() => {
      gb.refreshFeatures();
    }, 60000); // 每分钟刷新

    return () => {
      clearInterval(intervalId);
    };
  }, [gb]);

  return (
    <GBProvider growthbook={gb}>
      {children}
    </GBProvider>
  );
}

export default GrowthBookContextProvider;