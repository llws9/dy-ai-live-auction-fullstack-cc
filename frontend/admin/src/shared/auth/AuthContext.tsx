// 认证状态管理

import React, { createContext, useContext, useState, useEffect, useCallback } from 'react';
import { Navigate } from 'react-router-dom';
import { User, authApi } from '@/shared/api';

interface AuthContextType {
  user: User | null;
  token: string | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (token: string, user: User) => void;
  logout: () => void;
  refreshUser: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  // 初始化：从localStorage恢复认证状态
  useEffect(() => {
    const storedToken = localStorage.getItem('admin_auth_token');
    const storedUser = localStorage.getItem('admin_auth_user');

    if (storedToken && storedUser) {
      try {
        const parsedUser = JSON.parse(storedUser);
        setToken(storedToken);
        setUser(parsedUser);
      } catch (e) {
        localStorage.removeItem('admin_auth_token');
        localStorage.removeItem('admin_auth_user');
      }
    }

    setIsLoading(false);
  }, []);

  // 登录
  const login = useCallback((newToken: string, newUser: User) => {
    localStorage.setItem('admin_auth_token', newToken);
    localStorage.setItem('admin_auth_user', JSON.stringify(newUser));
    setToken(newToken);
    setUser(newUser);
  }, []);

  // 登出
  const logout = useCallback(() => {
    localStorage.removeItem('admin_auth_token');
    localStorage.removeItem('admin_auth_user');
    setToken(null);
    setUser(null);
  }, []);

  // 刷新用户信息
  const refreshUser = useCallback(async () => {
    if (!token) return;

    try {
      const currentUser = await authApi.getCurrentUser();
      setUser(currentUser);
      localStorage.setItem('admin_auth_user', JSON.stringify(currentUser));
    } catch (e) {
      // Token过期，清除认证状态
      logout();
    }
  }, [token, logout]);

  const value = {
    user,
    token,
    isAuthenticated: !!token && !!user,
    isLoading,
    login,
    logout,
    refreshUser,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}

// 路由保护组件
export function RequireAuth({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isLoading } = useAuth();

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-[#0f172a]">
        <div className="text-white">加载中...</div>
      </div>
    );
  }

  if (!isAuthenticated) {
    // 未登录，跳转到登录页
    return <Navigate to="/admin-login" replace />;
  }

  return <>{children}</>;
}