package dao

import (
	"strings"

	"gorm.io/gorm"
)

// EnsureAuctionActiveProductUniqueIndex adds the MySQL generated column and unique
// index that enforce one active auction per product.
func EnsureAuctionActiveProductUniqueIndex(db *gorm.DB) error {
	if db == nil || db.Dialector.Name() != "mysql" {
		return nil
	}

	var columnCount int64
	if err := db.Raw(`
		SELECT COUNT(*)
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE()
		  AND TABLE_NAME = 'auctions'
		  AND COLUMN_NAME = 'active_product_key'
	`).Scan(&columnCount).Error; err != nil {
		return err
	}
	if columnCount == 0 {
		if err := db.Exec(`
			ALTER TABLE auctions
			  ADD COLUMN active_product_key BIGINT AS
			    (CASE WHEN status IN (0,1,2) THEN product_id ELSE NULL END) STORED
		`).Error; err != nil && !isDuplicateSchemaError(err) {
			return err
		}
	}

	var indexCount int64
	if err := db.Raw(`
		SELECT COUNT(*)
		FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = DATABASE()
		  AND TABLE_NAME = 'auctions'
		  AND INDEX_NAME = 'uk_active_product'
	`).Scan(&indexCount).Error; err != nil {
		return err
	}
	if indexCount == 0 {
		if err := db.Exec(`
			ALTER TABLE auctions
			  ADD UNIQUE KEY uk_active_product (active_product_key)
		`).Error; err != nil && !isDuplicateSchemaError(err) {
			return err
		}
	}

	return nil
}

// EnsureAuctionActiveLiveStreamUniqueIndex enforces one active auction per live stream.
func EnsureAuctionActiveLiveStreamUniqueIndex(db *gorm.DB) error {
	if db == nil || db.Dialector.Name() != "mysql" {
		return nil
	}

	var columnCount int64
	if err := db.Raw(`
		SELECT COUNT(*)
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE()
		  AND TABLE_NAME = 'auctions'
		  AND COLUMN_NAME = 'active_live_stream_key'
	`).Scan(&columnCount).Error; err != nil {
		return err
	}
	if columnCount == 0 {
		if err := db.Exec(`
			ALTER TABLE auctions
			  ADD COLUMN active_live_stream_key BIGINT AS
			    (CASE WHEN status IN (0,1,2) THEN live_stream_id ELSE NULL END) STORED
		`).Error; err != nil && !isDuplicateSchemaError(err) {
			return err
		}
	}

	var indexCount int64
	if err := db.Raw(`
		SELECT COUNT(*)
		FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = DATABASE()
		  AND TABLE_NAME = 'auctions'
		  AND INDEX_NAME = 'uk_active_live_stream'
	`).Scan(&indexCount).Error; err != nil {
		return err
	}
	if indexCount == 0 {
		if err := db.Exec(`
			ALTER TABLE auctions
			  ADD UNIQUE KEY uk_active_live_stream (active_live_stream_key)
		`).Error; err != nil && !isDuplicateSchemaError(err) {
			return err
		}
	}

	return nil
}

func isDuplicateSchemaError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate column") ||
		strings.Contains(msg, "duplicate key name") ||
		strings.Contains(msg, "error 1060") ||
		strings.Contains(msg, "error 1061")
}
