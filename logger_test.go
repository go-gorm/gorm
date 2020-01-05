package gorm_test

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/jinzhu/gorm"
)

func TestNoColor(t *testing.T) {
	os.Setenv("NO_COLOR", "1")
	gorm.NoColor()
	l := gorm.Logger{log.New(os.Stdout, "\r\n", 0)}
	l.Print("info", "[info] NO_COLOR log test")
	os.Setenv("NO_COLOR", "")
	gorm.NoColor()
}

func TestSQLLog(t *testing.T) {
	l := gorm.Logger{log.New(os.Stdout, "\r\n", 0)}
	l.Print("sql", "SQL log test", time.Duration(9990000), "sql: log format test type ", []interface{}{"cover"}, int64(0))
}
