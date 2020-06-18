module gorm.io/gorm/tests

go 1.14

require (
	github.com/google/uuid v1.1.1
	github.com/jinzhu/now v1.1.1
	github.com/lib/pq v1.6.0
	gorm.io/driver/mysql v0.2.0
	gorm.io/driver/postgres v0.2.0
	gorm.io/driver/sqlite v1.0.2
	gorm.io/driver/sqlserver v0.2.0
	gorm.io/gorm v0.0.0-00010101000000-000000000000
)

replace gorm.io/gorm => ../
