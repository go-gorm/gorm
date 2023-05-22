module gorm.io/gorm/tests

go 1.16

require (
	github.com/google/uuid v1.3.0
	github.com/jinzhu/now v1.1.5
	github.com/lib/pq v1.10.8
	github.com/mattn/go-sqlite3 v1.14.16 // indirect
	gorm.io/driver/mysql v1.5.0
	gorm.io/driver/postgres v1.5.2
	gorm.io/driver/sqlite v1.5.0
	gorm.io/driver/sqlserver v1.4.3
	gorm.io/gorm v1.25.0
)

replace gorm.io/gorm => ../
