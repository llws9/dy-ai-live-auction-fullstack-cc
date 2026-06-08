ALTER TABLE auctions
  ADD COLUMN pending_live_stream_key BIGINT AS
    (CASE WHEN status = 0 THEN live_stream_id ELSE NULL END) STORED,
  ADD COLUMN running_live_stream_key BIGINT AS
    (CASE WHEN status IN (1,2) THEN live_stream_id ELSE NULL END) STORED,
  ADD UNIQUE KEY uk_pending_live_stream (pending_live_stream_key),
  ADD UNIQUE KEY uk_running_live_stream (running_live_stream_key);

ALTER TABLE auctions
  DROP INDEX uk_active_live_stream,
  DROP COLUMN active_live_stream_key;
