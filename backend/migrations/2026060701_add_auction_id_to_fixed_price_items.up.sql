ALTER TABLE fixed_price_items
  ADD COLUMN auction_id BIGINT NOT NULL DEFAULT 0 AFTER id,
  ADD INDEX idx_fixed_price_items_auction_id (auction_id);

UPDATE fixed_price_items AS f
JOIN (
  SELECT live_stream_id, MAX(id) AS auction_id
  FROM auctions
  WHERE live_stream_id IS NOT NULL
    AND status IN (0,1,2)
  GROUP BY live_stream_id
) AS active_auctions
  ON active_auctions.live_stream_id = f.live_stream_id
SET f.auction_id = active_auctions.auction_id
WHERE f.auction_id = 0;
