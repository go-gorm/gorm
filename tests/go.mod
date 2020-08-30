module gorm.io/gorm/tests

go 1.14

require (
	github.com/google/uuid v1.1.1
	github.com/jinzhu/now v1.1.1
	github.com/lib/pq v1.6.0
	github.com/stevefan1999-personal/gorm-driver-oracle v0.0.0-20200830154622-da7a71b7196d
	gorm.io/driver/mysql v1.0.0
	gorm.io/driver/postgres v1.0.0
	gorm.io/driver/sqlite v1.1.0
	gorm.io/driver/sqlserver v1.0.1
	gorm.io/gorm v1.9.19
)

replace gorm.io/gorm => ../
