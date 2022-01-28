module gorm.io/gorm/tests

go 1.14

require (
	github.com/denisenkom/go-mssqldb v0.12.0 // indirect
	github.com/google/uuid v1.3.0
	github.com/jackc/pgx/v4 v4.14.1 // indirect
	github.com/jinzhu/now v1.1.4
	github.com/lib/pq v1.10.4
	github.com/mattn/go-sqlite3 v1.14.10 // indirect
	golang.org/x/crypto v0.0.0-20220126234351-aa10faf2a1f8 // indirect
	gorm.io/driver/mysql v1.2.3
	gorm.io/driver/postgres v1.2.3
	gorm.io/driver/sqlite v1.2.6
	gorm.io/driver/sqlserver v1.2.1
	gorm.io/gorm v1.22.4
)

replace gorm.io/gorm => ../
