module gorm.io/gorm/tests

go 1.16

require (
	github.com/google/uuid v1.3.0
	github.com/jackc/pgx/v5 v5.3.1 // indirect
	github.com/jinzhu/now v1.1.5
	github.com/lib/pq v1.10.7
	github.com/mattn/go-sqlite3 v1.14.16 // indirect
	github.com/microsoft/go-mssqldb v0.20.0 // indirect
	golang.org/x/crypto v0.7.0 // indirect
	gorm.io/driver/mysql v1.4.7
	gorm.io/driver/postgres v1.5.0
	gorm.io/driver/sqlite v1.4.4
	gorm.io/driver/sqlserver v1.4.2
	gorm.io/gorm v1.24.7-0.20230306060331-85eaf9eeda11
)

replace gorm.io/gorm => ../
