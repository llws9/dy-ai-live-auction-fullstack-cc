# Seed 数据生成工具

## 用途
快速生成测试和演示数据，支持多种数据规模配置。

## 使用方法

### 基本用法
```bash
go run main.go --size medium
```

### 参数说明
| 参数 | 说明 | 默认值 |
|------|------|--------|
| --size | 数据规模 (small/medium/large) | medium |
| --db-host | 数据库地址 | localhost |
| --db-port | 数据库端口 | 3306 |
| --db-user | 数据库用户 | root |
| --db-password | 数据库密码 | - |
| --db-name | 数据库名 | live_auction |

## 数据规模配置
| 规模 | Users | Products | LiveStreams | AuctionRules | Orders |
|------|-------|----------|-------------|--------------|--------|
| small | 20 | 30 | 10 | 20 | 20 |
| medium | 30 | 50 | 20 | 40 | 40 |
| large | 100 | 200 | 50 | 100 | 100 |

## 生成顺序
1. Categories（类别）
2. Users（用户）
3. Products（商品）
4. LiveStreams（直播间）
5. AuctionRules（竞拍规则）
6. Orders（订单）

## 数据清理
重新运行会追加数据，如需清理请手动删除或使用SQL脚本。