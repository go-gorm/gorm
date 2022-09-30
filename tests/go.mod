module gorm.io/gorm/tests

go 1.16

require (
	github.com/denisenkom/go-mssqldb v0.12.2 // indirect
	github.com/golang-sql/civil v0.0.0-20220223132316-b832511892a9 // indirect
	github.com/google/uuid v1.3.0
	github.com/jinzhu/now v1.1.5
	github.com/lib/pq v1.10.7
	github.com/mattn/go-sqlite3 v1.14.15 // indirect
	golang.org/x/crypto v0.0.0-20220926161630-eccd6366d1be // indirect
	gorm.io/driver/mysql v1.3.6
	gorm.io/driver/postgres v1.3.10
	gorm.io/driver/sqlite v1.3.6
	gorm.io/driver/sqlserver v1.3.2
	gorm.io/gorm v1.23.10
)

replace gorm.io/gorm => ../
