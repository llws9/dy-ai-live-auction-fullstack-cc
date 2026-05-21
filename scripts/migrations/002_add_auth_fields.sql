-- 用户认证字段迁移脚本
-- Created: 2026-05-22
-- Purpose: 为users表添加认证相关字段，实现JWT认证系统

-- 添加认证相关字段到users表
ALTER TABLE users
ADD COLUMN email VARCHAR(128) NULL UNIQUE COMMENT '邮箱地址',
ADD COLUMN phone VARCHAR(20) NULL UNIQUE COMMENT '手机号码',
ADD COLUMN password VARCHAR(256) NOT NULL DEFAULT '' COMMENT '密码哈希(bcrypt)',
ADD COLUMN role TINYINT DEFAULT 0 COMMENT '角色: 0=普通用户, 1=管理员',
ADD COLUMN status TINYINT DEFAULT 1 COMMENT '状态: 0=禁用, 1=正常',
ADD COLUMN last_login_at TIMESTAMP NULL COMMENT '最后登录时间',
ADD INDEX idx_email (email),
ADD INDEX idx_phone (phone);

-- 创建登录日志表
CREATE TABLE IF NOT EXISTS login_logs (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL COMMENT '用户ID',
    ip VARCHAR(64) NULL COMMENT '登录IP',
    user_agent VARCHAR(256) NULL COMMENT '用户代理',
    login_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '登录时间',
    success TINYINT DEFAULT 1 COMMENT '是否成功: 0=失败, 1=成功',
    INDEX idx_user_id (user_id),
    INDEX idx_login_at (login_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='登录日志表';

-- 创建管理员账号
-- 注意：这里的密码哈希是 'admin123' 使用bcrypt成本因子10生成的
-- 在生产环境中，应该通过应用程序创建管理员账号
INSERT INTO users (id, name, email, password, role, status, created_at) VALUES
(999, '系统管理员', 'admin@example.com', '$2a$10$N9qo8yLOickgx2ZMRZoMyeIjZRGdjGn/n6.Pk7bFkLKHnFZJYQKCe', 1, 1, NOW())
ON DUPLICATE KEY UPDATE
    name = '系统管理员',
    email = 'admin@example.com',
    password = '$2a$10$N9qo8yLOickgx2ZMRZoMyeIjZRGdjGn/n6.Pk7bFkLKHnFZJYQKCe',
    role = 1,
    status = 1;

-- 为现有用户设置默认密码（空密码，需要用户重新设置）
UPDATE users SET password = '' WHERE password IS NULL OR password = '';
