-- 用户商品提醒订阅表
-- 用于存储用户订阅商品竞拍提醒的关系

CREATE TABLE IF NOT EXISTS user_product_reminders (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL COMMENT '用户ID',
    product_id BIGINT NOT NULL COMMENT '商品ID',
    auction_id BIGINT COMMENT '关联的竞拍ID',
    notification_enabled BOOLEAN DEFAULT TRUE COMMENT '是否接收通知',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '订阅时间',
    INDEX idx_user_product (user_id, product_id),
    INDEX idx_product (product_id),
    INDEX idx_auction (auction_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户商品提醒订阅表';