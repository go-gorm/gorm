module gorm.io/gorm/tests

go 1.14

require (
	github.com/jinzhu/now v1.1.1
	gorm.io/driver/mysql v0.0.0-20200602015408-0407d0c21cf0
	gorm.io/driver/postgres v0.0.0-20200602015520-15fcc29eb286
	gorm.io/driver/sqlite v0.0.0-20200602015323-284b563f81c8
	gorm.io/driver/sqlserver v0.0.0-20200602015206-ef9f739c6a30
	gorm.io/gorm v1.9.12
)

replace gorm.io/gorm => ../
