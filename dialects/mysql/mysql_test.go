package mysql_test

import (
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/dialects/mysql"
)

func TestOpen(t *testing.T) {
	gorm.Open(mysql.Open("gorm:gorm@tcp(localhost:9910)/gorm?charset=utf8&parseTime=True"), nil)
}
