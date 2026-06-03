-- T4: align admin live stream fields and control metadata.
ALTER TABLE live_streams
  ADD COLUMN IF NOT EXISTS streamer_name VARCHAR(128) DEFAULT '',
  ADD COLUMN IF NOT EXISTS streamer_avatar VARCHAR(255) DEFAULT '',
  ADD COLUMN IF NOT EXISTS viewer_count INT DEFAULT 0,
  ADD COLUMN IF NOT EXISTS ban_reason VARCHAR(255) NULL;

UPDATE live_streams
SET streamer_name = name
WHERE streamer_name IS NULL OR streamer_name = '';
