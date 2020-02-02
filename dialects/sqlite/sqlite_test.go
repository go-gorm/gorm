package sqlite_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jinzhu/gorm"
)

var DB *gorm.DB

func TestOpen(t *testing.T) {
	db, err = gorm.Open("sqlite3", filepath.Join(os.TempDir(), "gorm.db"))
}
