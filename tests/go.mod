module gorm.io/gorm/tests

go 1.16

require (
	github.com/go-sql-driver/mysql v1.7.0 // indirect
	github.com/google/uuid v1.3.0
	github.com/jinzhu/now v1.1.5
	github.com/lib/pq v1.10.7
	github.com/mattn/go-sqlite3 v1.14.16 // indirect
	github.com/microsoft/go-mssqldb v0.19.0 // indirect
	gorm.io/driver/mysql v1.4.4
	gorm.io/driver/postgres v1.4.6
	gorm.io/driver/sqlite v1.4.4
	gorm.io/driver/sqlserver v1.4.1
	gorm.io/gorm v1.24.2
)

replace gorm.io/gorm => ../
