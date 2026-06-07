ALTER TABLE auctions
  ADD COLUMN active_live_stream_key BIGINT AS
    (CASE WHEN status IN (0,1,2) THEN live_stream_id ELSE NULL END) STORED,
  ADD UNIQUE KEY uk_active_live_stream (active_live_stream_key);
