# 种子数据生成指南

## 概述
本项目提供种子数据生成工具，用于快速创建测试和演示数据。种子数据工具位于 `backend/seed` 目录，支持多种数据规模配置。

## 数据规模配置

| 规模 | 配置函数 | 总数据量 | 适用场景 |
|------|----------|----------|----------|
| Small | `SmallConfig()` | ~65条 | 快速测试、CI/CD |
| Medium | `DefaultConfig()` | ~148条 | 开发演示（默认） |
| Large | `LargeConfig()` | ~482条 | 性能测试、压力测试 |

### 详细配置对比

#### Small（小规模）
- Categories: 5
- Users: 20（管理员 15%, 主播 25%, 普通用户 60%）
- Products: 15
- LiveStreams: 5
- AuctionRules: 10
- Orders: 10

#### Medium（中等规模，默认）
- Categories: 8
- Users: 50（管理员 10%, 主播 20%, 普通用户 70%）
- Products: 30
- LiveStreams: 10
- AuctionRules: 20
- Orders: 30

#### Large（大规模）
- Categories: 12
- Users: 100（管理员 5%, 主播 15%, 普通用户 80%）
- Products: 100
- LiveStreams: 20
- AuctionRules: 50
- Orders: 100

## 使用方法

### 后端数据生成

```bash
cd backend/seed
go run main.go
```

当前默认使用 `DefaultConfig()`（中等规模）生成数据。

### 修改规模配置

编辑 `backend/seed/main.go` 文件，将 `DefaultConfig()` 替换为其他配置：

```go
// 使用小规模配置
cfg := SmallConfig()

// 或使用大规模配置
cfg := LargeConfig()
```

### 自定义配置

可以在 `backend/seed/config.go` 中创建自定义配置：

```go
func CustomConfig() *SeedConfig {
    return &SeedConfig{
        CategoriesCount:   10,
        UsersCount:        80,
        ProductsCount:     50,
        LiveStreamsCount:  15,
        AuctionRulesCount: 30,
        OrdersCount:       40,
        // ... 其他配置
    }
}
```

## 数据类型说明

### Categories（类别）
- 预设 12 个类别：数码电子、服装配饰、家居生活、美妆护肤、食品饮料、运动户外、母婴用品、珠宝首饰、图书文具、汽车用品、宠物用品、艺术品
- 每个类别包含名称、代码、描述和排序

### Users（用户）
根据配置的比例生成三种角色的用户：

| 角色 | Role值 | 说明 |
|------|--------|------|
| 普通用户 | 0 | 主要买家群体 |
| 主播 | 1 | 直播间创建者 |
| 管理员 | 2 | 系统管理员 |

生成的用户数据包含：
- 姓名（中文随机组合）
- 邮箱（admin_x@example.com、streamer_x@example.com、user_x@example.com）
- 手机号（不同前缀区分角色）
- 头像URL
- 密码（默认：password123_hash）

### Products（商品）
根据配置的比例生成三种状态的商品：

| 状态 | 状态值 | 比例（默认） |
|------|--------|--------------|
| 草稿 | draft | 30% |
| 已发布 | published | 60% |
| 已下架 | unpublished | 10% |

商品数据包含：
- 名称、描述
- 图片URL（1-3张）
- 所属类别
- 状态

### LiveStreams（直播间）
- 由主播用户创建
- 包含名称、描述、封面图
- 状态随机（禁用/正常）

### AuctionRules（竞拍规则）
- 仅关联已发布的商品
- 包含起拍价、加价幅度、封顶价（可选）
- 竞拍时长、延迟时长等配置

### Orders（订单）
根据配置的比例生成四种状态的订单：

| 状态 | 状态值 | 比例（默认） | 说明 |
|------|--------|--------------|------|
| 待支付 | pending | 20% | 新创建的订单 |
| 已支付 | paid | 30% | 已完成支付 |
| 已发货 | shipped | 30% | 商品已发出 |
| 已完成 | completed | 20% | 交易完成 |

订单数据包含：
- 关联的竞拍ID
- 商品ID
- 获胜者ID（普通用户）
- 最终价格
- 各阶段时间戳

## 数据生成顺序

种子数据按照以下顺序生成，以确保外键关系正确：

1. **Categories** - 无依赖
2. **Users** - 无依赖
3. **Products** - 依赖 Users, Categories
4. **LiveStreams** - 依赖 Users（主播）
5. **AuctionRules** - 依赖 Products（已发布）
6. **Orders** - 依赖 Users, Products

## 数据清理

### 清空所有数据

```bash
# 连接数据库并清空表数据
mysql -u root -p live_auction -e "
SET FOREIGN_KEY_CHECKS = 0;
TRUNCATE TABLE orders;
TRUNCATE TABLE auction_rules;
TRUNCATE TABLE live_streams;
TRUNCATE TABLE products;
TRUNCATE TABLE users;
TRUNCATE TABLE categories;
SET FOREIGN_KEY_CHECKS = 1;
"
```

### 使用初始化脚本

```bash
# 重新初始化数据库结构
mysql -u root -p live_auction < scripts/init.sql
```

## 环境配置

种子数据生成需要数据库连接配置。确保以下环境变量已设置：

```bash
# 数据库连接配置
export DB_HOST=localhost
export DB_PORT=3306
export DB_USER=root
export DB_PASSWORD=your_password
export DB_NAME=live_auction
```

或通过 `docker-compose.yml` 配置的数据库服务连接。

## 注意事项

1. **数据库连接**：运行前确保数据库服务已启动且可连接
2. **数据唯一性**：每次运行会生成新的数据，不会检查重复
3. **外键约束**：数据按依赖顺序生成，确保外键关系正确
4. **环境隔离**：建议仅在开发环境使用，避免在生产环境执行
5. **密码安全**：生成的用户密码为明文hash，实际应用需要使用bcrypt等加密方式

## 扩展开发

如需添加新的数据类型，参考 `generators.go` 中的实现模式：

1. 在 `SeedConfig` 中添加新配置字段
2. 实现 `GenerateXxx` 函数
3. 在 `main.go` 中添加调用逻辑
4. 实现对应的批量插入函数

## 示例输出

运行成功后的输出示例：

```
Seed configuration: &{CategoriesCount:8 UsersCount:50 ProductsCount:30 LiveStreamsCount:10 AuctionRulesCount:20 OrdersCount:30 StreamerRatio:0.2 AdminRatio:0.1 DraftRatio:0.3 PublishedRatio:0.6 UnpublishedRatio:0.1 UnpaidRatio:0.2 PaidRatio:0.3 ShippedRatio:0.3 CompletedRatio:0.2}
Starting seed data generation...
Generating categories...
Generated 8 categories
Generating users...
Generated 50 users
Generating products...
Generated 30 products
Generating live streams...
Generated 10 live streams
Generating auction rules...
Generated 20 auction rules
Generating orders...
Generated 30 orders
Seed data generation completed in 1.234s
Summary:
  - Categories: 8
  - Users: 50 (Admin: 5, Streamer: 10, User: rest)
  - Products: 30 (Draft: 9, Published: 18, Unpublished: 3)
  - Live Streams: 10
  - Auction Rules: 20
  - Orders: 30 (Pending: 6, Paid: 9, Shipped: 9, Completed: 6)
```