-- 点天灯订阅表
-- 用于存储用户点天灯（自动跟价）订阅信息

CREATE TABLE IF NOT EXISTS sky_lamp_subscriptions (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    auction_id BIGINT NOT NULL COMMENT '竞拍ID',
    user_id BIGINT NOT NULL COMMENT '用户ID',
    status TINYINT NOT NULL DEFAULT 1 COMMENT '状态: 1-活跃, 2-已停止, 3-已取消, 4-竞拍结束',
    initial_price DECIMAL(10,2) NOT NULL COMMENT '开启时的当前价格',
    initial_bid_amount DECIMAL(10,2) NOT NULL COMMENT '首次出价金额',
    max_price_limit DECIMAL(10,2) NOT NULL COMMENT '天灯上限金额',
    current_auto_bid_count INT NOT NULL DEFAULT 0 COMMENT '已自动跟价次数',
    total_bid_amount DECIMAL(10,2) NOT NULL DEFAULT 0 COMMENT '累计出价金额',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    stopped_at TIMESTAMP NULL COMMENT '停止时间',
    INDEX idx_auction_user (auction_id, user_id),
    INDEX idx_auction_status (auction_id, status),
    INDEX idx_user_status (user_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='点天灯订阅表';