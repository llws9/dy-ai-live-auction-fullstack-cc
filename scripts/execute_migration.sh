#!/bin/bash

# 一键执行数据库迁移

MIGRATION_FILE="/Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/scripts/migrations/001_remove_foreign_keys.sql"

echo "====================================="
echo "   外键优化 - 数据库迁移"
echo "====================================="
echo ""
echo "即将执行的迁移："
echo "  1. 移除bids表的user_id外键"
echo "  2. 移除auctions表的winner_id外键"
echo "  3. 创建索引以保证查询性能"
echo ""
echo "⚠️  警告：此操作将修改数据库结构"
echo ""

read -p "确认执行迁移？(yes/no): " CONFIRM

if [ "$CONFIRM" != "yes" ]; then
  echo "已取消迁移"
  exit 0
fi

echo ""
read -sp "请输入MySQL root密码: " DB_PASSWORD
echo ""

echo ""
echo "开始执行迁移..."

# 执行迁移
mysql -u root -p"$DB_PASSWORD" auction <<'EOF'
-- 移除bids表的user_id外键
ALTER TABLE bids DROP FOREIGN KEY IF EXISTS bids_ibfk_2;

-- 移除auctions表的winner_id外键
ALTER TABLE auctions DROP FOREIGN KEY IF EXISTS auctions_ibfk_2;

-- 创建索引（如果不存在）
CREATE INDEX IF NOT EXISTS idx_bids_user_id ON bids(user_id);
CREATE INDEX IF NOT EXISTS idx_auctions_winner_id ON auctions(winner_id);

-- 验证结果
SELECT
    'bids表外键' as table_name,
    COUNT(*) as foreign_key_count
FROM information_schema.TABLE_CONSTRAINTS
WHERE TABLE_SCHEMA = 'auction'
  AND TABLE_NAME = 'bids'
  AND CONSTRAINT_TYPE = 'FOREIGN KEY'

UNION ALL

SELECT
    'auctions表外键' as table_name,
    COUNT(*) as foreign_key_count
FROM information_schema.TABLE_CONSTRAINTS
WHERE TABLE_SCHEMA = 'auction'
  AND TABLE_NAME = 'auctions'
  AND CONSTRAINT_TYPE = 'FOREIGN KEY';
EOF

if [ $? -eq 0 ]; then
  echo ""
  echo "✅ 迁移执行成功"
  echo ""
  echo "下一步："
  echo "  1. 重启auction服务"
  echo "  2. 执行 ./scripts/verify_fk_optimization.sh 验证优化效果"
  echo "  3. 执行 ./scripts/concurrent_bid_test_optimized.sh 进行性能测试"
else
  echo ""
  echo "❌ 迁移执行失败，请检查数据库连接"
fi

echo ""
echo "====================================="
