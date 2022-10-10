module gorm.io/gorm/tests

go 1.16

require (
	github.com/golang-sql/civil v0.0.0-20220223132316-b832511892a9 // indirect
	github.com/google/uuid v1.3.0
	github.com/jinzhu/now v1.1.5
	github.com/lib/pq v1.10.7
	golang.org/x/crypto v0.0.0-20221005025214-4161e89ecf1b // indirect
	gorm.io/driver/mysql v1.4.1
	gorm.io/driver/postgres v1.4.4
	gorm.io/driver/sqlite v1.4.2
	gorm.io/driver/sqlserver v1.4.0
	gorm.io/gorm v1.24.0
)

replace gorm.io/gorm => ../
