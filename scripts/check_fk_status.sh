#!/bin/bash

# 检查数据库外键状态

echo "====================================="
echo "   数据库外键状态检查"
echo "====================================="
echo ""

# 提示输入数据库密码
read -sp "请输入MySQL root密码: " DB_PASSWORD
echo ""
echo ""

# 检查bids表的外键
echo "【1】检查bids表的外键约束..."
BIDS_FK=$(mysql -u root -p"$DB_PASSWORD" -D auction -e "
SELECT COUNT(*) as count
FROM information_schema.TABLE_CONSTRAINTS
WHERE TABLE_SCHEMA = 'auction'
  AND TABLE_NAME = 'bids'
  AND CONSTRAINT_TYPE = 'FOREIGN KEY';
" 2>/dev/null | tail -1)

if [ "$BIDS_FK" == "0" ]; then
  echo "✅ bids表的外键已移除"
else
  echo "⚠️  bids表仍有 $BIDS_FK 个外键约束"
fi
echo ""

# 检查auctions表的外键
echo "【2】检查auctions表的外键约束..."
AUCTIONS_FK=$(mysql -u root -p"$DB_PASSWORD" -D auction -e "
SELECT COUNT(*) as count
FROM information_schema.TABLE_CONSTRAINTS
WHERE TABLE_SCHEMA = 'auction'
  AND TABLE_NAME = 'auctions'
  AND CONSTRAINT_TYPE = 'FOREIGN KEY';
" 2>/dev/null | tail -1)

if [ "$AUCTIONS_FK" == "0" ]; then
  echo "✅ auctions表的外键已移除"
else
  echo "⚠️  auctions表仍有 $AUCTIONS_FK 个外键约束"
fi
echo ""

# 检查索引是否存在
echo "【3】检查索引状态..."
mysql -u root -p"$DB_PASSWORD" -D auction -e "
SHOW INDEX FROM bids WHERE Key_name = 'idx_bids_user_id';
" 2>/dev/null | grep -q "idx_bids_user_id"

if [ $? -eq 0 ]; then
  echo "✅ bids.user_id索引存在"
else
  echo "❌ bids.user_id索引不存在，需要创建"
fi

mysql -u root -p"$DB_PASSWORD" -D auction -e "
SHOW INDEX FROM auctions WHERE Key_name = 'idx_auctions_winner_id';
" 2>/dev/null | grep -q "idx_auctions_winner_id"

if [ $? -eq 0 ]; then
  echo "✅ auctions.winner_id索引存在"
else
  echo "❌ auctions.winner_id索引不存在，需要创建"
fi
echo ""

echo "====================================="
echo "   检查完成"
echo "====================================="
