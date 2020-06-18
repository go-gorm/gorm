module gorm.io/gorm/tests

go 1.14

require (
	github.com/google/uuid v1.1.1
	github.com/jinzhu/now v1.1.1
	github.com/lib/pq v1.6.0
	gorm.io/driver/mysql v0.0.0-20200609004954-b8310c61c3f2
	gorm.io/driver/postgres v0.0.0-20200602015520-15fcc29eb286
	gorm.io/driver/sqlite v1.0.0
	gorm.io/driver/sqlserver v0.0.0-20200610080012-25da0c25e81d
	gorm.io/gorm v0.0.0-00010101000000-000000000000
)

replace gorm.io/gorm => ../
