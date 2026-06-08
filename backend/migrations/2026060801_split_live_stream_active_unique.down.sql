ALTER TABLE auctions
  ADD COLUMN active_live_stream_key BIGINT AS
    (CASE WHEN status IN (0,1,2) THEN live_stream_id ELSE NULL END) STORED,
  ADD UNIQUE KEY uk_active_live_stream (active_live_stream_key);

ALTER TABLE auctions
  DROP INDEX uk_running_live_stream,
  DROP INDEX uk_pending_live_stream,
  DROP COLUMN running_live_stream_key,
  DROP COLUMN pending_live_stream_key;
