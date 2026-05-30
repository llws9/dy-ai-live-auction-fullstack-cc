import { post, ApiError } from './api';

export interface User {
  id: number;
  email?: string;
  phone?: string;
  name: string;
  role: number; // 0=用户, 1=商家, 2=管理员
}

// 后端登录契约支持 email / phone 任一
export interface LoginRequest {
  email?: string;
  phone?: string;
  password: string;
}

export interface LoginResponseData {
  user: User;
  token: string;
}

class AuthService {
  private TOKEN_KEY = 'auth_token';
  private USER_KEY = 'auth_user';

  /**
   * 用户登录：通过统一 api.ts 的 post()，自动处理业务码/超时/错误提示。
   */
  async login(req: LoginRequest): Promise<LoginResponseData> {
    if (!req.email && !req.phone) {
      throw new ApiError('请输入邮箱或手机号', 400, 'INVALID_PARAMS');
    }
    if (!req.password) {
      throw new ApiError('请输入密码', 400, 'INVALID_PARAMS');
    }

    const data = await post<LoginResponseData>('/auth/login', req, { showError: false });

    if (!data?.token || !data?.user) {
      throw new ApiError('登录响应缺少 token / user', 500, 'INVALID_RESPONSE');
    }

    this.setToken(data.token);
    this.setUser(data.user);

    return data;
  }

  /**
   * 用户登出：仅清理本地凭据，由调用方负责导航（避免硬刷新丢失 SPA 状态）。
   */
  logout(): void {
    localStorage.removeItem(this.TOKEN_KEY);
    localStorage.removeItem(this.USER_KEY);
    localStorage.removeItem('token');
    localStorage.removeItem('userInfo');
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
