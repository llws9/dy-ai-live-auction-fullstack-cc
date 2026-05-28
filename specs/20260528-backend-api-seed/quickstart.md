# Quickstart: 后端API补充与测试数据生成

**Feature**: `20260528-backend-api-seed`
**Date**: 2026-05-28

## 快速开始指南

### 1. 环境准备

确保以下服务正常运行：
```bash
# 检查服务状态
curl http://localhost:8080/health   # Gateway
curl http://localhost:8081/health   # Product Service (需添加健康检查)
curl http://localhost:8082/health   # Auction Service
```

### 2. 数据库迁移

Category 表和 Product.category_id 字段将通过 GORM AutoMigrate 自动创建：

```bash
# 启动 Product Service 时自动迁移
cd backend/product && go run main.go
```

### 3. 生成测试数据

```bash
# 运行 Seed 脚本
cd backend/seed && go run main.go

# 输出示例：
# ✅ Categories: 6 条
# ✅ Users: 53 条
# ✅ Products: 32 条
# ✅ LiveStreams: 12 条
# ✅ AuctionRules: 30 条
# ✅ Auctions: 20 条
# ✅ Bids: 105 条
# ✅ Orders: 15 条
# ✅ Notifications: 30 条
# 🎉 总计: 303 条数据生成完成
```

### 4. API 测试

#### 测试类别管理
```bash
# 获取类别列表
curl http://localhost:8080/api/v1/categories

# 创建新类别 (需管理员Token)
curl -X POST http://localhost:8080/api/v1/categories \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{"name":"运动户外","code":"sports"}'
```

#### 测试订单发货
```bash
# 发货 (需商家/管理员Token)
curl -X PUT http://localhost:8080/api/v1/orders/1/ship \
  -H "Authorization: Bearer <merchant-token>" \
  -H "Content-Type: application/json" \
  -d '{"tracking_number":"SF1234567890"}'
```

#### 测试直播间
```bash
# 管理端直播间列表
curl http://localhost:8080/api/v1/admin/live-streams \
  -H "Authorization: Bearer <admin-token>"

# 直播间详情 (无需认证)
curl http://localhost:8080/api/v1/live-streams/1
```

### 5. 前端验证

1. **Admin管理后台**: http://localhost:5175/#/
   - 验证商品列表显示类别
   - 验证订单管理发货功能
   
2. **H5用户端**: http://localhost:5178/#/
   - 验证首页竞拍列表加载真实数据
   - 验证订单历史页面

## 常见问题

### Q: Seed脚本执行失败？
检查数据库连接配置：
```bash
# 确认环境变量
cat backend/.env
```

### Q: 类别删除报错？
确保该类别下没有关联商品，否则需要先修改商品类别或删除商品。

### Q: API返回401？
检查JWT Token是否有效，权限是否满足要求（管理员/商家）。