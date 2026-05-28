import { GrowthBook, GrowthBookProvider as GBProvider } from '@growthbook/growthbook-react';
import React from 'react';
import { useAuth } from '../auth';

// GrowthBook 配置
const gb = new GrowthBook({
  apiHost: import.meta.env.VITE_GROWTHBOOK_API_HOST || 'http://localhost:3200',
  clientKey: import.meta.env.VITE_GROWTHBOOK_CLIENT_KEY || 'dev-client-key',
  enableDevMode: import.meta.env.DEV,
  trackingCallback: (experiment, result) => {
    // 可选：发送实验数据到后端
    console.log(`Experiment ${experiment.key} assigned variation ${result.variationId}`);
  },
});

// 自动刷新特性配置
setInterval(() => {
  gb.refreshFeatures();
}, 60000); // 每分钟刷新

interface GrowthBookContextProviderProps {
  children: React.ReactNode;
}

export function GrowthBookContextProvider({ children }: GrowthBookContextProviderProps) {
  const { user } = useAuth();

  // 更新用户属性
  React.useEffect(() => {
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
  }, [user]);

  // 初始加载特性配置
  React.useEffect(() => {
    gb.loadFeatures();
  }, []);

  return (
    <GBProvider growthbook={gb}>
      {children}
    </GBProvider>
  );
}

export default GrowthBookContextProvider;