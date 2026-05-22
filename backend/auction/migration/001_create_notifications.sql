-- Migration: Create notifications table
-- Created: 2026-05-22
-- Feature: MVP阶段功能完善 - 消息通知系统

CREATE TABLE IF NOT EXISTS `notifications` (
    `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键',
    `user_id` bigint NOT NULL COMMENT '接收用户ID',
    `type` varchar(32) NOT NULL COMMENT '通知类型: bid_outbid, auction_won, auction_lost, order_paid, order_shipped, order_completed',
    `title` varchar(128) NOT NULL COMMENT '通知标题',
    `content` text NOT NULL COMMENT '通知内容',
    `data` json DEFAULT NULL COMMENT '扩展数据(auction_id, order_id等)',
    `read_at` datetime DEFAULT NULL COMMENT '已读时间，NULL表示未读',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    PRIMARY KEY (`id`),
    INDEX `idx_user_id_created_at` (`user_id`, `created_at` DESC),
    INDEX `idx_user_id_read_at` (`user_id`, `read_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户通知表';
