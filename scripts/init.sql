-- 直播竞拍系统数据库初始化脚本
-- Created: 2026-05-21

-- 用户表
CREATE TABLE IF NOT EXISTS users (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(64) NOT NULL COMMENT '用户名',
    avatar VARCHAR(256) NULL COMMENT '头像URL',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户信息表';

-- 商品表
CREATE TABLE IF NOT EXISTS products (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(128) NOT NULL COMMENT '商品名称',
    description TEXT NULL COMMENT '商品描述',
    images JSON NULL COMMENT '商品图片列表',
    status TINYINT DEFAULT 0 COMMENT '状态: 0=draft, 1=published',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='商品信息表';

-- 竞拍场次表
CREATE TABLE IF NOT EXISTS auctions (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    product_id BIGINT NOT NULL COMMENT '关联商品ID',
    status TINYINT DEFAULT 0 COMMENT '状态: 0=pending, 1=ongoing, 2=delayed, 3=ended, 4=cancelled',
    current_price DECIMAL(10,2) DEFAULT 0 COMMENT '当前价格',
    winner_id BIGINT NULL COMMENT '当前领先者ID',
    start_time TIMESTAMP NOT NULL COMMENT '开始时间',
    end_time TIMESTAMP NOT NULL COMMENT '结束时间',
    delay_used INT DEFAULT 0 COMMENT '已延时秒数',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    INDEX idx_product_id (product_id),
    INDEX idx_status (status),
    INDEX idx_start_time (start_time),
    FOREIGN KEY (product_id) REFERENCES products(id),
    FOREIGN KEY (winner_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='竞拍场次表';

-- 竞拍规则表
CREATE TABLE IF NOT EXISTS auction_rules (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    product_id BIGINT NOT NULL COMMENT '关联商品ID',
    start_price DECIMAL(10,2) DEFAULT 0 COMMENT '起拍价（默认0元）',
    increment DECIMAL(10,2) NOT NULL COMMENT '加价幅度',
    cap_price DECIMAL(10,2) NULL COMMENT '封顶价',
    duration INT NOT NULL COMMENT '竞拍时长（秒）',
    delay_duration INT DEFAULT 30 COMMENT '单次延时时长（秒）',
    max_delay_time INT DEFAULT 180 COMMENT '最大延时时长（秒）',
    trigger_delay_before INT DEFAULT 30 COMMENT '延时触发时间（秒）',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    INDEX idx_auction_rules_product_id (product_id),
    FOREIGN KEY (product_id) REFERENCES products(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='竞拍规则表';

-- 出价记录表
CREATE TABLE IF NOT EXISTS bids (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    auction_id BIGINT NOT NULL COMMENT '关联竞拍ID',
    user_id BIGINT NOT NULL COMMENT '出价用户ID',
    amount DECIMAL(10,2) NOT NULL COMMENT '出价金额',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '出价时间',
    INDEX idx_auction_id (auction_id),
    INDEX idx_user_id (user_id),
    INDEX idx_auction_created (auction_id, created_at DESC),
    FOREIGN KEY (auction_id) REFERENCES auctions(id),
    FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='出价记录表';

-- 订单表
CREATE TABLE IF NOT EXISTS orders (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    auction_id BIGINT NOT NULL COMMENT '关联竞拍ID',
    product_id BIGINT NOT NULL COMMENT '关联商品ID',
    winner_id BIGINT NOT NULL COMMENT '中标者ID',
    final_price DECIMAL(10,2) NOT NULL COMMENT '成交价格',
    status TINYINT DEFAULT 0 COMMENT '状态: 0=pending, 1=paid, 2=shipped, 3=completed',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    FOREIGN KEY (auction_id) REFERENCES auctions(id),
    FOREIGN KEY (product_id) REFERENCES products(id),
    FOREIGN KEY (winner_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='订单表';

-- 插入测试用户数据
INSERT INTO users (id, name, avatar) VALUES
(1, '主播小明', 'https://example.com/avatar1.jpg'),
(2, '用户小红', 'https://example.com/avatar2.jpg'),
(3, '用户小刚', 'https://example.com/avatar3.jpg');

-- 插入测试商品数据
INSERT INTO products (id, name, description, status) VALUES
(1, '稀有珠宝', '限量版珠宝，全球仅发售10件', 1);
