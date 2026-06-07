ALTER TABLE auctions
  DROP INDEX uk_active_live_stream,
  DROP COLUMN active_live_stream_key;
