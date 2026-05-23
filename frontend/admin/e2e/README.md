# Admin端E2E测试文档

## 测试概述

本目录包含直播竞拍系统管理后台的端到端(E2E)测试,使用Playwright测试框架实现。

## 测试文件结构

```
e2e/
├── product.spec.ts           # 商品管理测试
├── auction-manage.spec.ts    # 竞拍管理测试
├── statistics.spec.ts        # 统计报表测试
├── utils/
│   └── test-helpers.ts       # 测试工具函数
└── README.md                 # 本文档
```

## 测试覆盖场景

### 1. 商品管理测试 (`product.spec.ts`)
- 创建商品
- 编辑商品
- 删除商品
- 查看商品列表
- 商品搜索与筛选
- 商品批量操作
- 商品导入导出

### 2. 竞拍管理测试 (`auction-manage.spec.ts`)
- 查看竞拍列表
- 查看竞拍详情
- 取消竞拍
- 创建竞拍
- 编辑竞拍
- 竞拍监控
- 竞拍数据导出

### 3. 统计报表测试 (`statistics.spec.ts`)
- 查看数据大屏
- 查看竞拍统计
- 查看收入统计
- 查看用户统计
- 时间范围筛选
- 数据对比
- 报表导出

## 环境要求

- Node.js >= 16.x
- npm >= 8.x

## 安装依赖

```bash
# 安装项目依赖
npm install

# 安装Playwright浏览器
npx playwright install
```

## 运行测试

### 运行所有测试
```bash
npm run test:e2e
```

### 运行特定测试文件
```bash
npx playwright test product.spec.ts
```

### 运行特定测试用例
```bash
npx playwright test -g "创建商品成功"
```

### 使用UI模式运行
```bash
npm run test:e2e:ui
```

### 调试模式运行
```bash
npm run test:e2e:debug
```

### 生成测试报告
```bash
npm run test:e2e:report
```

## 测试配置

### Playwright配置 (`playwright.config.ts`)

主要配置项:
- **测试目录**: `./e2e`
- **基础URL**: `http://localhost:5174`
- **浏览器**: Chromium, Firefox, WebKit
- **超时时间**: 60秒
- **重试次数**: CI环境2次,本地0次
- **并发执行**: CI环境单线程,本地自动

### 环境变量配置 (`.env.test`)

创建 `.env.test` 文件配置测试环境:

```bash
# 应用URL
E2E_BASE_URL=http://localhost:5174

# 管理员账号
ADMIN_USERNAME=admin
ADMIN_PASSWORD=Admin@123456
```

## 测试数据准备

### 管理员账号
系统需要预先创建管理员账号:
- 用户名: `admin`
- 密码: `Admin@123456`

### 测试数据
运行测试前,确保数据库中有:
- 至少3个商品
- 至少1个进行中的竞拍
- 至少1个已结束的竞拍
- 用户订单数据

## 测试报告

测试执行后,会生成以下报告:

1. **HTML报告**: `playwright-report/index.html`
2. **JSON报告**: `test-results.json`
3. **控制台输出**: 实时测试进度

## 特殊测试场景

### 权限测试
测试不同权限的管理员能访问的功能:
- 超级管理员: 所有功能
- 商品管理员: 商品管理相关功能
- 财务管理员: 统计报表相关功能

### 数据导出测试
测试各种数据导出功能:
- 商品列表导出
- 竞拍数据导出
- 统计报表导出
- 支持Excel、CSV、PDF格式

### 实时监控测试
测试竞拍实时监控功能:
- WebSocket连接
- 实时数据更新
- 图表动态渲染

## 调试技巧

### 1. 使用 page.pause()
```typescript
await page.pause();
```

### 2. 查看网络请求
```typescript
page.on('request', request => console.log('>>', request.method(), request.url()));
page.on('response', response => console.log('<<', response.status(), response.url()));
```

### 3. 模拟文件上传
```typescript
await page.setInputFiles('input[type="file"]', {
  name: 'test.jpg',
  mimeType: 'image/jpeg',
  buffer: Buffer.from('test-content'),
});
```

### 4. 等待表格加载
```typescript
await page.waitForSelector('.table-row');
```

## 最佳实践

1. **使用数据-testid属性**: 为管理后台元素添加 `data-testid` 属性
2. **测试数据隔离**: 每个测试用例创建独立测试数据,测试完成后清理
3. **模拟真实场景**: 测试覆盖正常流程和异常流程
4. **验证数据一致性**: 操作后验证数据库状态
5. **使用工具函数**: 复用登录、导航等通用操作

## CI/CD集成

### Jenkins Pipeline示例

```groovy
pipeline {
  agent any
  stages {
    stage('Install') {
      steps {
        sh 'npm ci'
        sh 'npx playwright install'
      }
    }
    stage('Test') {
      steps {
        sh 'npm run test:e2e'
      }
      post {
        always {
          publishHTML([
            allowMissing: false,
            alwaysLinkToLastBuild: true,
            keepAll: true,
            reportDir: 'playwright-report',
            reportFiles: 'index.html',
            reportName: 'Playwright Report'
          ])
        }
      }
    }
  }
}
```

## 维护指南

1. **定期更新测试用例**: 随着功能迭代更新测试
2. **监控测试稳定性**: 关注测试失败率,及时修复不稳定测试
3. **优化测试性能**: 减少不必要的等待,提高测试执行效率
4. **保持测试独立性**: 确保测试可以独立运行,不依赖执行顺序

## 联系方式

如有测试相关问题,请联系测试团队。
