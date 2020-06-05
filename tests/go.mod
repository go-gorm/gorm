module gorm.io/gorm/tests

go 1.14

require (
	github.com/jinzhu/now v1.1.1
	gorm.io/driver/mysql v0.0.0-20200602015408-0407d0c21cf0
	gorm.io/driver/postgres v0.0.0-20200602015520-15fcc29eb286
	gorm.io/driver/sqlite v1.0.0
	gorm.io/driver/sqlserver v0.0.0-20200602144728-79c224f6c1a2
	gorm.io/gorm v0.0.0-00010101000000-000000000000
)

replace gorm.io/gorm => ../
