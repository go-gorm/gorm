package utils

import (
	"strings"
	"testing"
)

func TestIsValidDBNameChar(t *testing.T) {
	for _, db := range []string{"db", "dbName", "db_name", "db1", "1dbname", "db$name"} {
		if fields := strings.FieldsFunc(db, IsValidDBNameChar); len(fields) != 1 {
			t.Fatalf("failed to parse db name %v", db)
		}
	}
}

func TestFileWithLineNum(t *testing.T) {
	t.Log("file line with num: ", FileWithLineNum())
}
