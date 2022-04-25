module gorm.io/gorm/tests

go 1.14

require (
	github.com/golang-sql/civil v0.0.0-20220223132316-b832511892a9 // indirect
	github.com/google/uuid v1.3.0
	github.com/jinzhu/now v1.1.5
	github.com/lib/pq v1.10.5
	golang.org/x/crypto v0.0.0-20220411220226-7b82a4e95df4 // indirect
	gorm.io/driver/mysql v1.3.3
	gorm.io/driver/postgres v1.3.4
	gorm.io/driver/sqlite v1.3.2
	gorm.io/driver/sqlserver v1.3.2
	gorm.io/gorm v1.23.4
)

replace gorm.io/gorm => ../
