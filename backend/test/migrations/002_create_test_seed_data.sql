-- 测试期间创建的业务数据 ref，用于结束后清理
CREATE TABLE IF NOT EXISTS test_seed_data (
    id        BIGINT AUTO_INCREMENT PRIMARY KEY,
    test_id   VARCHAR(36) NOT NULL,
    kind      VARCHAR(20) NOT NULL COMMENT 'product/auction/user/live_stream/order',
    ref_id    BIGINT      NOT NULL COMMENT '业务表主键',
    created_at TIMESTAMP  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_test_id (test_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='测试种子数据 ref，用于清理';
