import axios from 'axios';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1';

export interface User {
  id: number;
  email: string;
  name: string;
  role: number; // 0=用户, 1=商家, 2=管理员
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface LoginResponse {
  code: number;
  message: string;
  data: {
    user: User;
    token: string;
  };
}

class AuthService {
  private TOKEN_KEY = 'auth_token';
  private USER_KEY = 'auth_user';

  /**
   * 用户登录
   */
  async login(email: string, password: string): Promise<{ user: User; token: string }> {
    try {
      const response = await axios.post<LoginResponse>(`${API_BASE_URL}/auth/login`, {
        email,
        password,
      });

      if (response.data.code === 200 && response.data.data) {
        const { user, token } = response.data.data;

        // 存储到localStorage
        this.setToken(token);
        this.setUser(user);

        return { user, token };
      } else {
        throw new Error(response.data.message || '登录失败');
      }
    } catch (error: any) {
      if (error.response?.data?.message) {
        throw new Error(error.response.data.message);
      }
      throw new Error('登录失败，请检查网络连接');
    }
  }

  /**
   * 用户登出
   */
  logout(): void {
    localStorage.removeItem(this.TOKEN_KEY);
    localStorage.removeItem(this.USER_KEY);

    // 跳转到登录页
    window.location.href = '/login';
  }

  /**
   * 获取当前用户信息
   */
  getCurrentUser(): User | null {
    const userStr = localStorage.getItem(this.USER_KEY);
    if (userStr) {
      try {
        return JSON.parse(userStr);
      } catch (e) {
        return null;
      }
    }
    return null;
  }

  /**
   * 获取token
   */
  getToken(): string | null {
    return localStorage.getItem(this.TOKEN_KEY);
  }

  /**
   * 检查是否已登录
   */
  isAuthenticated(): boolean {
    const token = this.getToken();
    const user = this.getCurrentUser();
    return !!(token && user);
  }

  /**
   * 检查用户角色
   */
  hasRole(role: number): boolean {
    const user = this.getCurrentUser();
    return user ? user.role === role : false;
  }

  /**
   * 检查是否是管理员
   */
  isAdmin(): boolean {
    return this.hasRole(2);
  }

  /**
   * 检查是否是商家
   */
  isMerchant(): boolean {
    return this.hasRole(1);
  }

  /**
   * 存储token
   */
  private setToken(token: string): void {
    localStorage.setItem(this.TOKEN_KEY, token);
  }

  /**
   * 存储用户信息
   */
  private setUser(user: User): void {
    localStorage.setItem(this.USER_KEY, JSON.stringify(user));
  }
}

// 导出单例
export const authService = new AuthService();
