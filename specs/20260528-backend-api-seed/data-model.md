# Data Model: 后端API补充与测试数据生成

**Feature**: `20260528-backend-api-seed`
**Date**: 2026-05-28

## Entity Definitions

---

## Category (新增)

商品类别实体，支持动态增删改。

| 字段 | 类型 | GORM Tag | 说明 |
|------|------|----------|------|
| id | int64 | `primaryKey;autoIncrement` | 主键 |
| name | string | `type:varchar(64);not null` | 类别名称 |
| code | string | `type:varchar(32);uniqueIndex` | 类别代码（唯一） |
| description | string | `type:text` | 类别描述 |
| sort_order | int | `default:0` | 排序顺序 |
| status | int | `type:tinyint;default:1` | 状态：1=启用, 0=禁用 |
| created_at | time.Time | `autoCreateTime` | 创建时间 |
| updated_at | time.Time | `autoUpdateTime` | 更新时间 |

### 验证规则
- name: 非空，最大64字符
- code: 非空，最大32字符，唯一
- status: 仅允许 0 或 1

### 状态转换
```
启用(1) ←→ 禁用(0)
禁用状态不显示在商品选择列表
```

---

## Product (修改)

商品实体，新增category_id字段。

| 字段 | 类型 | GORM Tag | 说明 |
|------|------|----------|------|
| id | int64 | `primaryKey;autoIncrement` | 主键 |
| name | string | `type:varchar(128);not null` | 商品名称 |
| description | string | `type:text` | 商品描述 |
| images | JSONArray | `type:json` | 商品图片数组 |
| **category_id** | *int64 | `index` | **[新增] 逻辑外键关联Category** |
| status | int | `type:tinyint;default:0` | 状态：0=草稿, 1=已发布, 2=已下架 |
| created_at | time.Time | `autoCreateTime` | 创建时间 |

### 关联关系
- Category: 逻辑外键，无物理约束
- 删除Category时需检查Product.category_id是否有关联

---

## LiveStream (现有)

直播间实体，无结构变更。

| 字段 | 类型 | GORM Tag | 说明 |
|------|------|----------|------|
| id | int64 | `primaryKey;autoIncrement` | 主键 |
| creator_id | int64 | `uniqueIndex;not null` | 商家ID |
| name | string | `type:varchar(128);not null` | 直播间名称 |
| description | string | `type:text` | 直播间描述 |
| cover_image | string | `type:varchar(256)` | 封面图URL |
| status | int | `type:tinyint;default:1` | 状态：0=禁用, 1=正常 |
| created_at | time.Time | `autoCreateTime` | 创建时间 |
| updated_at | time.Time | `autoUpdateTime` | 更新时间 |

---

## Seed数据配置

### 类别初始数据
| code | name | sort_order |
|------|------|------------|
| art | 艺术收藏 | 1 |
| jewelry | 珠宝名表 | 2 |
| digital | 数码电子 | 3 |
| luxury | 奢侈品 | 4 |
| fashion | 时尚服饰 | 5 |
| home | 家居生活 | 6 |

### 用户角色分布
| 角色 | 数量 | 权限 |
|------|------|------|
| 买家(Role=0) | 40 | 出价、关注 |
| 商家(Role=1) | 8 | 商品管理、发货 |
| 主播(Role=1) | 3 | 竞拍创建、取消 |
| 管理员(Role=2) | 2 | 全权限 |

### 竞拍状态分布
| 状态 | 数量 | 说明 |
|------|------|------|
| 待开始(0) | 5 | start_time在未来 |
| 进行中(1) | 3 | 当前时间在竞拍时段内 |
| 延时中(2) | 2 | 触发延时机制 |
| 已结束(3) | 8 | 竞拍完成，有winner |
| 已取消(4) | 2 | 手动取消 |

---

## ER Diagram (简化)

```
Category ←(逻辑关联)→ Product ←→ Auction ←→ Bid
                              ↓
                          LiveStream ←→ User(商家)
                              ↓
                           Order ←→ User(买家)
```