module gorm.io/gorm/tests

go 1.14

require (
	github.com/google/uuid v1.3.0
	github.com/jinzhu/now v1.1.3
	github.com/lib/pq v1.10.4
	gorm.io/driver/mysql v1.1.3
	gorm.io/driver/postgres v1.2.2
	gorm.io/driver/sqlite v1.2.4
	gorm.io/driver/sqlserver v1.2.1
	gorm.io/gorm v1.22.2
)

replace gorm.io/gorm => ../
