import { GrowthBook, GrowthBookProvider as GBProvider } from '@growthbook/growthbook-react';
import React, { createContext, useContext, useEffect, useMemo, useState } from 'react';
import { useAuth } from './authContext';
import { post } from '../services/api';

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

// 上报实验曝光到后端,用于服务端 Prometheus 指标统计与离线分析。
// 与 console 日志相比,这是 SSOT 的实验埋点出口。
function reportExperimentViewed(experimentKey: string, variation: string) {
  void post('/experiments/viewed', { experiment: experimentKey, variation }, { showError: false }).catch(
    (err) => {
      // 不阻塞前端体验,失败仅在控制台告警
      console.warn('[experiment] viewed report failed:', err?.message || err);
    },
  );
}

export function GrowthBookContextProvider({ children }: GrowthBookContextProviderProps) {
  const { user, loading: authLoading } = useAuth();
  const [loaded, setLoaded] = useState(false);

  // 使用 useMemo 进行懒初始化，符合 React Hooks 规则
  const gb = useMemo(
    () =>
      new GrowthBook({
        apiHost: import.meta.env.VITE_GROWTHBOOK_API_HOST || 'http://localhost:3200',
        clientKey: import.meta.env.VITE_GROWTHBOOK_CLIENT_KEY || 'dev-client-key',
        enableDevMode: import.meta.env.DEV,
        trackingCallback: (experiment, result) => {
          reportExperimentViewed(experiment.key, String(result.variationId));
        },
      }),
    [],
  );

  // 初始加载特性配置（必须在 attributes 就绪后才发起，避免首次分桶用空属性）
  useEffect(() => {
    // 等待 auth 初始化完成（避免 token 已存在但 user 尚未注水时就分桶）
    if (authLoading) return;

    // 先注入 attributes，再加载/刷新特性
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

    if (!loaded) {
      gb.loadFeatures()
        .catch((err) => {
          console.warn('Failed to load GrowthBook features:', err);
        })
        .finally(() => {
          setLoaded(true);
        });
    } else {
      // 已加载过：用户态变化时只刷新一次特性
      gb.refreshFeatures().catch(() => {
        // 静默失败：旧 features 仍可用
      });
    }
  }, [authLoading, user, gb, loaded]);

  // 自动刷新特性配置，组件卸载时清理
  useEffect(() => {
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
      <GBProvider growthbook={gb}>{children}</GBProvider>
    </GrowthBookContext.Provider>
  );
}

export default GrowthBookContextProvider;