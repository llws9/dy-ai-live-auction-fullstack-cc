-- backend/migrations/2026060101_create_fixed_price_tables.up.sql
CREATE TABLE fixed_price_items (
  id              BIGINT AUTO_INCREMENT PRIMARY KEY,
  auction_id      BIGINT NOT NULL DEFAULT 0,
  live_stream_id  BIGINT NOT NULL,
  product_id      BIGINT NOT NULL,
  creator_id      BIGINT NOT NULL,
  price           DECIMAL(10,2) NOT NULL,
  total_stock     INT NOT NULL,
  remaining_stock INT NOT NULL,
  max_per_user    INT NOT NULL DEFAULT 1,
  status          TINYINT NOT NULL DEFAULT 1,
  version         INT NOT NULL DEFAULT 0,
  created_at      DATETIME NOT NULL,
  updated_at      DATETIME NOT NULL,
  INDEX idx_auction_id (auction_id),
  INDEX idx_live_stream (live_stream_id, status),
  INDEX idx_creator (creator_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE fixed_price_purchases (
  id         BIGINT AUTO_INCREMENT PRIMARY KEY,
  item_id    BIGINT NOT NULL,
  user_id    BIGINT NOT NULL,
  idempotency_key VARCHAR(64) NOT NULL DEFAULT '',
  price      DECIMAL(10,2) NOT NULL,
  created_at DATETIME NOT NULL,
  UNIQUE KEY uniq_item_user (item_id, user_id),
  INDEX idx_idempotency_key (idempotency_key),
  INDEX idx_user (user_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
