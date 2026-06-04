ALTER TABLE products
  ADD COLUMN owner_id BIGINT NULL COMMENT 'merchant user id owning this product' AFTER id,
  ADD INDEX idx_products_owner_id (owner_id),
  ADD INDEX idx_products_owner_status_created (owner_id, status, created_at);

ALTER TABLE orders
  ADD COLUMN seller_id BIGINT NULL COMMENT 'merchant user id owning the sold product at order creation time' AFTER product_id,
  ADD INDEX idx_orders_seller_id (seller_id),
  ADD INDEX idx_orders_seller_status_created (seller_id, status, created_at);

ALTER TABLE auctions
  ADD COLUMN creator_id BIGINT NULL COMMENT 'merchant user id creating this auction' AFTER live_stream_id,
  ADD INDEX idx_auctions_creator_id (creator_id),
  ADD INDEX idx_auctions_creator_status_created (creator_id, status, created_at);

UPDATE orders o
JOIN products p ON p.id = o.product_id
SET o.seller_id = p.owner_id
WHERE o.seller_id IS NULL AND p.owner_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS auction_rule_templates (
  id BIGINT NOT NULL AUTO_INCREMENT,
  owner_id BIGINT NOT NULL COMMENT 'merchant user id owning this template',
  name VARCHAR(128) NOT NULL,
  start_price DECIMAL(10,2) NOT NULL DEFAULT 0.00,
  increment DECIMAL(10,2) NOT NULL,
  cap_price DECIMAL(10,2) NULL,
  duration INT NOT NULL,
  delay_duration INT NOT NULL DEFAULT 30,
  max_delay_time INT NOT NULL DEFAULT 180,
  trigger_delay_before INT NOT NULL DEFAULT 30,
  is_default TINYINT(1) NOT NULL DEFAULT 0,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  KEY idx_rule_templates_owner_id (owner_id),
  KEY idx_rule_templates_owner_default (owner_id, is_default),
  UNIQUE KEY uniq_rule_templates_owner_name (owner_id, name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
