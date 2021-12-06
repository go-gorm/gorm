module gorm.io/gorm/tests

go 1.14

require (
	github.com/google/uuid v1.3.0
	github.com/jackc/pgx/v4 v4.14.1 // indirect
	github.com/jinzhu/now v1.1.4
	github.com/lib/pq v1.10.4
	golang.org/x/crypto v0.0.0-20211117183948-ae814b36b871 // indirect
	gorm.io/driver/mysql v1.2.0
	gorm.io/driver/postgres v1.2.3
	gorm.io/driver/sqlite v1.2.6
	gorm.io/driver/sqlserver v1.2.1
	gorm.io/gorm v1.22.3
)

replace gorm.io/gorm => ../
