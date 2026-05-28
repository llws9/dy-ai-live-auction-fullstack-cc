# 前端错误处理增强总结

## 概述
为管理后台(Admin)和H5端添加了完善的错误处理机制，包括统一错误拦截、友好提示、错误边界和日志记录。

## 新增文件

### Admin端

#### 1. `frontend/admin/src/utils/errorMessages.ts`
- 错误消息映射配置
- HTTP状态码错误处理
- 业务错误码处理
- 错误日志记录函数
- 用户友好的错误消息格式化

#### 2. `frontend/admin/src/utils/errorHandling.ts`
- 安全执行异步函数工具
- 重试机制
- 防抖错误处理
- 批量错误处理
- 错误报告生成
- 网络状态监测

#### 3. `frontend/admin/src/services/api.ts` (更新)
- 统一的错误拦截器
- 请求超时处理
- 网络错误处理
- 401自动跳转登录
- 业务错误码处理
- Toast提示集成

#### 4. `frontend/admin/src/components/ErrorBoundary/index.tsx`
- React错误边界组件
- 错误降级UI
- 开发环境错误详情显示
- 刷新/返回首页操作

#### 5. `frontend/admin/src/components/Toast/index.tsx`
- Toast提示组件
- 成功/错误/警告/信息四种类型
- 自动消失机制
- 可手动关闭

#### 6. `frontend/admin/src/hooks/useErrorHandler.ts`
- 统一错误处理Hook
- 自动显示错误提示
- 自动处理401跳转
- 异步函数包装器

#### 7. `frontend/admin/src/vite-env.d.ts`
- TypeScript环境类型定义

### H5端

#### 1. `frontend/h5/src/utils/errorMessages.ts`
- 错误消息映射配置
- 移动端适配的错误提示
- WebSocket错误处理

#### 2. `frontend/h5/src/utils/errorHandling.ts`
- 移动端震动反馈支持
- 网络状态监测
- 重试机制
- 错误日志导出

#### 3. `frontend/h5/src/services/api.ts`
- 统一的API请求封装
- 移动端优化
- 微信环境适配

#### 4. `frontend/h5/src/components/ErrorBoundary/index.tsx`
- 移动端适配的错误边界
- 简洁的错误UI
- 触摸友好的操作按钮

#### 5. `frontend/h5/src/components/Toast/index.tsx`
- 移动端居中Toast
- Loading状态支持
- 轻量级动画效果

#### 6. `frontend/h5/src/hooks/useErrorHandler.ts`
- 移动端错误处理Hook

#### 7. `frontend/h5/src/vite-env.d.ts`
- TypeScript环境类型定义

## 错误处理类型

### HTTP状态码错误
- 400: 请求参数有误
- 401: 登录已过期，自动跳转登录页
- 403: 权限不足
- 404: 资源不存在
- 408: 请求超时
- 409: 资源冲突
- 422: 参数验证失败
- 429: 请求过于频繁
- 500: 服务器错误
- 502: 网关错误
- 503: 服务不可用
- 504: 网关超时

### 网络错误
- 网络连接失败
- 请求超时
- WebSocket连接异常

### 业务错误
- 商品不存在
- 竞拍不存在/已结束/未开始
- 出价过低
- 余额不足
- 订单不存在/已支付

## 使用方式

### 在组件中使用
```tsx
import { useToast } from './components/Toast';
import { useErrorHandler } from './hooks/useErrorHandler';

function MyComponent() {
  const { showToast } = useToast();
  const { handleError, wrapAsync } = useErrorHandler();

  // 方式1: 手动处理错误
  const fetchData = async () => {
    try {
      const data = await api.getData();
    } catch (error) {
      handleError(error, 'fetchData');
    }
  };

  // 方式2: 包装异步函数
  const safeFetch = wrapAsync(async () => {
    return await api.getData();
  });

  // 方式3: 直接使用Toast
  const handleSuccess = () => {
    showToast('操作成功', 'success');
  };

  return <div>...</div>;
}
```

### API调用
```tsx
import { get, post } from './services/api';

// 自动错误处理
const data = await get('/api/users');

// 禁用错误提示
const data = await get('/api/users', { showError: false });

// 自定义超时
const data = await get('/api/users', { timeout: 60000 });
```

## 特性

1. **统一拦截**: 所有API请求自动拦截错误
2. **友好提示**: 用户友好的错误消息
3. **自动跳转**: 401错误自动跳转登录
4. **错误边界**: 捕获React渲染错误
5. **日志记录**: 本地存储错误日志，便于排查
6. **网络监测**: 检测网络状态变化
7. **重试机制**: 支持请求重试
8. **移动端优化**: H5端针对移动设备优化
9. **类型安全**: 完整的TypeScript类型支持

## 构建验证

- ✅ Admin端构建成功
- ✅ H5端构建成功
- ✅ TypeScript类型检查通过

## 后续建议

1. 集成错误监控服务（如Sentry）
2. 添加错误上报接口
3. 实现错误日志定期清理
4. 添加离线模式支持
