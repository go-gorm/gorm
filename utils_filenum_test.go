package gorm

import (
	"log"
	"testing"

	"gorm.io/gorm/utils"
)

func TestFileWithLineNum(t *testing.T) {
	log.Println("FileWithLineNum: ", utils.FileWithLineNum())
}
