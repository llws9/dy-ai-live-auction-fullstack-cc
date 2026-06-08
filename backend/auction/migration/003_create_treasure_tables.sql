CREATE TABLE IF NOT EXISTS user_coins (
  user_id    BIGINT   NOT NULL PRIMARY KEY,
  balance    BIGINT   NOT NULL DEFAULT 0,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS user_watch_duration (
  user_id       BIGINT      NOT NULL,
  stat_date     VARCHAR(10) NOT NULL,
  total_seconds INT         NOT NULL DEFAULT 0,
  updated_at    DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (user_id, stat_date)
);

CREATE TABLE IF NOT EXISTS treasure_claims (
  user_id    BIGINT      NOT NULL,
  stat_date  VARCHAR(10) NOT NULL,
  tier       TINYINT     NOT NULL,
  coins      BIGINT      NOT NULL,
  claimed_at DATETIME    NOT NULL,
  PRIMARY KEY (user_id, stat_date, tier)
);
