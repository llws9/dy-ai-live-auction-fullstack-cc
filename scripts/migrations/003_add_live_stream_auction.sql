-- Migration: 003_add_live_stream
-- Description: 创建直播间表、用户关注直播间表，并修改竞拍表
-- Date: 2026-05-23
-- Feature: 20260523-product-auction-live
-- Database: auction

-- 创建直播间表
CREATE TABLE IF NOT EXISTS live_streams (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    creator_id BIGINT NOT NULL UNIQUE COMMENT '商家ID，一对一',
    name VARCHAR(128) NOT NULL COMMENT '直播间名称',
    description TEXT COMMENT '直播间描述',
    cover_image VARCHAR(256) COMMENT '封面图',
    status TINYINT DEFAULT 1 COMMENT '状态：0=禁用，1=正常',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='直播间表';

-- 创建用户关注直播间表
CREATE TABLE IF NOT EXISTS user_live_stream_follows (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL COMMENT '用户ID',
    live_stream_id BIGINT NOT NULL COMMENT '直播间ID',
    notification_enabled TINYINT DEFAULT 1 COMMENT '是否接收通知',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '关注时间',
    UNIQUE KEY uk_user_live_stream (user_id, live_stream_id),
    INDEX idx_user_id (user_id),
    INDEX idx_live_stream_id (live_stream_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户关注直播间表';

-- 为auctions表新增live_stream_id字段
ALTER TABLE auctions
ADD COLUMN live_stream_id BIGINT NULL COMMENT '直播间ID' AFTER product_id,
ADD INDEX idx_live_stream_id (live_stream_id);

-- 为products表更新状态注释
ALTER TABLE products
MODIFY COLUMN status TINYINT DEFAULT 0 COMMENT '状态: 0=草稿, 1=已发布, 2=已下架';

-- 为现有商家创建直播间
INSERT INTO live_streams (creator_id, name, description, status, created_at)
SELECT
    id as creator_id,
    CONCAT(name, '的直播间') as name,
    CONCAT(name, '的个人直播间') as description,
    1 as status,
    created_at
FROM users
WHERE role = 1 -- 主播/商家
ON DUPLICATE KEY UPDATE
    name = VALUES(name);

-- 为现有竞拍记录设置live_stream_id
UPDATE auctions a
JOIN live_streams ls ON a.creator_id = ls.creator_id
SET a.live_stream_id = ls.id
WHERE a.live_stream_id IS NULL;
