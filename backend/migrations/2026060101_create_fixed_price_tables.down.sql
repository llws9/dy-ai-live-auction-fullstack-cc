-- backend/migrations/2026060101_create_fixed_price_tables.down.sql
ALTER TABLE orders DROP COLUMN source;
DROP TABLE fixed_price_purchases;
DROP TABLE fixed_price_items;
