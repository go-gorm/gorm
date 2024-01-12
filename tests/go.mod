module gorm.io/gorm/tests

go 1.18

require (
	github.com/google/uuid v1.5.0
	github.com/jinzhu/now v1.1.5
	github.com/lib/pq v1.10.9
	gorm.io/driver/mysql v1.5.2
	gorm.io/driver/postgres v1.5.4
	gorm.io/driver/sqlite v1.5.4
	gorm.io/driver/sqlserver v1.5.2
	gorm.io/gorm v1.25.5
)

require (
	github.com/go-sql-driver/mysql v1.7.1 // indirect
	github.com/golang-sql/civil v0.0.0-20220223132316-b832511892a9 // indirect
	github.com/golang-sql/sqlexp v0.1.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20231201235250-de7065d80cb9 // indirect
	github.com/jackc/pgx/v5 v5.5.1 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/mattn/go-sqlite3 v1.14.19 // indirect
	github.com/microsoft/go-mssqldb v1.6.0 // indirect
	golang.org/x/crypto v0.18.0 // indirect
	golang.org/x/text v0.14.0 // indirect
)

replace gorm.io/gorm => ../

replace github.com/jackc/pgx/v5 => github.com/jackc/pgx/v5 v5.4.3
