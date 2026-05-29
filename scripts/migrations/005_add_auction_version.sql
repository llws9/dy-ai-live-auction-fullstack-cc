-- Add version field for optimistic locking to auctions table
-- This prevents concurrent update issues (Lost Update problem)

ALTER TABLE auctions ADD COLUMN version INT NOT NULL DEFAULT 0 COMMENT 'Optimistic lock version number';