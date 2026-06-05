// 认证模块统一入口

export { AuthProvider, useAuth, RequireAuth, RequireRole } from './AuthContext';
export { authApi } from '@/shared/api/auth';
export type { LoginRequest, RegisterRequest } from '@/shared/api/auth';
