module gorm.io/gorm/tests

go 1.16

require (
	github.com/google/uuid v1.3.0
	github.com/jinzhu/now v1.1.5
	github.com/lib/pq v1.10.9
	gorm.io/driver/mysql v1.5.2-0.20230612053416-48b6526a21f0
	gorm.io/driver/postgres v1.5.3-0.20230607070428-18bc84b75196
	gorm.io/driver/sqlite v1.5.2
	gorm.io/driver/sqlserver v1.5.2-0.20230613072041-6e2cde390b0a
	gorm.io/gorm v1.25.2-0.20230610234218-206613868439
)

replace gorm.io/gorm => ../
