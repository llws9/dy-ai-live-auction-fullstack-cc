ALTER TABLE fixed_price_items
  DROP INDEX idx_fixed_price_items_auction_id,
  DROP COLUMN auction_id;
