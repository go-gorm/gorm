module gorm.io/gorm/tests

go 1.14

require (
	github.com/google/uuid v1.3.0
	github.com/jinzhu/now v1.1.2
	github.com/lib/pq v1.10.3
	github.com/mattn/go-sqlite3 v1.14.9 // indirect
	gorm.io/driver/mysql v1.1.2
	gorm.io/driver/postgres v1.1.2
	gorm.io/driver/sqlite v1.1.6
	gorm.io/driver/sqlserver v1.1.0
	gorm.io/gorm v1.21.16
)

replace gorm.io/gorm => ../
