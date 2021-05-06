module gorm.io/gorm/tests

go 1.14

require (
	github.com/google/uuid v1.1.1
	github.com/jinzhu/now v1.1.2
	github.com/lib/pq v1.6.0
	github.com/stretchr/testify v1.5.1
	gorm.io/driver/mysql v1.0.5
	gorm.io/driver/postgres v1.1.0
	gorm.io/driver/sqlite v1.1.4
	gorm.io/driver/sqlserver v1.0.7
	gorm.io/gorm v1.21.9
)

replace gorm.io/gorm => ../
