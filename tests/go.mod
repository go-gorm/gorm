module gorm.io/gorm/tests

go 1.16

require (
	github.com/google/uuid v1.3.0
	github.com/jinzhu/now v1.1.5
	github.com/lib/pq v1.10.7
	golang.org/x/crypto v0.0.0-20221012134737-56aed061732a // indirect
	golang.org/x/text v0.4.0 // indirect
	gorm.io/driver/mysql v1.4.3
	gorm.io/driver/postgres v1.4.5
	gorm.io/driver/sqlite v1.4.3
	gorm.io/driver/sqlserver v1.4.1
	gorm.io/gorm v1.24.1-0.20221019064659-5dd2bb482755
)

replace gorm.io/gorm => ../
