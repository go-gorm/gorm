module gorm.io/gorm/tests

go 1.14

require (
	github.com/google/uuid v1.1.1
	github.com/jinzhu/now v1.1.1
	github.com/lib/pq v1.6.0
	gorm.io/driver/mysql v1.0.1
	gorm.io/driver/postgres v1.0.0
	gorm.io/driver/sqlite v1.1.1
	gorm.io/driver/sqlserver v1.0.2
	gorm.io/gorm v1.9.19
)

replace gorm.io/gorm => ../

replace github.com/jackc/pgx/v4 => github.com/jinzhu/pgx/v4 v4.8.2
