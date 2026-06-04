DROP TABLE IF EXISTS auction_rule_templates;

ALTER TABLE auctions
  DROP INDEX idx_auctions_creator_status_created,
  DROP INDEX idx_auctions_creator_id,
  DROP COLUMN creator_id;

ALTER TABLE orders
  DROP INDEX idx_orders_seller_status_created,
  DROP INDEX idx_orders_seller_id,
  DROP COLUMN seller_id;

ALTER TABLE products
  DROP INDEX idx_products_owner_status_created,
  DROP INDEX idx_products_owner_id,
  DROP COLUMN owner_id;
