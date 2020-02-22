package mysql_test

import (
	"fmt"
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/dialects/mysql"
	"github.com/jinzhu/gorm/tests"
)

func TestOpen(t *testing.T) {
	gorm.Open(mysql.Open("gorm:gorm@tcp(localhost:9910)/gorm?charset=utf8&parseTime=True"), nil)
}

var (
	DB  *gorm.DB
	err error
)

func init() {
	if DB, err = gorm.Open(mysql.Open("gorm:gorm@tcp(localhost:9910)/gorm?charset=utf8&parseTime=True"), &gorm.Config{}); err != nil {
		panic(fmt.Sprintf("failed to initialize database, got error %v", err))
	}
}

func TestCURD(t *testing.T) {
	tests.RunTestsSuit(t, DB)
}

func TestMigrate(t *testing.T) {
	tests.TestMigrate(t, DB)
}
