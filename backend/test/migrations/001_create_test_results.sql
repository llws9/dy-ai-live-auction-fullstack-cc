-- 测试结果主表
CREATE TABLE IF NOT EXISTS test_results (
    id           VARCHAR(36) PRIMARY KEY,
    test_type    VARCHAR(20) NOT NULL COMMENT 'pressure/concurrent/websocket/skylamp/e2e/antisnipe/chaos/callback/consistency/risk/fairness/reconnect/dummy',
    status       VARCHAR(20) NOT NULL COMMENT 'running/completed/failed/cancelled',
    config_json  TEXT        NOT NULL COMMENT '场景配置 JSON',
    result_json  TEXT                 COMMENT '场景结果 JSON',
    replay_token VARCHAR(64)          COMMENT '复现 token',
    script_name  VARCHAR(64)          COMMENT '剧本名',
    error_msg    TEXT                 COMMENT '失败原因',
    created_at   TIMESTAMP   NOT NULL,
    completed_at TIMESTAMP   NULL,
    INDEX idx_test_type (test_type),
    INDEX idx_status (status),
    INDEX idx_created_at (created_at),
    INDEX idx_replay_token (replay_token)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='测试任务结果记录';
