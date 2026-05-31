CREATE TABLE IF NOT EXISTS live_stream_reminder_receipts (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id BIGINT NOT NULL,
  live_stream_id BIGINT NOT NULL,
  live_started_at BIGINT NOT NULL,
  reminded_at DATETIME NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_user_stream_started (user_id, live_stream_id, live_started_at),
  KEY idx_user_id (user_id)
);
