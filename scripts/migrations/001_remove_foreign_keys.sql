-- 移除物理外键约束，改为逻辑外键
-- 执行时间: 2026-05-21
-- 目的: 优化并发性能，支持更灵活的测试和生产环境

-- 1. 移除 bids 表的外键约束
ALTER TABLE bids DROP FOREIGN KEY IF EXISTS bids_ibfk_2;  -- user_id 外键

-- 2. 移除 auctions 表的 winner_id 外键
ALTER TABLE auctions DROP FOREIGN KEY IF EXISTS auctions_ibfk_2;  -- winner_id 外键

-- 3. 确保索引存在以保持查询性能
-- 如果索引不存在则创建（不使用 IF NOT EXISTS，使用存储过程判断）
CREATE INDEX IF NOT EXISTS idx_bids_user_id ON bids(user_id);
CREATE INDEX IF NOT EXISTS idx_auctions_winner_id ON auctions(winner_id);

-- 说明：
-- 1. 移除外键后，应用层需要在出价前校验用户是否存在
-- 2. 索引保留，确保按 user_id 和 winner_id 查询的性能
-- 3. 数据完整性由应用层保证（UserDAO.Exists 方法）

-- 验证脚本
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
