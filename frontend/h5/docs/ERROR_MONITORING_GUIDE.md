# 错误监控系统使用指南

## 概述

本项目集成了轻量级的错误监控系统，用于捕获、记录和上报前端应用中的错误。

## 功能特性

### 1. 自动错误捕获

错误监控系统会自动捕获以下类型的错误：

- **JavaScript运行时错误** - `window.onerror`
- **Promise未处理的rejection** - `unhandledrejection`
- **资源加载错误** - 图片、脚本、样式表等加载失败
- **React组件错误** - 通过ErrorBoundary捕获

### 2. 错误信息收集

每个错误报告包含以下信息：

```typescript
interface ErrorReport {
  timestamp: string;          // 错误发生时间
  message: string;            // 错误消息
  stack?: string;             // 错误堆栈
  url: string;                // 发生错误的页面URL
  userAgent: string;          // 用户浏览器信息
  userId?: number;            // 当前用户ID
  role?: number;              // 当前用户角色
  additionalData?: Record<string, any>;  // 额外数据
}
```

### 3. 智能上报策略

- **批量上报**: 错误累积到10条或延迟5秒后自动上报
- **离线支持**: 离线时保存到localStorage，上线后自动发送
- **失败重试**: 上报失败时保存错误，下次重试
- **用户关联**: 自动关联当前登录用户信息

## 使用方法

### 基础使用

错误监控系统已自动初始化，无需手动配置。它会自动捕获全局错误。

### 手动捕获错误

```typescript
import { captureException, captureMessage } from '../utils/errorMonitor';

// 捕获异常
try {
  // 可能出错的代码
} catch (error) {
  captureException(error as Error, {
    component: 'MyComponent',
    action: 'submitForm',
  });
}

// 捕获自定义消息
captureMessage('用户登录失败', 'warning');
captureMessage('重要操作完成', 'info');
captureMessage('严重错误发生', 'error');
```

### 在API调用中使用

```typescript
import { captureException } from '../utils/errorMonitor';

async function fetchData() {
  try {
    const response = await fetch('/api/data');
    return await response.json();
  } catch (error) {
    captureException(error as Error, {
      api: '/api/data',
      method: 'GET',
    });
    throw error;
  }
}
```

### 在React组件中使用

```typescript
import { captureException } from '../utils/errorMonitor';

function MyComponent() {
  const handleClick = () => {
    try {
      // 业务逻辑
    } catch (error) {
      captureException(error as Error, {
        component: 'MyComponent',
        event: 'handleClick',
      });
    }
  };

  return <button onClick={handleClick}>Click me</button>;
}
```

## 配置选项

### 修改上报端点

在 `src/utils/errorMonitor.ts` 中修改：

```typescript
class ErrorMonitor {
  private reportEndpoint = '/api/v1/errors/report'; // 修改为你的API端点
  // ...
}
```

### 调整批量上报大小

```typescript
class ErrorMonitor {
  private maxQueueSize = 10; // 修改为你需要的大小
  // ...
}
```

### 查看错误统计

```typescript
import { errorMonitor } from '../utils/errorMonitor';

// 获取错误统计
const stats = errorMonitor.getErrorStats();
console.log('错误总数:', stats.total);
console.log('最近10条错误:', stats.recent);

// 清除所有错误
errorMonitor.clearErrors();
```

## 后端API要求

错误监控系统会向 `/api/v1/errors/report` 发送POST请求，请求体格式：

```json
{
  "errors": [
    {
      "timestamp": "2026-05-23T12:00:00.000Z",
      "message": "Error message",
      "stack": "Error stack trace...",
      "url": "https://example.com/page",
      "userAgent": "Mozilla/5.0...",
      "userId": 123,
      "role": 0,
      "additionalData": {
        "component": "MyComponent"
      }
    }
  ]
}
```

### 后端实现示例（Go）

```go
type ErrorReport struct {
    Timestamp     string                 `json:"timestamp"`
    Message       string                 `json:"message"`
    Stack         string                 `json:"stack,omitempty"`
    URL           string                 `json:"url"`
    UserAgent     string                 `json:"userAgent"`
    UserID        *int                   `json:"userId,omitempty"`
    Role          *int                   `json:"role,omitempty"`
    AdditionalData map[string]interface{} `json:"additionalData,omitempty"`
}

type ErrorReportRequest struct {
    Errors []ErrorReport `json:"errors"`
}

// POST /api/v1/errors/report
func ReportErrors(c *gin.Context) {
    var req ErrorReportRequest
    if err := c.BindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": "Invalid request"})
        return
    }

    // 保存到数据库或日志系统
    for _, err := range req.Errors {
        log.Printf("[%s] Error: %s (User: %v, URL: %s)",
            err.Timestamp, err.Message, err.UserID, err.URL)

        // 可以保存到数据库
        // db.Create(&err)
    }

    c.JSON(200, gin.H{"success": true})
}
```

## 开发环境调试

在开发环境中，错误会输出到控制台：

```
🚨 Error Monitor Report
  Error: Something went wrong
    at Component.render (MyComponent.tsx:25)
    at ...
```

## 生产环境建议

### 1. 集成第三方服务

可以轻松集成Sentry、LogRocket等第三方错误监控服务：

```typescript
import * as Sentry from '@sentry/react';

class ErrorMonitor {
  private sendToServer(errors: ErrorReport[]) {
    // 发送到Sentry
    errors.forEach(error => {
      Sentry.captureException(new Error(error.message), {
        extra: error.additionalData,
        user: error.userId ? { id: error.userId } : undefined,
      });
    });

    // 同时发送到自己的服务器
    // ...
  }
}
```

### 2. 错误分类和过滤

```typescript
// 过滤掉不重要的错误
private shouldReport(error: ErrorReport): boolean {
  // 忽略网络错误
  if (error.message.includes('NetworkError')) {
    return false;
  }

  // 忽略第三方脚本错误
  if (error.additionalData?.source?.includes('googletagmanager')) {
    return false;
  }

  return true;
}
```

### 3. 错误聚合

对于频繁发生的相同错误，可以在后端进行聚合，避免重复记录：

```sql
-- 数据库表设计
CREATE TABLE error_reports (
    id SERIAL PRIMARY KEY,
    fingerprint VARCHAR(64), -- 错误指纹（基于message和stack生成）
    message TEXT,
    stack TEXT,
    count INTEGER DEFAULT 1,
    first_seen TIMESTAMP,
    last_seen TIMESTAMP,
    user_ids INTEGER[], -- 受影响的用户ID列表
    urls TEXT[] -- 出错的URL列表
);

-- 聚合相同错误
INSERT INTO error_reports (fingerprint, message, stack, first_seen, last_seen)
VALUES (?, ?, ?, NOW(), NOW())
ON CONFLICT (fingerprint)
DO UPDATE SET
    count = count + 1,
    last_seen = NOW(),
    user_ids = array_append(user_ids, EXCLUDED.user_id),
    urls = array_append(urls, EXCLUDED.url);
```

## 监控面板

可以基于存储的错误数据创建监控仪表板，显示：

- 错误趋势图
- 最常见的错误
- 受影响的用户数
- 错误分布（按浏览器、页面等）

## 最佳实践

### 1. 关键路径监控

对关键业务流程添加额外的错误监控：

```typescript
// 支付流程
async function processPayment() {
  try {
    // 支付逻辑
  } catch (error) {
    captureException(error as Error, {
      flow: 'payment',
      step: 'process',
      critical: true, // 标记为关键错误
    });
  }
}
```

### 2. 性能监控

结合Performance API监控性能：

```typescript
// 监控页面加载性能
window.addEventListener('load', () => {
  const perfData = performance.getEntriesByType('navigation')[0];
  if (perfData.loadEventEnd > 5000) {
    captureMessage('页面加载缓慢', 'warning', {
      loadTime: perfData.loadEventEnd,
      domReady: perfData.domContentLoadedEventEnd,
    });
  }
});
```

### 3. 用户体验监控

监控用户体验指标：

```typescript
// 监控API响应时间
const apiStartTime = Date.now();
await fetch('/api/data');
const duration = Date.now() - apiStartTime;

if (duration > 3000) {
  captureMessage('API响应缓慢', 'warning', {
    api: '/api/data',
    duration,
  });
}
```

## 隐私和合规

### 1. 敏感信息过滤

确保不上报敏感信息：

```typescript
private sanitizeError(error: ErrorReport): ErrorReport {
  // 过滤URL中的敏感参数
  error.url = error.url.replace(/token=[^&]+/, 'token=***');

  // 过滤堆栈中的敏感路径
  if (error.stack) {
    error.stack = error.stack.replace(/password=\S+/, 'password=***');
  }

  return error;
}
```

### 2. 用户同意

在收集错误数据前获取用户同意（GDPR要求）：

```typescript
// 在隐私政策中说明错误收集
// 提供退出选项
if (localStorage.getItem('allow_error_tracking') !== 'true') {
  // 禁用错误上报
  return;
}
```

## 故障排查

### 错误监控自身出错

如果错误监控系统本身出错，它会降级到console.error：

```typescript
try {
  await this.sendToServer(errors);
} catch (error) {
  console.error('Failed to report errors:', error);
  // 保存到localStorage以便下次发送
  this.saveToLocalStorage(errors);
}
```

### 查看存储的错误

```javascript
// 在浏览器控制台执行
JSON.parse(localStorage.getItem('error_monitor_queue'))
```

---

## 总结

错误监控系统提供了完整的前端错误捕获和上报解决方案，具有以下优势：

✅ **自动化** - 无需手动配置，自动捕获所有错误
✅ **智能上报** - 批量上报，离线支持，失败重试
✅ **轻量级** - 不依赖第三方服务，可独立运行
✅ **可扩展** - 易于集成第三方监控服务
✅ **开发友好** - 开发环境详细日志，生产环境安全上报

通过错误监控，可以：

- 🎯 快速发现和定位问题
- 📊 了解应用的健康状况
- 👥 追踪受影响的用户
- 🚀 提升应用质量和用户体验
