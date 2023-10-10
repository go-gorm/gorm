module gorm.io/gorm/tests

go 1.18

require (
	github.com/google/uuid v1.3.1
	github.com/jinzhu/now v1.1.5
	github.com/lib/pq v1.10.9
	gorm.io/driver/mysql v1.5.2-0.20230612053416-48b6526a21f0
	gorm.io/driver/postgres v1.5.3-0.20230607070428-18bc84b75196
	gorm.io/driver/sqlite v1.5.3
	gorm.io/driver/sqlserver v1.5.2-0.20230613072041-6e2cde390b0a
	gorm.io/gorm v1.25.3
)

require (
	github.com/go-sql-driver/mysql v1.7.1 // indirect
	github.com/golang-sql/civil v0.0.0-20220223132316-b832511892a9 // indirect
	github.com/golang-sql/sqlexp v0.1.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/pgx/v5 v5.4.3 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/mattn/go-sqlite3 v1.14.17 // indirect
	github.com/microsoft/go-mssqldb v1.5.0 // indirect
	golang.org/x/crypto v0.12.0 // indirect
	golang.org/x/text v0.12.0 // indirect
)

replace gorm.io/gorm => ../
