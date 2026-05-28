// 认证API

import { get, post } from './request';
import { User, ApiResponse } from './types';

export interface LoginRequest {
  email?: string;
  phone?: string;
  password: string;
}

export interface RegisterRequest {
  name: string;
  email?: string;
  phone?: string;
  password: string;
}

export interface LoginResponse {
  token: string;
  user: User;
}

export const authApi = {
  // 用户登录
  login: (data: LoginRequest) => post<LoginResponse>('/auth/login', data),

  // 用户注册
  register: (data: RegisterRequest) => post<LoginResponse>('/auth/register', data),

  // 获取当前用户信息
  getCurrentUser: () => get<User>('/users/me'),
};