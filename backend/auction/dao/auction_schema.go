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

// EnsureAuctionLiveStreamUniqueIndexes enforces at most one pending and one running
// auction per live stream.
func EnsureAuctionLiveStreamUniqueIndexes(db *gorm.DB) error {
	if db == nil || db.Dialector.Name() != "mysql" {
		return nil
	}

	if err := ensureAuctionGeneratedColumn(db, "pending_live_stream_key", `
		ALTER TABLE auctions
		  ADD COLUMN pending_live_stream_key BIGINT AS
		    (CASE WHEN status = 0 THEN live_stream_id ELSE NULL END) STORED
	`); err != nil {
		return err
	}
	if err := ensureAuctionGeneratedColumn(db, "running_live_stream_key", `
		ALTER TABLE auctions
		  ADD COLUMN running_live_stream_key BIGINT AS
		    (CASE WHEN status IN (1,2) THEN live_stream_id ELSE NULL END) STORED
	`); err != nil {
		return err
	}
	if err := ensureAuctionUniqueIndex(db, "uk_pending_live_stream", `
		ALTER TABLE auctions
		  ADD UNIQUE KEY uk_pending_live_stream (pending_live_stream_key)
	`); err != nil {
		return err
	}
	if err := ensureAuctionUniqueIndex(db, "uk_running_live_stream", `
		ALTER TABLE auctions
		  ADD UNIQUE KEY uk_running_live_stream (running_live_stream_key)
	`); err != nil {
		return err
	}

	if err := dropAuctionIndexIfExists(db, "uk_active_live_stream"); err != nil {
		return err
	}
	return dropAuctionColumnIfExists(db, "active_live_stream_key")
}

func ensureAuctionGeneratedColumn(db *gorm.DB, name, ddl string) error {
	var columnCount int64
	if err := db.Raw(`
		SELECT COUNT(*)
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE()
		  AND TABLE_NAME = 'auctions'
		  AND COLUMN_NAME = ?
	`, name).Scan(&columnCount).Error; err != nil {
		return err
	}
	if columnCount > 0 {
		return nil
	}
	if err := db.Exec(ddl).Error; err != nil && !isDuplicateSchemaError(err) {
		return err
	}
	return nil
}

func ensureAuctionUniqueIndex(db *gorm.DB, name, ddl string) error {
	var indexCount int64
	if err := db.Raw(`
		SELECT COUNT(*)
		FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = DATABASE()
		  AND TABLE_NAME = 'auctions'
		  AND INDEX_NAME = ?
	`, name).Scan(&indexCount).Error; err != nil {
		return err
	}
	if indexCount > 0 {
		return nil
	}
	if err := db.Exec(ddl).Error; err != nil && !isDuplicateSchemaError(err) {
		return err
	}
	return nil
}

func dropAuctionIndexIfExists(db *gorm.DB, name string) error {
	var indexCount int64
	if err := db.Raw(`
		SELECT COUNT(*)
		FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = DATABASE()
		  AND TABLE_NAME = 'auctions'
		  AND INDEX_NAME = ?
	`, name).Scan(&indexCount).Error; err != nil {
		return err
	}
	if indexCount == 0 {
		return nil
	}
	if err := db.Exec("ALTER TABLE auctions DROP INDEX " + name).Error; err != nil && !isDuplicateSchemaError(err) {
		return err
	}
	return nil
}

func dropAuctionColumnIfExists(db *gorm.DB, name string) error {
	var columnCount int64
	if err := db.Raw(`
		SELECT COUNT(*)
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE()
		  AND TABLE_NAME = 'auctions'
		  AND COLUMN_NAME = ?
	`, name).Scan(&columnCount).Error; err != nil {
		return err
	}
	if columnCount == 0 {
		return nil
	}
	if err := db.Exec("ALTER TABLE auctions DROP COLUMN " + name).Error; err != nil && !isDuplicateSchemaError(err) {
		return err
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
