module seed

go 1.24.5

require (
	gorm.io/gorm v1.31.1
	product-service v0.0.0
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/go-sql-driver/mysql v1.8.1 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	golang.org/x/text v0.21.0 // indirect
	gorm.io/driver/mysql v1.6.0 // indirect
)

replace product-service => ../product
