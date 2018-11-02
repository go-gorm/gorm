package gorm_test

import (
	"testing"
	"os"
	"crypto/md5"
	"fmt"
	"time"
	"github.com/jinzhu/gorm"
	"log"
)

func TestLoggerCtx(t *testing.T) {
	DB.SetLogger(gorm.Logger{log.New(os.Stdout, "\r\n", 0)})
	if debug := os.Getenv("DEBUG"); debug == "true" {
		DB.LogMode(true)
	} else if debug == "false" {
		DB.LogMode(false)
	}

	if logCtx := os.Getenv("LOGCTX"); logCtx == "true" {
		DB.LogCtx(true)
	} else if logCtx == "false" {
		DB.LogCtx(false)
	}

	i := 0
	for i < 10 {
		i++

		//Generating context information
		unixTime := fmt.Sprint(time.Now().Unix())
		traceId := fmt.Sprintf("%x", md5.Sum([]byte(unixTime)))
		ctxInfo:= "\n[context] trace_id="+traceId
		builder := DB.SetCtx(ctxInfo)
		if i > 5 {
			builder = builder.Where("Age = ?", i)
		} else {
			builder = builder.Where("Name = ?", i)
		}

		if builder.Find(&User{}).Error == nil {
			t.Errorf("Should got error with invalid SQL")
		}

		//Verify context information
		ctxTmp,_:=builder.GetCtx()
		ctxInfo2,_:=ctxTmp.(string)
		if ctxInfo!=ctxInfo2{
			t.Fatal("get context error")
		}
	}
}

