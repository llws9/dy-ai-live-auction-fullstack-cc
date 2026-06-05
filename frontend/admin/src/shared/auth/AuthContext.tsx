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

const WINDOWS_1252_REVERSE: Record<string, number> = {
  '€': 0x80,
  '‚': 0x82,
  'ƒ': 0x83,
  '„': 0x84,
  '…': 0x85,
  '†': 0x86,
  '‡': 0x87,
  'ˆ': 0x88,
  '‰': 0x89,
  'Š': 0x8a,
  '‹': 0x8b,
  'Œ': 0x8c,
  'Ž': 0x8e,
  '‘': 0x91,
  '’': 0x92,
  '“': 0x93,
  '”': 0x94,
  '•': 0x95,
  '–': 0x96,
  '—': 0x97,
  '˜': 0x98,
  '™': 0x99,
  'š': 0x9a,
  '›': 0x9b,
  'œ': 0x9c,
  'ž': 0x9e,
  'Ÿ': 0x9f,
};

function decodePossibleMojibake(value: string): string {
  if (!/[ÂÃÄÅÆÇÈÉÊËÌÍÎÏÐÑÒÓÔÕÖØÙÚÛÜÝÞßà-ÿ\u0080-\u009f]/.test(value)) {
    return value;
  }

  const bytes: number[] = [];
  for (const char of value) {
    const code = char.charCodeAt(0);
    if (code <= 0xff) {
      bytes.push(code);
      continue;
    }
    const mapped = WINDOWS_1252_REVERSE[char];
    if (mapped === undefined) {
      return value;
    }
    bytes.push(mapped);
  }

  try {
    return new TextDecoder('utf-8', { fatal: true }).decode(new Uint8Array(bytes));
  } catch {
    return value;
  }
}

function normalizeUser(user: User): User {
  return {
    ...user,
    name: decodePossibleMojibake(user.name),
  };
}

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
        const parsedUser = normalizeUser(JSON.parse(storedUser));
        setToken(storedToken);
        setUser(parsedUser);
        localStorage.setItem('admin_auth_user', JSON.stringify(parsedUser));
      } catch (e) {
        localStorage.removeItem('admin_auth_token');
        localStorage.removeItem('admin_auth_user');
      }
    }

    setIsLoading(false);
  }, []);

  // 登录
  const login = useCallback((newToken: string, newUser: User) => {
    const normalizedUser = normalizeUser(newUser);
    localStorage.setItem('admin_auth_token', newToken);
    localStorage.setItem('admin_auth_user', JSON.stringify(normalizedUser));
    setToken(newToken);
    setUser(normalizedUser);
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
      const currentUser = normalizeUser(await authApi.getCurrentUser());
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

export function RequireRole({
  allowedRoles,
  children,
  fallbackPath = '/dashboard',
}: {
  allowedRoles: number[];
  children: React.ReactNode;
  fallbackPath?: string;
}) {
  const { user, isAuthenticated, isLoading } = useAuth();

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-[#0f172a]">
        <div className="text-white">加载中...</div>
      </div>
    );
  }

  if (!isAuthenticated) {
    return <Navigate to="/admin-login" replace />;
  }

  if (!user || !allowedRoles.includes(user.role)) {
    return <Navigate to={fallbackPath} replace />;
  }

  return <>{children}</>;
}
