import { GrowthBook, GrowthBookProvider as GBProvider } from '@growthbook/growthbook-react';
import React, { useEffect, useRef } from 'react';
import { useAuth } from '../auth';

interface GrowthBookContextProviderProps {
  children: React.ReactNode;
}

export function GrowthBookContextProvider({ children }: GrowthBookContextProviderProps) {
  const { user } = useAuth();

  // 组件级实例，避免模块级单例的属性泄漏问题
  const gbRef = useRef<GrowthBook | null>(null);

  if (!gbRef.current) {
    gbRef.current = new GrowthBook({
      apiHost: import.meta.env.VITE_GROWTHBOOK_API_HOST || 'http://localhost:3200',
      clientKey: import.meta.env.VITE_GROWTHBOOK_CLIENT_KEY || 'dev-client-key',
      enableDevMode: import.meta.env.DEV,
      trackingCallback: (experiment, result) => {
        // 可选：发送实验数据到后端
        console.log(`Experiment ${experiment.key} assigned variation ${result.variationId}`);
      },
    });
  }

  const gb = gbRef.current;

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