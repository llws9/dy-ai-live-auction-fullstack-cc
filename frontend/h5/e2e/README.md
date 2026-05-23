# E2E测试文档

## 测试概述

本目录包含直播竞拍系统的端到端(E2E)测试,使用Playwright测试框架实现。

## 测试文件结构

```
e2e/
├── auth.spec.ts              # 用户认证测试
├── auction.spec.ts           # 竞拍流程测试
├── order.spec.ts             # 订单流程测试
├── utils/
│   └── test-helpers.ts       # 测试工具函数
└── README.md                 # 本文档
```

## 测试覆盖场景

### 1. 用户认证测试 (`auth.spec.ts`)
- 用户注册流程
- 用户登录流程
- Token验证
- 退出登录
- 表单验证
- 认证状态持久化

### 2. 竞拍流程测试 (`auction.spec.ts`)
- 查看竞拍列表
- 进入竞拍详情
- 出价操作
- 查看排名
- 竞拍结束
- 竞拍倒计时
- 实时竞拍更新

### 3. 订单流程测试 (`order.spec.ts`)
- 查看订单列表
- 订单支付
- 订单取消
- 查看历史记录
- 订单搜索
- 订单状态追踪
- 订单评价

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
npx playwright test auth.spec.ts
```

### 运行特定测试用例
```bash
npx playwright test -g "用户登录成功"
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
- **基础URL**: `http://localhost:5173`
- **浏览器**: Chromium, Mobile Chrome, Mobile Safari
- **超时时间**: 60秒
- **重试次数**: CI环境2次,本地0次
- **并发执行**: CI环境单线程,本地自动

### 环境变量配置 (`.env.test`)

创建 `.env.test` 文件配置测试环境:

```bash
# 应用URL
E2E_BASE_URL=http://localhost:5173

# 测试账号
TEST_USERNAME=testuser
TEST_PASSWORD=Test@123456
```

## 测试数据准备

### 测试账号
系统需要预先创建以下测试账号:

1. **普通用户账号**
   - 用户名: `testuser`
   - 密码: `Test@123456`

2. **管理员账号**
   - 用户名: `admin`
   - 密码: `Admin@123456`

### 测试数据
运行测试前,确保数据库中有:
- 至少1个竞拍中的商品
- 至少1个已结束的竞拍
- 至少1个待付款的订单

## 测试报告

测试执行后,会生成以下报告:

1. **HTML报告**: `playwright-report/index.html`
2. **JSON报告**: `test-results.json`
3. **控制台输出**: 实时测试进度

## 调试技巧

### 1. 使用 page.pause()
```typescript
await page.pause();
```

### 2. 查看页面快照
```typescript
await page.screenshot({ path: 'debug.png' });
```

### 3. 查看控制台日志
```typescript
page.on('console', msg => console.log(msg.text()));
```

### 4. 查看网络请求
```typescript
page.on('request', request => console.log(request.url()));
```

## 最佳实践

1. **使用数据-testid属性**: 为关键元素添加 `data-testid` 属性,提高测试稳定性
2. **避免硬编码等待**: 使用 `waitFor` 系列方法替代 `waitForTimeout`
3. **测试隔离**: 每个测试用例独立运行,不依赖其他测试
4. **清理测试数据**: 测试完成后清理创建的测试数据
5. **使用工具函数**: 将通用操作封装到 `test-helpers.ts`

## 常见问题

### Q: 测试运行超时怎么办?
A: 增加 `playwright.config.ts` 中的 `timeout` 配置,或优化测试逻辑

### Q: 元素定位失败怎么办?
A: 检查选择器是否正确,或使用 `data-testid` 属性

### Q: 如何跳过某些测试?
A: 使用 `test.skip()` 或 `test.describe.skip()`

### Q: 如何只运行特定浏览器?
A: 使用 `--project=chromium` 参数

## CI/CD集成

### GitHub Actions示例

```yaml
name: E2E Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
      - run: npm ci
      - run: npx playwright install --with-deps
      - run: npm run test:e2e
      - uses: actions/upload-artifact@v3
        if: always()
        with:
          name: playwright-report
          path: playwright-report/
```

## 维护指南

1. **定期更新依赖**: 保持Playwright和浏览器版本最新
2. **审查测试用例**: 定期检查测试覆盖率,添加缺失的测试场景
3. **修复失败测试**: 及时修复失败的测试,保持测试套件健康
4. **优化测试性能**: 减少不必要的等待,并行执行独立测试
