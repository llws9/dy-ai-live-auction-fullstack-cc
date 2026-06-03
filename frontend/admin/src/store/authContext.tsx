import { createContext, useContext, useState, useEffect, ReactNode } from 'react';

interface AdminUser {
  id: number;
  name: string;
  email?: string;
  role: number;
}

interface AdminAuthContextType {
  user: AdminUser | null;
  token: string | null;
  loading: boolean;
  login: (token: string, user: AdminUser) => void;
  logout: () => void;
  isAuthenticated: boolean;
  isAdmin: boolean;
  isStaff: boolean; // 商家或管理员
}

const AdminAuthContext = createContext<AdminAuthContextType | undefined>(undefined);

export function AdminAuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<AdminUser | null>(null);
  const [token, setToken] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const storedToken = localStorage.getItem('admin_auth_token');
    const storedUser = localStorage.getItem('admin_auth_user');

    if (storedToken && storedUser) {
      const parsedUser = JSON.parse(storedUser);
      // 验证是否为商家(role=1)或管理员(role=2)
      if (parsedUser.role >= 1) {
        setToken(storedToken);
        setUser(parsedUser);
      } else {
        // 清除无效token
        localStorage.removeItem('admin_auth_token');
        localStorage.removeItem('admin_auth_user');
      }
    }
    setLoading(false);
  }, []);

  const login = (newToken: string, newUser: AdminUser) => {
    if (newUser.role < 1) {
      throw new Error('权限不足');
    }
    setToken(newToken);
    setUser(newUser);
    localStorage.setItem('admin_auth_token', newToken);
    localStorage.setItem('admin_auth_user', JSON.stringify(newUser));
  };

  const logout = () => {
    setToken(null);
    setUser(null);
    localStorage.removeItem('admin_auth_token');
    localStorage.removeItem('admin_auth_user');
  };

  return (
    <AdminAuthContext.Provider value={{
      user,
      token,
      loading,
      login,
      logout,
      isAuthenticated: !!token && !!user,
      isAdmin: user?.role === 2,
      isStaff: (user?.role === 1 || user?.role === 2) // 商家或管理员都可以访问
    }}>
      {children}
    </AdminAuthContext.Provider>
  );
}

export function useAdminAuth() {
  const context = useContext(AdminAuthContext);
  if (context === undefined) {
    throw new Error('useAdminAuth must be used within an AdminAuthProvider');
  }
  return context;
}
